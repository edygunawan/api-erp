package factory_test

import (
	"context"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/repository/factory"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
	"golang.org/x/crypto/bcrypt"
)

func TestDefaultAdminMigrationIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		db, cfg := testutil.Isolate(t, cfg)
		repos, err := factory.New(cfg.Driver, db)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		user, err := repos.Auth.FindByLogin(ctx, "admin")
		if err != nil {
			t.Fatal(err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("Admin123!")); err != nil {
			t.Fatal(err)
		}
		stored, err := repos.User.FindByID(ctx, user.ID)
		if err != nil || len(stored.Roles) != 1 || stored.Roles[0] != adminRole {
			t.Fatalf("default admin role: %v %v", stored, err)
		}
		permissions, err := repos.Permission.GetPermissionsByUserID(ctx, user.ID)
		if err != nil || len(permissions) != 17 {
			t.Fatalf("default admin permissions: %v %v", permissions, err)
		}
		if err := database.Migrate(cfg, -6); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("seed rollback users=%d err=%v", count, err)
		}

		// Upgrading an existing installation must preserve its administrator and password.
		id, err := bootstrap.New(repos.User).Run(ctx, bootstrap.Request{FullName: "Existing Administrator", Username: "custom_admin", Password: "existing password"})
		if err != nil {
			t.Fatal(err)
		}
		if err := database.Migrate(cfg, 0); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("existing admin users=%d err=%v", count, err)
		}
		existing, err := repos.User.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte("existing password")); err != nil {
			t.Fatal(err)
		}
		if err := database.Migrate(cfg, -6); err != nil {
			t.Fatal(err)
		}
		if _, err := repos.User.FindByID(ctx, id); err != nil {
			t.Fatalf("rollback deleted pre-existing admin: %v", err)
		}
	})
}
