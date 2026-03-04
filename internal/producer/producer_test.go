package producer

import (
	"context"
	"errors"
	"testing"
)

func TestMockProducer_Publish(t *testing.T) {
	mock := NewMockProducer()
	ctx := context.Background()

	err := mock.Publish(ctx, "100", []byte(`{"test":true}`))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	if mock.Count() != 1 {
		t.Errorf("Count() = %d, want 1", mock.Count())
	}

	msgs := mock.Messages()
	if len(msgs) != 1 {
		t.Fatalf("Messages() len = %d, want 1", len(msgs))
	}
	if msgs[0].Key != "100" {
		t.Errorf("Key = %q, want %q", msgs[0].Key, "100")
	}
}

func TestMockProducer_Error(t *testing.T) {
	mock := NewMockProducer()
	mock.SetError(errors.New("kafka down"))

	err := mock.Publish(context.Background(), "1", []byte("data"))
	if err == nil {
		t.Error("expected error from Publish")
	}
	if mock.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after error", mock.Count())
	}
}

func TestMockProducer_Close(t *testing.T) {
	mock := NewMockProducer()
	if err := mock.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
