package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dabom/simulator-usage/internal/api"
	"github.com/dabom/simulator-usage/internal/config"
	"github.com/dabom/simulator-usage/internal/metrics"
	"github.com/dabom/simulator-usage/internal/producer"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	setupLogger(cfg.Logging)

	slog.Info("simulator-usage starting",
		"mode", cfg.Simulation.Mode,
		"tps", cfg.Simulation.TPS,
		"brokers", cfg.Kafka.Brokers,
		"topic", cfg.Kafka.Topic,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize Prometheus metrics
	_ = metrics.New()

	// Initialize Kafka producer
	kafkaProducer := producer.NewKafkaProducer(cfg.Kafka)

	// Initialize Simulator (family registry + event generator)
	sim := api.InitSimulator(cfg, kafkaProducer)

	// Start control API server
	controlServer := api.NewServer(sim, cfg.Server.ControlPort)
	go func() {
		slog.Info("control API listening", "port", cfg.Server.ControlPort)
		if err := controlServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("control API server error", "error", err)
		}
	}()

	// Start metrics server
	metricsServer := metrics.NewServer(cfg.Server.MetricsPort)
	go func() {
		slog.Info("metrics server listening", "port", cfg.Server.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	// Auto-start if configured
	if err := sim.Start(); err != nil {
		slog.Error("failed to auto-start simulator", "error", err)
	}

	<-ctx.Done()
	slog.Info("simulator-usage shutting down")

	// Graceful shutdown
	if sim.IsRunning() {
		sim.Stop()
	}
	controlServer.Shutdown(context.Background())
	metricsServer.Shutdown(context.Background())
	kafkaProducer.Close()

	slog.Info("simulator-usage stopped")
}

func setupLogger(lcfg config.LoggingConfig) {
	var level slog.Level
	switch strings.ToLower(lcfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.ToLower(lcfg.Format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
