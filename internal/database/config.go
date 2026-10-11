package database

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	Postgres     = "postgres"
	MySQL        = "mysql"
	PasswordAuth = "password"
)

type Config struct {
	Driver          string
	AuthMode        string
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnectTimeout  time.Duration
	// TLS settings stay driver-specific; CAFile is shared.
	PostgresSSLMode string
	MySQLTLSMode    string
	CAFile          string
	// Schema is used for isolated PostgreSQL integration tests and custom schemas.
	Schema string
}

func (cfg Config) Normalize() (Config, error) {
	cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
	if cfg.Driver == "" {
		cfg.Driver = MySQL
	}
	switch cfg.Driver {
	case MySQL:
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
	case Postgres:
		if cfg.Port == "" {
			cfg.Port = "5432"
		}
	default:
		return cfg, fmt.Errorf("unsupported DB_DRIVER %q", cfg.Driver)
	}
	cfg.AuthMode = strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	if cfg.AuthMode == "" {
		cfg.AuthMode = PasswordAuth
	}
	if cfg.Host == "" || cfg.Name == "" || cfg.User == "" {
		return cfg, errors.New("DB_HOST, DB_NAME and DB_USER are required")
	}
	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return cfg, errors.New("invalid DB_PORT")
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxOpenConns < 1 || cfg.MaxIdleConns < 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
		return cfg, errors.New("invalid database pool limits")
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 3 * time.Minute
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.ConnMaxLifetime < 0 || cfg.ConnectTimeout < 0 {
		return cfg, errors.New("database timeouts must be positive")
	}
	if cfg.PostgresSSLMode == "" {
		cfg.PostgresSSLMode = "disable"
	}
	if cfg.MySQLTLSMode == "" {
		cfg.MySQLTLSMode = "disable"
	}
	if cfg.Driver == Postgres {
		switch cfg.PostgresSSLMode {
		case "disable", "require", "verify-ca", "verify-full":
		default:
			return cfg, errors.New("unsupported POSTGRES_SSL_MODE")
		}
	} else {
		switch cfg.MySQLTLSMode {
		case "disable", "verify-full":
		default:
			return cfg, errors.New("MYSQL_TLS_MODE must be disable or verify-full")
		}
	}
	return cfg, nil
}
