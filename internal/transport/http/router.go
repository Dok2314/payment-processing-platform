package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	applicationpayment "payment-processing-platform/internal/application/payment"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		if r.URL.Path != "/metrics" {
			httpRequestsTotal.WithLabelValues(
				r.Method,
				strconv.Itoa(ww.Status()),
			).Inc()
		}

		slog.Info(
			"HTTP request",
			"request_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
		)
	})
}

func NewRouter(service *applicationpayment.PaymentService) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger)

	r.Get("/health", healthHandler)
	r.Get("/version", versionHandler)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

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
