package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	Bucket string
	Client *minio.Client
}

func NewMinioClient(bucket string) (*MinioClient, error) {
	minioClient, err := getMinio()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, errBucketExists := minioClient.BucketExists(ctx, bucket)
	if errBucketExists == nil && !exists {
		log.Printf("bucket do not exist creating it ... %v", bucket)
		if err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	} else if errBucketExists != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", errBucketExists)
	}

	return &MinioClient{
		Client: minioClient,
		Bucket: bucket,
	}, nil
}

func getMinio() (*minio.Client, error) {
	clnt, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_ID"), os.Getenv("MINIO_SECRET_ACCESS_KEY"), ""),
		Secure: false,
	})
	return clnt, err
}

func (m *MinioClient) GeneratePresignedURL(objectName string, expiryTime time.Duration) (string, error) {
	ctx := context.Background()
	presignedURL, err := m.Client.PresignedPutObject(ctx, m.Bucket, objectName, expiryTime)
	if err != nil {
		return "", fmt.Errorf("failed to generate presignedUrl: %v", err)
	}
	return presignedURL.String(), nil

}

func (m *MinioClient) GenerateAccessURL(objectName string, expiryTime time.Duration) (string, error) {
	ctx := context.Background()
	presignedURL, err := m.Client.PresignedGetObject(ctx, m.Bucket, objectName, expiryTime, url.Values{})
	if err != nil {
		return "", fmt.Errorf("error creating get link for the object: %v %v", err, objectName)
	}
	return presignedURL.String(), nil
}
