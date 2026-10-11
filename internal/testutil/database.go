// Package testutil isolates integration databases and runs the same suite on
// every configured driver. It is used only by tests, never application wiring.
package testutil

import (
	"context"
	"database/sql"
	"flag"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/helper"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var postgresURL = flag.String("test-postgres-url", os.Getenv("TEST_POSTGRES_URL"), "PostgreSQL URL for isolated integration schemas")
var mysqlDSN = flag.String("test-mysql-dsn", os.Getenv("TEST_MYSQL_DSN"), "MySQL DSN for isolated integration databases")
var requireBoth = flag.Bool("require-test-databases", false, "fail if either integration database is missing")

func ForEachDatabase(t *testing.T, run func(*testing.T, database.Config)) {
	t.Helper()
	if *requireBoth && (*postgresURL == "" || *mysqlDSN == "") {
		t.Fatal("both TEST_POSTGRES_URL and TEST_MYSQL_DSN (or test flags) are required")
	}
	for _, driver := range []string{database.Postgres, database.MySQL} {
		t.Run(driver, func(t *testing.T) {
			cfg := database.Config{Driver: driver, MaxOpenConns: 10, MaxIdleConns: 5}
			switch driver {
			case database.Postgres:
				if *postgresURL == "" {
					t.Skip("set TEST_POSTGRES_URL or -test-postgres-url")
				}
				u, err := url.Parse(*postgresURL)
				if err != nil {
					t.Fatal("invalid test PostgreSQL URL")
				}
				cfg.Host = u.Hostname()
				cfg.Port = u.Port()
				cfg.Name = strings.TrimPrefix(u.Path, "/")
				if u.User != nil {
					cfg.User = u.User.Username()
					cfg.Password, _ = u.User.Password()
				}
				if host := u.Query().Get("host"); host != "" {
					cfg.Host = host
				}
				if port := u.Query().Get("port"); port != "" {
					cfg.Port = port
				}
				cfg.PostgresSSLMode = u.Query().Get("sslmode")
				cfg.CAFile = u.Query().Get("sslrootcert")
			case database.MySQL:
				if *mysqlDSN == "" {
					t.Skip("set TEST_MYSQL_DSN or -test-mysql-dsn")
				}
				parsed, err := mysql.ParseDSN(*mysqlDSN)
				if err != nil {
					t.Fatal("invalid test MySQL DSN")
				}
				if parsed.Net != "tcp" {
					t.Fatal("test MySQL DSN requires tcp")
				}
				cfg.Host, cfg.Port, err = net.SplitHostPort(parsed.Addr)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Name = parsed.DBName
				cfg.User = parsed.User
				cfg.Password = parsed.Passwd
				if parsed.TLSConfig == "true" {
					cfg.MySQLTLSMode = "verify-full"
				} else if parsed.TLSConfig != "" && parsed.TLSConfig != "false" {
					t.Fatal("test MySQL DSN supports only verified TLS or disabled TLS")
				}
			}
			cfg, err := cfg.Normalize()
			if err != nil {
				t.Fatal(err)
			}
			run(t, cfg)
		})
	}
}

// Isolate creates only a unique schema/database; supplied base databases are
// never migrated or dropped. Cleanup closes pools before removing that namespace.
func Isolate(t *testing.T, cfg database.Config) (*sql.DB, database.Config) {
	t.Helper()
	ctx := context.Background()
	base, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	id, err := helper.NewUUID()
	if err != nil {
		_ = base.Close()
		t.Fatal(err)
	}
	name := "foundation_test_" + strings.ReplaceAll(id, "-", "")
	create := `CREATE SCHEMA ` + name
	drop := `DROP SCHEMA ` + name + ` CASCADE`
	if cfg.Driver == database.MySQL {
		create = "CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"
		drop = "DROP DATABASE `" + name + "`"
	}
	if _, err := base.ExecContext(ctx, create); err != nil {
		_ = base.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := base.ExecContext(context.Background(), drop); err != nil {
			t.Error(err)
		}
		_ = base.Close()
	})
	if cfg.Driver == database.Postgres {
		cfg.Schema = name
	} else {
		cfg.Name = name
	}
	if err := database.Migrate(cfg, 0); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(cfg, 0); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	db, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, cfg
}

// Integration SQL is written with question marks; production repositories keep
// driver-specific SQL and semantics instead of hiding dialect differences here.
func Bind(driver, query string) string {
	if driver == database.Postgres {
		return sqlx.Rebind(sqlx.DOLLAR, query)
	}
	return query
}
