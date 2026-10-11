package middleware

import (
	"context"
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/http/response"
)

type PermissionChecker interface {
	HasPermission(context.Context, string) (bool, error)
}

func RequirePermission(checker PermissionChecker, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := auth.GetIdentity(r.Context())
			if !ok || id == nil {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			allowed, err := checker.HasPermission(r.Context(), permission)
			if err != nil {
				response.FromError(w, err)
				return
			}
			if !allowed {
				response.Error(w, http.StatusForbidden, "permission denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
