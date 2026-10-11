package settinggroup

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Usecase interface {
	Create(context.Context, CreateSettingGroupRequest) (*SettingGroupResponse, error)
	Update(context.Context, string, UpdateSettingGroupRequest) (*SettingGroupResponse, error)
	Delete(context.Context, string) error
	GetByID(context.Context, string) (*SettingGroupResponse, error)
	List(context.Context, query.Options) (query.Page[SettingGroupResponse], error)
}
type Repository interface {
	Create(context.Context, *domain.SettingGroup) error
	Update(context.Context, *domain.SettingGroup) error
	Delete(context.Context, string) error
	FindByID(context.Context, string) (*domain.SettingGroup, error)
	List(context.Context, query.Options) ([]domain.SettingGroup, int64, error)
}
