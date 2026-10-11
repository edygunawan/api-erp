package user

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Usecase interface {
	Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
	Update(ctx context.Context, id string, req UpdateUserRequest) (*UserResponse, error)
	GetByID(ctx context.Context, id string) (*UserResponse, error)
	List(ctx context.Context, options query.Options) (query.Page[UserResponse], error)
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User, passwordChanged bool) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context, options query.Options) ([]domain.User, int64, error)
}
