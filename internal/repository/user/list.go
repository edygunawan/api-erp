package userrepo

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/jmoiron/sqlx"
)

var userListDefinition = sqlbuilder.Definition{
	SearchColumns: []string{"full_name", "username", "email", "phone"},
	FilterColumns: map[string]string{"status": "status"},
	SortColumns:   map[string]string{"id": "id", "full_name": "full_name", "username": "username", "email": "email", "status": "status", "created_at": "created_at", "updated_at": "updated_at", "last_login_at": "last_login_at"},
	DefaultSort:   []sqlbuilder.Sort{{Key: "created_at", Order: query.ASC}},
	TieBreaker:    "id",
}

// Load assignments only for the SQL page, in one query instead of one per user.
func attachRoles(ctx context.Context, db *sqlx.DB, users []domain.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]string, len(users))
	positions := make(map[string]int, len(users))
	for i := range users {
		ids[i] = users[i].ID
		positions[users[i].ID] = i
		users[i].Roles = []string{}
		users[i].RoleDetails = []domain.UserRole{}
	}
	sql, args, err := sqlx.In(`SELECT ur.user_id, ur.role_id, r.code, r.name FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id IN (?) ORDER BY ur.user_id, ur.role_id`, ids)
	if err != nil {
		return err
	}
	var assignments []struct {
		UserID string `db:"user_id"`
		RoleID string `db:"role_id"`
		Code   string `db:"code"`
		Name   string `db:"name"`
	}
	if err := db.SelectContext(ctx, &assignments, db.Rebind(sql), args...); err != nil {
		return err
	}
	for _, assignment := range assignments {
		if i, ok := positions[assignment.UserID]; ok {
			users[i].Roles = append(users[i].Roles, assignment.RoleID)
			users[i].RoleDetails = append(users[i].RoleDetails, domain.UserRole{ID: assignment.RoleID, Code: assignment.Code, Name: assignment.Name})
		}
	}
	return nil
}
