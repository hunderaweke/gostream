package grpcserver

import (
	"context"

	authpb "github.com/hunderaweke/gostream/gen/go/auth"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/hunderaweke/gostream/pkg/utils"
)

type authService struct {
	authpb.UnimplementedAuthServiceServer
	usecase domain.UserService
}

func NewAuthService(usecase domain.UserService) authpb.AuthServiceServer {
	return &authService{usecase: usecase}
}
func (s *authService) Login(ctx context.Context, credentials *authpb.UserCredentials) (*authpb.AuthorizedUser, error) {
	user, err := s.usecase.Login(credentials.GetUsername(), credentials.GetPassword())
	if err != nil {
		return nil, err
	}
	accessToken, err := utils.GenerateToken(*user, utils.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateToken(*user, utils.RefreshToken)
	if err != nil {
		return nil, err
	}

	return &authpb.AuthorizedUser{Token: &authpb.TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken}, User: &authpb.User{
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Id:        user.ID.String(),
	}}, nil
}
func (s *authService) Validate(ctx context.Context, req *authpb.ValidateRequest) (*authpb.ValidateResponse, error) {
	claims, err := utils.ValidateToken(req.Token, string(utils.AccessToken))
	if err != nil {
		return nil, err
	}
	return &authpb.ValidateResponse{UserId: claims.ID.String()}, nil
}

func (s *authService) Register(ctx context.Context, req *authpb.UserRegisterRequest) (*authpb.User, error) {
	user := domain.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Username:  req.Username,
	}
	createdUser, err := s.usecase.CreateUser(&user)
	if err != nil {
		return nil, err
	}
	return &authpb.User{
		FirstName: createdUser.FirstName,
		LastName:  createdUser.LastName,
		Username:  createdUser.Username,
		Id:        createdUser.ID.String(),
	}, nil
}
