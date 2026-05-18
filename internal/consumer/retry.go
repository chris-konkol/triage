package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

const DLQTopic = "ticket.dlq"

// MessageWriter is the subset of *kafka.Writer used for DLQ publishing.
type MessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type dlqMessage struct {
	OriginalTopic string `json:"original_topic"`
	Error         string `json:"error"`
	Attempts      int    `json:"attempts"`
	Payload       []byte `json:"payload"`
}

// NewDLQWriter returns a kafka.Writer that publishes to the dead letter queue.
func NewDLQWriter(brokers string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers),
		Topic:                  DLQTopic,
		AllowAutoTopicCreation: true,
	}
}

// ProcessWithRetry runs fn up to maxAttempts times with linear backoff.
// After all retries are exhausted it writes the original message to the DLQ
// and returns an error. A cancelled ctx aborts immediately.
func ProcessWithRetry(ctx context.Context, dlq MessageWriter, msg kafka.Message, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt < maxAttempts {
				backoff := time.Duration(attempt) * 500 * time.Millisecond
				log.Warn().Err(err).
					Int("attempt", attempt).
					Str("topic", msg.Topic).
					Dur("backoff", backoff).
					Msg("processing failed, retrying")
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			continue
		}
		return nil
	}

	d := dlqMessage{
		OriginalTopic: msg.Topic,
		Error:         lastErr.Error(),
		Attempts:      maxAttempts,
		Payload:       msg.Value,
	}
	data, _ := json.Marshal(d)

	dlqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dlq.WriteMessages(dlqCtx, kafka.Message{Key: msg.Key, Value: data}); err != nil {
		log.Error().Err(err).Str("topic", msg.Topic).Msg("failed to write to DLQ")
	} else {
		log.Error().Str("original_topic", msg.Topic).Str("error", lastErr.Error()).Msg("message sent to DLQ after max retries")
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
