package producer

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/dabom/simulator-usage/internal/generator"
	"golang.org/x/time/rate"
)

func TestWorkerPool_PublishesEvents(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := generator.NewFamilyRegistry(100, rng)
	gen := generator.NewEventGenerator(reg, rng)

	mock := NewMockProducer()
	limiter := rate.NewLimiter(rate.Limit(500), 50)
	pool := NewWorkerPool(mock, gen, limiter, 4)

	ctx := context.Background()
	pool.Start(ctx)
	time.Sleep(1 * time.Second)
	pool.Stop()

	count := mock.Count()
	if count < 400 || count > 600 {
		t.Errorf("published %d events in 1s at 500 TPS, want ~500 (±20%%)", count)
	}

	if pool.Published() != count {
		t.Errorf("Published() = %d, want %d", pool.Published(), count)
	}
	if pool.Failed() != 0 {
		t.Errorf("Failed() = %d, want 0", pool.Failed())
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := generator.NewFamilyRegistry(10, rng)
	gen := generator.NewEventGenerator(reg, rng)

	mock := NewMockProducer()
	limiter := rate.NewLimiter(rate.Limit(100), 10)
	pool := NewWorkerPool(mock, gen, limiter, 2)

	ctx := context.Background()
	pool.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	// After stop, active workers should be 0
	if pool.ActiveWorkers() != 0 {
		t.Errorf("ActiveWorkers() = %d after stop, want 0", pool.ActiveWorkers())
	}
}

func TestWorkerPool_MessageKeyIsFamilyID(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := generator.NewFamilyRegistry(10, rng)
	gen := generator.NewEventGenerator(reg, rng)

	mock := NewMockProducer()
	limiter := rate.NewLimiter(rate.Limit(1000), 100)
	pool := NewWorkerPool(mock, gen, limiter, 2)

	ctx := context.Background()
	pool.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	pool.Stop()

	msgs := mock.Messages()
	if len(msgs) == 0 {
		t.Fatal("no messages published")
	}

	for _, msg := range msgs {
		if msg.Key == "" {
			t.Error("message key is empty, expected familyId")
		}
	}
}
