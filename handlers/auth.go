package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"receipt-splitter-backend/db"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	MonzoID   string    `json:"monzo_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	// Insert user into the database
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

	user.Password = "" // Omit the password from the response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

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

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	user.Password = "" // Omit the password from the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
