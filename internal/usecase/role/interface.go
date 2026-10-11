package role

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Usecase interface {
	Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error)
	Update(ctx context.Context, id string, req UpdateRoleRequest) (*RoleResponse, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*RoleResponse, error)
	List(ctx context.Context, options query.Options) (query.Page[RoleResponse], error)
}

type Repository interface {
	Create(ctx context.Context, role *domain.Role) error
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*domain.Role, error)
	List(ctx context.Context, options query.Options) ([]domain.Role, int64, error)
}
