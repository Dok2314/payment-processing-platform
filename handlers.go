package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("1.0.0"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok"))
}

func createPaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		http.Error(w, "invalid header", http.StatusBadRequest)
		return
	}

	var payment PaymentRequest

	err := json.NewDecoder(r.Body).Decode(&payment)

	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	createdPayment, err := service.Create(key, payment)

	if err != nil {
		if errors.Is(err, ErrInvalidPayment) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("create payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdPayment)
}

func getPaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	payment, ok := service.Get(id)
	if !ok {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func updatePaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var payment PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	updatedPayment, err := service.Update(id, payment)

	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if errors.Is(err, ErrInvalidPayment) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("update payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedPayment)
}

func deletePaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !service.Delete(id) {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
