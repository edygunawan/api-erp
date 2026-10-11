package permission

import (
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/helper/sqlbuilder"
)

var permissionListDefinition = sqlbuilder.Definition{
	SearchColumns: []string{"name", "code", "module", "description"},
	FilterColumns: map[string]string{"module": "module"},
	SortColumns:   map[string]string{"id": "id", "name": "name", "code": "code", "module": "module", "created_at": "created_at", "updated_at": "updated_at"},
	DefaultSort:   []sqlbuilder.Sort{{Key: "module", Order: query.ASC}, {Key: "code", Order: query.ASC}},
	TieBreaker:    "id",
}
