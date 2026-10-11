package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/joho/godotenv"
)

type Config struct {
	AppName            string
	AppEnv             string
	DebugMode          bool
	HttpPort           string
	Database           database.Config
	AutoMigrate        bool
	JWTSecret          string
	JWTIssuer          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	CORSAllowedOrigins []string
	SentryDSN          string
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func LoadConfig(envFiles ...string) (Config, error) {
	if len(envFiles) > 0 {
		_ = godotenv.Load(envFiles...)
	} else {
		_ = godotenv.Load()
	}
	cfg := Config{
		AppName: env("APP_NAME", "rest-api"), AppEnv: env("APP_ENV", "local"), HttpPort: env("HTTP_PORT", "8080"),
		JWTSecret: env("JWT_SECRET", ""), JWTIssuer: env("JWT_ISSUER", "go-rest-api"), SentryDSN: env("SENTRY_DSN", ""),
		Database: database.Config{
			Driver: env("DB_DRIVER", "mysql"), AuthMode: env("DB_AUTH_MODE", "password"),
			Host: env("DB_HOST", "127.0.0.1"), Port: env("DB_PORT", ""), Name: env("DB_NAME", "rest_api"),
			User: env("DB_USER", "rest_api"), Password: env("DB_PASSWORD", ""),
			PostgresSSLMode: env("POSTGRES_SSL_MODE", "disable"), MySQLTLSMode: env("MYSQL_TLS_MODE", "disable"), CAFile: env("DB_TLS_CA_FILE", ""),
		},
	}

	var err error
	parseBool := func(key string, fallback bool) (bool, error) {
		value := env(key, strconv.FormatBool(fallback))
		v, e := strconv.ParseBool(value)
		if e != nil {
			return false, fmt.Errorf("invalid %s", key)
		}
		return v, nil
	}
	cfg.DebugMode, err = parseBool("DEBUG_MODE", false)
	if err != nil {
		return cfg, err
	}
	cfg.AutoMigrate, err = parseBool("AUTO_MIGRATE", false)
	if err != nil {
		return cfg, err
	}
	parseInt := func(key string, fallback int) (int, error) {
		v, e := strconv.Atoi(env(key, strconv.Itoa(fallback)))
		if e != nil {
			return 0, fmt.Errorf("invalid %s", key)
		}
		return v, nil
	}
	cfg.Database.MaxOpenConns, err = parseInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return cfg, err
	}
	cfg.Database.MaxIdleConns, err = parseInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return cfg, err
	}
	lifetime, err := parseInt("DB_CONN_MAX_LIFETIME_SECONDS", 180)
	if err != nil || lifetime < 1 || lifetime > 86400 {
		return cfg, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME_SECONDS")
	}
	cfg.Database.ConnMaxLifetime = time.Duration(lifetime) * time.Second
	if cfg.Database.MaxOpenConns < 1 || cfg.Database.MaxIdleConns < 0 || cfg.Database.MaxIdleConns > cfg.Database.MaxOpenConns {
		return cfg, fmt.Errorf("invalid database pool limits")
	}
	cfg.Database, err = cfg.Database.Normalize()
	if err != nil {
		return cfg, err
	}
	access, err := parseInt("JWT_EXPIRE_MINUTES", 15)
	if err != nil {
		return cfg, err
	}
	refresh, err := parseInt("JWT_REFRESH_EXPIRE_MINUTES", 10080)
	if err != nil {
		return cfg, err
	}
	if access <= 0 || refresh <= access || refresh > 525600 {
		return cfg, fmt.Errorf("JWT lifetimes must be positive, refresh longer than access, and at most one year")
	}
	cfg.JWTAccessTTL = time.Duration(access) * time.Minute
	cfg.JWTRefreshTTL = time.Duration(refresh) * time.Minute
	port, err := strconv.Atoi(cfg.HttpPort)
	if err != nil || port < 1 || port > 65535 {
		return cfg, fmt.Errorf("invalid HTTP_PORT")
	}
	for _, origin := range strings.Split(env("CORS_ALLOWED_ORIGINS", ""), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, origin)
		}
	}
	return cfg, nil
}
