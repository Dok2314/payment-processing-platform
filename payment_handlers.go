package main

import (
	"encoding/json"
	"log"
	"net/http"
	applicationpayment "payment-processing-platform/internal/application/payment"
)

func createPaymentHTTPHandler(service *applicationpayment.PaymentService, w http.ResponseWriter, r *http.Request) {
	var paymentRequest applicationpayment.PaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&paymentRequest); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Idempotency-Key")
	payment, err := service.Create(r.Context(), key, paymentRequest)
	if err != nil {
		http.Error(w, "failed to create payment", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(payment); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}
