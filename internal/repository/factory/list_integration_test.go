package factory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/testutil"
)

func TestListCancellationAndValidation(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		_, _, repos := fixture(t, cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := repos.User.List(ctx, query.Options{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("user cancellation: %v", err)
		}
		_, _, err = repos.Role.List(ctx, query.Options{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("role cancellation: %v", err)
		}
		_, _, err = repos.Permission.List(ctx, query.Options{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("permission cancellation: %v", err)
		}
		_, _, err = repos.User.List(context.Background(), query.Options{Sort: "password;--"})
		if err == nil {
			t.Fatal("sort validation failed")
		}
	})
}
