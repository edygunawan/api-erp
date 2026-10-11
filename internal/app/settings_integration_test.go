package app

import (
	"context"
	"strings"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	roleuc "github.com/edygunawan/go-rest-api/internal/usecase/role"
	settinguc "github.com/edygunawan/go-rest-api/internal/usecase/setting"
	groupuc "github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	useruc "github.com/edygunawan/go-rest-api/internal/usecase/user"
)

func TestSettingsAPIIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		f := setupFoundation(t, cfg)
		admin := f.login(t, "admin", " admin password ").AccessToken
		wantStatus(t, f.request("GET", "/settings", "", nil), 401)
		wantStatus(t, f.request("GET", "/setting-groups", "", nil), 401)

		response := f.request("GET", "/settings?group_code=appearance&page_size=1&sort=setting_key&order=desc", admin, nil)
		wantStatus(t, response, 200)
		page := decodeResponse[query.Page[settinguc.SettingResponse]](t, response)
		if page.Meta.Total != 2 || page.Meta.TotalPages != 2 || len(page.Data) != 1 || page.Data[0].SettingKey != "color_theme" || *page.Data[0].SettingValue != "mono" {
			t.Fatalf("page=%+v", page)
		}
		if page.Data[0].Group.ID != page.Data[0].GroupID || page.Data[0].Group.Code != "APPEARANCE" || page.Data[0].Group.Name == "" || page.Data[0].Group.CreatedAt.IsZero() {
			t.Fatalf("missing list group: %+v", page.Data[0].Group)
		}
		response = f.request("GET", "/settings?group_code=APPEARANCE&setting_key=appearance_mode", admin, nil)
		wantStatus(t, response, 200)
		appearance := decodeResponse[query.Page[settinguc.SettingResponse]](t, response).Data[0]
		dark := "dark"
		response = f.request("PUT", "/settings/"+appearance.ID, admin, settinguc.UpdateSettingRequest{GroupID: appearance.GroupID, SettingKey: appearance.SettingKey, SettingValue: &dark, DataType: "STRING"})
		wantStatus(t, response, 200)
		changed := decodeResponse[settinguc.SettingResponse](t, response)
		if *changed.SettingValue != "dark" || !changed.UpdatedAt.After(appearance.UpdatedAt) {
			t.Fatalf("update=%+v", changed)
		}
		wantStatus(t, f.request("GET", "/settings/"+appearance.ID, admin, nil), 200)

		response = f.request("POST", "/setting-groups", admin, groupuc.CreateSettingGroupRequest{Code: " test_general ", Name: "General"})
		wantStatus(t, response, 201)
		group := decodeResponse[groupuc.SettingGroupResponse](t, response)
		if group.Code != "TEST_GENERAL" || group.CreatedAt.IsZero() {
			t.Fatalf("group=%+v", group)
		}
		wantStatus(t, f.request("POST", "/setting-groups", admin, groupuc.CreateSettingGroupRequest{Code: "TEST_GENERAL", Name: "Duplicate"}), 409)
		response = f.request("PUT", "/setting-groups/"+group.ID, admin, groupuc.UpdateSettingGroupRequest{Code: "TEST_GENERAL", Name: "General preferences"})
		wantStatus(t, response, 200)
		if decodeResponse[groupuc.SettingGroupResponse](t, response).Name != "General preferences" {
			t.Fatal("group update missing")
		}
		wantStatus(t, f.request("GET", "/setting-groups/"+group.ID, admin, nil), 200)
		response = f.request("GET", "/setting-groups?code=test_general&search=preferences&sort=name&page_size=1", admin, nil)
		wantStatus(t, response, 200)
		if decodeResponse[query.Page[groupuc.SettingGroupResponse]](t, response).Meta.Total != 1 {
			t.Fatal("group filter/search failed")
		}

		value := " true "
		req := settinguc.CreateSettingRequest{GroupID: group.ID, SettingKey: " Enabled ", SettingValue: &value, DataType: "boolean"}
		response = f.request("POST", "/settings", admin, req)
		wantStatus(t, response, 201)
		item := decodeResponse[settinguc.SettingResponse](t, response)
		if item.Group.ID != group.ID || item.Group.Name != "General preferences" {
			t.Fatalf("missing created group: %+v", item.Group)
		}
		if item.SettingKey != "enabled" || item.DataType != "BOOLEAN" || *item.SettingValue != "true" {
			t.Fatalf("item=%+v", item)
		}
		// Group changes must be visible in the joined list without a detail lookup.
		wantStatus(t, f.request("PUT", "/setting-groups/"+group.ID, admin, groupuc.UpdateSettingGroupRequest{Code: group.Code, Name: "Renamed preferences"}), 200)
		response = f.request("GET", "/settings?group_id="+group.ID, admin, nil)
		wantStatus(t, response, 200)
		renamedPage := decodeResponse[query.Page[settinguc.SettingResponse]](t, response)
		if renamedPage.Meta.Total != 1 || len(renamedPage.Data) != 1 || renamedPage.Data[0].Group.Name != "Renamed preferences" {
			t.Fatalf("joined group rename: %+v", renamedPage)
		}
		wantStatus(t, f.request("POST", "/settings", admin, req), 409)
		req.GroupID = missingID
		wantStatus(t, f.request("POST", "/settings", admin, req), 400)
		// PUT can move a setting to another valid group and persists a nullable value.
		response = f.request("PUT", "/settings/"+item.ID, admin, settinguc.UpdateSettingRequest{GroupID: appearance.GroupID, SettingKey: "enabled", DataType: "BOOLEAN"})
		wantStatus(t, response, 200)
		moved := decodeResponse[settinguc.SettingResponse](t, response)
		if moved.Group.ID != appearance.GroupID || moved.Group.Code != "APPEARANCE" {
			t.Fatalf("stale moved group: %+v", moved.Group)
		}
		if moved.GroupID != appearance.GroupID || moved.SettingValue != nil {
			t.Fatalf("move=%+v", moved)
		}
		response = f.request("GET", "/settings?group_id="+appearance.GroupID+"&data_type=boolean&search=enabled", admin, nil)
		wantStatus(t, response, 200)
		if decodeResponse[query.Page[settinguc.SettingResponse]](t, response).Meta.Total != 1 {
			t.Fatal("setting filters failed")
		}
		wantStatus(t, f.request("DELETE", "/settings/"+item.ID, admin, nil), 200)
		wantStatus(t, f.request("GET", "/settings/"+item.ID, admin, nil), 404)
		wantStatus(t, f.request("DELETE", "/settings/"+item.ID, admin, nil), 404)

		// A group delete cascades only its settings.
		req.GroupID = group.ID
		response = f.request("POST", "/settings", admin, req)
		wantStatus(t, response, 201)
		child := decodeResponse[settinguc.SettingResponse](t, response)
		wantStatus(t, f.request("DELETE", "/setting-groups/"+group.ID, admin, nil), 200)
		wantStatus(t, f.request("GET", "/settings/"+child.ID, admin, nil), 404)
		wantStatus(t, f.request("DELETE", "/setting-groups/"+group.ID, admin, nil), 404)
		wantStatus(t, f.request("GET", "/settings/"+appearance.ID, admin, nil), 200)

		for _, path := range []string{"/settings/not-a-uuid", "/setting-groups/not-a-uuid", "/settings?sort=setting_value", "/settings?group_id=bad", "/settings?data_type=BAD", "/settings?group_code=" + strings.Repeat("x", 51), "/settings?unknown=1", "/settings?page_size=101", "/setting-groups?sort=password", "/setting-groups?code=" + strings.Repeat("x", 51)} {
			wantStatus(t, f.request("GET", path, admin, nil), 400)
		}
		for _, path := range []string{"/settings/" + missingID, "/setting-groups/" + missingID} {
			wantStatus(t, f.request("GET", path, admin, nil), 404)
		}
		for _, path := range []string{"/settings", "/setting-groups"} {
			wantStatus(t, f.request("POST", path, admin, map[string]string{"unknown": "field"}), 400)
		}
		response = f.request("GET", "/settings?search=no_match_at_all", admin, nil)
		wantStatus(t, response, 200)
		empty := decodeResponse[query.Page[settinguc.SettingResponse]](t, response)
		if empty.Data == nil || len(empty.Data) != 0 || empty.Meta.Total != 0 {
			t.Fatalf("empty=%+v", empty)
		}

		// Read-only settings access must not authorize mutation of either resource.
		ctx := context.Background()
		role, err := roleuc.New(f.repos.Role).Create(ctx, roleuc.CreateRoleRequest{Code: "SETTINGS_READER", Name: "Settings reader", Permissions: []string{"00000000-0000-4000-8001-000000000012"}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = useruc.New(f.repos.User).Create(ctx, useruc.CreateUserRequest{FullName: "Settings Reader", Username: "settings_reader", Password: "ReaderPassword123!", Roles: []string{role.ID}})
		if err != nil {
			t.Fatal(err)
		}
		reader := f.login(t, "settings_reader", "ReaderPassword123!").AccessToken
		for _, path := range []string{"/settings", "/setting-groups"} {
			wantStatus(t, f.request("GET", path, reader, nil), 200)
			wantStatus(t, f.request("POST", path, reader, map[string]string{}), 403)
			wantStatus(t, f.request("PUT", path+"/"+appearance.ID, reader, map[string]string{}), 403)
			wantStatus(t, f.request("DELETE", path+"/"+appearance.ID, reader, nil), 403)
		}
		_, err = useruc.New(f.repos.User).Create(ctx, useruc.CreateUserRequest{FullName: "No permissions", Username: "settings_denied", Password: "DeniedPassword123!"})
		if err != nil {
			t.Fatal(err)
		}
		denied := f.login(t, "settings_denied", "DeniedPassword123!").AccessToken
		for _, path := range []string{"/settings", "/setting-groups"} {
			wantStatus(t, f.request("GET", path, denied, nil), 403)
		}
		wantStatus(t, f.request("DELETE", "/permissions/00000000-0000-4000-8001-000000000012", admin, nil), 403)
	})
}
