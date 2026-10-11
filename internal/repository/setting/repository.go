package setting

import (
	"context"
	"database/sql"
	"time"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/edygunawan/go-rest-api/internal/repository"
	uc "github.com/edygunawan/go-rest-api/internal/usecase/setting"
	"github.com/jmoiron/sqlx"
)

// SQL belongs to this feature; sqlx only rebinds placeholders for the selected driver.
type SettingRepository struct {
	dbx          *sqlx.DB
	updatedAtSQL string
}

func NewPostgresSettingRepository(db *sql.DB) *SettingRepository {
	return &SettingRepository{dbx: sqlx.NewDb(db, "pgx"), updatedAtSQL: "NOW()"}
}
func NewMySQLSettingRepository(db *sql.DB) *SettingRepository {
	return &SettingRepository{dbx: sqlx.NewDb(db, "mysql"), updatedAtSQL: "UTC_TIMESTAMP(6)"}
}

// The same many-to-one join supplies relation data and preserves list/count filters.
const settingFrom = " FROM settings s JOIN setting_groups g ON g.id=s.group_id"
const settingSelect = "SELECT s.id,s.group_id,s.setting_key,s.setting_value,s.data_type,s.description,s.created_at,s.updated_at,g.code AS group_code,g.name AS group_name,g.created_at AS group_created_at,g.updated_at AS group_updated_at"

var _ uc.Repository = (*SettingRepository)(nil)

type settingModel struct {
	ID             string    `db:"id"`
	GroupID        string    `db:"group_id"`
	GroupCode      string    `db:"group_code"`
	GroupName      string    `db:"group_name"`
	GroupCreatedAt time.Time `db:"group_created_at"`
	GroupUpdatedAt time.Time `db:"group_updated_at"`
	SettingKey     string    `db:"setting_key"`
	SettingValue   *string   `db:"setting_value"`
	DataType       string    `db:"data_type"`
	Description    *string   `db:"description"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (r *SettingRepository) Create(ctx context.Context, item *domain.Setting) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("INSERT INTO settings (id,group_id,setting_key,setting_value,data_type,description) VALUES (?,?,?,?,?,?)"), item.ID, item.GroupID, item.SettingKey, item.SettingValue, item.DataType, item.Description)
	return repository.CheckResult(result, err)
}
func (r *SettingRepository) Update(ctx context.Context, item *domain.Setting) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("UPDATE settings SET group_id=?,setting_key=?,setting_value=?,data_type=?,description=?,updated_at="+r.updatedAtSQL+" WHERE id=?"), item.GroupID, item.SettingKey, item.SettingValue, item.DataType, item.Description, item.ID)
	return repository.CheckResult(result, err)
}
func (r *SettingRepository) Delete(ctx context.Context, id string) error {
	result, err := r.dbx.ExecContext(ctx, r.dbx.Rebind("DELETE FROM settings WHERE id=?"), id)
	return repository.CheckResult(result, err)
}
func (r *SettingRepository) FindByID(ctx context.Context, id string) (*domain.Setting, error) {
	var model settingModel
	if err := r.dbx.GetContext(ctx, &model, r.dbx.Rebind(settingSelect+settingFrom+" WHERE s.id=?"), id); err != nil {
		return nil, repository.Normalize(err)
	}
	item := model.domain()
	return &item, nil
}

var listDefinition = sqlbuilder.Definition{
	SearchColumns: []string{"s.setting_key", "s.description"},
	FilterColumns: map[string]string{"group_id": "s.group_id", "group_code": "g.code", "setting_key": "s.setting_key", "data_type": "s.data_type"},
	SortColumns:   map[string]string{"id": "s.id", "setting_key": "s.setting_key", "data_type": "s.data_type", "created_at": "s.created_at", "updated_at": "s.updated_at"},
	DefaultSort:   []sqlbuilder.Sort{{Key: "setting_key", Order: query.ASC}}, TieBreaker: "id",
}

func (r *SettingRepository) List(ctx context.Context, options query.Options) ([]domain.Setting, int64, error) {
	plan, err := sqlbuilder.New(options, listDefinition)
	if err != nil {
		return nil, 0, repository.Normalize(err)
	}
	where, args := plan.Where()
	var total int64
	if err := r.dbx.GetContext(ctx, &total, r.dbx.Rebind("SELECT COUNT(*)"+settingFrom+where), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	suffix, args := plan.Suffix()
	var models []settingModel
	if err := r.dbx.SelectContext(ctx, &models, r.dbx.Rebind(settingSelect+settingFrom+suffix), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	items := make([]domain.Setting, len(models))
	for i, model := range models {
		items[i] = model.domain()
	}
	return items, total, nil
}

func (model settingModel) domain() domain.Setting {
	return domain.Setting{
		ID: model.ID, GroupID: model.GroupID,
		Group:      domain.SettingGroup{ID: model.GroupID, Code: model.GroupCode, Name: model.GroupName, CreatedAt: model.GroupCreatedAt, UpdatedAt: model.GroupUpdatedAt},
		SettingKey: model.SettingKey, SettingValue: model.SettingValue, DataType: model.DataType, Description: model.Description, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}
