package handlers

import (
	"net/http"
	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/helpers"
	"receipt-splitter-backend/models"
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
	var user models.User
	err := db.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to query user")
		return
	}

	// Respond with the user data
	helpers.JSONResponse(w, http.StatusOK, user)
}
