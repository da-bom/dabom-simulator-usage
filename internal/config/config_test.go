package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	yaml := `
kafka:
  brokers:
    - "localhost:9092"
  topic: "usage-events"
  acks: 1
  batchSize: 65536
  lingerMs: 5
simulation:
  mode: "constant"
  tps: 5000
  workerCount: 8
  families:
    count: 250000
    maxMembers: 10
server:
  controlPort: 8080
  metricsPort: 9090
logging:
  level: "info"
  format: "json"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "localhost:9092" {
		t.Errorf("Kafka.Brokers = %v, want [localhost:9092]", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.Topic != "usage-events" {
		t.Errorf("Kafka.Topic = %q, want %q", cfg.Kafka.Topic, "usage-events")
	}
	if cfg.Simulation.TPS != 5000 {
		t.Errorf("Simulation.TPS = %d, want 5000", cfg.Simulation.TPS)
	}
	if cfg.Simulation.Mode != "constant" {
		t.Errorf("Simulation.Mode = %q, want %q", cfg.Simulation.Mode, "constant")
	}
	if cfg.Simulation.Families.Count != 250000 {
		t.Errorf("Families.Count = %d, want 250000", cfg.Simulation.Families.Count)
	}
	if cfg.Server.ControlPort != 8080 {
		t.Errorf("Server.ControlPort = %d, want 8080", cfg.Server.ControlPort)
	}
}

func TestEnvOverrides(t *testing.T) {
	yaml := `
kafka:
  brokers:
    - "localhost:9092"
  topic: "usage-events"
simulation:
  mode: "constant"
  tps: 5000
server:
  controlPort: 8080
  metricsPort: 9090
logging:
  level: "info"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("SIM_TPS", "10000")
	t.Setenv("SIM_MODE", "burst")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Kafka.Brokers) != 2 {
		t.Errorf("Kafka.Brokers length = %d, want 2", len(cfg.Kafka.Brokers))
	}
	if cfg.Simulation.TPS != 10000 {
		t.Errorf("Simulation.TPS = %d, want 10000", cfg.Simulation.TPS)
	}
	if cfg.Simulation.Mode != "burst" {
		t.Errorf("Simulation.Mode = %q, want %q", cfg.Simulation.Mode, "burst")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
