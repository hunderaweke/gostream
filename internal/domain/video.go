package domain

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	VideoStatusPending    VideoStatus = "PENDING"
	VideoStatusProcessing VideoStatus = "PROCESSING"
	VideoStatusReady      VideoStatus = "READY"
	VideoStatusFailed     VideoStatus = "FAILED"
)

type VideoStatus string

type Video struct {
	Model
	Title        string      `gorm:"not null" json:"title" validate:"required,min=1,max=200"`
	Description  string      `json:"description" validate:"omitempty,max=2000"`
	FileName     string      `gorm:"not null" json:"file_name" validate:"required"`
	HLSUrl       string      `json:"hls_url"`
	ThumbnailUrl string      `json:"thumbnail_url"`
	Status       VideoStatus `gorm:"default:'PENDING'" json:"status" validate:"omitempty,oneof=PENDING PROCESSING READY FAILED"`
	UserID       uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	Views        int64       `gorm:"default:0" json:"views"`
}

func (v *Video) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	v.FileName = fmt.Sprintf("%s.%s", v.ID.String(), v.FileName)
	return nil
}

type VideoFetchOptions struct {
	BaseFetchOptions
	UserID string
	Status VideoStatus
}

type MultipleVideoResponse struct {
	Videos []Video `json:"videos"`
	Total  int64   `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}

type VideoRepository interface {
	Create(video *Video) (*Video, error)
	FindByID(id uuid.UUID) (*Video, error)
	Find(opts VideoFetchOptions) ([]Video, int64, error)
	Update(video *Video) error
	Delete(id uuid.UUID) error
	IncrementViews(id uuid.UUID) error
}

type VideoService interface {
	CreateVideo(video *Video) (*Video, error)
	FindByID(id string) (*Video, error)
	Find(opts VideoFetchOptions) (*MultipleVideoResponse, error)
	Update(id string, video *Video) (*Video, error)
	Delete(id string) error
	IncrementViews(id string) error
	CompleteUpload(userID, videoID string) error
	UpdateStatus(videoID string, status VideoStatus) error
}
