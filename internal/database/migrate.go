package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/edygunawan/go-rest-api/migrations"
	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	mysql "github.com/golang-migrate/migrate/v4/database/mysql"
	postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Migrate(cfg Config, steps int, opts ...Option) error {
	return MigrateContext(context.Background(), cfg, steps, opts...)
}

// Migrations open their own pool with the same credentials provider as the API.
// Multi-statement execution is enabled only on the MySQL migration connection.
func MigrateContext(ctx context.Context, cfg Config, steps int, opts ...Option) (err error) {
	cfg, err = cfg.Normalize()
	if err != nil {
		return err
	}
	migrationOptions := append([]Option{}, opts...)
	migrationOptions = append(migrationOptions, func(o *options) { o.migrations = true })
	db, err := Open(ctx, cfg, migrationOptions...)
	if err != nil {
		return err
	}
	var driver migratedb.Driver
	switch cfg.Driver {
	case Postgres:
		driver, err = postgres.WithInstance(db, &postgres.Config{})
	case MySQL:
		driver, err = mysql.WithInstance(db, &mysql.Config{DatabaseName: cfg.Name})
	}
	if err != nil {
		_ = db.Close()
		return err
	}
	source, err := iofs.New(migrations.Files, cfg.Driver)
	if err != nil {
		_ = driver.Close()
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, cfg.Driver, driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()
		return err
	}
	defer func() {
		se, de := m.Close()
		if err == nil {
			err = errors.Join(se, de)
		}
	}()
	if steps == 0 {
		err = m.Up()
	} else {
		err = m.Steps(steps)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
