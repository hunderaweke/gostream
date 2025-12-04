package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	videopb "github.com/hunderaweke/gostream/gen/go/video"
	"github.com/hunderaweke/gostream/internal/database"
	"github.com/hunderaweke/gostream/internal/domain"
	"github.com/hunderaweke/gostream/internal/queue"
	"github.com/hunderaweke/gostream/pkg/utils"
)

type videoService struct {
	videopb.UnimplementedVideoServiceServer
	usecase     domain.VideoService
	minioClient *database.MinioClient
	rmq         *queue.RabbitMQ
}

func NewVideoService(minioClient *database.MinioClient, usecase domain.VideoService, rmq *queue.RabbitMQ) videopb.VideoServiceServer {
	return &videoService{usecase: usecase, minioClient: minioClient, rmq: rmq}
}
func (s *videoService) CreateVideo(ctx context.Context, req *videopb.CreateVideoRequest) (*videopb.CreateVideoResponse, error) {
	userId, err := utils.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding user id: %v", err)
	}
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return nil, fmt.Errorf("error parsing user id: %v", err)
	}
	video := &domain.Video{
		UserID:      userUUID,
		Title:       req.Title,
		Description: req.Description,
		FileName:    req.FileExtension,
	}
	video, err = s.usecase.CreateVideo(video)
	if err != nil {
		return nil, fmt.Errorf("error creating video: %v", err)
	}
	objectName := fmt.Sprintf("%s.%s", video.ID.String(), req.FileExtension)
	uploadUrl, err := s.minioClient.GeneratePresignedURL(objectName, 10*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error creating upload url: %v", err)
	}
	return &videopb.CreateVideoResponse{VideoId: video.ID.String(), UploadUrl: uploadUrl}, nil
}
func (s *videoService) CompleteUpload(ctx context.Context, req *videopb.CompleteUploadRequest) (*videopb.CompleteUploadResponse, error) {
	userId, err := utils.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding user id: %v", err)
	}
	if err := s.usecase.CompleteUpload(userId, req.GetVideoId()); err != nil {
		return nil, err
	}
	return &videopb.CompleteUploadResponse{
		VideoId: req.VideoId,
		Status:  string(domain.VideoStatusProcessing),
	}, nil
}
func convertToGrpcVideo(v domain.Video) *videopb.Video {
	return &videopb.Video{
		Id:           v.ID.String(),
		Title:        v.Title,
		Description:  v.Description,
		HlsUrl:       v.HLSUrl,
		ThumbnailUrl: v.ThumbnailUrl,
		Status:       string(v.Status),
		Views:        v.Views,
		CreatedAt:    v.CreatedAt.Format(time.RFC3339),
	}
}
func convertToGrpcVideos(videos []domain.Video) []*videopb.Video {
	result := make([]*videopb.Video, len(videos))
	for i, v := range videos {
		result[i] = convertToGrpcVideo(v)
	}
	return result
}

func (s *videoService) GetVideos(ctx context.Context, req *videopb.GetVideosRequest) (*videopb.GetVideosResponse, error) {
	userId, err := utils.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding user id: %v", err)
	}
	resp, err := s.usecase.Find(domain.VideoFetchOptions{
		UserID: userId,
		Status: domain.VideoStatus(req.GetStatus()),
		BaseFetchOptions: domain.BaseFetchOptions{
			Page:  int(req.GetPage()),
			Limit: int(req.GetLimit()),
			Query: req.GetQuery(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting multiple videos: %v", err)
	}
	return &videopb.GetVideosResponse{
		Videos: convertToGrpcVideos(resp.Videos),
		Page:   int32(resp.Page),
		Limit:  int32(resp.Limit),
		Total:  resp.Total,
	}, nil
}
