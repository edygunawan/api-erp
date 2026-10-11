package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"
)

// Open is the single connection factory for the API, migrations, and bootstrap.
func Open(ctx context.Context, cfg Config, opts ...Option) (*sql.DB, error) {
	connector, normalized, err := newConnector(cfg, opts...)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(normalized.MaxOpenConns)
	db.SetMaxIdleConns(normalized.MaxIdleConns)
	db.SetConnMaxLifetime(normalized.ConnMaxLifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, normalized.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
func newConnector(cfg Config, opts ...Option) (driver.Connector, Config, error) {
	cfg, err := cfg.Normalize()
	if err != nil {
		return nil, cfg, err
	}
	settings := options{}
	for _, opt := range opts {
		opt(&settings)
	}
	if settings.provider == nil {
		if cfg.AuthMode != PasswordAuth {
			return nil, cfg, errors.New("DB_AUTH_MODE requires an explicitly configured credentials provider")
		}
		settings.provider = passwordProvider{password: cfg.Password}
	}
	var connector driver.Connector
	switch cfg.Driver {
	case Postgres:
		connector, err = postgresConnector(cfg, settings)
	case MySQL:
		connector, err = mysqlConnector(cfg, settings)
	}
	return connector, cfg, err
}
func target(cfg Config) Target {
	return Target{Driver: cfg.Driver, Host: cfg.Host, Port: cfg.Port, Name: cfg.Name, User: cfg.User}
}
