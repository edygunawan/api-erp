package app

import (
	"context"
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	roleuc "github.com/edygunawan/go-rest-api/internal/usecase/role"
	useruc "github.com/edygunawan/go-rest-api/internal/usecase/user"
	"testing"
)

func TestUserRoleDetailsIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		f := setupFoundation(t, cfg)
		admin := f.login(t, "admin", " admin password ").AccessToken
		ctx := context.Background()
		roles := roleuc.New(f.repos.Role)
		first, err := roles.Create(ctx, roleuc.CreateRoleRequest{Code: "REL_FIRST", Name: "First role"})
		if err != nil {
			t.Fatal(err)
		}
		second, err := roles.Create(ctx, roleuc.CreateRoleRequest{Code: "REL_SECOND", Name: "Second role"})
		if err != nil {
			t.Fatal(err)
		}
		users := useruc.New(f.repos.User)
		assigned, err := users.Create(ctx, useruc.CreateUserRequest{FullName: "Relation User Alpha", Username: "rel_alpha", Password: "password", Roles: []string{first.ID, second.ID}})
		if err != nil {
			t.Fatal(err)
		}
		if len(assigned.Roles) != 2 || len(assigned.RoleDetails) != 2 {
			t.Fatalf("create roles: %+v", assigned)
		}
		_, err = users.Create(ctx, useruc.CreateUserRequest{FullName: "Relation User Beta", Username: "rel_beta", Password: "password"})
		if err != nil {
			t.Fatal(err)
		}
		response := f.request("GET", "/users?search=Relation%20User&sort=full_name&page_size=1", admin, nil)
		wantStatus(t, response, 200)
		page := decodeResponse[query.Page[useruc.UserResponse]](t, response)
		if page.Meta.Total != 2 || page.Meta.TotalPages != 2 || len(page.Data) != 1 || len(page.Data[0].RoleDetails) != 2 {
			t.Fatalf("pagination with multiple roles: %+v", page)
		}
		found := map[string]string{}
		for _, role := range page.Data[0].RoleDetails {
			found[role.Code] = role.Name
		}
		if found["REL_FIRST"] != "First role" || found["REL_SECOND"] != "Second role" {
			t.Fatalf("role names: %+v", found)
		}
		response = f.request("GET", "/users?search=Relation%20User&sort=full_name&page_size=1&page=2", admin, nil)
		wantStatus(t, response, 200)
		empty := decodeResponse[query.Page[useruc.UserResponse]](t, response).Data[0]
		if empty.Roles == nil || empty.RoleDetails == nil || len(empty.Roles) != 0 || len(empty.RoleDetails) != 0 {
			t.Fatalf("empty assignments must be arrays: %+v", empty)
		}
		_, err = roles.Update(ctx, first.ID, roleuc.UpdateRoleRequest{Code: first.Code, Name: "Renamed role"})
		if err != nil {
			t.Fatal(err)
		}
		response = f.request("GET", "/users/"+assigned.ID, admin, nil)
		wantStatus(t, response, 200)
		detail := decodeResponse[useruc.UserResponse](t, response)
		for _, role := range detail.RoleDetails {
			if role.ID == first.ID && role.Name != "Renamed role" {
				t.Fatal("stale role name")
			}
		}
		updated, err := users.Update(ctx, assigned.ID, useruc.UpdateUserRequest{FullName: assigned.FullName, Username: assigned.Username, Status: assigned.Status, Roles: []string{second.ID}})
		if err != nil {
			t.Fatal(err)
		}
		if len(updated.RoleDetails) != 1 || updated.RoleDetails[0].ID != second.ID {
			t.Fatalf("removed assignment persisted: %+v", updated)
		}
	})
}
