package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the simulator.
type Metrics struct {
	EventsPublished     prometheus.Counter
	EventsFailed        prometheus.Counter
	EventsPublishedBytes prometheus.Counter
	CurrentTPS          prometheus.Gauge
	ActualTPS           prometheus.Gauge
	PublishLatency      prometheus.Histogram
	KafkaQueueLength    prometheus.Gauge
	KafkaDeliveryErrors prometheus.Counter
	WorkerActive        prometheus.Gauge
	UptimeSeconds       prometheus.Gauge
}

// New creates and registers all Prometheus metrics.
func New() *Metrics {
	m := &Metrics{
		EventsPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "simulator_events_published_total",
			Help: "Total number of successfully published events",
		}),
		EventsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "simulator_events_failed_total",
			Help: "Total number of failed events",
		}),
		EventsPublishedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "simulator_events_published_bytes_total",
			Help: "Total bytes of published events",
		}),
		CurrentTPS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "simulator_current_tps",
			Help: "Current configured TPS",
		}),
		ActualTPS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "simulator_actual_tps",
			Help: "Actual measured TPS (1-second window)",
		}),
		PublishLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "simulator_publish_latency_seconds",
			Help:    "Kafka publish latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
		}),
		KafkaQueueLength: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "simulator_kafka_queue_length",
			Help: "Kafka producer internal queue length",
		}),
		KafkaDeliveryErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "simulator_kafka_delivery_errors_total",
			Help: "Total Kafka delivery errors",
		}),
		WorkerActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "simulator_worker_active",
			Help: "Number of active workers",
		}),
		UptimeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "simulator_uptime_seconds",
			Help: "Simulator uptime in seconds",
		}),
	}

	prometheus.MustRegister(
		m.EventsPublished,
		m.EventsFailed,
		m.EventsPublishedBytes,
		m.CurrentTPS,
		m.ActualTPS,
		m.PublishLatency,
		m.KafkaQueueLength,
		m.KafkaDeliveryErrors,
		m.WorkerActive,
		m.UptimeSeconds,
	)

	return m
}

// NewServer creates an HTTP server serving Prometheus metrics.
func NewServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}
