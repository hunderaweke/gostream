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
	"github.com/hunderaweke/gostream/internal/database"
	grpcserver "github.com/hunderaweke/gostream/internal/grpc_server"
	"github.com/hunderaweke/gostream/internal/repository"
	"github.com/hunderaweke/gostream/internal/usecase"
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
	db, err := database.GetPostgresDB()
	if err != nil {
		log.Fatalf("error creating postgres connection: %v", err)
	}
	usecase := usecase.NewUserUsecase(repository.NewUserRepository(db))
	// err = usecase.CreateUser(&domain.User{
	// 	Username:  "hundera",
	// 	FirstName: "Hundera",
	// 	LastName:  "Awoke",
	// 	Password:  "password",
	// })
	// if err != nil {
	// 	log.Printf("error creatin the user: %v", err)
	// }
	authService := grpcserver.NewAuthService(usecase)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("error creating tcp server: %v", err)
	}
	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, authService)
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
	if err = authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, ":50051", opts); err != nil {
		log.Fatalf("error registering handlers: %v", err)
	}
	httpServer := http.Server{
		Addr:    ":8080",
		Handler: allowCORS(mux),
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("HTTP Gateway listening to the port :8080")
		if err := httpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("failed to listen to HTTP Server: %w", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case err := <-errChan:
			log.Printf("shutdown triggered by server error: %v", err)
		case sig := <-sigCh:
			log.Printf("shutdown triggeted by signal: %v", sig)
		}
		cancel()
		go func() {
			grpcServer.GracefulStop()
		}()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		if err := httpServer.Shutdown(shutCtx); err != nil {
			log.Printf("http server shutdown error: %v", err)
		}
	}()
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
