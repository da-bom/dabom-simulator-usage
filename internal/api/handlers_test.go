package api

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dabom/simulator-usage/internal/config"
	"github.com/dabom/simulator-usage/internal/generator"
	"github.com/dabom/simulator-usage/internal/producer"
)

func testSimulator() *Simulator {
	cfg := &config.Config{
		Simulation: config.SimulationConfig{
			Mode:        "constant",
			TPS:         1000,
			WorkerCount: 2,
			Families:    config.FamilyConfig{Count: 100, MaxMembers: 6},
			RampUp:      config.RampUpConfig{StartTPS: 100, TargetTPS: 1000, DurationSeconds: 10},
			Burst:       config.BurstConfig{BaseTPS: 1000, BurstTPS: 5000, BurstDurationSeconds: 5, IntervalSeconds: 30},
		},
		Kafka: config.KafkaConfig{Topic: "usage-events"},
	}
	rng := rand.New(rand.NewPCG(42, 0))
	reg := generator.NewFamilyRegistry(100, rng)
	gen := generator.NewEventGenerator(reg, rng)
	mock := producer.NewMockProducer()
	return NewSimulator(cfg, gen, mock)
}

func TestHealth(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q, want ok status", rec.Body.String())
	}
}

func TestStatus(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	req := httptest.NewRequest("GET", "/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"running":false`) {
		t.Errorf("body = %q, want running=false", body)
	}
	if !strings.Contains(body, `"mode":"constant"`) {
		t.Errorf("body = %q, want mode=constant", body)
	}
}

func TestUpdateTPS(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"tps": 5000}`
	req := httptest.NewRequest("PUT", "/config/tps", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateTPS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"currentTps":5000`) {
		t.Errorf("body = %q, want currentTps=5000", rec.Body.String())
	}
}

func TestUpdateTPS_Invalid(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"tps": -1}`
	req := httptest.NewRequest("PUT", "/config/tps", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateTPS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateMode(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"mode": "burst"}`
	req := httptest.NewRequest("PUT", "/config/mode", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateMode(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestUpdateMode_Invalid(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"mode": "invalid"}`
	req := httptest.NewRequest("PUT", "/config/mode", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateMode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestControlStartStop(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	// Start
	req := httptest.NewRequest("POST", "/control/start", nil)
	rec := httptest.NewRecorder()
	h.ControlStart(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("start: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !sim.IsRunning() {
		t.Error("simulator should be running after start")
	}

	// Start again should conflict
	req = httptest.NewRequest("POST", "/control/start", nil)
	rec = httptest.NewRecorder()
	h.ControlStart(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("double start: status = %d, want %d", rec.Code, http.StatusConflict)
	}

	// Stop
	req = httptest.NewRequest("POST", "/control/stop", nil)
	rec = httptest.NewRecorder()
	h.ControlStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("stop: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if sim.IsRunning() {
		t.Error("simulator should not be running after stop")
	}

	// Stop again should conflict
	req = httptest.NewRequest("POST", "/control/stop", nil)
	rec = httptest.NewRecorder()
	h.ControlStop(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("double stop: status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestControlBurst_NotRunning(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"count": 10000, "durationSeconds": 5}`
	req := httptest.NewRequest("POST", "/control/burst", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ControlBurst(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestUpdateBurst(t *testing.T) {
	sim := testSimulator()
	h := &handler{sim: sim}

	body := `{"baseTps": 2000, "burstTps": 10000, "burstDurationSeconds": 10, "intervalSeconds": 60}`
	req := httptest.NewRequest("PUT", "/config/burst", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateBurst(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
