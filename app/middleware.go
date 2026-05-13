package app

import (
	"context"
	"net/http"
	"strings"
	"todo/config"
)

type contextKey string

const ctxAuthKey contextKey = "authenticated"

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !config.Get().HasToken() {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxAuthKey, true)))
			return
		}
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !config.Get().VerifyToken(strings.TrimPrefix(authHeader, "Bearer ")) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxAuthKey, true)))
	})
}
