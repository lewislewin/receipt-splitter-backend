package auth

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// UserIDKey is the context key for the authenticated user's ID
type UserIDKey struct{}

// JWTMiddleware validates the JWT token and adds the user ID to the request context
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		// Extract the token from the header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		// Get JWT secret dynamically
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			http.Error(w, "Server misconfigured: JWT secret missing", http.StatusInternalServerError)
			return
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrUseLastResponse
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract user ID from the token claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}

		// Add the user ID to the request context
		ctx := context.WithValue(r.Context(), UserIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey{}).(string)
	return userID, ok
}
