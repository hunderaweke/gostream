package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authpb "github.com/hunderaweke/gostream/gen/go/auth"
	videopb "github.com/hunderaweke/gostream/gen/go/video"
	"github.com/hunderaweke/gostream/internal/database"
	grpcserver "github.com/hunderaweke/gostream/internal/grpc_server"
	"github.com/hunderaweke/gostream/internal/queue"
	"github.com/hunderaweke/gostream/internal/repository"
	"github.com/hunderaweke/gostream/internal/server/handlers"
	"github.com/hunderaweke/gostream/internal/usecase"
	"github.com/hunderaweke/gostream/pkg/interceptors"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	var wg sync.WaitGroup
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
	minioClient, err := database.NewMinioClient("gostream")
	if err != nil {
		log.Fatalf("error creating minio client: %v", err)
	}
	db, err := database.GetPostgresDB()
	if err != nil {
		log.Fatalf("error creating postgres connection: %v", err)
	}
	authUsecase := usecase.NewUserUsecase(repository.NewUserRepository(db))
	rmq, err := queue.NewRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer rmq.Close()
	videoUsecase := usecase.NewVideoUsecase(repository.NewVideoRepository(db), minioClient, rmq)
	authService := grpcserver.NewAuthService(authUsecase)
	videoService := grpcserver.NewVideoService(minioClient, videoUsecase, rmq)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("error creating tcp server: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.NewAuthInterceptor().Unary()),
	)
	authpb.RegisterAuthServiceServer(grpcServer, authService)
	videopb.RegisterVideoServiceServer(grpcServer, videoService)
	errChan := make(chan error, 2)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Auth service has started running on port :50051....")
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("error serving gRPC: %w", err)
		}
	}()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		runtime.WithErrorHandler(customErrorHandler),
	)
	rootMux := http.NewServeMux()
	rootMux.Handle("/", mux)
	rootMux.HandleFunc("/v1/stream/", handlers.SecureStreamHandler(minioClient, videoUsecase))
	if err = authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, ":50051", opts); err != nil {
		log.Fatalf("error registering auth handlers: %v", err)
	}
	if err = videopb.RegisterVideoServiceHandlerFromEndpoint(ctx, mux, ":50051", opts); err != nil {
		log.Fatalf("error registering video handlers: %v", err)
	}
	httpServer := http.Server{
		Addr:    ":8080",
		Handler: allowCORS(rootMux),
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("HTTP Gateway listening to the port :8080")
		if err := httpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("failed to listen to HTTP Server: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := rmq.ConsumeVideoQueue(ctx, minioClient, videoUsecase); err != nil {
			if ctx.Err() != nil {
				errChan <- err
			}
			return
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errChan:
		log.Printf("shutdown triggered by server error: %v", err)
	case sig := <-sigCh:
		log.Printf("shutdown triggered by signal: %v", sig)
	}
	cancel()
	go func() {
		grpcServer.GracefulStop()
	}()
	rmq.Close()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := httpServer.Shutdown(shutCtx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}
	wg.Wait()
	log.Printf("servers stopped, exiting")
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
func customErrorHandler(ctx context.Context, sm *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	s := status.Convert(err)
	httpStatusCode := runtime.HTTPStatusFromCode(s.Code())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	response := map[string]string{
		"error": s.Message(),
	}
	json.NewEncoder(w).Encode(response)
}
