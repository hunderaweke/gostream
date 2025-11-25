package database

import (
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func GetMinio() (*minio.Client, error) {
	clnt, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_ID"), os.Getenv("MINIO_SECRET_ACCESS_KEY"), ""),
		Secure: true,
	})
	return clnt, err
}
