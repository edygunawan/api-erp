package permission

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Usecase interface {
	Create(ctx context.Context, req CreatePermissionRequest) (*PermissionResponse, error)
	Update(ctx context.Context, id string, req UpdatePermissionRequest) (*PermissionResponse, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*PermissionResponse, error)
	List(ctx context.Context, options query.Options) (query.Page[PermissionResponse], error)
}

type Repository interface {
	Create(ctx context.Context, permission *domain.Permission) error
	Update(ctx context.Context, permission *domain.Permission) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*domain.Permission, error)
	List(ctx context.Context, options query.Options) ([]domain.Permission, int64, error)
}
