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

type PostgresAuthRepository struct {
	db    *sql.DB
	users *userrepo.PostgresUserRepository
}

func NewPostgresAuthRepository(db *sql.DB) *PostgresAuthRepository {
	return &PostgresAuthRepository{db: db, users: userrepo.NewPostgresUserRepository(db)}
}

var _ auth.Repository = (*PostgresAuthRepository)(nil)

func (r *PostgresAuthRepository) FindByLogin(ctx context.Context, login string) (*domain.User, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(username)=lower($1) OR lower(email)=lower($1) ORDER BY id LIMIT 1`, login).Scan(&id)
	if err != nil {
		return nil, repository.Normalize(err)
	}
	return r.FindByID(ctx, id)
}
func (r *PostgresAuthRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := r.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=$1 ORDER BY r.code`, id)
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
func (r *PostgresAuthRepository) CreateSession(ctx context.Context, s auth.Session) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Recheck status under the same transaction as session creation.
	var status domain.UserStatus
	var passwordHash string
	if err := tx.QueryRowContext(ctx, `SELECT status,password FROM users WHERE id=$1 FOR UPDATE`, s.UserID).Scan(&status, &passwordHash); err != nil {
		return repository.Normalize(err)
	}
	if status != domain.UserStatusActive || passwordHash != s.PasswordHash {
		return errors.New(errors.Unauthorized, "invalid credentials or session")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO refresh_sessions(id,user_id,token_hash,expires_at,token_version) SELECT $1,$2,$3,$4,auth_version FROM users WHERE id=$2`, s.ID, s.UserID, s.TokenHash, s.ExpiresAt)
	if err != nil {
		return repository.Normalize(err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE users SET last_login_at=NOW() WHERE id=$1`, s.UserID)
	if err != nil {
		return err
	}
	return repository.Normalize(tx.Commit())
}
func (r *PostgresAuthRepository) RotateSession(ctx context.Context, oldHash string, s auth.Session) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE refresh_sessions SET token_hash=$1,expires_at=$2,updated_at=NOW() WHERE id=$3 AND user_id=$4 AND token_hash=$5 AND revoked_at IS NULL AND expires_at>NOW() AND EXISTS(SELECT 1 FROM users WHERE id=$4 AND status='ACTIVE' AND auth_version=refresh_sessions.token_version)`, s.TokenHash, s.ExpiresAt, s.ID, s.UserID, oldHash)
	if err != nil {
		return false, repository.Normalize(err)
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
func (r *PostgresAuthRepository) SessionActive(ctx context.Context, id, userID string) (bool, error) {
	if !helper.ValidUUID(id) || !helper.ValidUUID(userID) {
		return false, nil
	}
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM refresh_sessions WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at>NOW() AND EXISTS(SELECT 1 FROM users WHERE id=$2 AND status='ACTIVE' AND auth_version=refresh_sessions.token_version))`, id, userID).Scan(&active)
	return active, err
}
func (r *PostgresAuthRepository) RevokeSession(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,NOW()),updated_at=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	return repository.Normalize(err)
}
