package main

import (
	"net/http"
	applicationpayment "payment-processing-platform/internal/application/payment"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(service *applicationpayment.PaymentService) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	r.Get("/version", versionHandler)

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		createPaymentHTTPHandler(service, w, r)
	})

	return r
}
