package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"errors"
	"net"
	"os"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

func mysqlConnector(cfg Config, settings options) (driver.Connector, error) {
	connection := mysql.NewConfig()
	connection.User = cfg.User
	connection.Net = "tcp"
	connection.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	connection.DBName = cfg.Name
	connection.ParseTime = true
	connection.Loc = time.UTC
	connection.Timeout = cfg.ConnectTimeout
	// Matched rows make unchanged updates have the same contract as PostgreSQL.
	connection.ClientFoundRows = true
	connection.MultiStatements = settings.migrations
	connection.Params = map[string]string{"time_zone": "'+00:00'"}
	if cfg.MySQLTLSMode == "verify-full" {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: cfg.Host}
		if cfg.CAFile != "" {
			ca, err := os.ReadFile(cfg.CAFile)
			if err != nil {
				return nil, errors.New("could not read DB_TLS_CA_FILE")
			}
			roots, err := x509.SystemCertPool()
			if err != nil {
				roots = x509.NewCertPool()
			}
			if !roots.AppendCertsFromPEM(ca) {
				return nil, errors.New("invalid DB_TLS_CA_FILE")
			}
			tlsConfig.RootCAs = roots
		}
		connection.TLS = tlsConfig
	}
	if err := connection.Apply(mysql.BeforeConnect(func(ctx context.Context, connection *mysql.Config) error {
		credentials, err := settings.provider.Credentials(ctx, target(cfg))
		if err != nil {
			return err
		}
		if credentials.AllowCleartext && cfg.MySQLTLSMode != "verify-full" {
			return errors.New("cleartext token authentication requires verified TLS")
		}
		connection.Passwd = credentials.Password
		connection.AllowCleartextPasswords = credentials.AllowCleartext
		return nil
	})); err != nil {
		return nil, err
	}
	return mysql.NewConnector(connection)
}
