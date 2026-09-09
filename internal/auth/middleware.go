package auth

import (
	"context"
	"net/http"
	"strings"

	"ticket-system/internal/httpx"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// RequireAuth returns middleware that validates the Authorization:
// Bearer <token> header and injects the authenticated user id into the
// request context. Requests without a valid token get 401 Unauthorized.
func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				httpx.WriteError(w, http.StatusUnauthorized, "Authorization header must use Bearer scheme")
				return
			}
			tokenString := strings.TrimPrefix(authHeader, prefix)

			claims, err := ParseToken(tokenString, secret)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user id set by RequireAuth.
// The bool is false if called on a request that didn't go through the
// middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}
