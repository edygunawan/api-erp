package database_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/app"
	"github.com/edygunawan/go-rest-api/internal/config"
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
)

func TestDynamicCredentialsIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		_, cfg = testutil.Isolate(t, cfg)
		ctx := context.Background()
		realPassword := cfg.Password
		cfg.Password = "wrong-static-password"
		cfg.AuthMode = "injected-token-provider"
		var calls atomic.Int32
		var unavailable atomic.Bool
		credentialError := errors.New("credentials unavailable")
		provider := database.CredentialsFunc(func(context.Context, database.Target) (database.Credentials, error) {
			calls.Add(1)
			if unavailable.Load() {
				return database.Credentials{}, credentialError
			}
			return database.Credentials{Password: realPassword}, nil
		})
		option := database.WithCredentialsProvider(provider)
		db, err := database.Open(ctx, cfg, option)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		db.SetMaxIdleConns(0)
		// Force new physical connections, proving that a startup token is not retained.
		var value int
		for i := 0; i < 3; i++ {
			if err := db.QueryRowContext(ctx, `SELECT 1`).Scan(&value); err != nil {
				t.Fatal(err)
			}
		}
		if calls.Load() < 4 {
			t.Fatal("pool reused a startup credential instead of invoking provider")
		}
		unavailable.Store(true)
		if err := db.QueryRowContext(ctx, `SELECT 1`).Scan(&value); !errors.Is(err, credentialError) {
			t.Fatal("provider failure did not stop connection")
		}
		unavailable.Store(false)
		if err := db.QueryRowContext(ctx, `SELECT 1`).Scan(&value); err != nil {
			t.Fatal("pool did not recover after provider refresh")
		}
		before := calls.Load()
		if err := database.Migrate(cfg, 0, option); err != nil {
			t.Fatal(err)
		}
		if calls.Load() <= before {
			t.Fatal("migrations bypassed credential provider")
		}
		before = calls.Load()
		_, err = app.BootstrapAdmin(ctx, config.Config{Database: cfg}, bootstrap.Request{FullName: "Dynamic Admin", Username: "dynamic-admin", Password: "admin-password"}, option)
		if err != nil {
			t.Fatal(err)
		}
		if calls.Load() <= before {
			t.Fatal("bootstrap bypassed credential provider")
		}
	})
}
