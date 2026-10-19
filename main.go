package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func main() {
	mux := newRouter()

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("1.0.0"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok"))
}

type PaymentRequest struct {
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Payment struct {
	ID       string  `json:"id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

func createPaymentHandler(w http.ResponseWriter, r *http.Request) {
	var payment PaymentRequest

	err := json.NewDecoder(r.Body).Decode(&payment)

	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if payment.Amount <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return
	}

	if payment.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if payment.Currency == "" {
		http.Error(w, "currency is required", http.StatusBadRequest)
		return
	}

	createdPayment := Payment{
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	createdPayment.ID = uuid.NewString()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdPayment)
}

func newRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)

	r.Get("/version", versionHandler)

	r.Post("/payments", createPaymentHandler)

	return r
}
