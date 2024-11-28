package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func CreateReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context
	_, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Proceed with creating the receipt, associating it with the user ID
	var receipt models.ParsedReceipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Use userID to associate the receipt with the authenticated user
	receipt.ID = uuid.New().String()
	receipt.CreatedAt = time.Now()

	// Insert into the database (example)
	query := `INSERT INTO parsed_receipts (id, name, monzo_id, reason, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.DB.Exec(query, receipt.ID, receipt.Name, receipt.MonzoID, receipt.Reason, receipt.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to store receipt", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(receipt)
}

func ParseReceiptHandler(w http.ResponseWriter, r *http.Request) {
	type ParseRequest struct {
		Receipt string `json:"receipt"`
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	base64Image := req.Receipt
	if base64Image == "" {
		http.Error(w, "Receipt image in Base64 format is required", http.StatusBadRequest)
		return
	}

	if strings.HasPrefix(base64Image, "receipt:") {
		base64Image = strings.TrimPrefix(base64Image, "receipt:")
	}
	if strings.HasPrefix(base64Image, "data:image") {
		base64Image = strings.Split(base64Image, ",")[1]
	}

	_, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		http.Error(w, "Invalid Base64 image data", http.StatusBadRequest)
		return
	}

	// Simulate OCR and parsing
	parsed := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"item": "Milk", "price": 1.5, "qty": 2},
		},
		"modifiers": []interface{}{
			map[string]interface{}{"type": "discount", "value": 2.0},
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(parsed)
}

func GetAllReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`SELECT id, name, monzo_id, reason, created_at FROM parsed_receipts`)
	if err != nil {
		http.Error(w, "Failed to fetch receipts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var receipts []models.ParsedReceipt
	for rows.Next() {
		var receipt models.ParsedReceipt
		err := rows.Scan(&receipt.ID, &receipt.Name, &receipt.MonzoID, &receipt.Reason, &receipt.CreatedAt)
		if err != nil {
			http.Error(w, "Failed to parse receipt data", http.StatusInternalServerError)
			return
		}
		receipts = append(receipts, receipt)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipts)
}

func GetReceiptByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var receipt models.ParsedReceipt
	query := `SELECT id, name, monzo_id, reason, created_at FROM parsed_receipts WHERE id = $1`
	err := db.DB.QueryRow(query, id).Scan(&receipt.ID, &receipt.Name, &receipt.MonzoID, &receipt.Reason, &receipt.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Receipt not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve receipt", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipt)
}
