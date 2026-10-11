package settinggroup

import (
	"context"
	"database/sql"
	"time"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/edygunawan/go-rest-api/internal/repository"
	uc "github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	"github.com/jmoiron/sqlx"
)

// SQL belongs to this feature; sqlx only rebinds placeholders for the selected driver.
type SettingGroupRepository struct {
	dbx          *sqlx.DB
	updatedAtSQL string
}

func NewPostgresSettingGroupRepository(db *sql.DB) *SettingGroupRepository {
	return &SettingGroupRepository{dbx: sqlx.NewDb(db, "pgx"), updatedAtSQL: "NOW()"}
}
func NewMySQLSettingGroupRepository(db *sql.DB) *SettingGroupRepository {
	return &SettingGroupRepository{dbx: sqlx.NewDb(db, "mysql"), updatedAtSQL: "UTC_TIMESTAMP(6)"}
}

var _ uc.Repository = (*SettingGroupRepository)(nil)

type settinggroupModel struct {
	ID        string    `db:"id"`
	Code      string    `db:"code"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r *SettingGroupRepository) Create(ctx context.Context, item *domain.SettingGroup) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("INSERT INTO setting_groups (id,code,name) VALUES (?,?,?)"), item.ID, item.Code, item.Name)
	return repository.CheckResult(result, err)
}
func (r *SettingGroupRepository) Update(ctx context.Context, item *domain.SettingGroup) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("UPDATE setting_groups SET code=?,name=?,updated_at="+r.updatedAtSQL+" WHERE id=?"), item.Code, item.Name, item.ID)
	return repository.CheckResult(result, err)
}
func (r *SettingGroupRepository) Delete(ctx context.Context, id string) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("DELETE FROM setting_groups WHERE id=?"), id)
	return repository.CheckResult(result, err)
}
func (r *SettingGroupRepository) FindByID(ctx context.Context, id string) (*domain.SettingGroup, error) {
	var model settinggroupModel
	if err := r.dbx.GetContext(ctx, &model, r.dbx.Rebind("SELECT id,code,name,created_at,updated_at FROM setting_groups WHERE id=?"), id); err != nil {
		return nil, repository.Normalize(err)
	}
	item := domain.SettingGroup(model)
	return &item, nil
}

var listDefinition = sqlbuilder.Definition{
	SearchColumns: []string{"code", "name"},
	FilterColumns: map[string]string{"code": "code"},
	SortColumns:   map[string]string{"id": "id", "code": "code", "name": "name", "created_at": "created_at", "updated_at": "updated_at"},
	DefaultSort:   []sqlbuilder.Sort{{Key: "code", Order: query.ASC}}, TieBreaker: "id",
}

func (r *SettingGroupRepository) List(ctx context.Context, options query.Options) ([]domain.SettingGroup, int64, error) {
	plan, err := sqlbuilder.New(options, listDefinition)
	if err != nil {
		return nil, 0, repository.Normalize(err)
	}
	where, args := plan.Where()
	var total int64
	if err := r.dbx.GetContext(ctx, &total, r.dbx.Rebind("SELECT COUNT(*) FROM setting_groups"+where), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	suffix, args := plan.Suffix()
	var models []settinggroupModel
	if err := r.dbx.SelectContext(ctx, &models, r.dbx.Rebind("SELECT id,code,name,created_at,updated_at FROM setting_groups"+suffix), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	items := make([]domain.SettingGroup, len(models))
	for i, model := range models {
		items[i] = domain.SettingGroup(model)
	}
	return items, total, nil
}
