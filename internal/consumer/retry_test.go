package consumer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chris-konkol/triage/internal/consumer"
	"github.com/segmentio/kafka-go"
)

// mockWriter captures messages written to it, satisfying consumer.MessageWriter.
type mockWriter struct {
	messages []kafka.Message
	err      error
}

func (m *mockWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	m.messages = append(m.messages, msgs...)
	return m.err
}

var testMsg = kafka.Message{Topic: "test.topic", Value: []byte(`{"test":true}`)}

func TestProcessWithRetry_SuccessFirstAttempt(t *testing.T) {
	w := &mockWriter{}
	calls := 0

	err := consumer.ProcessWithRetry(context.Background(), w, testMsg, 3, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
	if len(w.messages) != 0 {
		t.Errorf("want 0 DLQ messages, got %d", len(w.messages))
	}
}

func TestProcessWithRetry_SuccessAfterOneRetry(t *testing.T) {
	w := &mockWriter{}
	calls := 0

	err := consumer.ProcessWithRetry(context.Background(), w, testMsg, 3, func() error {
		calls++
		if calls < 2 {
			return errors.New("transient failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("fn called %d times, want 2", calls)
	}
	if len(w.messages) != 0 {
		t.Error("want no DLQ messages on eventual success")
	}
}

func TestProcessWithRetry_ExhaustsRetries_SendsToDLQ(t *testing.T) {
	w := &mockWriter{}
	sentinel := errors.New("permanent failure")

	// Use maxAttempts=2 to keep test fast (one 500ms backoff).
	err := consumer.ProcessWithRetry(context.Background(), w, testMsg, 2, func() error {
		return sentinel
	})

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error chain should wrap sentinel; got: %v", err)
	}
	if len(w.messages) != 1 {
		t.Errorf("want 1 DLQ message, got %d", len(w.messages))
	}
}

func TestProcessWithRetry_ContextCancelled_StopsRetrying(t *testing.T) {
	w := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	err := consumer.ProcessWithRetry(ctx, w, testMsg, 5, func() error {
		calls++
		cancel() // cancel while fn is executing; backoff will notice immediately
		return errors.New("failure")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn should only be called once before context cancels; called %d times", calls)
	}
	if len(w.messages) != 0 {
		t.Error("must not send to DLQ when context is cancelled")
	}
}

