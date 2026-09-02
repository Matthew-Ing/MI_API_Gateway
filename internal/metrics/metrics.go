package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	Requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gateway_http_requests_total", Help: "HTTP requests"},
		[]string{"method", "code"},
	)
	Duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "gateway_http_request_duration_seconds", Help: "Request duration"},
		[]string{"method"},
	)
	RateLimited = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "gateway_rate_limited_total", Help: "429 responses"},
	)
	BreakerOpen = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gateway_circuit_open_total", Help: "Requests rejected because circuit is open"},
		[]string{"upstream"},
	)
	BreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_state",
			Help: "0=closed 1=open 2=half-open",
		},
		[]string{"upstream"},
	)
)

func init() {
	prometheus.MustRegister(Requests, Duration, RateLimited, BreakerOpen, BreakerState)
}
