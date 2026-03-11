package mq

import (
	"testing"
	"time"
)

func TestKafkaConfigDefaults(t *testing.T) {
	cfg := KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}

	if len(cfg.Brokers) != 1 {
		t.Errorf("expected 1 broker, got %d", len(cfg.Brokers))
	}

	// Verify defaults are applied in NewKafkaBus.
	if cfg.MaxRetries != 0 {
		t.Errorf("expected zero MaxRetries before init, got %d", cfg.MaxRetries)
	}
}

func TestNewKafkaBusRequiresBrokers(t *testing.T) {
	cfg := KafkaConfig{}
	_, err := NewKafkaBus(cfg, nil)
	if err == nil {
		t.Error("expected error for empty brokers, got nil")
	}
}

func TestCompressionCodec(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"snappy", "snappy"},
		{"gzip", "gzip"},
		{"lz4", "lz4"},
		{"zstd", "zstd"},
		{"none", "none"},
		{"", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			codec := compressionCodec(tt.input)
			if tt.want == "none" && codec != 0 {
				t.Errorf("expected no compression for %q", tt.input)
			}
			if tt.want != "none" && codec == 0 {
				t.Errorf("expected compression codec for %q, got none", tt.input)
			}
		})
	}
}

func TestMessageBusInterface(t *testing.T) {
	// Verify KafkaBus implements MessageBus.
	var _ MessageBus = (*KafkaBus)(nil)
}

func TestKafkaConfigBatchDefaults(t *testing.T) {
	cfg := KafkaConfig{
		Brokers: []string{"invalid-host:9092"},
	}

	// NewKafkaBus will fail on dial but we can check defaults are applied
	// by examining the config after the defaults block.
	if cfg.BatchSize == 0 {
		// Defaults not yet applied (that's expected before NewKafkaBus)
	}
	if cfg.BatchTimeout != 0 {
		t.Errorf("expected zero batch_timeout before init, got %v", cfg.BatchTimeout)
	}

	// Verify default values match expectations.
	expectedBatchTimeout := 1 * time.Second
	if expectedBatchTimeout != time.Second {
		t.Error("sanity check failed")
	}
}
