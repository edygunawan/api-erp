package sqlbuilder

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/jmoiron/sqlx"
)

func definition() Definition {
	return Definition{
		SearchColumns: []string{"u.name", "u.email"},
		FilterColumns: map[string]string{"status": "u.status", "module": "u.module"},
		SortColumns:   map[string]string{"name": "u.name", "created_at": "u.created_at", "id": "u.id"},
		DefaultSort:   []Sort{{Key: "created_at", Order: query.DESC}}, TieBreaker: "id",
	}
}
func mustPlan(t *testing.T, opts query.Options, d Definition) Plan {
	t.Helper()
	p, err := New(opts, d)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestSearchAndCountSuffix(t *testing.T) {
	for _, columns := range [][]string{{"u.name"}, {"u.name", "u.email"}} {
		d := definition()
		d.SearchColumns = columns
		p := mustPlan(t, query.Options{Page: 3, PageSize: 20, Search: " Jane%_! ", Filters: map[string][]string{"status": {"ACTIVE"}}}, d)
		where, countArgs := p.Where()
		suffix, listArgs := p.Suffix()
		parts := make([]string, len(columns))
		for i, column := range columns {
			parts[i] = "LOWER(" + column + ") LIKE LOWER(?) ESCAPE '!'"
		}
		want := " WHERE (" + strings.Join(parts, " OR ") + ") AND u.status = ?"
		if where != want || strings.Contains(where, "ORDER BY") || strings.Contains(where, "LIMIT") || !strings.HasPrefix(suffix, where+" ORDER BY u.created_at DESC, u.id DESC LIMIT ? OFFSET ?") {
			t.Fatalf("where=%s suffix=%s", where, suffix)
		}
		args := make([]any, 0, len(columns)+1)
		for range columns {
			args = append(args, "%Jane!%!_!!%")
		}
		args = append(args, "ACTIVE")
		if !reflect.DeepEqual(countArgs, args) || !reflect.DeepEqual(listArgs, append(args, 20, int64(40))) {
			t.Fatalf("count=%v list=%v", countArgs, listArgs)
		}
		countArgs[0] = "mutated"
		_, again := p.Where()
		if again[0] != "%Jane!%!_!!%" {
			t.Fatal("args alias internal plan")
		}
	}
}
func TestEmptySearchAndFilters(t *testing.T) {
	p := mustPlan(t, query.Options{Search: "  ", Filters: map[string][]string{"status": {}, "module": {" "}}}, definition())
	where, args := p.Where()
	if where != "" || len(args) != 0 {
		t.Fatalf("where=%s args=%v", where, args)
	}
	suffix, args := p.Suffix()
	if suffix != " ORDER BY u.created_at DESC, u.id DESC LIMIT ? OFFSET ?" || !reflect.DeepEqual(args, []any{20, int64(0)}) {
		t.Fatalf("suffix=%s args=%v", suffix, args)
	}
}
func TestExactAndINFiltersAreDeterministic(t *testing.T) {
	p := mustPlan(t, query.Options{Filters: map[string][]string{"status": {"ACTIVE", "INACTIVE"}, "module": {"AUDIT"}}}, definition())
	where, args := p.Where()
	if where != " WHERE u.module = ? AND u.status IN (?, ?)" || !reflect.DeepEqual(args, []any{"AUDIT", "ACTIVE", "INACTIVE"}) {
		t.Fatalf("where=%s args=%v", where, args)
	}
	for range 25 {
		other := mustPlan(t, query.Options{Filters: map[string][]string{"status": {"ACTIVE", "INACTIVE"}, "module": {"AUDIT"}}}, definition())
		sql, values := other.Where()
		if sql != where || !reflect.DeepEqual(args, values) {
			t.Fatal("nondeterministic map iteration")
		}
	}
}
func TestSortWhitelistDirectionsAndTieBreaker(t *testing.T) {
	for _, tc := range []struct {
		opts query.Options
		want string
	}{
		{query.Options{Sort: "name"}, "u.name ASC, u.id ASC"},
		{query.Options{Sort: "name", Order: query.DESC}, "u.name DESC, u.id DESC"},
		{query.Options{Sort: "id", Order: query.ASC}, "u.id ASC"},
		{query.Options{Order: query.ASC}, "u.created_at ASC, u.id ASC"},
	} {
		p := mustPlan(t, tc.opts, definition())
		sql, _ := p.Suffix()
		if sql != " ORDER BY "+tc.want+" LIMIT ? OFFSET ?" {
			t.Fatal(sql)
		}
	}
}
func TestInvalidClientInputAndMaliciousValues(t *testing.T) {
	for _, opts := range []query.Options{
		{Sort: "name;DROP TABLE users"}, {Sort: "password"}, {Order: "desc;--"},
		{Filters: map[string][]string{"status;DROP TABLE users": {"ACTIVE"}}},
	} {
		_, err := New(opts, definition())
		var invalid *query.ValidationError
		if !errors.As(err, &invalid) {
			t.Fatalf("error=%v", err)
		}
	}
	malicious := "' OR 1=1 --"
	p := mustPlan(t, query.Options{Search: malicious, Filters: map[string][]string{"status": {malicious}}}, definition())
	sql, args := p.Suffix()
	if strings.Contains(sql, malicious) || !reflect.DeepEqual(args, []any{"%" + malicious + "%", "%" + malicious + "%", malicious, 20, int64(0)}) {
		t.Fatalf("sql=%s args=%v", sql, args)
	}
	d := definition()
	d.SearchColumns = nil
	if _, err := New(query.Options{Search: "test"}, d); err == nil {
		t.Fatal("unsupported search accepted")
	}
}
func TestInvalidServerConfiguration(t *testing.T) {
	for _, change := range []func(*Definition){
		func(d *Definition) { d.SearchColumns = []string{"name;DROP TABLE users"} },
		func(d *Definition) { d.SortColumns["name"] = "name DESC;--" },
		func(d *Definition) { d.FilterColumns["status"] = "u.status OR 1=1" },
		func(d *Definition) { d.SearchColumns = []string{"u;evil.name"} },
		func(d *Definition) { d.TieBreaker = "missing" },
		func(d *Definition) { d.DefaultSort = nil },
		func(d *Definition) { d.DefaultSort = []Sort{{Key: "name", Order: "DESC;--"}} },
		func(d *Definition) { d.DefaultSort = []Sort{{Key: "missing", Order: query.ASC}} },
	} {
		d := definition()
		change(&d)
		_, err := New(query.Options{}, d)
		var config *ConfigError
		if !errors.As(err, &config) {
			t.Fatalf("error=%v", err)
		}
	}
}
func TestCompleteQueryRebindAndIndependentCountArgs(t *testing.T) {
	p := mustPlan(t, query.Options{Search: "person", Filters: map[string][]string{"status": {"ACTIVE", "INACTIVE"}}}, definition())
	where, countArgs := p.Where()
	suffix, args := p.Suffix()
	countSQL := sqlx.Rebind(sqlx.DOLLAR, "SELECT COUNT(*) FROM users u"+where)
	dataSQL := sqlx.Rebind(sqlx.DOLLAR, "SELECT u.id FROM users u"+suffix)
	if !strings.Contains(countSQL, "IN ($3, $4)") || strings.Contains(countSQL, "$5") || !strings.HasSuffix(dataSQL, "LIMIT $5 OFFSET $6") {
		t.Fatalf("count=%s data=%s", countSQL, dataSQL)
	}
	args[0] = "changed"
	if len(countArgs) != 4 || countArgs[0] != "%person%" {
		t.Fatal("list args aliased count args")
	}
}
