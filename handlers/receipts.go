package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"receipt-splitter-backend/db"
	"receipt-splitter-backend/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func CreateReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var receipt models.ParsedReceipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	receipt.ID = uuid.New().String()
	receipt.CreatedAt = time.Now()

	// Insert receipt into the database
	query := `INSERT INTO parsed_receipts (id, name, monzo_id, reason, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.DB.Exec(query, receipt.ID, receipt.Name, receipt.MonzoID, receipt.Reason, receipt.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to store receipt", http.StatusInternalServerError)
		return
	}

	// Insert associated items
	for _, item := range receipt.Items {
		item.ID = uuid.New().String()
		query = `INSERT INTO receipt_items (id, item, price, qty) VALUES ($1, $2, $3, $4)`
		_, err := db.DB.Exec(query, item.ID, item.Item, item.Price, item.Qty)
		if err != nil {
			http.Error(w, "Failed to store receipt items", http.StatusInternalServerError)
			return
		}

		// Link items to the receipt
		query = `INSERT INTO parsed_receipts_items (parsed_receipt_id, receipt_item_id) VALUES ($1, $2)`
		_, err = db.DB.Exec(query, receipt.ID, item.ID)
		if err != nil {
			http.Error(w, "Failed to link receipt items", http.StatusInternalServerError)
			return
		}
	}

	// Insert associated modifiers
	for _, modifier := range receipt.Modifiers {
		modifier.ID = uuid.New().String()
		query = `INSERT INTO modifiers (id, type, value, percentage, include) VALUES ($1, $2, $3, $4, $5)`
		_, err := db.DB.Exec(query, modifier.ID, modifier.Type, modifier.Value, modifier.Percentage, modifier.Include)
		if err != nil {
			http.Error(w, "Failed to store receipt modifiers", http.StatusInternalServerError)
			return
		}

		// Link modifiers to the receipt
		query = `INSERT INTO parsed_receipts_modifiers (parsed_receipt_id, modifier_id) VALUES ($1, $2)`
		_, err = db.DB.Exec(query, receipt.ID, modifier.ID)
		if err != nil {
			http.Error(w, "Failed to link receipt modifiers", http.StatusInternalServerError)
			return
		}
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
