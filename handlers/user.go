package handlers

import (
	"database/sql"
	"net/http"
	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/helpers"
	"time"
)

// GetCurrentUser retrieves and returns the currently authenticated user
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the request context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		helpers.JSONErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Query the user from the database
	var user struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		MonzoID   string    `json:"monzo_id"`
		CreatedAt time.Time `json:"created_at"`
	}
	query := `SELECT id, name, email, monzo_id, created_at FROM users WHERE id = $1`
	err := db.DB.QueryRow(query, userID).Scan(&user.ID, &user.Name, &user.Email, &user.MonzoID, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.JSONErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve user")
		}
		return
	}

	// Respond with the user data
	helpers.JSONResponse(w, http.StatusOK, user)
}
