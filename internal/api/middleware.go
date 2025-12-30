package api

import (
	"net/http"
	"os"
	"strings"
)

func CORSMiddleware(next http.Handler) http.Handler {
	raw := os.Getenv("ALLOWED_ORIGINS")
	origins := map[string]struct{}{}

	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = struct{}{}
		}
	}

	allowAll := false
	if _, ok := origins["*"]; ok {
		allowAll = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := origins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin") // Caching güvenliği için kritik
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				http.Error(w, "CORS origin denied", http.StatusForbidden)
				return
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight kontrolü
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
