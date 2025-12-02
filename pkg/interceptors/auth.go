package interceptors

import (
	"context"

	authpb "github.com/hunderaweke/gostream/gen/go/auth"
	"github.com/hunderaweke/gostream/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	authClient authpb.AuthServiceClient
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if info.FullMethod == "/auth.AuthService/login" {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadeta is not provided")
		}
		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token not found")
		}
		token := values[0]
		res, err := i.authClient.Validate(ctx, &authpb.ValidateRequest{Token: token})
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "token is invalid")
		}
		newCtx := utils.SetUserID(ctx, res.UserId)
		return handler(newCtx, nil)
	}
}
