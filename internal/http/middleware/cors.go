package middleware

import (
	"net/http"
	"slices"

	"github.com/edygunawan/go-rest-api/internal/http/response"
)

func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			w.Header().Add("Vary", "Origin")
			if origin != "" && (slices.Contains(origins, origin) || slices.Contains(origins, "*")) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Add("Vary", "Access-Control-Request-Method")
					w.Header().Add("Vary", "Access-Control-Request-Headers")
					if !slices.Contains([]string{"GET", "POST", "PUT", "DELETE"}, r.Header.Get("Access-Control-Request-Method")) {
						response.Error(w, http.StatusForbidden, "cors method not allowed")
						return
					}
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
					w.Header().Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			} else if origin != "" && r.Method == http.MethodOptions {
				response.Error(w, http.StatusForbidden, "cors origin not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
