// Package sqlbuilder builds only WHERE/ORDER BY/LIMIT/OFFSET suffixes for ordinary
// lists. Repositories retain SELECT, FROM, joins, scanning and context handling.
package sqlbuilder

import (
	"regexp"
	"slices"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

type Sort struct {
	Key   string
	Order query.Direction
}

type Definition struct {
	SearchColumns []string
	FilterColumns map[string]string
	SortColumns   map[string]string
	DefaultSort   []Sort
	// TieBreaker is a whitelisted unique key, appended using the primary direction.
	TieBreaker string
}

// ConfigError is a programmer error; its details must not reach API clients.
type ConfigError struct{ Message string }

func (e *ConfigError) Error() string { return "sqlbuilder configuration: " + e.Message }

// Identifiers are bare columns or alias.column, not expressions or SQL fragments.
var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

func (d Definition) validate() error {
	for _, column := range d.SearchColumns {
		if !identifier.MatchString(column) {
			return &ConfigError{"invalid search column"}
		}
	}
	for key, column := range d.FilterColumns {
		if key == "" || !identifier.MatchString(column) {
			return &ConfigError{"invalid filter column"}
		}
	}
	for key, column := range d.SortColumns {
		if key == "" || !identifier.MatchString(column) {
			return &ConfigError{"invalid sort column"}
		}
	}
	if len(d.DefaultSort) == 0 || d.SortColumns[d.TieBreaker] == "" {
		return &ConfigError{"default sort and unique tie breaker required"}
	}
	for _, sort := range d.DefaultSort {
		if d.SortColumns[sort.Key] == "" || (sort.Order != query.ASC && sort.Order != query.DESC) {
			return &ConfigError{"invalid default sort"}
		}
	}
	return nil
}

// Plan is immutable after construction. Returned args are independent slices.
// Question-mark placeholders are rebound on the COMPLETE SQL by sqlx for pgx.
type Plan struct {
	where   string
	order   string
	args    []any
	options query.Options
}

func New(options query.Options, definition Definition) (Plan, error) {
	if err := definition.validate(); err != nil {
		return Plan{}, err
	}
	o, err := options.Normalize()
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{options: o}
	conditions := []string{}
	if o.Search != "" {
		if len(definition.SearchColumns) == 0 {
			return Plan{}, query.Invalid("search is not supported")
		}
		// Treat %, _ and ! as literal user text, identically on both drivers.
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(o.Search) + "%"
		parts := make([]string, len(definition.SearchColumns))
		for i, column := range definition.SearchColumns {
			parts[i] = "LOWER(" + column + ") LIKE LOWER(?) ESCAPE '!'"
			plan.args = append(plan.args, pattern)
		}
		conditions = append(conditions, "("+strings.Join(parts, " OR ")+")")
	}
	keys := make([]string, 0, len(o.Filters))
	for key := range o.Filters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		column, allowed := definition.FilterColumns[key]
		if !allowed {
			return Plan{}, query.Invalid("unsupported filter")
		}
		values := o.Filters[key]
		if len(values) == 0 {
			continue
		}
		placeholders := make([]string, len(values))
		for i, value := range values {
			placeholders[i] = "?"
			plan.args = append(plan.args, value)
		}
		if len(values) == 1 {
			conditions = append(conditions, column+" = ?")
		} else {
			conditions = append(conditions, column+" IN ("+strings.Join(placeholders, ", ")+")")
		}
	}
	if len(conditions) > 0 {
		plan.where = " WHERE " + strings.Join(conditions, " AND ")
	}
	sorts := slices.Clone(definition.DefaultSort)
	if o.Sort != "" {
		if _, ok := definition.SortColumns[o.Sort]; !ok {
			return Plan{}, query.Invalid("unsupported sort field")
		}
		direction := o.Order
		if direction == "" {
			direction = query.ASC
		}
		sorts = []Sort{{Key: o.Sort, Order: direction}}
	} else if o.Order != "" {
		for i := range sorts {
			sorts[i].Order = o.Order
		}
	}
	parts := []string{}
	seenColumns := map[string]bool{}
	for _, sort := range sorts {
		column := definition.SortColumns[sort.Key]
		parts = append(parts, column+" "+string(sort.Order))
		seenColumns[column] = true
	}
	if column := definition.SortColumns[definition.TieBreaker]; !seenColumns[column] {
		parts = append(parts, column+" "+string(sorts[0].Order))
	}
	plan.order = " ORDER BY " + strings.Join(parts, ", ")
	return plan, nil
}

// Where applies the same search/filters to COUNT(*) without sorting or paging.
func (p Plan) Where() (string, []any) { return p.where, slices.Clone(p.args) }

// Suffix appends the complete list suffix and parameterized LIMIT/OFFSET.
func (p Plan) Suffix() (string, []any) {
	args := make([]any, 0, len(p.args)+2)
	args = append(args, p.args...)
	args = append(args, p.options.PageSize, p.options.Offset())
	return p.where + p.order + " LIMIT ? OFFSET ?", args
}
