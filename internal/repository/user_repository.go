package repository

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hunderaweke/gostream/internal/domain"
)

type GormUserRepository struct {
	db       *gorm.DB
	validate *validator.Validate
}

func NewUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{
		db:       db,
		validate: validator.New(),
	}
}

func (r *GormUserRepository) Create(user *domain.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	if err := r.validate.Struct(user); err != nil {
		return err
	}

	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *GormUserRepository) Update(user *domain.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}
	if err := r.validate.Struct(user); err != nil {
		return err
	}
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

func (r *GormUserRepository) Delete(id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid id")
	}
	if err := r.db.Delete(&domain.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (r *GormUserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("invalid id")
	}
	var user domain.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *GormUserRepository) GetAll(opts domain.UserFetchOptions) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	tx := r.db.Model(&domain.User{})

	if opts.Username != "" {
		tx = tx.Where("username LIKE ?", "%"+opts.Username+"%")
	}
	if opts.Query != "" {
		like := "%" + opts.Query + "%"
		tx = tx.Where("(username LIKE ? OR first_name LIKE ? OR last_name LIKE ?)", like, like, like)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	if opts.Sort != "" {
		tx = tx.Order(opts.Sort)
	} else {
		tx = tx.Order("created_at desc")
	}

	limit := opts.Limit
	offset := opts.Offset
	if opts.Page > 0 && limit > 0 {
		offset = (opts.Page - 1) * limit
	}
	if limit > 0 {
		tx = tx.Limit(limit).Offset(offset)
	}

	if err := tx.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("finding users: %w", err)
	}
	return users, total, nil
}
