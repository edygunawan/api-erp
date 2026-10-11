package config

import "testing"

func defaults(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{"APP_NAME": "rest-api", "APP_ENV": "local", "HTTP_PORT": "8080", "DB_DRIVER": "mysql", "DB_AUTH_MODE": "password", "DB_HOST": "127.0.0.1", "DB_PORT": "", "DB_NAME": "rest_api", "DB_USER": "rest_api", "DB_PASSWORD": "", "POSTGRES_SSL_MODE": "disable", "MYSQL_TLS_MODE": "disable", "DB_TLS_CA_FILE": "", "DB_CONN_MAX_LIFETIME_SECONDS": "180", "JWT_SECRET": "", "JWT_ISSUER": "go-rest-api", "SENTRY_DSN": "", "DEBUG_MODE": "false", "AUTO_MIGRATE": "false", "DB_MAX_OPEN_CONNS": "25", "DB_MAX_IDLE_CONNS": "10", "JWT_EXPIRE_MINUTES": "15", "JWT_REFRESH_EXPIRE_MINUTES": "10080", "CORS_ALLOWED_ORIGINS": ""} {
		t.Setenv(key, value)
	}
}

func TestInvalidConfigValues(t *testing.T) {
	for _, tc := range []struct{ key, value string }{{"DB_DRIVER", "sqlite"}, {"DB_PORT", "0"}, {"DB_PORT", "65536"}, {"DB_HOST", ""}, {"DB_NAME", ""}, {"DB_USER", ""}, {"MYSQL_TLS_MODE", "skip-verify"}, {"DB_CONN_MAX_LIFETIME_SECONDS", "0"}, {"DEBUG_MODE", "invalid"}, {"AUTO_MIGRATE", "invalid"}, {"HTTP_PORT", "0"}, {"HTTP_PORT", "65536"}, {"DB_MAX_OPEN_CONNS", "0"}, {"DB_MAX_IDLE_CONNS", "26"}, {"JWT_EXPIRE_MINUTES", "0"}, {"JWT_REFRESH_EXPIRE_MINUTES", "1"}, {"JWT_REFRESH_EXPIRE_MINUTES", "525601"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			defaults(t)
			t.Setenv(tc.key, tc.value)
			if _, err := LoadConfig("/nonexistent-foundation-env"); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
func TestConfigGenericDefaults(t *testing.T) {
	defaults(t)
	cfg, err := LoadConfig("/nonexistent-foundation-env")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JWTIssuer != "go-rest-api" || cfg.AutoMigrate || cfg.Database.Driver != "mysql" || cfg.Database.Port != "3306" || cfg.Database.Password != "" || cfg.JWTSecret != "" || len(cfg.CORSAllowedOrigins) != 0 {
		t.Fatal("unexpected defaults")
	}
}

func TestPostgresConfiguration(t *testing.T) {
	defaults(t)
	t.Setenv("DB_DRIVER", "postgres")
	cfg, err := LoadConfig("/nonexistent-foundation-env")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Port != "5432" {
		t.Fatal("PostgreSQL default port missing")
	}
	t.Setenv("POSTGRES_SSL_MODE", "invalid")
	if _, err := LoadConfig("/nonexistent-foundation-env"); err == nil {
		t.Fatal("invalid PostgreSQL TLS mode accepted")
	}
}
