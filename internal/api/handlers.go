package api

import (
	"encoding/json"
	"net/http"

	"github.com/dabom/simulator-usage/internal/generator"
)

type handler struct {
	sim *Simulator
}

// StatusResponse is returned by GET /status.
type StatusResponse struct {
	Running           bool   `json:"running"`
	Mode              string `json:"mode"`
	CurrentTPS        int    `json:"currentTps"`
	TargetTPS         int    `json:"targetTps"`
	TotalPublished    int64  `json:"totalPublished"`
	TotalFailed       int64  `json:"totalFailed"`
	UptimeSeconds     int64  `json:"uptimeSeconds"`
	FixedTargetCount  int    `json:"fixedTargetCount"`
}

func (h *handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.sim.Status())
}

type tpsRequest struct {
	TPS int `json:"tps"`
}

func (h *handler) UpdateTPS(w http.ResponseWriter, r *http.Request) {
	var req tpsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.TPS <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tps must be positive"})
		return
	}

	prev := h.sim.SetTPS(req.TPS)
	writeJSON(w, http.StatusOK, map[string]any{
		"previousTps": prev,
		"currentTps":  req.TPS,
		"message":     "TPS updated",
	})
}

type modeRequest struct {
	Mode            string `json:"mode"`
	StartTPS        int    `json:"startTps,omitempty"`
	TargetTPS       int    `json:"targetTps,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
}

func (h *handler) UpdateMode(w http.ResponseWriter, r *http.Request) {
	var req modeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	validModes := map[string]bool{"constant": true, "ramp-up": true, "burst": true, "realistic": true}
	if !validModes[req.Mode] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mode"})
		return
	}

	cfg := h.sim.Config()
	if req.Mode == "ramp-up" && req.StartTPS > 0 {
		cfg.Simulation.RampUp.StartTPS = req.StartTPS
		cfg.Simulation.RampUp.TargetTPS = req.TargetTPS
		cfg.Simulation.RampUp.DurationSeconds = req.DurationSeconds
	}

	h.sim.SetMode(req.Mode)
	writeJSON(w, http.StatusOK, map[string]string{
		"mode":    req.Mode,
		"message": "mode updated",
	})
}

type burstConfigRequest struct {
	BaseTPS              int `json:"baseTps"`
	BurstTPS             int `json:"burstTps"`
	BurstDurationSeconds int `json:"burstDurationSeconds"`
	IntervalSeconds      int `json:"intervalSeconds"`
}

func (h *handler) UpdateBurst(w http.ResponseWriter, r *http.Request) {
	var req burstConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.sim.SetBurstConfig(req.BaseTPS, req.BurstTPS, req.BurstDurationSeconds, req.IntervalSeconds)
	writeJSON(w, http.StatusOK, map[string]string{"message": "burst config updated"})
}

func (h *handler) ControlStart(w http.ResponseWriter, _ *http.Request) {
	if err := h.sim.Start(); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "simulator started"})
}

func (h *handler) ControlStop(w http.ResponseWriter, _ *http.Request) {
	if err := h.sim.Stop(); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "simulator stopped"})
}

type burstTriggerRequest struct {
	Count           int `json:"count"`
	DurationSeconds int `json:"durationSeconds"`
}

func (h *handler) ControlBurst(w http.ResponseWriter, r *http.Request) {
	var req burstTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Count <= 0 || req.DurationSeconds <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "count and durationSeconds must be positive"})
		return
	}

	if !h.sim.IsRunning() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "simulator not running"})
		return
	}

	h.sim.TriggerBurst(req.Count, req.DurationSeconds)
	writeJSON(w, http.StatusOK, map[string]string{"message": "burst triggered"})
}

type fixedTargetsRequest struct {
	Targets []generator.FixedTarget `json:"targets"`
}

func (h *handler) UpdateFixedTargets(w http.ResponseWriter, r *http.Request) {
	var req fixedTargetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Targets) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "targets must not be empty"})
		return
	}
	for _, t := range req.Targets {
		if t.FamilyID <= 0 || len(t.CustomerIDs) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "each target must have familyId > 0 and at least one customerId"})
			return
		}
	}

	h.sim.SetFixedTargets(req.Targets)
	writeJSON(w, http.StatusOK, map[string]any{
		"message":     "fixed targets updated",
		"targetCount": len(req.Targets),
	})
}

func (h *handler) ClearFixedTargets(w http.ResponseWriter, _ *http.Request) {
	h.sim.ClearFixedTargets()
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "fixed targets cleared, returning to random mode",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
