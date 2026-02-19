package producer

import (
	"context"
	"time"

	"github.com/dabom/simulator-usage/internal/config"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

// KafkaProducer implements Producer using segmentio/kafka-go.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a KafkaProducer from the given config.
func NewKafkaProducer(cfg config.KafkaConfig) *KafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: time.Duration(cfg.LingerMs) * time.Millisecond,
		RequiredAcks: kafka.RequiredAcks(cfg.Acks),
		Compression:  compress.Lz4,
		Async:        false,
	}
	return &KafkaProducer{writer: w}
}

func (p *KafkaProducer) Publish(ctx context.Context, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
