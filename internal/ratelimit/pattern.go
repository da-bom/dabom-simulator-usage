package ratelimit

import (
	"context"
	"log/slog"
	"time"
)

// LoadPattern defines a strategy for adjusting TPS over time.
type LoadPattern interface {
	Name() string
	Run(ctx context.Context, limiter *AdaptiveLimiter)
}

// ConstantPattern maintains a fixed TPS.
type ConstantPattern struct {
	TPS int
}

func (p *ConstantPattern) Name() string { return "constant" }

func (p *ConstantPattern) Run(ctx context.Context, limiter *AdaptiveLimiter) {
	limiter.SetTPS(p.TPS)
	<-ctx.Done()
}

// RampUpPattern linearly increases TPS from Start to Target over Duration.
type RampUpPattern struct {
	StartTPS int
	TargetTPS int
	Duration  time.Duration
}

func (p *RampUpPattern) Name() string { return "ramp-up" }

func (p *RampUpPattern) Run(ctx context.Context, limiter *AdaptiveLimiter) {
	limiter.SetTPS(p.StartTPS)

	steps := int(p.Duration / (100 * time.Millisecond))
	if steps <= 0 {
		steps = 1
	}
	increment := float64(p.TargetTPS-p.StartTPS) / float64(steps)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	current := float64(p.StartTPS)
	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current += increment
			tps := int(current)
			if tps > p.TargetTPS {
				tps = p.TargetTPS
			}
			limiter.SetTPS(tps)
		}
	}

	limiter.SetTPS(p.TargetTPS)
	slog.Info("ramp-up complete", "tps", p.TargetTPS)
	<-ctx.Done()
}

// BurstPattern maintains a base TPS with periodic bursts.
type BurstPattern struct {
	BaseTPS       int
	BurstTPS      int
	BurstDuration time.Duration
	Interval      time.Duration
}

func (p *BurstPattern) Name() string { return "burst" }

func (p *BurstPattern) Run(ctx context.Context, limiter *AdaptiveLimiter) {
	limiter.SetTPS(p.BaseTPS)

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.Info("burst triggered", "burstTps", p.BurstTPS, "duration", p.BurstDuration)
			limiter.SetTPS(p.BurstTPS)

			select {
			case <-ctx.Done():
				return
			case <-time.After(p.BurstDuration):
				limiter.SetTPS(p.BaseTPS)
				slog.Info("burst ended", "baseTps", p.BaseTPS)
			}
		}
	}
}

// RealisticPattern applies time-of-day multipliers to a base TPS.
type RealisticPattern struct {
	BaseTPS int
}

func (p *RealisticPattern) Name() string { return "realistic" }

// hourMultipliers maps hour ranges to TPS multipliers (DESIGN.md §4.5).
var hourMultipliers = []struct {
	startHour  int
	endHour    int
	multiplier float64
}{
	{0, 6, 0.2},
	{6, 9, 0.8},
	{9, 12, 1.0},
	{12, 14, 1.5},
	{14, 18, 1.0},
	{18, 22, 1.8},
	{22, 24, 1.2},
}

func (p *RealisticPattern) Run(ctx context.Context, limiter *AdaptiveLimiter) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	applyMultiplier := func() {
		hour := time.Now().Hour()
		mult := 1.0
		for _, hm := range hourMultipliers {
			if hour >= hm.startHour && hour < hm.endHour {
				mult = hm.multiplier
				break
			}
		}
		tps := int(float64(p.BaseTPS) * mult)
		limiter.SetTPS(tps)
	}

	applyMultiplier()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			applyMultiplier()
		}
	}
}

// TriggerBurst temporarily sets TPS to burstTPS for the given duration, then restores.
func TriggerBurst(ctx context.Context, limiter *AdaptiveLimiter, burstTPS int, duration time.Duration) {
	prevTPS := limiter.TPS()
	limiter.SetTPS(burstTPS)
	slog.Info("manual burst triggered", "burstTps", burstTPS, "duration", duration)

	select {
	case <-ctx.Done():
	case <-time.After(duration):
	}

	limiter.SetTPS(prevTPS)
	slog.Info("manual burst ended", "restoredTps", prevTPS)
}
