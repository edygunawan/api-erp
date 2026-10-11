package factory

import (
	"database/sql"
	"fmt"

	"github.com/edygunawan/go-rest-api/internal/database"
	authrepo "github.com/edygunawan/go-rest-api/internal/repository/auth"
	permissionrepo "github.com/edygunawan/go-rest-api/internal/repository/permission"
	rolerepo "github.com/edygunawan/go-rest-api/internal/repository/role"
	settingrepo "github.com/edygunawan/go-rest-api/internal/repository/setting"
	settinggrouprepo "github.com/edygunawan/go-rest-api/internal/repository/settinggroup"
	userrepo "github.com/edygunawan/go-rest-api/internal/repository/user"
	authuc "github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
	"github.com/edygunawan/go-rest-api/internal/usecase/permission"
	"github.com/edygunawan/go-rest-api/internal/usecase/role"
	"github.com/edygunawan/go-rest-api/internal/usecase/setting"
	"github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	"github.com/edygunawan/go-rest-api/internal/usecase/user"
)

type UserRepository interface {
	user.UserRepository
	bootstrap.Repository
}
type PermissionRepository interface {
	permission.Repository
	authuc.PermissionRepository
}
type Repositories struct {
	Setting      setting.Repository
	SettingGroup settinggroup.Repository
	User         UserRepository
	Role         role.Repository
	Permission   PermissionRepository
	Auth         authuc.Repository
}

func New(driver string, db *sql.DB) (Repositories, error) {
	if db == nil {
		return Repositories{}, fmt.Errorf("repository factory requires a database")
	}
	switch driver {
	case database.Postgres:
		return Repositories{
			Setting:      settingrepo.NewPostgresSettingRepository(db),
			SettingGroup: settinggrouprepo.NewPostgresSettingGroupRepository(db),
			User:         userrepo.NewPostgresUserRepository(db),
			Role:         rolerepo.NewPostgresRoleRepository(db),
			Permission:   permissionrepo.NewPostgresPermissionRepository(db),
			Auth:         authrepo.NewPostgresAuthRepository(db),
		}, nil
	case database.MySQL:
		return Repositories{
			Setting:      settingrepo.NewMySQLSettingRepository(db),
			SettingGroup: settinggrouprepo.NewMySQLSettingGroupRepository(db),
			User:         userrepo.NewMySQLUserRepository(db),
			Role:         rolerepo.NewMySQLRoleRepository(db),
			Permission:   permissionrepo.NewMySQLPermissionRepository(db),
			Auth:         authrepo.NewMySQLAuthRepository(db),
		}, nil
	default:
		return Repositories{}, fmt.Errorf("unsupported DB_DRIVER %q", driver)
	}
}
