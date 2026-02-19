package api

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/dabom/simulator-usage/internal/config"
	"github.com/dabom/simulator-usage/internal/generator"
	"github.com/dabom/simulator-usage/internal/producer"
	"github.com/dabom/simulator-usage/internal/ratelimit"
)

// Simulator holds the shared state between the API and the worker pool.
type Simulator struct {
	mu        sync.RWMutex
	running   bool
	startTime time.Time
	config    *config.Config
	generator *generator.EventGenerator
	producer  producer.Producer
	limiter   *ratelimit.AdaptiveLimiter
	pool      *producer.WorkerPool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewSimulator creates a Simulator from the given components.
func NewSimulator(cfg *config.Config, gen *generator.EventGenerator, prod producer.Producer) *Simulator {
	return &Simulator{
		config:    cfg,
		generator: gen,
		producer:  prod,
		limiter:   ratelimit.NewAdaptiveLimiter(cfg.Simulation.TPS),
	}
}

// Start begins event publishing.
func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("simulator already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.pool = producer.NewWorkerPool(
		s.producer,
		s.generator,
		s.limiter.InnerLimiter(),
		s.config.Simulation.WorkerCount,
	)

	s.applyPattern()
	s.pool.Start(s.ctx)
	s.running = true
	s.startTime = time.Now()

	slog.Info("simulator started", "mode", s.config.Simulation.Mode, "tps", s.limiter.TPS())
	return nil
}

// Stop halts event publishing.
func (s *Simulator) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("simulator not running")
	}

	s.limiter.StopPattern()
	s.cancel()
	s.pool.Stop()
	s.running = false

	slog.Info("simulator stopped")
	return nil
}

// IsRunning returns whether the simulator is actively publishing.
func (s *Simulator) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Status returns the current simulator status.
func (s *Simulator) Status() StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := StatusResponse{
		Running:   s.running,
		Mode:      s.config.Simulation.Mode,
		CurrentTPS: s.limiter.TPS(),
		TargetTPS:  s.limiter.TPS(),
	}

	if s.pool != nil {
		resp.TotalPublished = s.pool.Published()
		resp.TotalFailed = s.pool.Failed()
	}

	if s.running {
		resp.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	}

	return resp
}

// SetTPS changes the target TPS at runtime.
func (s *Simulator) SetTPS(tps int) int {
	prev := s.limiter.TPS()
	s.limiter.SetTPS(tps)
	s.config.Simulation.TPS = tps
	slog.Info("TPS updated", "previous", prev, "current", tps)
	return prev
}

// SetMode changes the load pattern mode.
func (s *Simulator) SetMode(mode string) {
	s.mu.Lock()
	s.config.Simulation.Mode = mode
	s.mu.Unlock()

	if s.IsRunning() {
		s.mu.RLock()
		s.applyPattern()
		s.mu.RUnlock()
	}
	slog.Info("mode changed", "mode", mode)
}

// SetBurstConfig updates burst parameters.
func (s *Simulator) SetBurstConfig(baseTPS, burstTPS, burstDuration, interval int) {
	s.mu.Lock()
	s.config.Simulation.Burst.BaseTPS = baseTPS
	s.config.Simulation.Burst.BurstTPS = burstTPS
	s.config.Simulation.Burst.BurstDurationSeconds = burstDuration
	s.config.Simulation.Burst.IntervalSeconds = interval
	s.mu.Unlock()
	slog.Info("burst config updated")
}

// TriggerBurst starts an immediate burst.
func (s *Simulator) TriggerBurst(count int, durationSeconds int) {
	dur := time.Duration(durationSeconds) * time.Second
	burstTPS := count / durationSeconds
	go ratelimit.TriggerBurst(s.ctx, s.limiter, burstTPS, dur)
}

// Config returns the current config.
func (s *Simulator) Config() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *Simulator) applyPattern() {
	cfg := s.config.Simulation
	var pattern ratelimit.LoadPattern

	switch cfg.Mode {
	case "ramp-up":
		pattern = &ratelimit.RampUpPattern{
			StartTPS:  cfg.RampUp.StartTPS,
			TargetTPS: cfg.RampUp.TargetTPS,
			Duration:  time.Duration(cfg.RampUp.DurationSeconds) * time.Second,
		}
	case "burst":
		pattern = &ratelimit.BurstPattern{
			BaseTPS:       cfg.Burst.BaseTPS,
			BurstTPS:      cfg.Burst.BurstTPS,
			BurstDuration: time.Duration(cfg.Burst.BurstDurationSeconds) * time.Second,
			Interval:      time.Duration(cfg.Burst.IntervalSeconds) * time.Second,
		}
	case "realistic":
		pattern = &ratelimit.RealisticPattern{BaseTPS: cfg.TPS}
	default:
		pattern = &ratelimit.ConstantPattern{TPS: cfg.TPS}
	}

	s.limiter.StartPattern(s.ctx, pattern)
}

// NewServer creates and returns an HTTP server for the control API.
func NewServer(sim *Simulator, port int) *http.Server {
	mux := http.NewServeMux()
	h := &handler{sim: sim}

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /status", h.Status)
	mux.HandleFunc("PUT /config/tps", h.UpdateTPS)
	mux.HandleFunc("PUT /config/mode", h.UpdateMode)
	mux.HandleFunc("PUT /config/burst", h.UpdateBurst)
	mux.HandleFunc("POST /control/start", h.ControlStart)
	mux.HandleFunc("POST /control/stop", h.ControlStop)
	mux.HandleFunc("POST /control/burst", h.ControlBurst)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

// InitSimulator creates a fully wired Simulator from config.
func InitSimulator(cfg *config.Config, prod producer.Producer) *Simulator {
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	reg := generator.NewFamilyRegistry(cfg.Simulation.Families.Count, rng)
	gen := generator.NewEventGenerator(reg, rng)

	slog.Info("family registry initialized",
		"families", reg.Count(),
		"members", reg.TotalMembers(),
	)

	return NewSimulator(cfg, gen, prod)
}
