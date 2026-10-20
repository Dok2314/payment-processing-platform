package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(store *PaymentStore) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)

	r.Get("/version", versionHandler)

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		createPaymentHandler(store, w, r)
	})

	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		getPaymentHandler(store, w, r)
	})

	return r
}
