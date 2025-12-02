package utils

import (
	"context"
	"fmt"
)

type contextKey string

const userIDKey contextKey = "user_id"

func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (string, error) {
	var userID string
	val := ctx.Value(userIDKey)
	userID, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("error getting the user_id in context")
	}
	return userID, nil
}
