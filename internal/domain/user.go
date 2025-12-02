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
	GetByUsername(username string) (*User, error)
	GetAll(opts UserFetchOptions) ([]User, int64, error)
}

// UserService defines the application-level behaviour for Users.
// Keep the service contract in the domain so callers can depend on the
// behaviour rather than concrete implementations.
type UserService interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	DeleteUser(id uuid.UUID) error
	GetUserByID(id uuid.UUID) (*User, error)
	ListUsers(opts UserFetchOptions) ([]User, int64, error)
	GetByUsername(username string) (*User, error)
	Login(username, password string) (*User, error)
	ResetPassword(username,newPassword string)error
	ChangePassword(username,oldPassword,newPassword string)error
}
