package main

import (
	"encoding/json"
	"errors"
	"io"
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

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&payment)
	if err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "request body must contain only one JSON object", http.StatusBadRequest)
		return
	}

	createdPayment, err := service.Create(r.Context(), key, payment)
	if err != nil {
		if errors.Is(err, ErrInvalidPayment) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrIdempotencyKeyReuse) {
			http.Error(w, "idempotency key reused with different payload", http.StatusConflict)
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

	payment, err := service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}

		log.Printf("get payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func cancelPaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	payment, err := service.Cancel(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if errors.Is(err, ErrPaymentCannotBeCancelled) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("cancel payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func moveToProcessingHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	payment, err := service.MoveToProcessing(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if errors.Is(err, ErrPaymentCannotBeProcessed) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("process payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func completePaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	payment, err := service.Complete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if errors.Is(err, ErrPaymentCannotBeCompleted) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("complete payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

func failPaymentHandler(service *PaymentService, w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	payment, err := service.Fail(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if errors.Is(err, ErrPaymentCannotBeFailed) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("fail payment error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}
