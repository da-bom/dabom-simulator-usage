package producer

import (
	"context"
	"sync"
	"sync/atomic"
)

// MockProducer is a test double that records published messages.
type MockProducer struct {
	mu       sync.Mutex
	messages []MockMessage
	count    atomic.Int64
	failErr  error
}

// MockMessage stores a captured message.
type MockMessage struct {
	Key   string
	Value []byte
}

func NewMockProducer() *MockProducer {
	return &MockProducer{}
}

func (m *MockProducer) Publish(_ context.Context, key string, value []byte) error {
	if m.failErr != nil {
		return m.failErr
	}
	m.mu.Lock()
	m.messages = append(m.messages, MockMessage{Key: key, Value: value})
	m.mu.Unlock()
	m.count.Add(1)
	return nil
}

func (m *MockProducer) Close() error {
	return nil
}

// Count returns the number of published messages.
func (m *MockProducer) Count() int64 {
	return m.count.Load()
}

// Messages returns a copy of all captured messages.
func (m *MockProducer) Messages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]MockMessage, len(m.messages))
	copy(cp, m.messages)
	return cp
}

// SetError makes all future Publish calls return this error.
func (m *MockProducer) SetError(err error) {
	m.failErr = err
}
