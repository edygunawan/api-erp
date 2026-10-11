package userrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/edygunawan/go-rest-api/internal/repository"
	"github.com/edygunawan/go-rest-api/internal/usecase/user"
	"github.com/jmoiron/sqlx"
)

type postgresUserModel struct {
	ID string `db:"id"`

	FullName string `db:"full_name"`
	Username string `db:"username"`
	Password string `db:"password"`

	Email *string `db:"email"`
	Phone *string `db:"phone"`

	Status domain.UserStatus `db:"status"`

	LastLoginAt *time.Time `db:"last_login_at"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	CreatedBy *string `db:"created_by"`
	UpdatedBy *string `db:"updated_by"`
}

func (m postgresUserModel) ToDomainUser() domain.User {
	return domain.User{
		ID:          m.ID,
		FullName:    m.FullName,
		Username:    m.Username,
		Password:    m.Password,
		Email:       m.Email,
		Phone:       m.Phone,
		Status:      m.Status,
		LastLoginAt: m.LastLoginAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		CreatedBy:   m.CreatedBy,
		UpdatedBy:   m.UpdatedBy,
	}
}

var _ user.UserRepository = (*PostgresUserRepository)(nil)

type PostgresUserRepository struct {
	db  *sql.DB
	dbx *sqlx.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db:  db,
		dbx: sqlx.NewDb(db, "pgx"),
	}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Normalize(err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO users (
			id,
			full_name,
			username,
			password,
			email,
			phone,
			status,
			created_by,
			updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		user.ID,
		user.FullName,
		user.Username,
		user.Password,
		user.Email,
		user.Phone,
		user.Status,
		user.CreatedBy,
		user.UpdatedBy,
	)
	if err := repository.CheckResult(result, err); err != nil {
		return err
	}

	if err := insertUserRoles(
		ctx,
		tx,
		user.ID,
		user.Roles,
		user.CreatedBy,
	); err != nil {
		return repository.Normalize(err)
	}

	return repository.Normalize(tx.Commit())
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User, passwordChanged bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Normalize(err)
	}
	defer tx.Rollback()

	const query = `
		UPDATE users
		SET
			full_name = $1,
			username = $2,
			password = CASE WHEN $9 THEN $3 ELSE password END,
			email = $4,
			phone = $5,
			status = $6,
			updated_at = NOW(),
			updated_by = $7
		WHERE id = $8
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		user.FullName,
		user.Username,
		user.Password,
		user.Email,
		user.Phone,
		user.Status,
		user.UpdatedBy,
		user.ID,
		passwordChanged,
	)
	if err := repository.CheckResult(result, err); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,NOW()) WHERE user_id=$1 AND ( $2='INACTIVE' OR token_version <> (SELECT auth_version FROM users WHERE id=$1) )`, user.ID, user.Status)
	if err != nil {
		return repository.Normalize(err)
	}
	if err := replaceUserRoles(
		ctx,
		tx,
		user.ID,
		user.Roles,
		user.UpdatedBy,
	); err != nil {
		return repository.Normalize(err)
	}

	return repository.Normalize(tx.Commit())
}

func (r *PostgresUserRepository) FindByID(
	ctx context.Context,
	userID string,
) (*domain.User, error) {
	const query = `
		SELECT
			id,
			full_name,
			username,
			password,
			email,
			phone,
			status,
			last_login_at,
			created_at,
			updated_at,
			created_by,
			updated_by
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	var model postgresUserModel

	err := r.dbx.GetContext(ctx, &model, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.Normalize(err)
		}
		return nil, repository.Normalize(err)
	}

	user := model.ToDomainUser()
	users := []domain.User{user}
	if err := attachRoles(ctx, r.dbx, users); err != nil {
		return nil, repository.Normalize(err)
	}
	user = users[0]

	return &user, nil
}

func (r *PostgresUserRepository) List(ctx context.Context, options query.Options) ([]domain.User, int64, error) {
	plan, err := sqlbuilder.New(options, userListDefinition)
	if err != nil {
		return nil, 0, repository.Normalize(err)
	}
	where, args := plan.Where()
	var total int64
	if err := r.dbx.GetContext(ctx, &total, r.dbx.Rebind(`SELECT COUNT(*) FROM users`+where), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	suffix, args := plan.Suffix()
	const selectSQL = `SELECT id, full_name, username, email, phone, status, last_login_at, created_at, updated_at, created_by, updated_by FROM users`
	var models []postgresUserModel
	if err := r.dbx.SelectContext(ctx, &models, r.dbx.Rebind(selectSQL+suffix), args...); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	items := make([]domain.User, len(models))
	for i, model := range models {
		items[i] = model.ToDomainUser()
	}
	if err := attachRoles(ctx, r.dbx, items); err != nil {
		return nil, 0, repository.Normalize(err)
	}
	return items, total, nil
}

func replaceUserRoles(
	ctx context.Context,
	tx *sql.Tx,
	userID string,
	roleIDs []string,
	createdBy *string,
) error {
	const deleteQuery = `DELETE FROM user_roles WHERE user_id = $1`

	if _, err := tx.ExecContext(ctx, deleteQuery, userID); err != nil {
		return repository.Normalize(err)
	}

	return insertUserRoles(ctx, tx, userID, roleIDs, createdBy)
}

func insertUserRoles(
	ctx context.Context,
	tx *sql.Tx,
	userID string,
	roleIDs []string,
	createdBy *string,
) error {
	if len(roleIDs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO user_roles (
			user_id,
			role_id,
			created_by
		) VALUES ($1, $2, $3)
	`

	for _, roleID := range roleIDs {
		if _, err := tx.ExecContext(
			ctx,
			query,
			userID,
			roleID,
			createdBy,
		); err != nil {
			return repository.Normalize(err)
		}
	}

	return nil
}
