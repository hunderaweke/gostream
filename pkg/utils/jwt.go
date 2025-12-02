package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hunderaweke/gostream/internal/domain"
)

const (
	AccessToken          = TokenType("access")
	RefreshToken         = TokenType("refresh")
	AccessTokenDuration  = time.Hour * 1
	RefreshTokenDuration = time.Hour * 3
)

type TokenType string
type UserClaims struct {
	jwt.RegisteredClaims
	ID uuid.UUID
	Type TokenType
}

func GenerateToken(user domain.User, tokenType TokenType) (string, error) {
	if tokenType != AccessToken && tokenType != RefreshToken {
		return "", fmt.Errorf("invalid token type: %q", tokenType)
	}
	expiresAt := time.Now().Add(AccessTokenDuration)
	if tokenType == RefreshToken {
		expiresAt = time.Now().Add(RefreshTokenDuration)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{Type:tokenType,ID: user.ID, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)}})
	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", fmt.Errorf("error signing the token %v", err)
	}
	return tokenStr, nil
}
func ValidateToken(tokenStr string, tokenType string) (*UserClaims, error) {
	var claims UserClaims
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing the token string: %v", err)
	}
	if !token.Valid || claims.Type != TokenType(tokenType){
		return nil, fmt.Errorf("invalid token")
	}
	return &claims, nil
}
