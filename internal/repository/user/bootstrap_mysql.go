package userrepo

import (
	"context"
	"database/sql"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/repository"
	"github.com/edygunawan/golang-pkgs/errors"
)

// BootstrapAdmin is an explicit operator action; it never overwrites credentials.
func (r *MySQLUserRepository) BootstrapAdmin(ctx context.Context, user *domain.User) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// The seeded ADMIN row lock below serializes concurrent bootstrap requests.
	var roleID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM roles WHERE code='ADMIN' AND is_system FOR UPDATE`).Scan(&roleID); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New(errors.Invalid, "seeded ADMIN role missing; run migrations first")
		}
		return "", repository.Normalize(err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,username,email,status FROM users WHERE lower(username)=lower(?) OR (? IS NOT NULL AND lower(email)=lower(?)) FOR UPDATE`, user.Username, user.Email, user.Email)
	if err != nil {
		return "", repository.Normalize(err)
	}
	count := 0
	for rows.Next() {
		var id, username string
		var email *string
		var status domain.UserStatus
		if err := rows.Scan(&id, &username, &email, &status); err != nil {
			_ = rows.Close()
			return "", err
		}
		count++
		if !strings.EqualFold(username, user.Username) || status != domain.UserStatusActive || (user.Email != nil && (email == nil || !strings.EqualFold(*email, *user.Email))) {
			_ = rows.Close()
			return "", errors.New(errors.Conflict, "existing user does not match the requested active admin")
		}
		user.ID = id
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return "", err
	}
	_ = rows.Close()
	if count > 1 {
		return "", errors.New(errors.Conflict, "admin username and email match different users")
	}
	if count == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO users(id,full_name,username,password,email,status) VALUES(?,?,?,?,?,'ACTIVE')`, user.ID, user.FullName, user.Username, user.Password, user.Email)
		if err != nil {
			return "", repository.Normalize(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles(user_id,role_id) VALUES(?,?) ON DUPLICATE KEY UPDATE role_id=VALUES(role_id)`, user.ID, roleID); err != nil {
		return "", repository.Normalize(err)
	}
	return user.ID, repository.Normalize(tx.Commit())
}
