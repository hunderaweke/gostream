package main

import (
	"log"
	"net"

	authpb "github.com/hunderaweke/gostream/gen/go/auth"
	"github.com/hunderaweke/gostream/internal/database"
	grpcserver "github.com/hunderaweke/gostream/internal/grpc_server"
	"github.com/hunderaweke/gostream/internal/repository"
	"github.com/hunderaweke/gostream/internal/usecase"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
	db, err := database.GetPostgresDB()
	if err != nil {
		log.Fatalf("error creating postgres connection: %v", err)
	}
	usecase := usecase.NewUserUsecase(repository.NewUserRepository(db))
	authService := grpcserver.NewAuthService(usecase)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("error creating tcp server: %v", err)
	}
	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, authService)
	log.Println("Auth service has started running on port :50051....")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("error listening to the server: %v", err)
	}
}
