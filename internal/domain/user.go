package domain

import (
	"github.com/google/uuid"
)

type User struct {
	Model
	Username  string `gorm:"column:username;uniqueIndex;not null" json:"username" validate:"required,min=3,max=50"`
	Password  string `gorm:"column:password;not null" json:"password" validate:"required,min=6"`
	FirstName string `gorm:"column:first_name" json:"first_name" validate:"omitempty,max=100"`
	LastName  string `gorm:"column:last_name" json:"last_name" validate:"omitempty,max=100"`
}

type UserFetchOptions struct {
	BaseFetchOptions
	// Additional user-specific filters can go here (e.g. Username)
	Username string
}

type UserRepository interface {
	Create(user *User) error
	Delete(id uuid.UUID) error
	Update(user *User) error
	GetByID(id uuid.UUID) (*User, error)
	// GetAll returns a slice of users and the total count (for pagination)
	GetAll(opts UserFetchOptions) ([]User, int64, error)
}
