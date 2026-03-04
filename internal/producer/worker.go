package producer

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/dabom/simulator-usage/internal/generator"
	"golang.org/x/time/rate"
)

// WorkerPool manages goroutines that generate and publish events.
type WorkerPool struct {
	producer  Producer
	generator *generator.EventGenerator
	limiter   *rate.Limiter
	workers   int

	cancel context.CancelFunc
	wg     sync.WaitGroup

	published atomic.Int64
	failed    atomic.Int64
	active    atomic.Int32
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(p Producer, gen *generator.EventGenerator, limiter *rate.Limiter, workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	return &WorkerPool{
		producer:  p,
		generator: gen,
		limiter:   limiter,
		workers:   workers,
	}
}

// Start launches all workers. It is safe to call only once.
func (wp *WorkerPool) Start(ctx context.Context) {
	ctx, wp.cancel = context.WithCancel(ctx)
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.run(ctx, i)
	}
	slog.Info("worker pool started", "workers", wp.workers)
}

// Stop signals all workers to stop and waits for them to finish.
func (wp *WorkerPool) Stop() {
	if wp.cancel != nil {
		wp.cancel()
	}
	wp.wg.Wait()
	slog.Info("worker pool stopped",
		"published", wp.published.Load(),
		"failed", wp.failed.Load(),
	)
}

func (wp *WorkerPool) run(ctx context.Context, id int) {
	defer wp.wg.Done()
	wp.active.Add(1)
	defer wp.active.Add(-1)

	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	_ = rng // each worker could have its own rng if needed

	for {
		if err := wp.limiter.Wait(ctx); err != nil {
			return // context cancelled
		}

		env := wp.generator.Generate()
		data, err := json.Marshal(env)
		if err != nil {
			slog.Error("marshal event", "error", err)
			wp.failed.Add(1)
			continue
		}

		key := strconv.FormatInt(env.Payload.FamilyID, 10)
		if err := wp.producer.Publish(ctx, key, data); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("publish event", "error", err, "familyId", env.Payload.FamilyID)
			wp.failed.Add(1)
			continue
		}

		wp.published.Add(1)
	}
}

// Published returns the total number of successfully published events.
func (wp *WorkerPool) Published() int64 {
	return wp.published.Load()
}

// Failed returns the total number of failed events.
func (wp *WorkerPool) Failed() int64 {
	return wp.failed.Load()
}

// ActiveWorkers returns the number of currently active workers.
func (wp *WorkerPool) ActiveWorkers() int32 {
	return wp.active.Load()
}

// SetLimiter updates the rate limiter.
func (wp *WorkerPool) SetLimiter(l *rate.Limiter) {
	wp.limiter = l
}
