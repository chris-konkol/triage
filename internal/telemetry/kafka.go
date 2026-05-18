package telemetry

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

// kafkaHeaders adapts kafka.Message headers to the OTel TextMapCarrier interface.
type kafkaHeaders struct {
	headers *[]kafka.Header
}

func (h kafkaHeaders) Get(key string) string {
	for _, hdr := range *h.headers {
		if hdr.Key == key {
			return string(hdr.Value)
		}
	}
	return ""
}

func (h kafkaHeaders) Set(key, value string) {
	for i, hdr := range *h.headers {
		if hdr.Key == key {
			(*h.headers)[i].Value = []byte(value)
			return
		}
	}
	*h.headers = append(*h.headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (h kafkaHeaders) Keys() []string {
	keys := make([]string, len(*h.headers))
	for i, hdr := range *h.headers {
		keys[i] = hdr.Key
	}
	return keys
}

// InjectKafka injects the current trace context into a Kafka message's headers.
func InjectKafka(ctx context.Context, msg *kafka.Message) {
	otel.GetTextMapPropagator().Inject(ctx, kafkaHeaders{headers: &msg.Headers})
}

// ExtractKafka extracts the trace context from a Kafka message's headers into a new context.
func ExtractKafka(ctx context.Context, msg kafka.Message) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, kafkaHeaders{headers: &msg.Headers})
}
