package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
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
	"github.com/hunderaweke/gostream/internal/usecase"
	"github.com/hunderaweke/gostream/pkg/interceptors"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
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
	// err = usecase.CreateUser(&domain.User{
	// 	Username:  "hundera",
	// 	FirstName: "Hundera",
	// 	LastName:  "Awoke",
	// 	Password:  "password",
	// })
	// if err != nil {
	// 	log.Printf("error creatin the user: %v", err)
	// }
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
	if err = authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, ":50051", opts); err != nil {
		log.Fatalf("error registering auth handlers: %v", err)
	}
	if err = videopb.RegisterVideoServiceHandlerFromEndpoint(ctx, mux, ":50051", opts); err != nil {
		log.Fatalf("error registering video handlers: %v", err)
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
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		msgs, err := rmq.Channel.Consume(
			"video_encoding_queue",
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			errChan <- err
			return
		}
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var job queue.VideoMessage
			if err := json.Unmarshal(d.Body, &job); err != nil {
				log.Printf("Error decoding message: %v", err)
				d.Nack(false, false) // Drop bad message
				continue
			}

			if err := processVideo(minioClient, job); err != nil {
				d.Nack(false, false) // Requeue logic (simplistic)
				errChan <- fmt.Errorf("❌ Job Failed: %v", err)
			} else {
				d.Ack(false)
			}
		}
		sig := <-sigCh
		if sig != nil {
			log.Printf("shutdown triggeted by signal: %v", sig)
			return
		}
	}()
	wg.Wait()
	log.Printf("servers stopped, exiting")
}
func processVideo(minioClient *database.MinioClient, job queue.VideoMessage) error {
	// 1. Create Temp Directory
	tempDir := filepath.Join(os.TempDir(), "transcoder", job.VideoID)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir) // Cleanup

	localInput := filepath.Join(tempDir, "input.mp4")

	// 2. Download Raw Video
	log.Printf("Downloading raw video %s...", job.FilePath)
	// Note: You need to implement DownloadFile in your minio.go helper or use FGetObject directly
	// For now, assuming you add this helper or use the raw client:
	err := minioClient.Client.FGetObject(context.Background(), minioClient.Bucket, job.FilePath, localInput, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// 3. Run FFmpeg (Convert to HLS)
	outputPlaylist := filepath.Join(tempDir, "index.m3u8")
	cmd := exec.Command("ffmpeg",
		"-i", localInput,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(tempDir, "segment_%03d.ts"),
		"-start_number", "0",
		outputPlaylist,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %s", string(output))
	}

	// 4. Upload HLS Files to Public Bucket
	// Iterate over the generated files (.m3u8 and .ts)
	files, _ := os.ReadDir(tempDir)
	for _, f := range files {
		if f.Name() == "input.mp4" {
			continue
		}

		localPath := filepath.Join(tempDir, f.Name())
		remotePath := fmt.Sprintf("%s/%s", job.VideoID, f.Name())

		contentType := "application/octet-stream"
		if strings.HasSuffix(f.Name(), ".m3u8") {
			contentType = "application/x-mpegURL"
		} else if strings.HasSuffix(f.Name(), ".ts") {
			contentType = "video/MP2T"
		}
		ctx := context.Background()
		bucket := "hls-videos"
		exists, errBucketExists := minioClient.Client.BucketExists(ctx, bucket)
		if errBucketExists == nil && !exists {
			log.Printf("bucket do not exist creating it ... %v", bucket)
			if err := minioClient.Client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}
		} else if errBucketExists != nil {
			return fmt.Errorf("failed to check if bucket exists: %w", errBucketExists)
		}
		policy := fmt.Sprintf(`{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {"AWS": ["*"]},
                "Action": ["s3:GetObject"],
                "Resource": ["arn:aws:s3:::%s/*"]
            }
        ]
    	}`, bucket)
		minioClient.Client.SetBucketPolicy(ctx, bucket, policy)
		log.Printf("Uploading %s...", remotePath)
		_, err := minioClient.Client.FPutObject(context.Background(), bucket, remotePath, localPath, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("upload failed for %s: %w", f.Name(), err)
		}
	}
	return nil
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
