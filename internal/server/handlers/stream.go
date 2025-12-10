package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hunderaweke/gostream/internal/database"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/minio/minio-go/v7"
)

func SecureStreamHandler(minioClient *database.MinioClient, videoService domain.VideoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/stream/")
		path = strings.TrimSuffix(path, "/")

		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		videoID := parts[0]
		fileName := ""
		if len(parts) == 2 && parts[1] != "" {
			fileName = parts[1]
		}

		if fileName == "" && r.URL.Query().Get("info") == "true" {
			video, err := videoService.FindByID(videoID)
			if err != nil {
				http.Error(w, "Video not found", http.StatusNotFound)
				return
			}
			if video.Status != domain.VideoStatusReady {
				http.Error(w, "Video not ready", http.StatusServiceUnavailable)
				return
			}

			obj, err := minioClient.Client.GetObject(r.Context(), "hls-videos", videoID+"/index.m3u8", minio.GetObjectOptions{})
			if err != nil {
				http.Error(w, "Playlist not found", http.StatusNotFound)
				return
			}
			defer obj.Close()

			var rewritten strings.Builder
			scanner := bufio.NewScanner(obj)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".ts") {
					line = fmt.Sprintf("/v1/stream/%s/%s", videoID, line)
				}
				rewritten.WriteString(line + "\n")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"video_id":    video.ID.String(),
				"title":       video.Title,
				"description": video.Description,
				"status":      video.Status,
				"hls_url":     fmt.Sprintf("/v1/stream/%s", videoID),
				"playlist":    rewritten.String(),
			})
			return
		}

		if fileName == "" {
			fileName = "index.m3u8"
		}
		objectPath := fmt.Sprintf("%s/%s", videoID, fileName)
		obj, err := minioClient.Client.GetObject(r.Context(), "hls-videos", objectPath, minio.GetObjectOptions{})
		if err != nil {
			http.Error(w, "Video not found", http.StatusNotFound)
			return
		}
		defer obj.Close()
		stat, err := obj.Stat()
		if err != nil {
			http.Error(w, "Video segment not found", http.StatusNotFound)
			return
		}
		if strings.HasSuffix(fileName, ".m3u8") {
			w.Header().Set("Content-Type", "application/x-mpegURL")
			w.Header().Set("Cache-Control", "no-cache")
			scanner := bufio.NewScanner(obj)
			var rewritten strings.Builder
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".ts") {
					line = fmt.Sprintf("/v1/stream/%s/%s", videoID, line)
				}
				rewritten.WriteString(line + "\n")
			}

			if err := scanner.Err(); err != nil {
				http.Error(w, "Error reading playlist", http.StatusInternalServerError)
				return
			}

			content := rewritten.String()
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Write([]byte(content))
			return
		}

		if strings.HasSuffix(fileName, ".ts") {
			w.Header().Set("Content-Type", "video/MP2T")
			w.Header().Set("Cache-Control", "max-age=3600")
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))
		_, err = io.Copy(w, obj)
		if err != nil {
			log.Println("Stream interrupted:", err)
		}
	}
}
