package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/http/response"
)

type Authenticator interface {
	Authenticate(context.Context, string) (*auth.Identity, error)
}

func AuthMiddleware(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			id, err := authenticator.Authenticate(r.Context(), parts[1])
			if err != nil {
				response.FromError(w, err)
				return
			}
			if id == nil || id.UserID == "" {
				response.Error(w, http.StatusUnauthorized, "invalid session")
				return
			}
			ctx := context.WithValue(r.Context(), auth.IdentityKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
