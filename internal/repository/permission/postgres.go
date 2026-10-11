package permission

import (
	"context"
	"database/sql"
	"time"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/edygunawan/go-rest-api/internal/repository"
	permissionusecase "github.com/edygunawan/go-rest-api/internal/usecase/permission"
	"github.com/jmoiron/sqlx"
)

type postgresPermissionModel struct {
	ID string `db:"id"`

	Code        string  `db:"code"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	Module      string  `db:"module"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	CreatedBy *string `db:"created_by"`
	UpdatedBy *string `db:"updated_by"`
}

func (m postgresPermissionModel) toDomainPermission() domain.Permission {
	return domain.Permission{
		ID:          m.ID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		Module:      m.Module,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		CreatedBy:   m.CreatedBy,
		UpdatedBy:   m.UpdatedBy,
	}
}

type PostgresPermissionRepository struct {
	db  *sql.DB
	dbx *sqlx.DB
}

func NewPostgresPermissionRepository(db *sql.DB) *PostgresPermissionRepository {
	return &PostgresPermissionRepository{
		db:  db,
		dbx: sqlx.NewDb(db, "pgx"),
	}
}

var _ permissionusecase.Repository = (*PostgresPermissionRepository)(nil)

func (r *PostgresPermissionRepository) Create(
	ctx context.Context,
	permission *domain.Permission,
) error {
	const query = `
		INSERT INTO permissions (
			id,
			code,
			name,
			description,
			module,
			created_by,
			updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		permission.ID,
		permission.Code,
		permission.Name,
		permission.Description,
		permission.Module,
		permission.CreatedBy,
		permission.UpdatedBy,
	)

	return repository.CheckResult(result, err)
}

func (r *PostgresPermissionRepository) Update(
	ctx context.Context,
	permission *domain.Permission,
) error {
	const query = `
		UPDATE permissions
		SET
			code = $1,
			name = $2,
			description = $3,
			module = $4,
			updated_at = NOW(),
			updated_by = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		permission.Code,
		permission.Name,
		permission.Description,
		permission.Module,
		permission.UpdatedBy,
		permission.ID,
	)

	return repository.CheckResult(result, err)
}

func (r *PostgresPermissionRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM permissions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	return repository.CheckResult(result, err)
}

func (r *PostgresPermissionRepository) FindByID(
	ctx context.Context,
	id string,
) (*domain.Permission, error) {
	const query = `
		SELECT
			id,
			code,
			name,
			description,
			module,
			created_at,
			updated_at,
			created_by,
			updated_by
		FROM permissions
		WHERE id = $1
		LIMIT 1
	`

	var model postgresPermissionModel
	err := r.dbx.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.Normalize(err)
		}
		return nil, repository.Normalize(err)
	}

	permission := model.toDomainPermission()
	return &permission, nil
}

func (r *PostgresPermissionRepository) List(ctx context.Context, options query.Options) ([]domain.Permission, int64, error) {
	plan, err := sqlbuilder.New(options, permissionListDefinition)
	if err != nil {
		return nil, 0, repository.Normalize(err)
	}
	where, args := plan.Where()
	var total int64
	if err := r.dbx.GetContext(ctx, &total, r.dbx.Rebind(`SELECT COUNT(*) FROM permissions`+where), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	suffix, args := plan.Suffix()
	const selectSQL = `SELECT id, code, name, description, module, created_at, updated_at, created_by, updated_by FROM permissions`
	var models []postgresPermissionModel
	if err := r.dbx.SelectContext(ctx, &models, r.dbx.Rebind(selectSQL+suffix), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	items := make([]domain.Permission, len(models))
	for i, model := range models {
		items[i] = model.toDomainPermission()
	}

	return items, total, nil
}

// Resolve permission codes using current assignments, including active status.
func (r *PostgresPermissionRepository) GetPermissionsByUserID(ctx context.Context, userID string) ([]string, error) {
	permissions := []string{}
	err := r.dbx.SelectContext(ctx, &permissions, `SELECT DISTINCT p.code FROM permissions p JOIN role_permissions rp ON rp.permission_id=p.id JOIN user_roles ur ON ur.role_id=rp.role_id JOIN users u ON u.id=ur.user_id WHERE u.id=$1 AND u.status='ACTIVE' ORDER BY p.code`, userID)
	return permissions, repository.Normalize(err)
}
