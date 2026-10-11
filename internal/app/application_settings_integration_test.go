package app

import (
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	settinguc "github.com/edygunawan/go-rest-api/internal/usecase/setting"
	"testing"
)

func TestApplicationSettingsIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		f := setupFoundation(t, cfg)
		admin := f.login(t, "admin", " admin password ").AccessToken
		wantStatus(t, f.request("GET", "/settings/application", "", nil), 401)
		response := f.request("GET", "/settings/application", admin, nil)
		wantStatus(t, response, 200)
		original := decodeResponse[settinguc.ApplicationSettingsResponse](t, response)
		if original.ApplicationName != "React Admin Kit" || original.Timezone != "Asia/Makassar" || original.DateFormat != "DD/MM/YYYY" {
			t.Fatalf("defaults=%+v", original)
		}
		changed := original
		changed.ApplicationName = " My Application "
		changed.Description = "Updated description"
		changed.Timezone = "Asia/Jakarta"
		changed.DateFormat = "YYYY-MM-DD"
		response = f.request("PUT", "/settings/application", admin, changed)
		wantStatus(t, response, 200)
		saved := decodeResponse[settinguc.ApplicationSettingsResponse](t, response)
		if saved.ApplicationName != "My Application" {
			t.Fatalf("saved=%+v", saved)
		}
		response = f.request("GET", "/settings/application", admin, nil)
		wantStatus(t, response, 200)
		if decodeResponse[settinguc.ApplicationSettingsResponse](t, response) != saved {
			t.Fatal("values were not persisted")
		}
		bad := saved
		bad.Timezone = "Invalid/Timezone"
		wantStatus(t, f.request("PUT", "/settings/application", admin, bad), 400)
		bad = saved
		bad.ApplicationName = " "
		wantStatus(t, f.request("PUT", "/settings/application", admin, bad), 400)
		bad = saved
		bad.DateFormat = "arbitrary"
		wantStatus(t, f.request("PUT", "/settings/application", admin, bad), 400)
		wantStatus(t, f.request("PUT", "/settings/application", admin, map[string]string{"unexpected": "field"}), 400)
		// SETTING permissions alone do not grant access to the application form.
		if _, err := f.db.Exec(`DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('APPLICATIONSETTING.READ','APPLICATIONSETTING.UPDATE'))`); err != nil {
			t.Fatal(err)
		}
		wantStatus(t, f.request("GET", "/settings/application", admin, nil), 403)
		wantStatus(t, f.request("GET", "/settings", admin, nil), 200)
		wantStatus(t, f.request("PUT", "/settings/application", admin, saved), 403)
		if _, err := f.db.Exec(`INSERT INTO role_permissions(role_id,permission_id) SELECT r.id,p.id FROM roles r CROSS JOIN permissions p WHERE r.code='ADMIN' AND p.code IN ('APPLICATIONSETTING.READ','APPLICATIONSETTING.UPDATE')`); err != nil {
			t.Fatal(err)
		}
		// Application permissions alone must not grant generic settings access.
		if _, err := f.db.Exec("DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('SETTING.READ','SETTING.UPDATE'))"); err != nil {
			t.Fatal(err)
		}
		wantStatus(t, f.request("GET", "/settings/application", admin, nil), 200)
		wantStatus(t, f.request("PUT", "/settings/application", admin, saved), 200)
		wantStatus(t, f.request("GET", "/settings", admin, nil), 403)
		if _, err := f.db.Exec("INSERT INTO role_permissions(role_id,permission_id) SELECT r.id,p.id FROM roles r CROSS JOIN permissions p WHERE r.code='ADMIN' AND p.code IN ('SETTING.READ','SETTING.UPDATE')"); err != nil {
			t.Fatal(err)
		}
		// Missing the final field must roll back updates to the preceding three fields.
		if _, err := f.db.Exec(`DELETE FROM settings WHERE setting_key='date_format' AND group_id IN (SELECT id FROM setting_groups WHERE code='LOCALIZATION')`); err != nil {
			t.Fatal(err)
		}
		changed = saved
		changed.ApplicationName = "Must not persist"
		changed.Description = "Must not persist"
		changed.Timezone = "UTC"
		wantStatus(t, f.request("PUT", "/settings/application", admin, changed), 404)
		var name, description, zone string
		if err := f.db.QueryRow(`SELECT setting_value FROM settings WHERE setting_key='application_name' AND group_id IN (SELECT id FROM setting_groups WHERE code='GENERAL')`).Scan(&name); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRow(`SELECT setting_value FROM settings WHERE setting_key='description' AND group_id IN (SELECT id FROM setting_groups WHERE code='GENERAL')`).Scan(&description); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRow(`SELECT setting_value FROM settings WHERE setting_key='timezone' AND group_id IN (SELECT id FROM setting_groups WHERE code='LOCALIZATION')`).Scan(&zone); err != nil {
			t.Fatal(err)
		}
		if name != saved.ApplicationName || description != saved.Description || zone != saved.Timezone {
			t.Fatal("partial save was committed")
		}
		wantStatus(t, f.request("GET", "/settings/application", admin, nil), 404)
	})
}
