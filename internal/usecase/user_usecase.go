package usecase

import (
    "fmt"

    "github.com/go-playground/validator/v10"
    "golang.org/x/crypto/bcrypt"
    "github.com/google/uuid"

    "github.com/hunderaweke/gostream/internal/domain"
)

// UserUsecase contains business logic for users.
type UserUsecase struct {
    repo        domain.UserRepository
    validate    *validator.Validate
    passwordCost int
}

// NewUserUsecase creates a new usecase with sensible defaults.
func NewUserUsecase(repo domain.UserRepository) *UserUsecase {
    return &UserUsecase{
        repo:        repo,
        validate:    validator.New(),
        passwordCost: bcrypt.DefaultCost,
    }
}

// CreateUser validates, hashes password and persists a new user.
func (u *UserUsecase) CreateUser(user *domain.User) error {
    if user == nil {
        return fmt.Errorf("user is nil")
    }

    // validation: username and password required on create
    if err := u.validate.Struct(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // hash password
    hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), u.passwordCost)
    if err != nil {
        return fmt.Errorf("hashing password: %w", err)
    }
    user.Password = string(hashed)

    if err := u.repo.Create(user); err != nil {
        return fmt.Errorf("create user: %w", err)
    }
    return nil
}

// UpdateUser updates allowed fields. If Password is non-empty it will be hashed.
func (u *UserUsecase) UpdateUser(user *domain.User) error {
    if user == nil {
        return fmt.Errorf("user is nil")
    }

    // Validate individual fields that are present.
    if user.Username != "" {
        if err := u.validate.Var(user.Username, "min=3,max=50"); err != nil {
            return fmt.Errorf("invalid username: %w", err)
        }
    }
    if user.FirstName != "" {
        if err := u.validate.Var(user.FirstName, "max=100"); err != nil {
            return fmt.Errorf("invalid first name: %w", err)
        }
    }
    if user.LastName != "" {
        if err := u.validate.Var(user.LastName, "max=100"); err != nil {
            return fmt.Errorf("invalid last name: %w", err)
        }
    }

    if user.Password != "" {
        if err := u.validate.Var(user.Password, "min=6"); err != nil {
            return fmt.Errorf("invalid password: %w", err)
        }
        hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), u.passwordCost)
        if err != nil {
            return fmt.Errorf("hashing password: %w", err)
        }
        user.Password = string(hashed)
    }

    if err := u.repo.Update(user); err != nil {
        return fmt.Errorf("update user: %w", err)
    }
    return nil
}

func (u *UserUsecase) DeleteUser(id uuid.UUID) error {
    if id == uuid.Nil {
        return fmt.Errorf("invalid id")
    }
    if err := u.repo.Delete(id); err != nil {
        return fmt.Errorf("delete user: %w", err)
    }
    return nil
}

func (u *UserUsecase) GetUserByID(id uuid.UUID) (*domain.User, error) {
    if id == uuid.Nil {
        return nil, fmt.Errorf("invalid id")
    }
    return u.repo.GetByID(id)
}

// ListUsers returns a page of users and the total count for pagination.
func (u *UserUsecase) ListUsers(opts domain.UserFetchOptions) ([]domain.User, int64, error) {
    // apply sane defaults
    if opts.Limit <= 0 && opts.Page <= 0 {
        opts.Limit = 25
    }
    return u.repo.GetAll(opts)
}
