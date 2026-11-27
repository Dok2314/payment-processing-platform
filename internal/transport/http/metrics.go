package http

import "github.com/prometheus/client_golang/prometheus"

var httpRequestDurationBuckets = []float64{
	0.0001, // 0.1 ms
	0.0005, // 0.5 ms
	0.001,  // 1 ms
	0.0025, // 2.5 ms
	0.005,  // 5 ms
	0.01,   // 10 ms
	0.025,  // 25 ms
	0.05,   // 50 ms
	0.1,    // 100 ms
	0.25,   // 250 ms
	0.5,    // 500 ms
	1,      // 1 sec
}

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "payment_api_http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "payment_api_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: httpRequestDurationBuckets,
	},
	[]string{"method", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}
