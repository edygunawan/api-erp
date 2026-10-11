package rolerepo

import (
	"context"
	"database/sql"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/edygunawan/go-rest-api/internal/repository"
	"github.com/edygunawan/go-rest-api/internal/usecase/role"
	"github.com/jmoiron/sqlx"
)

type postgresRoleModel struct {
	ID string `db:"id"`

	Code        string  `db:"code"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	IsSystem    bool    `db:"is_system"`

	CreatedAt sql.NullTime `db:"created_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`

	CreatedBy *string `db:"created_by"`
	UpdatedBy *string `db:"updated_by"`
}

func (m postgresRoleModel) toDomainRole() domain.Role {
	return domain.Role{
		ID:          m.ID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		IsSystem:    m.IsSystem,
		CreatedAt:   m.CreatedAt.Time,
		UpdatedAt:   m.UpdatedAt.Time,
		CreatedBy:   m.CreatedBy,
		UpdatedBy:   m.UpdatedBy,
	}
}

type PostgresRoleRepository struct {
	db  *sql.DB
	dbx *sqlx.DB
}

func NewPostgresRoleRepository(db *sql.DB) role.Repository {
	return &PostgresRoleRepository{
		db:  db,
		dbx: sqlx.NewDb(db, "pgx"),
	}
}

func (r *PostgresRoleRepository) Create(ctx context.Context, role *domain.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Normalize(err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO roles (
			id,
			code,
			name,
			description,
			is_system,
			created_by,
			updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		role.ID,
		role.Code,
		role.Name,
		role.Description,
		role.IsSystem,
		role.CreatedBy,
		role.UpdatedBy,
	)
	if err := repository.CheckResult(result, err); err != nil {
		return err
	}

	if err := insertRolePermissions(
		ctx,
		tx,
		role.ID,
		role.Permissions,
		role.CreatedBy,
	); err != nil {
		return repository.Normalize(err)
	}

	return repository.Normalize(tx.Commit())
}

func (r *PostgresRoleRepository) Update(ctx context.Context, role *domain.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Normalize(err)
	}
	defer tx.Rollback()

	const query = `
		UPDATE roles
		SET
			code = $1,
			name = $2,
			description = $3,
			updated_at = NOW(),
			updated_by = $4
		WHERE id = $5 AND (NOT is_system OR code = $1)
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		role.Code,
		role.Name,
		role.Description,
		role.UpdatedBy,
		role.ID,
	)
	if err := repository.CheckResult(result, err); err != nil {
		return err
	}

	if err := replaceRolePermissions(
		ctx,
		tx,
		role.ID,
		role.Permissions,
		role.UpdatedBy,
	); err != nil {
		return repository.Normalize(err)
	}

	return repository.Normalize(tx.Commit())
}

func (r *PostgresRoleRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM roles WHERE id = $1 AND NOT is_system`

	result, err := r.db.ExecContext(ctx, query, id)
	return repository.CheckResult(result, err)
}

func (r *PostgresRoleRepository) FindByID(ctx context.Context, id string) (*domain.Role, error) {
	const query = `
		SELECT
			id,
			code,
			name,
			description,
			is_system,
			created_at,
			updated_at,
			created_by,
			updated_by
		FROM roles
		WHERE id = $1
		LIMIT 1
	`

	var model postgresRoleModel
	err := r.dbx.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.Normalize(err)
		}
		return nil, repository.Normalize(err)
	}

	role := model.toDomainRole()
	permissionIDs, err := r.findPermissionIDsByRoleID(ctx, role.ID)
	if err != nil {
		return nil, repository.Normalize(err)
	}
	role.Permissions = permissionIDs

	return &role, nil
}

func (r *PostgresRoleRepository) List(ctx context.Context, options query.Options) ([]domain.Role, int64, error) {
	plan, err := sqlbuilder.New(options, roleListDefinition)
	if err != nil {
		return nil, 0, repository.Normalize(err)
	}
	where, args := plan.Where()
	var total int64
	if err := r.dbx.GetContext(ctx, &total, r.dbx.Rebind(`SELECT COUNT(*) FROM roles`+where), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	suffix, args := plan.Suffix()
	const selectSQL = `SELECT id, code, name, description, is_system, created_at, updated_at, created_by, updated_by FROM roles`
	var models []postgresRoleModel
	if err := r.dbx.SelectContext(ctx, &models, r.dbx.Rebind(selectSQL+suffix), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	items := make([]domain.Role, len(models))
	for i, model := range models {
		items[i] = model.toDomainRole()
	}
	if err := attachPermissions(ctx, r.dbx, items); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	return items, total, nil
}

func (r *PostgresRoleRepository) findPermissionIDsByRoleID(
	ctx context.Context,
	roleID string,
) ([]string, error) {
	const query = `
		SELECT permission_id
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY permission_id ASC
	`

	permissionIDs := []string{}
	if err := r.dbx.SelectContext(ctx, &permissionIDs, query, roleID); err != nil {
		return nil, repository.Normalize(err)
	}

	return permissionIDs, nil
}

func replaceRolePermissions(
	ctx context.Context,
	tx *sql.Tx,
	roleID string,
	permissionIDs []string,
	createdBy *string,
) error {
	const deleteQuery = `DELETE FROM role_permissions WHERE role_id = $1`

	if _, err := tx.ExecContext(ctx, deleteQuery, roleID); err != nil {
		return repository.Normalize(err)
	}

	return insertRolePermissions(ctx, tx, roleID, permissionIDs, createdBy)
}

func insertRolePermissions(
	ctx context.Context,
	tx *sql.Tx,
	roleID string,
	permissionIDs []string,
	createdBy *string,
) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO role_permissions (
			role_id,
			permission_id,
			created_by
		) VALUES ($1, $2, $3)
	`

	for _, permissionID := range permissionIDs {
		if _, err := tx.ExecContext(
			ctx,
			query,
			roleID,
			permissionID,
			createdBy,
		); err != nil {
			return repository.Normalize(err)
		}
	}

	return nil
}
