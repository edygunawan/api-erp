package rolerepo

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
	"github.com/jmoiron/sqlx"
)

var roleListDefinition = sqlbuilder.Definition{
	SearchColumns: []string{"name", "code", "description"},
	SortColumns:   map[string]string{"id": "id", "name": "name", "code": "code", "created_at": "created_at", "updated_at": "updated_at"},
	DefaultSort:   []sqlbuilder.Sort{{Key: "code", Order: query.ASC}},
	TieBreaker:    "id",
}

func attachPermissions(ctx context.Context, db *sqlx.DB, roles []domain.Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]string, len(roles))
	positions := make(map[string]int, len(roles))
	for i := range roles {
		ids[i] = roles[i].ID
		positions[roles[i].ID] = i
		roles[i].Permissions = []string{}
	}
	sql, args, err := sqlx.In(`SELECT role_id, permission_id FROM role_permissions WHERE role_id IN (?) ORDER BY role_id, permission_id`, ids)
	if err != nil {
		return err
	}
	var assignments []struct {
		RoleID       string `db:"role_id"`
		PermissionID string `db:"permission_id"`
	}
	if err := db.SelectContext(ctx, &assignments, db.Rebind(sql), args...); err != nil {
		return err
	}
	for _, assignment := range assignments {
		if i, ok := positions[assignment.RoleID]; ok {
			roles[i].Permissions = append(roles[i].Permissions, assignment.PermissionID)
		}
	}
	return nil
}
