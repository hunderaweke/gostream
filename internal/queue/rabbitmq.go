package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hunderaweke/gostream/internal/database"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/minio/minio-go/v7"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	queueName string
	Conn      *amqp.Connection
	Channel   *amqp.Channel
}

func NewRabbitMQ() (*RabbitMQ, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return nil, fmt.Errorf("error connecting to rabbitmq: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error getting connection channel: %v", err)
	}
	_, err = ch.QueueDeclare(
		"video_encoding_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error declaring video encoding queue: %v", err)
	}
	return &RabbitMQ{queueName: "video_encoding_queue", Conn: conn, Channel: ch}, nil
}

type VideoMessage struct {
	VideoID  string `json:"video_id,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

func (r *RabbitMQ) PublishVideoUploaded(videoId, filePath string) error {
	msg := VideoMessage{
		VideoID:  videoId,
		FilePath: filePath,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error parsing video message: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = r.Channel.PublishWithContext(ctx, "", r.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("error publishing the event %v", err)
	}
	return nil
}
func (r *RabbitMQ) ConsumeVideoQueue(ctx context.Context, minioClient *database.MinioClient, usecase domain.VideoService) error {
	msgs, err := r.Channel.Consume(
		r.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			log.Println("context cancelled")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return nil
			}
			log.Printf("Received a message: %s", d.Body)
			var job VideoMessage
			if err := json.Unmarshal(d.Body, &job); err != nil {
				log.Printf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}
			if err := processVideo(minioClient, job); err != nil {
				d.Nack(false, false)
				log.Printf("Job Failed: %v", err)
				usecase.UpdateStatus(job.VideoID, domain.VideoStatusFailed)
				continue
			}
			usecase.UpdateStatus(job.VideoID, domain.VideoStatusReady)
			d.Ack(false)

		}
	}
}

func processVideo(minioClient *database.MinioClient, job VideoMessage) error {
	tempDir := filepath.Join(os.TempDir(), "transcoder", job.VideoID)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)
	localInput := filepath.Join(tempDir, "input.mp4")
	log.Printf("Downloading raw video %s...", job.FilePath)
	err := minioClient.Client.FGetObject(context.Background(), minioClient.Bucket, job.FilePath, localInput, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	outputPlaylist := filepath.Join(tempDir, "index.m3u8")
	cmd := exec.Command("ffmpeg",
		"-i", localInput,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(tempDir, "segment_%03d.ts"),
		"-start_number", "0",
		outputPlaylist,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %s", string(output))
	}
	files, _ := os.ReadDir(tempDir)
	for _, f := range files {
		if f.Name() == "input.mp4" {
			continue
		}
		localPath := filepath.Join(tempDir, f.Name())
		remotePath := fmt.Sprintf("%s/%s", job.VideoID, f.Name())
		contentType := "application/octet-stream"
		if strings.HasSuffix(f.Name(), ".m3u8") {
			contentType = "application/x-mpegURL"
		} else if strings.HasSuffix(f.Name(), ".ts") {
			contentType = "video/MP2T"
		}
		ctx := context.Background()
		bucket := "hls-videos"
		exists, errBucketExists := minioClient.Client.BucketExists(ctx, bucket)
		if errBucketExists == nil && !exists {
			log.Printf("bucket do not exist creating it ... %v", bucket)
			if err := minioClient.Client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}
		} else if errBucketExists != nil {
			return fmt.Errorf("failed to check if bucket exists: %w", errBucketExists)
		}
		log.Printf("Uploading %s...", remotePath)
		_, err := minioClient.Client.FPutObject(context.Background(), bucket, remotePath, localPath, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("upload failed for %s: %w", f.Name(), err)
		}
	}
	return nil
}
func (r *RabbitMQ) Close() {
	r.Channel.Close()
	r.Conn.Close()
}
