package helpers

import (
	"net/http"
	"os"
	"strings"
)

func ValidateAdminToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if parts[1] != os.Getenv("ADMIN_TOKEN") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
		next(w, r)
	}
}
