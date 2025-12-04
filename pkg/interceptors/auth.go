package interceptors

import (
	"context"
	"strings"

	"github.com/hunderaweke/gostream/pkg/utils" // Your JWT utils
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
}

func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{}
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {

		if strings.Contains(info.FullMethod, "/Login") ||
			strings.Contains(info.FullMethod, "/Register") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
		}
		accessToken := values[0]
		accessToken = strings.TrimPrefix(accessToken, "Bearer ")

		claims, err := utils.ValidateToken(accessToken, string(utils.AccessToken))
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "access token is invalid: "+err.Error())
		}

		newCtx := utils.SetUserID(ctx, claims.ID.String())

		return handler(newCtx, req)
	}
}
