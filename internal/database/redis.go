package database

import (
	"fmt"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func GetRedis() (*redis.Client, error) {
	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("error converting redis_db to int: %v", err)
	}
	redis := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	return redis, nil
}
