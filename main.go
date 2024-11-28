package main

import (
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
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(r)

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
