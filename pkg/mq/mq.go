// Package mq provides a message bus abstraction for VC Stack.
// The primary implementation uses Apache Kafka (segmentio/kafka-go),
// but the interface allows future backends (NATS, RabbitMQ, etc.).
//
// When Kafka is not configured, callers should check for nil and
// fall back to synchronous REST or in-memory dispatch.
package mq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// ---------- Interface ----------

// Message represents a message to publish or that was received.
type Message struct {
	Topic   string            `json:"topic"`
	Key     string            `json:"key"`               // routing/partition key
	Value   []byte            `json:"value"`             // payload (JSON)
	Headers map[string]string `json:"headers,omitempty"` // metadata (trace_id, source, etc.)
	Offset  int64             `json:"offset,omitempty"`  // set on received messages
	Time    time.Time         `json:"time,omitempty"`
}

// Handler processes incoming messages. Return nil to ACK, error to NACK.
type Handler func(ctx context.Context, msg Message) error

// MessageBus defines the interface for publishing and subscribing to messages.
type MessageBus interface {
	// Publish sends a message to the specified topic.
	Publish(ctx context.Context, msg Message) error

	// Subscribe starts consuming from a topic with the given consumer group.
	// The handler is called for each message. Blocks until ctx is cancelled.
	Subscribe(ctx context.Context, topic, groupID string, handler Handler) error

	// Close shuts down all connections.
	Close() error
}

// ---------- Kafka Config ----------

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	Compression   string   `mapstructure:"compression"` // none, gzip, snappy, lz4, zstd
	TLS           bool     `mapstructure:"tls"`

	// Retry / DLQ
	MaxRetries int    `mapstructure:"max_retries"` // default 3
	DLQTopic   string `mapstructure:"dlq_topic"`   // default "vc.system.dlq"
	DLQEnabled bool   `mapstructure:"dlq_enabled"`

	// Tuning
	BatchSize    int           `mapstructure:"batch_size"`    // default 100
	BatchTimeout time.Duration `mapstructure:"batch_timeout"` // default 1s
}

// ---------- Kafka Implementation ----------

// KafkaBus implements MessageBus using segmentio/kafka-go.
type KafkaBus struct {
	writers map[string]*kafka.Writer // per-topic writers
	mu      sync.RWMutex
	cfg     KafkaConfig
	logger  *zap.Logger
}

// NewKafkaBus creates a new Kafka-backed message bus.
func NewKafkaBus(cfg KafkaConfig, logger *zap.Logger) (*KafkaBus, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("mq: at least one Kafka broker is required")
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.DLQTopic == "" {
		cfg.DLQTopic = "vc.system.dlq"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 1 * time.Second
	}

	// Verify connectivity by dialing one broker.
	conn, err := kafka.DialContext(context.Background(), "tcp", cfg.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("mq: kafka dial failed: %w", err)
	}
	_ = conn.Close()

	logger.Info("mq: connected to Kafka",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("compression", cfg.Compression))

	return &KafkaBus{
		writers: make(map[string]*kafka.Writer),
		cfg:     cfg,
		logger:  logger,
	}, nil
}

// compressionCodec converts string config to kafka.Compression.
func compressionCodec(s string) kafka.Compression {
	switch strings.ToLower(s) {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return 0 // no compression
	}
}

