package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveLimiter_SetTPS(t *testing.T) {
	l := NewAdaptiveLimiter(1000)
	if l.TPS() != 1000 {
		t.Errorf("TPS() = %d, want 1000", l.TPS())
	}

	l.SetTPS(5000)
	if l.TPS() != 5000 {
		t.Errorf("TPS() = %d after SetTPS, want 5000", l.TPS())
	}
}

func TestAdaptiveLimiter_Wait(t *testing.T) {
	l := NewAdaptiveLimiter(500)
	ctx := context.Background()

	count := 0
	start := time.Now()
	deadline := start.Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait() error: %v", err)
		}
		count++
	}

	// 500 TPS ±20%
	if count < 400 || count > 600 {
		t.Errorf("count = %d in 1s at 500 TPS, want ~500 (±20%%)", count)
	}
}

func TestAdaptiveLimiter_WaitCancelled(t *testing.T) {
	l := NewAdaptiveLimiter(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := l.Wait(ctx)
	if err == nil {
		t.Error("Wait() should return error on cancelled context")
	}
}

func TestAdaptiveLimiter_RuntimeChange(t *testing.T) {
	l := NewAdaptiveLimiter(100)

	// Count at 100 TPS for 500ms
	ctx := context.Background()
	count1 := 0
	start := time.Now()
	for time.Since(start) < 500*time.Millisecond {
		if err := l.Wait(ctx); err != nil {
			break
		}
		count1++
	}

	// Change to 1000 TPS
	l.SetTPS(1000)

	count2 := 0
	start = time.Now()
	for time.Since(start) < 500*time.Millisecond {
		if err := l.Wait(ctx); err != nil {
			break
		}
		count2++
	}

	// count2 should be significantly higher than count1
	if count2 <= count1 {
		t.Errorf("after TPS increase: count2=%d should be > count1=%d", count2, count1)
	}
}
