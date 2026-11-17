package http

import (
	"net/http"
	applicationpayment "payment-processing-platform/internal/application/payment"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(service *applicationpayment.PaymentService) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	r.Get("/version", versionHandler)

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		createPaymentHTTPHandler(service, w, r)
	})

	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		getPaymentHTTPHandler(service, w, r)
	})

	r.Post("/payments/{id}/process", func(w http.ResponseWriter, r *http.Request) {
		moveToProcessingHTTPHandler(service, w, r)
	})

	r.Post("/payments/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		completePaymentHTTPHandler(service, w, r)
	})

	r.Post("/payments/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancelPaymentHTTPHandler(service, w, r)
	})

	return r
}
