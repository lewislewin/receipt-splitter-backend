package handlers

import (
	"net/http"
	"receipt-splitter-backend/helpers"
)

// HealthCheckHandler handles health check requests
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	helpers.JSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// HealthCheckHandler handles health check requests
func HealthCheckAuthHandler(w http.ResponseWriter, r *http.Request) {
	helpers.JSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}
