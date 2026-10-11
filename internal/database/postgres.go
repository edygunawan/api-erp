package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"net"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func postgresConnector(cfg Config, settings options) (driver.Connector, error) {
	// Use a URL builder rather than interpolating credentials or keyword values.
	// Passwords/tokens are supplied only inside the per-connection callback.
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort("localhost", cfg.Port), Path: "/" + cfg.Name, User: url.User(cfg.User)}
	q := u.Query()
	q.Set("host", cfg.Host)
	q.Set("port", cfg.Port)
	q.Set("sslmode", cfg.PostgresSSLMode)
	if cfg.CAFile != "" {
		q.Set("sslrootcert", cfg.CAFile)
	}
	if cfg.Schema != "" {
		q.Set("search_path", cfg.Schema)
	}
	u.RawQuery = q.Encode()
	parsed, err := pgx.ParseConfig(u.String())
	if err != nil {
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}
	// pgx also reads PGPASSWORD; clear it so only the selected provider is used.
	parsed.Password = ""
	parsed.ConnectTimeout = cfg.ConnectTimeout
	return stdlib.GetConnector(*parsed, stdlib.OptionBeforeConnect(func(ctx context.Context, connection *pgx.ConnConfig) error {
		credentials, err := settings.provider.Credentials(ctx, target(cfg))
		if err != nil {
			return err
		}
		connection.Password = credentials.Password
		return nil
	})), nil
}
