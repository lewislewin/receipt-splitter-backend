package handlers

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
)

// Receipt struct to simulate a database entity
type Receipt struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// In-memory "database"
var receipts = []Receipt{}

func CreateReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var newReceipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&newReceipt); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Mock ID generation
	newReceipt.ID = "R" + string(len(receipts)+1)
	receipts = append(receipts, newReceipt)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newReceipt)
}

func ParseReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Mock receipt parsing logic
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Receipt parsed"})
}

func GetAllReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch all receipts
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipts)
}

func GetReceiptByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, receipt := range receipts {
		if receipt.ID == id {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(receipt)
			return
		}
	}
	http.Error(w, "Receipt not found", http.StatusNotFound)
}
