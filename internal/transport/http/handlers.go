package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	applicationpayment "payment-processing-platform/internal/application/payment"

	"github.com/go-chi/chi/v5"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("1.0.0"))
}

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

	if err := json.NewEncoder(w).Encode(newPaymentResponse(payment)); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}

func getPaymentHTTPHandler(
	service *applicationpayment.PaymentService,
	w http.ResponseWriter,
	r *http.Request,
) {
	id := chi.URLParam(r, "id")

	payment, err := service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, applicationpayment.ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to get payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(newPaymentResponse(payment)); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}

func moveToProcessingHTTPHandler(
	service *applicationpayment.PaymentService,
	w http.ResponseWriter,
	r *http.Request,
) {
	id := chi.URLParam(r, "id")

	payment, err := service.Process(r.Context(), id)
	if err != nil {
		if errors.Is(err, applicationpayment.ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}

		if errors.Is(err, applicationpayment.ErrPaymentCannotBeProcessed) {
			http.Error(w, "payment cannot be processed", http.StatusConflict)
			return
		}

		http.Error(w, "failed to process payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(newPaymentResponse(payment)); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}

func completePaymentHTTPHandler(
	service *applicationpayment.PaymentService,
	w http.ResponseWriter,
	r *http.Request,
) {
	id := chi.URLParam(r, "id")

	payment, err := service.Complete(r.Context(), id)
	if err != nil {
		if errors.Is(err, applicationpayment.ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}

		if errors.Is(err, applicationpayment.ErrPaymentCannotBeCompleted) {
			http.Error(w, "payment cannot be completed", http.StatusConflict)
			return
		}

		http.Error(w, "failed to complete payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(newPaymentResponse(payment)); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}

func cancelPaymentHTTPHandler(
	service *applicationpayment.PaymentService,
	w http.ResponseWriter,
	r *http.Request,
) {
	id := chi.URLParam(r, "id")

	payment, err := service.Cancel(r.Context(), id)
	if err != nil {
		if errors.Is(err, applicationpayment.ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}

		if errors.Is(err, applicationpayment.ErrPaymentCannotBeCancelled) {
			http.Error(w, "payment cannot be canceled", http.StatusConflict)
			return
		}

		http.Error(w, "failed to cancel payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(newPaymentResponse(payment)); err != nil {
		log.Printf("failed to encode payment response: %v", err)
	}
}
