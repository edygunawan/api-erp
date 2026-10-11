package database_test

import (
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/testutil"
)

func TestSettingsMigrationIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		db, cfg := testutil.Isolate(t, cfg)
		exec := func(query string, args ...any) {
			t.Helper()
			if _, err := db.Exec(testutil.Bind(cfg.Driver, query), args...); err != nil {
				t.Fatal(err)
			}
		}
		count := func(query string, want int) {
			t.Helper()
			var got int
			if err := db.QueryRow(query).Scan(&got); err != nil || got != want {
				t.Fatalf("count=%d want=%d err=%v", got, want, err)
			}
		}
		migrate := func(steps int) {
			t.Helper()
			if err := database.Migrate(cfg, steps); err != nil {
				t.Fatal(err)
			}
		}
		value := func(key, want string) {
			t.Helper()
			var got string
			if err := db.QueryRow(testutil.Bind(cfg.Driver, `SELECT s.setting_value FROM settings s JOIN setting_groups g ON g.id=s.group_id WHERE g.code='APPEARANCE' AND s.setting_key=?`), key).Scan(&got); err != nil || got != want {
				t.Fatalf("%s=%s want=%s err=%v", key, got, want, err)
			}
		}

		count("SELECT COUNT(*) FROM permissions WHERE module='APPLICATIONSETTING'", 2)
		count("SELECT COUNT(*) FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id JOIN roles r ON r.id=rp.role_id WHERE p.module='APPLICATIONSETTING' AND r.code='ADMIN'", 2)
		migrate(-1)
		count("SELECT COUNT(*) FROM permissions WHERE module='APPLICATIONSETTING'", 0)
		count("SELECT COUNT(*) FROM permissions WHERE module='SETTING'", 4)
		migrate(0)
		value("appearance_mode", "light")
		value("color_theme", "mono")
		count(`SELECT COUNT(*) FROM settings WHERE data_type='STRING' AND created_at IS NOT NULL AND updated_at IS NOT NULL`, 6)
		exec(`INSERT INTO setting_groups (code,name) VALUES ('OTHER','Other settings')`)
		var otherID string
		if err := db.QueryRow(`SELECT id FROM setting_groups WHERE code='OTHER'`).Scan(&otherID); err != nil || len(otherID) != 36 {
			t.Fatalf("UUID default=%q err=%v", otherID, err)
		}
		for _, dataType := range []string{"STRING", "NUMBER", "BOOLEAN", "JSON"} {
			exec(`INSERT INTO settings (group_id,setting_key,data_type) VALUES (?,?,?)`, otherID, dataType, dataType)
		}
		for _, check := range []struct {
			name, query string
			args        []any
		}{
			{"duplicate group", `INSERT INTO setting_groups (code,name) VALUES ('OTHER','Duplicate')`, nil},
			{"duplicate key", `INSERT INTO settings (group_id,setting_key) VALUES (?,'STRING')`, []any{otherID}},
			{"invalid type", `INSERT INTO settings (group_id,setting_key,data_type) VALUES (?,'bad_type','INVALID')`, []any{otherID}},
			{"blank key", `INSERT INTO settings (group_id,setting_key) VALUES (?,' ')`, []any{otherID}},
			{"missing group", `INSERT INTO settings (group_id,setting_key) VALUES ('ffffffff-ffff-4fff-8fff-ffffffffffff','missing')`, nil},
		} {
			if _, err := db.Exec(testutil.Bind(cfg.Driver, check.query), check.args...); err == nil {
				t.Fatalf("constraint accepted %s", check.name)
			}
		}
		exec(`DELETE FROM setting_groups WHERE code='OTHER'`)
		count(`SELECT COUNT(*) FROM settings`, 6)

		// Seed-only rollback removes its records but retains a group with custom settings.
		exec(`INSERT INTO settings (group_id,setting_key,setting_value) SELECT id,'density','compact' FROM setting_groups WHERE code='APPEARANCE'`)
		migrate(-4)
		value("density", "compact")
		count(`SELECT COUNT(*) FROM setting_groups`, 1)
		count(`SELECT COUNT(*) FROM settings`, 1)
		exec(`DELETE FROM setting_groups WHERE code='APPEARANCE'`)
		count(`SELECT COUNT(*) FROM setting_groups`, 0)
		count(`SELECT COUNT(*) FROM settings`, 0)

		// Upgrading an existing configuration preserves names, IDs and custom values.
		exec(`INSERT INTO setting_groups (code,name) VALUES ('APPEARANCE','Existing appearance')`)
		var existingID string
		if err := db.QueryRow(`SELECT id FROM setting_groups WHERE code='APPEARANCE'`).Scan(&existingID); err != nil {
			t.Fatal(err)
		}
		exec(`INSERT INTO settings (group_id,setting_key,setting_value) VALUES (?,'appearance_mode','dark')`, existingID)
		exec(`INSERT INTO settings (group_id,setting_key,setting_value) VALUES (?,'density','compact')`, existingID)
		migrate(0)
		value("appearance_mode", "dark")
		value("color_theme", "mono")
		value("density", "compact")
		count(`SELECT COUNT(*) FROM setting_groups WHERE name='Existing appearance'`, 1)
		migrate(-4)
		value("appearance_mode", "dark")
		value("density", "compact")
		count(`SELECT COUNT(*) FROM settings`, 2)
		count(`SELECT COUNT(*) FROM setting_groups`, 1)

		// Schema rollback must remove the child table before the parent table.
		migrate(-1)
		tableQuery := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name IN ('settings','setting_groups')`
		if cfg.Driver == database.MySQL {
			tableQuery = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name IN ('settings','setting_groups')`
		}
		count(tableQuery, 0)
		migrate(0)
		value("appearance_mode", "light")
		value("color_theme", "mono")
	})
}
