package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(service *PaymentService) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)

	r.Get("/version", versionHandler)

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		createPaymentHandler(service, w, r)
	})

	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		getPaymentHandler(service, w, r)
	})

	r.Put("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		updatePaymentHandler(service, w, r)
	})

	r.Post("/payments/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancelPaymentHandler(service, w, r)
	})

	return r
}
