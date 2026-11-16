package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	applicationpayment "payment-processing-platform/internal/application/payment"
)

func createPaymentHTTPHandler(
	service *applicationpayment.PaymentService,
	w http.ResponseWriter,
	r *http.Request,
) {
	var paymentRequest applicationpayment.PaymentRequest

	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" {
		http.Error(w, "invalid Idempotency-Key", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&paymentRequest); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	payment, err := service.Create(r.Context(), key, paymentRequest)
	if err != nil {
		if errors.Is(err, applicationpayment.ErrIdempotencyKeyReuse) {
			http.Error(w, "Idempotency key reused with different request payload", http.StatusConflict)
			return
		}

		http.Error(w, "failed to create payment", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(payment); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}
