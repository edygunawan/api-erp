package database

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func configFor(driver string) Config {
	return Config{Driver: driver, Host: "localhost", Name: "test_db", User: "test_user", Password: "unused-static-password", MaxOpenConns: 5, MaxIdleConns: 2}
}
func TestProviderCalledForEveryPhysicalConnection(t *testing.T) {
	for _, driver := range []string{Postgres, MySQL} {
		t.Run(driver, func(t *testing.T) {
			cfg := configFor(driver)
			cfg.AuthMode = "future-token-auth"
			stopped := errors.New("credentials temporarily unavailable")
			var calls atomic.Int32
			provider := CredentialsFunc(func(ctx context.Context, request Target) (Credentials, error) {
				calls.Add(1)
				if request.Driver != driver || request.Host != cfg.Host || request.User != cfg.User || request.Name != cfg.Name || request.Port == "" {
					t.Error("wrong credential target")
				}
				return Credentials{}, stopped
			})
			connector, _, err := newConnector(cfg, WithCredentialsProvider(provider))
			if err != nil {
				t.Fatal(err)
			}
			const attempts = 8
			var wg sync.WaitGroup
			for i := 0; i < attempts; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := connector.Connect(context.Background())
					if !errors.Is(err, stopped) {
						t.Error("credential failure fell back to static password")
					}
				}()
			}
			wg.Wait()
			if calls.Load() != attempts {
				t.Fatalf("provider calls=%d", calls.Load())
			}
		})
	}
}
func TestProviderContextCancellation(t *testing.T) {
	for _, driver := range []string{Postgres, MySQL} {
		cfg := configFor(driver)
		connector, _, err := newConnector(cfg, WithCredentialsProvider(CredentialsFunc(func(ctx context.Context, _ Target) (Credentials, error) { return Credentials{}, ctx.Err() })))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = connector.Connect(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("provider did not receive cancellation")
		}
	}
}
func TestUnknownAuthModeRequiresProvider(t *testing.T) {
	cfg := configFor(MySQL)
	cfg.AuthMode = "aws_rds_iam"
	if _, _, err := newConnector(cfg); err == nil {
		t.Fatal("unsupported auth mode silently used password")
	}
}
func TestMySQLCleartextTokensRequireVerifiedTLS(t *testing.T) {
	connector, _, err := newConnector(configFor(MySQL), WithCredentialsProvider(CredentialsFunc(func(context.Context, Target) (Credentials, error) {
		return Credentials{Password: "token", AllowCleartext: true}, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	_, err = connector.Connect(context.Background())
	if err == nil || !strings.Contains(err.Error(), "verified TLS") {
		t.Fatal("token authentication allowed without verified TLS")
	}
}
func TestConnectionConfigRejectsUnknownDriver(t *testing.T) {
	cfg := configFor("sqlite")
	if _, _, err := newConnector(cfg); err == nil {
		t.Fatal("unsupported driver accepted")
	}
}
