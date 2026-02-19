package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsRegistration(t *testing.T) {
	// Use a custom registry to avoid conflicts with other tests
	reg := prometheus.NewRegistry()

	m := &Metrics{
		EventsPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_events_published_total",
		}),
		EventsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_events_failed_total",
		}),
		EventsPublishedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_events_published_bytes_total",
		}),
		CurrentTPS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_current_tps",
		}),
		ActualTPS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_actual_tps",
		}),
		PublishLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_publish_latency_seconds",
			Buckets: prometheus.DefBuckets,
		}),
		KafkaQueueLength: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_kafka_queue_length",
		}),
		KafkaDeliveryErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_kafka_delivery_errors_total",
		}),
		WorkerActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_worker_active",
		}),
		UptimeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_uptime_seconds",
		}),
	}

	reg.MustRegister(
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

	// Record some metrics
	m.EventsPublished.Add(100)
	m.EventsFailed.Add(5)
	m.EventsPublishedBytes.Add(1024000)
	m.CurrentTPS.Set(5000)
	m.ActualTPS.Set(4950)
	m.PublishLatency.Observe(0.002)
	m.KafkaQueueLength.Set(42)
	m.KafkaDeliveryErrors.Add(2)
	m.WorkerActive.Set(8)
	m.UptimeSeconds.Set(300)

	// Serve metrics and check output
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	expectedMetrics := []string{
		"test_events_published_total 100",
		"test_events_failed_total 5",
		"test_events_published_bytes_total 1.024e+06",
		"test_current_tps 5000",
		"test_actual_tps 4950",
		"test_publish_latency_seconds",
		"test_kafka_queue_length 42",
		"test_kafka_delivery_errors_total 2",
		"test_worker_active 8",
		"test_uptime_seconds 300",
	}

	for _, expected := range expectedMetrics {
		if !strings.Contains(body, expected) {
			t.Errorf("metrics output missing %q", expected)
		}
	}
}
