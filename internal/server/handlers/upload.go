package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hunderaweke/gostream/internal/database"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/minio/minio-go/v7"
)

func SecureUploadHandler(minioClient *database.MinioClient, videoUsecase domain.VideoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idString := (r.PathValue("video_id"))
		video, err := videoUsecase.FindByID(idString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		info, err := minioClient.Client.PutObject(
			r.Context(),
			minioClient.Bucket,
			video.FileName,
			r.Body,
			r.ContentLength,
			minio.PutObjectOptions{
				ContentType: contentType,
			},
		)
		if err != nil {
			http.Error(w, "upload failed", http.StatusInternalServerError)
			log.Printf("minio upload error: %v", err)
			return
		}
		w.Header().Set("Location", video.FileName)
		w.Header().Set("X-Object-Size", fmt.Sprintf("%d", info.Size))
		w.WriteHeader(http.StatusCreated)
		writeJson(w, map[string]string{"message": "video upload finished"})
	}
}
func writeJson(writer io.Writer, data any) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	writer.Write(encoded)
	return nil
}
