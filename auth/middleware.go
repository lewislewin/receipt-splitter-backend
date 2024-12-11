package auth

import (
	"context"
	"net/http"
	"os"
	"receipt-splitter-backend/helpers"
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
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Server misconfigured: JWT secret missing")
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrUseLastResponse
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid token payload")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey{}).(string)
	return userID, ok
}
