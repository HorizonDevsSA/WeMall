// Package middleware provides HTTP middleware for the GraphQL gateway.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyRole   contextKey = "role"
)

// Claims is the JWT payload shape issued by user-service.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Auth returns an HTTP middleware that extracts and validates a Bearer JWT.
// On success it injects userID and role into the request context.
// On failure (missing / invalid token) the request continues unauthenticated —
// field-level @hasRole directives enforce access control.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				claims := &Claims{}
				token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
					return []byte(jwtSecret), nil
				})
				if err == nil && token.Valid {
					ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
					ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromCtx extracts the authenticated user ID from context.
// Returns ("", false) when the request is unauthenticated.
func UserIDFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(string)
	return v, ok && v != ""
}

// RoleFromCtx extracts the authenticated user role from context.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}
