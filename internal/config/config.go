package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Kafka      KafkaConfig      `yaml:"kafka"`
	Database   DatabaseConfig   `yaml:"database"`
	Simulation SimulationConfig `yaml:"simulation"`
	Server     ServerConfig     `yaml:"server"`
	Logging    LoggingConfig    `yaml:"logging"`
}

type DatabaseConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbName"`
}

type KafkaConfig struct {
	Brokers         []string `yaml:"brokers"`
	Topic           string   `yaml:"topic"`
	Acks            int      `yaml:"acks"`
	BatchSize       int      `yaml:"batchSize"`
	LingerMs        int      `yaml:"lingerMs"`
	BufferMemory    int      `yaml:"bufferMemory"`
	CompressionType string   `yaml:"compressionType"`
	MaxBlockMs      int      `yaml:"maxBlockMs"`
}

type SimulationConfig struct {
	Mode         string              `yaml:"mode"`
	TPS          int                 `yaml:"tps"`
	WorkerCount  int                 `yaml:"workerCount"`
	Families     FamilyConfig        `yaml:"families"`
	RampUp       RampUpConfig        `yaml:"rampUp"`
	Burst        BurstConfig         `yaml:"burst"`
	FixedTargets []FixedTargetConfig `yaml:"fixedTargets"`
}

type FixedTargetConfig struct {
	FamilyID    int64   `yaml:"familyId"`
	CustomerIDs []int64 `yaml:"customerIds"`
}

type FamilyConfig struct {
	Count      int `yaml:"count"`
	MaxMembers int `yaml:"maxMembers"`
}

type RampUpConfig struct {
	StartTPS        int `yaml:"startTps"`
	TargetTPS       int `yaml:"targetTps"`
	DurationSeconds int `yaml:"durationSeconds"`
}

type BurstConfig struct {
	BaseTPS              int `yaml:"baseTps"`
	BurstTPS             int `yaml:"burstTps"`
	BurstDurationSeconds int `yaml:"burstDurationSeconds"`
	IntervalSeconds      int `yaml:"intervalSeconds"`
}

type ServerConfig struct {
	ControlPort int `yaml:"controlPort"`
	MetricsPort int `yaml:"metricsPort"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.Kafka.Brokers = strings.Split(v, ",")
	}
	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
	if v := os.Getenv("SIM_MODE"); v != "" {
		cfg.Simulation.Mode = v
	}
	if v, err := strconv.Atoi(os.Getenv("SIM_TPS")); err == nil {
		cfg.Simulation.TPS = v
	}
	if v, err := strconv.Atoi(os.Getenv("SIM_WORKER_COUNT")); err == nil {
		cfg.Simulation.WorkerCount = v
	}
	if v, err := strconv.Atoi(os.Getenv("CONTROL_PORT")); err == nil {
		cfg.Server.ControlPort = v
	}
	if v, err := strconv.Atoi(os.Getenv("METRICS_PORT")); err == nil {
		cfg.Server.MetricsPort = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("DB_ENABLED"); v == "true" {
		cfg.Database.Enabled = true
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v, err := strconv.Atoi(os.Getenv("DB_PORT")); err == nil {
		cfg.Database.Port = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
}