// getWriter returns (or creates) a writer for the given topic.
func (b *KafkaBus) getWriter(topic string) *kafka.Writer {
	b.mu.RLock()
	w, ok := b.writers[topic]
	b.mu.RUnlock()
	if ok {
		return w
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check after acquiring write lock.
	if w, ok = b.writers[topic]; ok {
		return w
	}

	w = &kafka.Writer{
		Addr:         kafka.TCP(b.cfg.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // partition by key
		Compression:  compressionCodec(b.cfg.Compression),
		BatchSize:    b.cfg.BatchSize,
		BatchTimeout: b.cfg.BatchTimeout,
		Async:        false, // synchronous for reliability
		RequiredAcks: kafka.RequireAll,
		Logger:       kafka.LoggerFunc(func(msg string, args ...interface{}) { b.logger.Debug(fmt.Sprintf(msg, args...)) }),
		ErrorLogger:  kafka.LoggerFunc(func(msg string, args ...interface{}) { b.logger.Error(fmt.Sprintf(msg, args...)) }),
	}
	b.writers[topic] = w
	return w
}

// Publish sends a message to Kafka.
func (b *KafkaBus) Publish(ctx context.Context, msg Message) error {
	w := b.getWriter(msg.Topic)

	headers := make([]kafka.Header, 0, len(msg.Headers))
	for k, v := range msg.Headers {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	km := kafka.Message{
		Key:     []byte(msg.Key),
		Value:   msg.Value,
		Headers: headers,
		Time:    time.Now(),
	}

	if err := w.WriteMessages(ctx, km); err != nil {
		b.logger.Error("mq: publish failed",
			zap.String("topic", msg.Topic),
			zap.String("key", msg.Key),
			zap.Error(err))
		return fmt.Errorf("mq: publish to %q failed: %w", msg.Topic, err)
	}

	b.logger.Debug("mq: published",
		zap.String("topic", msg.Topic),
		zap.String("key", msg.Key),
		zap.Int("size", len(msg.Value)))

	return nil
}

// Subscribe starts a consumer that reads from the topic in an infinite loop.
// It blocks until ctx is cancelled. Errors from the handler trigger DLQ
// after MaxRetries attempts.
func (b *KafkaBus) Subscribe(ctx context.Context, topic, groupID string, handler Handler) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  b.cfg.Brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
		MaxWait:  3 * time.Second,
		Logger:   kafka.LoggerFunc(func(msg string, args ...interface{}) { b.logger.Debug(fmt.Sprintf(msg, args...)) }),
	})
	defer func() { _ = r.Close() }()

	b.logger.Info("mq: subscribed",
		zap.String("topic", topic),
		zap.String("group", groupID))

	for {
		km, err := r.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				b.logger.Info("mq: consumer shutting down",
					zap.String("topic", topic),
					zap.String("group", groupID))
				return nil
			}
			b.logger.Error("mq: fetch failed", zap.Error(err))
			continue
		}

		// Convert kafka headers to map.
		headers := make(map[string]string, len(km.Headers))
		for _, h := range km.Headers {
			headers[h.Key] = string(h.Value)
		}

		msg := Message{
			Topic:   km.Topic,
			Key:     string(km.Key),
			Value:   km.Value,
			Headers: headers,
			Offset:  km.Offset,
			Time:    km.Time,
		}

		// Retry loop with DLQ.
		var lastErr error
		for attempt := 1; attempt <= b.cfg.MaxRetries; attempt++ {
			if herr := handler(ctx, msg); herr != nil {
				lastErr = herr
				b.logger.Warn("mq: handler failed, retrying",
					zap.String("topic", topic),
					zap.Int("attempt", attempt),
					zap.Int("max_retries", b.cfg.MaxRetries),
					zap.Error(herr))
				// Exponential backoff: 100ms, 200ms, 400ms...
				time.Sleep(time.Duration(100<<uint(attempt-1)) * time.Millisecond)
				continue
			}
			lastErr = nil
			break
		}

		if lastErr != nil {
			b.logger.Error("mq: handler exhausted retries, sending to DLQ",
				zap.String("topic", topic),
				zap.String("key", msg.Key),
				zap.Error(lastErr))

			// Send to Dead Letter Queue.
			if b.cfg.DLQEnabled {
				msg.Headers["dlq_original_topic"] = topic
				msg.Headers["dlq_error"] = lastErr.Error()
				msg.Headers["dlq_timestamp"] = time.Now().Format(time.RFC3339)
				if dlqErr := b.Publish(ctx, Message{
					Topic:   b.cfg.DLQTopic,
					Key:     msg.Key,
					Value:   msg.Value,
					Headers: msg.Headers,
				}); dlqErr != nil {
					b.logger.Error("mq: DLQ publish failed", zap.Error(dlqErr))
				}
			}
		}

		// Commit offset regardless (message either processed or DLQ'd).
		if err := r.CommitMessages(ctx, km); err != nil {
			b.logger.Error("mq: commit failed", zap.Error(err))
		}
	}
}

// Close shuts down all Kafka writers.
func (b *KafkaBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	for topic, w := range b.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close writer %q: %w", topic, err))
		}
	}
	b.writers = make(map[string]*kafka.Writer)

	if len(errs) > 0 {
		return fmt.Errorf("mq: close errors: %v", errs)
	}
	return nil
}
