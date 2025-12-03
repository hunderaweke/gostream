package usecase

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hunderaweke/gostream/internal/domain"
)

type userUsecase struct {
	repo         domain.UserRepository
	validate     *validator.Validate
	passwordCost int
}

func NewUserUsecase(repo domain.UserRepository) domain.UserService {
	return &userUsecase{
		repo:         repo,
		validate:     validator.New(),
		passwordCost: bcrypt.DefaultCost,
	}
}

func (u *userUsecase) CreateUser(user *domain.User) (*domain.User, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	if err := u.validate.Struct(user); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), u.passwordCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	user.Password = string(hashed)

	return u.repo.Create(user)
}

func (u *userUsecase) UpdateUser(user *domain.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

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

func (u *userUsecase) DeleteUser(id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid id")
	}
	if err := u.repo.Delete(id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (u *userUsecase) GetUserByID(id uuid.UUID) (*domain.User, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("invalid id")
	}
	return u.repo.GetByID(id)
}

func (u *userUsecase) ListUsers(opts domain.UserFetchOptions) ([]domain.User, int64, error) {
	if opts.Limit <= 0 && opts.Page <= 0 {
		opts.Limit = 25
	}
	return u.repo.GetAll(opts)
}

func (u *userUsecase) Authenticate(username, password string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password required")
	}
	user, err := u.repo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return user, nil
}

func (u *userUsecase) ChangePassword(userID string, currentPassword, newPassword string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("error parsing user id : %v", err)
	}
	if id == uuid.Nil {
		return fmt.Errorf("invalid id")
	}
	if newPassword == "" {
		return fmt.Errorf("new password required")
	}
	user, err := u.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password incorrect")
	}
	if err := u.validate.Var(newPassword, "min=6"); err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), u.passwordCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	user.Password = string(hashed)
	if err := u.repo.Update(user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (u *userUsecase) ResetPassword(username, newPassword string) error {
	user, err := u.GetByUsername(username)
	if err != nil {
		return fmt.Errorf("error getting the user by username: %v", err)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %v", err)
	}
	user.Password = string(hashedPassword)

	return nil
}

func (u *userUsecase) GetByUsername(username string) (*domain.User, error) {
	return u.repo.GetByUsername(username)
}
func (u *userUsecase) Login(username, password string) (*domain.User, error) {
	user, err := u.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}
	return user, nil
}
