package setting

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Usecase interface {
	GetApplication(context.Context) (*ApplicationSettingsResponse, error)
	SaveApplication(context.Context, ApplicationSettingsRequest) (*ApplicationSettingsResponse, error)
	Create(context.Context, CreateSettingRequest) (*SettingResponse, error)
	Update(context.Context, string, UpdateSettingRequest) (*SettingResponse, error)
	Delete(context.Context, string) error
	GetByID(context.Context, string) (*SettingResponse, error)
	List(context.Context, query.Options) (query.Page[SettingResponse], error)
}
type Repository interface {
	GetApplication(context.Context) (*domain.ApplicationSettings, error)
	SaveApplication(context.Context, domain.ApplicationSettings) error
	Create(context.Context, *domain.Setting) error
	Update(context.Context, *domain.Setting) error
	Delete(context.Context, string) error
	FindByID(context.Context, string) (*domain.Setting, error)
	List(context.Context, query.Options) ([]domain.Setting, int64, error)
}
