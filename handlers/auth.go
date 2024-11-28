package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"receipt-splitter-backend/db"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Omit from responses
	MonzoID   string    `json:"monzo_id"`
	CreatedAt time.Time `json:"created_at"`
}

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
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate input fields
	if input.Name == "" || input.Email == "" || input.Password == "" {
		http.Error(w, "Name, email, and password are required", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Create the user struct
	user := User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword), // Store the hash
		MonzoID:  input.MonzoID,
	}

	// Insert the user into the database
	query := `INSERT INTO users (name, email, password, monzo_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err = db.DB.QueryRow(query, user.Name, user.Email, user.Password, user.MonzoID).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Omit the password from the response
	user.Password = ""

	// Return the created user
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate input
	if credentials.Email == "" || credentials.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Fetch user from the database
	var user User
	query := `SELECT id, name, email, password, monzo_id, created_at FROM users WHERE email = $1`
	err := db.DB.QueryRow(query, credentials.Email).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.MonzoID, &user.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Failed to query user", http.StatusInternalServerError)
		return
	}

	// Debug: Log retrieved user
	fmt.Println("Retrieved user from DB:", user)

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		fmt.Println("Password comparison failed:", err)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Omit password from the response
	user.Password = ""

	// Return successful login
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
