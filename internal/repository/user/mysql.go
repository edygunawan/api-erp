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

type mysqlUserModel struct {
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

func (m mysqlUserModel) ToDomainUser() domain.User {
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

var _ user.UserRepository = (*MySQLUserRepository)(nil)

type MySQLUserRepository struct {
	db  *sql.DB
	dbx *sqlx.DB
}

func NewMySQLUserRepository(db *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{
		db:  db,
		dbx: sqlx.NewDb(db, "mysql"),
	}
}

func (r *MySQLUserRepository) Create(ctx context.Context, user *domain.User) error {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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

	if err := insertMySQLUserRoles(
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

func (r *MySQLUserRepository) Update(ctx context.Context, user *domain.User, passwordChanged bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Normalize(err)
	}
	defer tx.Rollback()
	var password any
	if passwordChanged {
		password = user.Password
	}

	const query = `
		UPDATE users
		SET
			full_name = ?,
			username = ?,
			password = COALESCE(?, password),
			email = ?,
			phone = ?,
			status = ?,
			updated_at = UTC_TIMESTAMP(6),
			updated_by = ?
		WHERE id = ?
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		user.FullName,
		user.Username,
		password,
		user.Email,
		user.Phone,
		user.Status,
		user.UpdatedBy,
		user.ID,
	)
	if err := repository.CheckResult(result, err); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE refresh_sessions s JOIN users u ON u.id=s.user_id SET s.revoked_at=COALESCE(s.revoked_at,UTC_TIMESTAMP(6)),s.updated_at=UTC_TIMESTAMP(6) WHERE s.user_id=? AND (u.status='INACTIVE' OR s.token_version<>u.auth_version)`, user.ID)
	if err != nil {
		return repository.Normalize(err)
	}
	if err := replaceMySQLUserRoles(
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

func (r *MySQLUserRepository) FindByID(
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
		WHERE id = ?
		LIMIT 1
	`

	var model mysqlUserModel

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

func (r *MySQLUserRepository) List(ctx context.Context, options query.Options) ([]domain.User, int64, error) {
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
	var models []mysqlUserModel
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

func replaceMySQLUserRoles(
	ctx context.Context,
	tx *sql.Tx,
	userID string,
	roleIDs []string,
	createdBy *string,
) error {
	const deleteQuery = `DELETE FROM user_roles WHERE user_id = ?`

	if _, err := tx.ExecContext(ctx, deleteQuery, userID); err != nil {
		return repository.Normalize(err)
	}

	return insertMySQLUserRoles(ctx, tx, userID, roleIDs, createdBy)
}

func insertMySQLUserRoles(
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
		) VALUES (?, ?, ?)
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
