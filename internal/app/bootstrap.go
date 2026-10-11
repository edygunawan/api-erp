package app

import (
	"context"

	"github.com/edygunawan/go-rest-api/internal/config"
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/repository/factory"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
)

func BootstrapAdmin(ctx context.Context, cfg config.Config, req bootstrap.Request, options ...database.Option) (string, error) {
	dbCfg, err := cfg.Database.Normalize()
	if err != nil {
		return "", err
	}
	db, err := database.Open(ctx, dbCfg, options...)
	if err != nil {
		return "", err
	}
	defer db.Close()
	repos, err := factory.New(dbCfg.Driver, db)
	if err != nil {
		return "", err
	}
	return bootstrap.New(repos.User).Run(ctx, req)
}
