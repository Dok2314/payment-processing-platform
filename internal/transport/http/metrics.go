package http

import "github.com/prometheus/client_golang/prometheus"

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "payment_api_http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}
