package middleware

import (
	"net/http"

	"github.com/edygunawan/golang-pkgs/logger"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func LoggerMiddleware(baseLog *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			reqLog := baseLog.WithMeta(map[string]any{
				"trace_id": chiMiddleware.GetReqID(r.Context()),
				"method":   r.Method,
				"path":     r.URL.Path,
			})

			ctx := logger.WithContext(r.Context(), reqLog)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
