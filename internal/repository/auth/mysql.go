package authrepo

import (
	"context"
	"database/sql"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/repository"
	userrepo "github.com/edygunawan/go-rest-api/internal/repository/user"
	"github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/golang-pkgs/errors"
)

type MySQLAuthRepository struct {
	db    *sql.DB
	users *userrepo.MySQLUserRepository
}

func NewMySQLAuthRepository(db *sql.DB) *MySQLAuthRepository {
	return &MySQLAuthRepository{db: db, users: userrepo.NewMySQLUserRepository(db)}
}

var _ auth.Repository = (*MySQLAuthRepository)(nil)

func (r *MySQLAuthRepository) FindByLogin(ctx context.Context, login string) (*domain.User, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(username)=lower(?) OR lower(email)=lower(?) ORDER BY id LIMIT 1`, login, login).Scan(&id)
	if err != nil {
		return nil, repository.Normalize(err)
	}
	return r.FindByID(ctx, id)
}
func (r *MySQLAuthRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := r.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=? ORDER BY r.code`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	user.Roles = []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		user.Roles = append(user.Roles, code)
	}
	return user, rows.Err()
}
func (r *MySQLAuthRepository) CreateSession(ctx context.Context, s auth.Session) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status domain.UserStatus
	var hash string
	if err := tx.QueryRowContext(ctx, `SELECT status,password FROM users WHERE id=? FOR UPDATE`, s.UserID).Scan(&status, &hash); err != nil {
		return repository.Normalize(err)
	}
	if status != domain.UserStatusActive || hash != s.PasswordHash {
		return errors.New(errors.Unauthorized, "invalid credentials or session")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO refresh_sessions(id,user_id,token_hash,expires_at,token_version) SELECT ?,id,?,?,auth_version FROM users WHERE id=?`, s.ID, s.TokenHash, s.ExpiresAt, s.UserID)
	if err != nil {
		return repository.Normalize(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET last_login_at=UTC_TIMESTAMP(6) WHERE id=?`, s.UserID); err != nil {
		return err
	}
	return repository.Normalize(tx.Commit())
}
func (r *MySQLAuthRepository) RotateSession(ctx context.Context, oldHash string, s auth.Session) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Lock user before session, matching password updates and login. This avoids
	// inverse lock ordering with InnoDB and rechecks the current authentication version.
	var status domain.UserStatus
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT status,auth_version FROM users WHERE id=? FOR UPDATE`, s.UserID).Scan(&status, &version); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, repository.Normalize(err)
	}
	if status != domain.UserStatusActive {
		return false, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE refresh_sessions SET token_hash=?,expires_at=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND user_id=? AND token_hash=? AND revoked_at IS NULL AND expires_at>UTC_TIMESTAMP(6) AND token_version=?`, s.TokenHash, s.ExpiresAt, s.ID, s.UserID, oldHash, version)
	if err != nil {
		return false, repository.Normalize(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	return true, repository.Normalize(tx.Commit())
}
func (r *MySQLAuthRepository) SessionActive(ctx context.Context, id, userID string) (bool, error) {
	if !helper.ValidUUID(id) || !helper.ValidUUID(userID) {
		return false, nil
	}
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM refresh_sessions s JOIN users u ON u.id=s.user_id WHERE s.id=? AND s.user_id=? AND s.revoked_at IS NULL AND s.expires_at>UTC_TIMESTAMP(6) AND u.status='ACTIVE' AND u.auth_version=s.token_version)`, id, userID).Scan(&active)
	return active, err
}
func (r *MySQLAuthRepository) RevokeSession(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)),updated_at=UTC_TIMESTAMP(6) WHERE id=? AND user_id=?`, id, userID)
	return repository.Normalize(err)
}
