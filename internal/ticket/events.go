package ticket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/chris-konkol/triage/internal/telemetry"
)

const (
	TopicCreated       = "ticket.created"
	TopicUpdated       = "ticket.updated"
	TopicStatusChanged = "ticket.status-changed"
	TopicCommented     = "ticket.commented"
)

type Event struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type Producer struct {
	writers map[string]*kafka.Writer
}

func NewProducer(brokers string) *Producer {
	topics := []string{TopicCreated, TopicUpdated, TopicStatusChanged, TopicCommented}
	writers := make(map[string]*kafka.Writer, len(topics))
	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:                   kafka.TCP(brokers),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
		}
	}
	return &Producer{writers: writers}
}

func (p *Producer) Close() {
	for _, w := range p.writers {
		w.Close()
	}
}

func (p *Producer) Publish(ctx context.Context, eventType string, payload any) error {
	tracer := otel.Tracer("ticket-svc")
	ctx, span := tracer.Start(ctx, "kafka.publish "+eventType)
	defer span.End()
	span.SetAttributes(attribute.String("messaging.destination", eventType))

	body, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	evt := Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   body,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	msg := kafka.Message{
		Key:   []byte(eventType),
		Value: data,
	}
	telemetry.InjectKafka(ctx, &msg)

	w, ok := p.writers[eventType]
	if !ok {
		return nil
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
