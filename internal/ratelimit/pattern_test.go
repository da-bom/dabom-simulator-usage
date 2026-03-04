package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestConstantPattern(t *testing.T) {
	l := NewAdaptiveLimiter(100)
	p := &ConstantPattern{TPS: 5000}

	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx, l)

	time.Sleep(50 * time.Millisecond)
	if l.TPS() != 5000 {
		t.Errorf("TPS = %d, want 5000", l.TPS())
	}
	cancel()
}

func TestRampUpPattern(t *testing.T) {
	l := NewAdaptiveLimiter(100)
	p := &RampUpPattern{
		StartTPS:  100,
		TargetTPS: 1000,
		Duration:  500 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx, l)

	time.Sleep(50 * time.Millisecond)
	tps1 := l.TPS()

	time.Sleep(300 * time.Millisecond)
	tps2 := l.TPS()

	if tps2 <= tps1 {
		t.Errorf("TPS should increase: tps1=%d, tps2=%d", tps1, tps2)
	}

	time.Sleep(300 * time.Millisecond)
	tpsFinal := l.TPS()
	if tpsFinal != 1000 {
		t.Errorf("final TPS = %d, want 1000", tpsFinal)
	}
}

func TestBurstPattern(t *testing.T) {
	l := NewAdaptiveLimiter(100)
	p := &BurstPattern{
		BaseTPS:       100,
		BurstTPS:      1000,
		BurstDuration: 200 * time.Millisecond,
		Interval:      500 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx, l)

	// Wait for burst to trigger (interval=500ms + 100ms margin)
	time.Sleep(600 * time.Millisecond)
	burstTPS := l.TPS()
	if burstTPS != 1000 {
		t.Errorf("during burst: TPS = %d, want 1000", burstTPS)
	}

	// Wait for burst to end (burstDuration=200ms + 100ms margin)
	time.Sleep(300 * time.Millisecond)
	baseTPS := l.TPS()
	if baseTPS != 100 {
		t.Errorf("after burst: TPS = %d, want 100", baseTPS)
	}
}

func TestRealisticPattern(t *testing.T) {
	l := NewAdaptiveLimiter(100)
	p := &RealisticPattern{BaseTPS: 1000}

	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx, l)

	time.Sleep(50 * time.Millisecond)
	tps := l.TPS()

	// TPS should be baseTPS * some multiplier (0.2 to 1.8)
	if tps < 200 || tps > 1800 {
		t.Errorf("realistic TPS = %d, want [200, 1800]", tps)
	}
	cancel()
}

func TestTriggerBurst(t *testing.T) {
	l := NewAdaptiveLimiter(500)
	ctx := context.Background()

	go TriggerBurst(ctx, l, 5000, 200*time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	if l.TPS() != 5000 {
		t.Errorf("during burst: TPS = %d, want 5000", l.TPS())
	}

	time.Sleep(300 * time.Millisecond)
	if l.TPS() != 500 {
		t.Errorf("after burst: TPS = %d, want 500", l.TPS())
	}
}

func TestPatternName(t *testing.T) {
	tests := []struct {
		pattern LoadPattern
		name    string
	}{
		{&ConstantPattern{TPS: 100}, "constant"},
		{&RampUpPattern{StartTPS: 100, TargetTPS: 1000, Duration: time.Second}, "ramp-up"},
		{&BurstPattern{BaseTPS: 100, BurstTPS: 1000, BurstDuration: time.Second, Interval: 5 * time.Second}, "burst"},
		{&RealisticPattern{BaseTPS: 1000}, "realistic"},
	}
	for _, tt := range tests {
		if tt.pattern.Name() != tt.name {
			t.Errorf("Name() = %q, want %q", tt.pattern.Name(), tt.name)
		}
	}
}
