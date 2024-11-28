package main

import (
	"log"
	"net/http"

	"receipt-splitter-backend/db"
	"receipt-splitter-backend/handlers"

	"github.com/gorilla/mux"
)

func main() {
	db.InitDB()

	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	// Receipt routes
	r.HandleFunc("/receipts", handlers.CreateReceiptHandler).Methods("POST")
	r.HandleFunc("/receipts/parse", handlers.ParseReceiptHandler).Methods("POST")
	r.HandleFunc("/receipts", handlers.GetAllReceiptsHandler).Methods("GET")
	r.HandleFunc("/receipts/{id}", handlers.GetReceiptByIDHandler).Methods("GET")

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
