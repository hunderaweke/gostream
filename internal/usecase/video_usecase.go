package usecase

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/hunderaweke/gostream/internal/database"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/hunderaweke/gostream/internal/queue"
)

type videoUsecase struct {
	repo        domain.VideoRepository
	validate    *validator.Validate
	minioClient *database.MinioClient
	rmq         *queue.RabbitMQ
}

func NewVideoUsecase(repo domain.VideoRepository, minioClient *database.MinioClient, rmq *queue.RabbitMQ) domain.VideoService {
	return &videoUsecase{
		repo:        repo,
		validate:    validator.New(),
		minioClient: minioClient,
		rmq:         rmq,
	}
}

func (u *videoUsecase) CreateVideo(video *domain.Video) (*domain.Video, error) {
	// if err := u.validate.Struct(video); err != nil {
	// 	return nil, fmt.Errorf("validation failed: %w", err)
	// }

	if video.Status == "" {
		video.Status = domain.VideoStatusPending
	}

	if video.Status != domain.VideoStatusPending &&
		video.Status != domain.VideoStatusProcessing &&
		video.Status != domain.VideoStatusReady &&
		video.Status != domain.VideoStatusFailed {
		return nil, fmt.Errorf("invalid video status: %s", video.Status)
	}
	if video.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	createdVideo, err := u.repo.Create(video)
	if err != nil {
		return nil, fmt.Errorf("failed to create video: %w", err)
	}
	return createdVideo, nil
}

func (u *videoUsecase) FindByID(id string) (*domain.Video, error) {
	videoID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid video id: %w", err)
	}

	video, err := u.repo.FindByID(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find video: %w", err)
	}

	return video, nil
}

func (u *videoUsecase) Find(opts domain.VideoFetchOptions) (*domain.MultipleVideoResponse, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	if opts.Limit > 100 {
		opts.Limit = 100
	}

	videos, total, err := u.repo.Find(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find videos: %w", err)
	}
	page := opts.Page
	if page == 0 {
		page = 1
	}
	return &domain.MultipleVideoResponse{
		Videos: videos,
		Total:  total,
		Page:   page,
		Limit:  opts.Limit,
	}, nil
}

func (u *videoUsecase) Update(id string, video *domain.Video) (*domain.Video, error) {
	videoID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid video id: %w", err)
	}

	// Fetch existing video
	existing, err := u.repo.FindByID(videoID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	// Update only provided fields
	if video.Title != "" {
		existing.Title = video.Title
	}
	if video.Description != "" {
		existing.Description = video.Description
	}
	if video.HLSUrl != "" {
		existing.HLSUrl = video.HLSUrl
	}
	if video.ThumbnailUrl != "" {
		existing.ThumbnailUrl = video.ThumbnailUrl
	}
	if video.Status != "" {
		// Validate status transition
		if video.Status != domain.VideoStatusPending &&
			video.Status != domain.VideoStatusProcessing &&
			video.Status != domain.VideoStatusReady &&
			video.Status != domain.VideoStatusFailed {
			return nil, fmt.Errorf("invalid video status: %s", video.Status)
		}
		existing.Status = video.Status
	}

	// Validate before update
	if err := u.validate.Struct(existing); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := u.repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	return existing, nil
}

func (u *videoUsecase) Delete(id string) error {
	videoID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid video id: %w", err)
	}

	if err := u.repo.Delete(videoID); err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	return nil
}

func (u *videoUsecase) IncrementViews(id string) error {
	videoID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid video id: %w", err)
	}

	if err := u.repo.IncrementViews(videoID); err != nil {
		return fmt.Errorf("failed to increment views: %w", err)
	}

	return nil
}
func (u *videoUsecase) CompleteUpload(userID, videoID string) error {
	video, err := u.FindByID(videoID)
	if err != nil {
		return err
	}
	if video.UserID.String() != userID {
		return fmt.Errorf("video does not belong to the current user")
	}
	ctx := context.Background()
	_, err = u.minioClient.Client.StatObject(ctx, u.minioClient.Bucket, video.FileName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("video file not found in storage (did you upload it?): %w", err)
	}
	if err := u.UpdateStatus(videoID, domain.VideoStatusProcessing); err != nil {
		return fmt.Errorf("error updating the video status: %w", err)
	}
	if err := u.rmq.PublishVideoUploaded(videoID, video.FileName); err != nil {
		return fmt.Errorf("failed to add to video encoding queue: %w", err)
	}
	return nil
}

func (u *videoUsecase) UpdateStatus(videoID string, status domain.VideoStatus) error {
	_, err := u.Update(videoID, &domain.Video{Status: status})
	return err
}
