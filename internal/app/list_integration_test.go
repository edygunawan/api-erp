package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	permissionuc "github.com/edygunawan/go-rest-api/internal/usecase/permission"
	roleuc "github.com/edygunawan/go-rest-api/internal/usecase/role"
	useruc "github.com/edygunawan/go-rest-api/internal/usecase/user"
)

func TestPaginatedLists(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		f := setupFoundation(t, cfg)
		ctx := context.Background()
		for i := 0; i < 24; i++ {
			uid, err := helper.NewUUID()
			if err != nil {
				t.Fatal(err)
			}
			status := domain.UserStatusActive
			if i%2 == 1 {
				status = domain.UserStatusInactive
			}
			name := fmt.Sprintf("Query Person %02d", i)
			if i == 23 {
				name = "Query Literal %_! Person"
			}
			email := fmt.Sprintf("query%02d@example.test", i)
			u := &domain.User{ID: uid, FullName: name, Username: fmt.Sprintf("query_%02d", i), Email: &email, Status: status, Password: "not-used-for-login", Roles: []string{adminRole}}
			if err := f.repos.User.Create(ctx, u); err != nil {
				t.Fatal(err)
			}
			pid, err := helper.NewUUID()
			if err != nil {
				t.Fatal(err)
			}
			module := "QUERY"
			if i%2 == 1 {
				module = "OTHER"
			}
			p := &domain.Permission{ID: pid, Code: fmt.Sprintf("QUERY.ACTION_%02d", i), Name: fmt.Sprintf("Query Permission %02d", i), Module: module}
			if err := f.repos.Permission.Create(ctx, p); err != nil {
				t.Fatal(err)
			}
			rid, err := helper.NewUUID()
			if err != nil {
				t.Fatal(err)
			}
			r := &domain.Role{ID: rid, Code: fmt.Sprintf("QUERY_ROLE_%02d", i), Name: fmt.Sprintf("Query Role %02d", i), Permissions: []string{userReadPermission, pid}}
			if err := f.repos.Role.Create(ctx, r); err != nil {
				t.Fatal(err)
			}
		}
		token := f.login(t, "admin", " admin password ").AccessToken
		for _, entity := range []struct{ path, sort string }{{"users", "username"}, {"roles", "code"}, {"permissions", "code"}} {
			t.Run(entity.path, func(t *testing.T) {
				type row struct {
					ID          string   `json:"id"`
					Roles       []string `json:"roles"`
					Permissions []string `json:"permissions"`
				}
				list := func(page int) query.Page[row] {
					w := f.request("GET", fmt.Sprintf("/%s?search=QUERY&page=%d&page_size=5&sort=%s&order=asc", entity.path, page, entity.sort), token, nil)
					wantStatus(t, w, 200)
					if strings.Contains(w.Body.String(), `"password"`) {
						t.Fatal("list exposed password")
					}
					return decodeResponse[query.Page[row]](t, w)
				}
				one, two := list(1), list(2)
				if len(one.Data) != 5 || len(two.Data) != 5 || one.Meta.Total != 24 || two.Meta.Total != 24 || one.Meta.TotalPages != 5 || two.Meta.Page != 2 || two.Meta.PageSize != 5 {
					t.Fatalf("pages=%+v %+v", one, two)
				}
				ids := map[string]bool{}
				for _, r := range one.Data {
					ids[r.ID] = true
					if entity.path == "users" && (len(r.Roles) != 1 || r.Roles[0] != adminRole) {
						t.Fatal("user roles missing")
					}
					if entity.path == "roles" && len(r.Permissions) != 2 {
						t.Fatal("role permissions missing")
					}
				}
				for _, r := range two.Data {
					if ids[r.ID] {
						t.Fatal("page 2 reused page 1 row")
					}
				}
				again := list(1)
				for i, r := range again.Data {
					if r.ID != one.Data[i].ID {
						t.Fatal("ordering changed between identical requests")
					}
				}
				far := list(999)
				if far.Data == nil || len(far.Data) != 0 || far.Meta.Page != 999 || far.Meta.Total != 24 || far.Meta.TotalPages != 5 {
					t.Fatalf("out of range=%+v", far)
				}
				w := f.request("GET", "/"+entity.path+"?search=missing-record", token, nil)
				wantStatus(t, w, 200)
				empty := decodeResponse[query.Page[row]](t, w)
				if empty.Data == nil || len(empty.Data) != 0 || empty.Meta.Total != 0 || empty.Meta.TotalPages != 0 {
					t.Fatal("empty page contract failed")
				}
				for _, raw := range []string{"sort=" + url.QueryEscape("name;DROP TABLE users"), "order=" + url.QueryEscape("desc;--"), "sort=password", "page=0", "page_size=101", "unexpected=field"} {
					wantStatus(t, f.request("GET", "/"+entity.path+"?"+raw, token, nil), 400)
				}
				w = f.request("GET", "/"+entity.path+"?search="+url.QueryEscape("' OR 1=1 --"), token, nil)
				wantStatus(t, w, 200)
				if decodeResponse[query.Page[row]](t, w).Meta.Total != 0 {
					t.Fatal("malicious search returned records")
				}
			})
		}
		w := f.request("GET", "/users?search=QUERY&status=active&page=2&page_size=5&sort=username", token, nil)
		wantStatus(t, w, 200)
		active := decodeResponse[query.Page[useruc.UserResponse]](t, w)
		if len(active.Data) != 5 || active.Meta.Total != 12 || active.Meta.TotalPages != 3 {
			t.Fatalf("active page=%+v", active)
		}
		for _, u := range active.Data {
			if u.Status != domain.UserStatusActive {
				t.Fatal("status predicate was not applied")
			}
		}
		w = f.request("GET", "/users?search=QUERY&status=ACTIVE,INACTIVE&page_size=100", token, nil)
		wantStatus(t, w, 200)
		if page := decodeResponse[query.Page[useruc.UserResponse]](t, w); len(page.Data) != 24 || page.Meta.Total != 24 {
			t.Fatal("IN filter failed")
		}
		wantStatus(t, f.request("GET", "/users?status=UNKNOWN", token, nil), 400)
		w = f.request("GET", "/users?search="+url.QueryEscape("%_!"), token, nil)
		wantStatus(t, w, 200)
		if page := decodeResponse[query.Page[useruc.UserResponse]](t, w); len(page.Data) != 1 || page.Meta.Total != 1 || page.Data[0].FullName != "Query Literal %_! Person" {
			t.Fatalf("literal wildcard=%+v", page)
		}
		w = f.request("GET", "/permissions?search=query&module=query&page_size=5&sort=code&order=desc", token, nil)
		wantStatus(t, w, 200)
		permissionPage := decodeResponse[query.Page[permissionuc.PermissionResponse]](t, w)
		if len(permissionPage.Data) != 5 || permissionPage.Meta.Total != 12 || permissionPage.Meta.TotalPages != 3 {
			t.Fatalf("permission page=%+v", permissionPage)
		}
		if permissionPage.Data[0].Code != "QUERY.ACTION_22" || permissionPage.Data[4].Code != "QUERY.ACTION_14" {
			t.Fatal("descending sort not applied")
		}
		w = f.request("GET", "/permissions?module=QUERY,OTHER&page_size=100", token, nil)
		wantStatus(t, w, 200)
		if page := decodeResponse[query.Page[permissionuc.PermissionResponse]](t, w); len(page.Data) != 24 || page.Meta.Total != 24 {
			t.Fatal("module IN filter failed")
		}
		// Tie-break by ID remains stable when every primary sort value is identical.
		if _, err := f.db.ExecContext(ctx, testutil.Bind(cfg.Driver, `UPDATE users SET created_at=? WHERE username LIKE ?`), "2026-01-01 00:00:00", "query_%"); err != nil {
			t.Fatal(err)
		}
		for _, pageNum := range []int{1, 2} {
			w = f.request("GET", fmt.Sprintf("/users?search=query&sort=created_at&page=%d&page_size=5", pageNum), token, nil)
			wantStatus(t, w, 200)
			page := decodeResponse[query.Page[useruc.UserResponse]](t, w)
			for i := 1; i < len(page.Data); i++ {
				if page.Data[i-1].ID >= page.Data[i].ID {
					t.Fatal("missing deterministic tie-breaker")
				}
			}
		}
		w = f.request("GET", "/roles?search=QUERY&page_size=100", token, nil)
		wantStatus(t, w, 200)
		if page := decodeResponse[query.Page[roleuc.RoleResponse]](t, w); len(page.Data) != 24 {
			t.Fatal("large role page failed")
		}
	})
}
