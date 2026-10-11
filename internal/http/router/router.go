package router

import (
	"net/http"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/http/handler"
	"github.com/edygunawan/go-rest-api/internal/http/middleware"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	"github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	Logger             *logger.Logger
	Handlers           *handler.Handlers
	AuthUC             auth.Usecase
	CORSAllowedOrigins []string
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	// standard middlewares
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORSMiddleware(deps.CORSAllowedOrigins))
	r.Use(chiMiddleware.StripSlashes)
	r.Use(middleware.LoggerMiddleware(deps.Logger))

	// routes
	r.Get("/health", handler.HealtCheck)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", deps.Handlers.Auth.Login)
		r.Post("/refresh-token", deps.Handlers.Auth.RefreshToken)

		r.Group(func(r chi.Router) {
			// jwt middleware
			r.Use(middleware.AuthMiddleware(deps.AuthUC))

			r.Post("/logout", deps.Handlers.Auth.Logout)
			r.Get("/me", deps.Handlers.Auth.Me)
		})
	})

	r.Group(func(r chi.Router) {
		// jwt middleware
		r.Use(middleware.AuthMiddleware(deps.AuthUC))

		r.Route("/settings", func(r chi.Router) {
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.ApplicationSettingRead)).Get("/application", deps.Handlers.Setting.GetApplication)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.ApplicationSettingUpdate)).Put("/application", deps.Handlers.Setting.SaveApplication)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingCreate)).Post("/", deps.Handlers.Setting.CreateSetting)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingRead)).Get("/", deps.Handlers.Setting.ListSetting)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingRead)).Get("/{id}", deps.Handlers.Setting.GetSetting)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingUpdate)).Put("/{id}", deps.Handlers.Setting.UpdateSetting)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingDelete)).Delete("/{id}", deps.Handlers.Setting.DeleteSetting)
		})

		r.Route("/setting-groups", func(r chi.Router) {
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingCreate)).Post("/", deps.Handlers.SettingGroup.CreateSettingGroup)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingRead)).Get("/", deps.Handlers.SettingGroup.ListSettingGroup)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingRead)).Get("/{id}", deps.Handlers.SettingGroup.GetSettingGroup)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingUpdate)).Put("/{id}", deps.Handlers.SettingGroup.UpdateSettingGroup)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.SettingDelete)).Delete("/{id}", deps.Handlers.SettingGroup.DeleteSettingGroup)
		})

		r.Route("/users", func(r chi.Router) {
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.UserCreate)).Post("/", deps.Handlers.User.CreateUser)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.UserRead)).Get("/", deps.Handlers.User.GetAllUser)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.UserRead)).Get("/{id}", deps.Handlers.User.GetUser)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.UserUpdate)).Put("/{id}", deps.Handlers.User.UpdateUser)
		})

		r.Route("/roles", func(r chi.Router) {
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.RoleCreate)).Post("/", deps.Handlers.Role.CreateRole)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.RoleRead)).Get("/", deps.Handlers.Role.GetAllRoles)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.RoleRead)).Get("/{id}", deps.Handlers.Role.GetRole)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.RoleUpdate)).Put("/{id}", deps.Handlers.Role.UpdateRole)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.RoleDelete)).Delete("/{id}", deps.Handlers.Role.DeleteRole)
		})

		r.Route("/permissions", func(r chi.Router) {
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.PermissionCreate)).Post("/", deps.Handlers.Permission.CreatePermission)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.PermissionRead)).Get("/", deps.Handlers.Permission.GetAllPermissions)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.PermissionRead)).Get("/{id}", deps.Handlers.Permission.GetPermission)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.PermissionUpdate)).Put("/{id}", deps.Handlers.Permission.UpdatePermission)
			r.With(middleware.RequirePermission(deps.AuthUC, tokenauth.PermissionDelete)).Delete("/{id}", deps.Handlers.Permission.DeletePermission)
		})

	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusNotFound, "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	return r
}
