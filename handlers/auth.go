package handlers

import (
	"encoding/json"
	"net/http"
	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/helpers"
	"receipt-splitter-backend/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterInput represents the input for the RegisterHandler
type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	MonzoID  string `json:"monzo_id"`
}

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Invalid input")
		return
	}

	// Validate input fields
	if input.Name == "" || input.Email == "" || input.Password == "" {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Name, email, and password are required")
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	// Create the user struct
	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword), // Store the hash
		MonzoID:  input.MonzoID,
	}

	// Insert the user into the database using GORM
	err = db.DB.Create(&user).Error
	if err != nil {
		if gorm.ErrDuplicatedKey == err {
			helpers.JSONErrorResponse(w, http.StatusConflict, "Email already exists")
			return
		}
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Omit the password from the response
	user.Password = ""

	// Return the created user
	helpers.JSONResponse(w, http.StatusCreated, user)
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Decode request
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Invalid input")
		return
	}

	// Validate input
	if credentials.Email == "" || credentials.Password == "" {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Fetch user from the database using GORM
	var user models.User
	err := db.DB.Where("email = ?", credentials.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to query user")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Generate JWT
	token, err := auth.GenerateJWT(user.ID)
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Omit password from the response
	user.Password = ""

	// Respond with user info and token
	helpers.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}
