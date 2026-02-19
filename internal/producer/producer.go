package producer

import "context"

// Producer defines the interface for publishing messages.
type Producer interface {
	Publish(ctx context.Context, key string, value []byte) error
	Close() error
}
