package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	db.InitDB()
	handlers.InitOpenAIClient(os.Getenv("OPENAPI_API_KEY"))

	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	// Receipt routes (protected)
	r.Handle("/receipts", auth.JWTMiddleware(http.HandlerFunc(handlers.CreateReceiptHandler))).Methods("POST")
	r.Handle("/receipts/parse", auth.JWTMiddleware(http.HandlerFunc(handlers.ParseReceiptHandler))).Methods("POST")
	r.Handle("/receipts", auth.JWTMiddleware(http.HandlerFunc(handlers.GetAllReceiptsHandler))).Methods("GET")
	r.Handle("/receipts/{id}", http.HandlerFunc(handlers.GetReceiptByIDHandler)).Methods("GET")

	// CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}).Handler(r)

	host := os.Getenv("APP_PORT")

	log.Println(fmt.Sprintf("Server running on port %s", host))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", host), corsHandler))
}
