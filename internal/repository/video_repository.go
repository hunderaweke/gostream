package repository

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hunderaweke/gostream/internal/domain"
)

type gormVideoRepository struct {
	db       *gorm.DB
	validate *validator.Validate
}

func NewVideoRepository(db *gorm.DB) domain.VideoRepository {
	db.AutoMigrate(&domain.Video{})
	return &gormVideoRepository{
		db:       db,
		validate: validator.New(),
	}
}

func (r *gormVideoRepository) Create(video *domain.Video) (*domain.Video, error) {
	if err := r.validate.Struct(video); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create video: %w", err)
	}

	return video, nil
}

func (r *gormVideoRepository) FindByID(id uuid.UUID) (*domain.Video, error) {
	var video domain.Video
	if err := r.db.Where("id = ?", id).First(&video).Error; err != nil {
		return nil, fmt.Errorf("failed to find video: %w", err)
	}
	return &video, nil
}

func (r *gormVideoRepository) Find(opts domain.VideoFetchOptions) ([]domain.Video, int64, error) {
	var videos []domain.Video
	var total int64

	query := r.db.Model(&domain.Video{})

	// Apply filters
	if opts.UserID != "" {
		userID, err := uuid.Parse(opts.UserID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid user_id: %w", err)
		}
		query = query.Where("user_id = ?", userID)
	}

	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}

	// Apply search query (title or description)
	if opts.Query != "" {
		searchPattern := "%" + opts.Query + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count videos: %w", err)
	}

	// Apply sorting
	if opts.Sort != "" {
		query = query.Order(opts.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	limit := opts.Limit
	if limit == 0 {
		limit = 20 // default
	}

	offset := opts.Offset
	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	query = query.Limit(limit).Offset(offset)

	// Execute query
	if err := query.Find(&videos).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find videos: %w", err)
	}

	return videos, total, nil
}

func (r *gormVideoRepository) Update(video *domain.Video) error {
	if err := r.validate.Struct(video); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.Save(video).Error; err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}

func (r *gormVideoRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&domain.Video{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete video: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("video not found")
	}

	return nil
}

func (r *gormVideoRepository) IncrementViews(id uuid.UUID) error {
	result := r.db.Model(&domain.Video{}).Where("id = ?", id).Update("views", gorm.Expr("views + ?", 1))
	if result.Error != nil {
		return fmt.Errorf("failed to increment views: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("video not found")
	}

	return nil
}
