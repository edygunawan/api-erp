package app

import (
	"context"
	"database/sql"
	"errors"
	stdhttp "net/http"
	"time"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/config"
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/http"
	"github.com/edygunawan/go-rest-api/internal/http/handler"
	"github.com/edygunawan/go-rest-api/internal/http/router"
	"github.com/edygunawan/go-rest-api/internal/repository/factory"
	authuc "github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/go-rest-api/internal/usecase/permission"
	"github.com/edygunawan/go-rest-api/internal/usecase/role"
	"github.com/edygunawan/go-rest-api/internal/usecase/setting"
	"github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	"github.com/edygunawan/go-rest-api/internal/usecase/user"
	"github.com/edygunawan/golang-pkgs/logger"
)

type App struct {
	db     *sql.DB
	logger *logger.Logger
	server *http.Server
}

func buildHTTPHandlers(repos factory.Repositories, token tokenauth.TokenService) (*handler.Handlers, authuc.Usecase) {
	authentication := authuc.NewUsecase(repos.Auth, repos.Permission, token)
	return &handler.Handlers{
		Setting:      handler.NewSettingHandler(setting.New(repos.Setting)),
		SettingGroup: handler.NewSettingGroupHandler(settinggroup.New(repos.SettingGroup)),
		User:         handler.NewUserHandler(user.New(repos.User)),
		Role:         handler.NewRoleHandler(role.New(repos.Role)),
		Permission:   handler.NewPermissionHandler(permission.New(repos.Permission)),
		Auth:         handler.NewAuthHandler(authentication),
	}, authentication
}

func New(cfg config.Config, options ...database.Option) (*App, error) {
	token, err := tokenauth.NewJWTTokenService(tokenauth.JWTConfig{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, AccessTTL: cfg.JWTAccessTTL, RefreshTTL: cfg.JWTRefreshTTL})
	if err != nil {
		return nil, err
	}
	log := logger.New(cfg.AppName, cfg.DebugMode)
	logger.SetDefault(log)
	if cfg.SentryDSN != "" {
		if err := logger.InitSentry(cfg.SentryDSN, cfg.AppEnv, true); err != nil {
			log.Error(err, "sentry initialization failed")
		} else {
			log.SetReporter(logger.SentryReporter{})
		}
	}
	dbCfg, err := cfg.Database.Normalize()
	if err != nil {
		return nil, err
	}
	db, err := database.Open(context.Background(), dbCfg, options...)
	if err != nil {
		return nil, err
	}
	if cfg.AutoMigrate {
		if err := database.Migrate(dbCfg, 0, options...); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	repos, err := factory.New(dbCfg.Driver, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	handlers, authentication := buildHTTPHandlers(repos, token)
	server := http.NewServer(http.Config{Port: cfg.HttpPort}, log, router.Dependencies{Logger: log, Handlers: handlers, AuthUC: authentication, CORSAllowedOrigins: cfg.CORSAllowedOrigins})
	return &App{db: db, logger: log, server: server}, nil
}
func (a *App) Run(ctx context.Context) error {
	defer a.db.Close()
	a.logger.Info("application started")
	errCh := make(chan error, 1)
	go func() { errCh <- a.server.Start() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, stdhttp.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if errors.Is(err, stdhttp.ErrServerClosed) {
			return nil
		}
		return err
	}
}
