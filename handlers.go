package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("1.0.0"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok"))
}

func createPaymentHandler(store *PaymentStore, w http.ResponseWriter, r *http.Request) {
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

	store.Save(createdPayment)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdPayment)
}
