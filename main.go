package main

import (
	"log"
	"net/http"

	"receipt-splitter-backend/auth"
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

	// Receipt routes (protected)
	r.Handle("/receipts", auth.JWTMiddleware(http.HandlerFunc(handlers.CreateReceiptHandler))).Methods("POST")
	r.Handle("/receipts/parse", auth.JWTMiddleware(http.HandlerFunc(handlers.ParseReceiptHandler))).Methods("POST")
	r.Handle("/receipts", auth.JWTMiddleware(http.HandlerFunc(handlers.GetAllReceiptsHandler))).Methods("GET")
	r.Handle("/receipts/{id}", http.HandlerFunc(handlers.GetReceiptByIDHandler)).Methods("GET")

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
