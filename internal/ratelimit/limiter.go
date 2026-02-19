package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// AdaptiveLimiter wraps rate.Limiter with runtime TPS changes and load pattern support.
type AdaptiveLimiter struct {
	mu      sync.RWMutex
	limiter *rate.Limiter
	tps     atomic.Int64
	pattern LoadPattern
	cancel  context.CancelFunc
}

// NewAdaptiveLimiter creates a limiter with the given initial TPS.
func NewAdaptiveLimiter(tps int) *AdaptiveLimiter {
	l := &AdaptiveLimiter{
		limiter: rate.NewLimiter(rate.Limit(tps), max(tps/10, 1)),
	}
	l.tps.Store(int64(tps))
	return l
}

// Wait blocks until a token is available or the context is cancelled.
func (l *AdaptiveLimiter) Wait(ctx context.Context) error {
	l.mu.RLock()
	lim := l.limiter
	l.mu.RUnlock()
	return lim.Wait(ctx)
}

// SetTPS updates the rate limit at runtime.
func (l *AdaptiveLimiter) SetTPS(tps int) {
	l.mu.Lock()
	l.limiter.SetLimit(rate.Limit(tps))
	l.limiter.SetBurst(max(tps/10, 1))
	l.mu.Unlock()
	l.tps.Store(int64(tps))
}

// TPS returns the current target TPS.
func (l *AdaptiveLimiter) TPS() int {
	return int(l.tps.Load())
}

// InnerLimiter returns the underlying rate.Limiter for direct use by WorkerPool.
func (l *AdaptiveLimiter) InnerLimiter() *rate.Limiter {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.limiter
}

// StartPattern begins executing a load pattern in the background.
func (l *AdaptiveLimiter) StartPattern(ctx context.Context, p LoadPattern) {
	l.StopPattern()
	l.mu.Lock()
	l.pattern = p
	l.mu.Unlock()

	var pctx context.Context
	pctx, l.cancel = context.WithCancel(ctx)
	go p.Run(pctx, l)
}

// StopPattern stops the currently running load pattern.
func (l *AdaptiveLimiter) StopPattern() {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	l.mu.Lock()
	l.pattern = nil
	l.mu.Unlock()
}

// Pattern returns the current load pattern name, or "" if none.
func (l *AdaptiveLimiter) PatternName() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.pattern == nil {
		return ""
	}
	return l.pattern.Name()
}
