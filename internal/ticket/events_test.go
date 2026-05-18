package ticket_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chris-konkol/triage/internal/ticket"
)

func TestEvent_JSONRoundTrip(t *testing.T) {
	payload := map[string]any{"id": "123", "title": "test ticket"}
	payloadBytes, _ := json.Marshal(payload)

	original := ticket.Event{
		EventID:   "550e8400-e29b-41d4-a716-446655440000",
		EventType: ticket.TopicCreated,
		Timestamp: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Payload:   payloadBytes,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ticket.Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.EventID != original.EventID {
		t.Errorf("EventID = %q, want %q", decoded.EventID, original.EventID)
	}
	if decoded.EventType != original.EventType {
		t.Errorf("EventType = %q, want %q", decoded.EventType, original.EventType)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}

	var decodedPayload map[string]any
	if err := json.Unmarshal(decoded.Payload, &decodedPayload); err != nil {
		t.Fatalf("Unmarshal payload: %v", err)
	}
	if decodedPayload["id"] != "123" {
		t.Errorf("Payload[id] = %v, want %q", decodedPayload["id"], "123")
	}
}

func TestEvent_PayloadPreservesRawJSON(t *testing.T) {
	// Nested JSON should survive the round-trip without being re-encoded.
	raw := json.RawMessage(`{"nested":{"key":"value"},"arr":[1,2,3]}`)
	evt := ticket.Event{
		EventID:   "test-id",
		EventType: ticket.TopicUpdated,
		Timestamp: time.Now(),
		Payload:   raw,
	}

	data, _ := json.Marshal(evt)
	var decoded ticket.Event
	json.Unmarshal(data, &decoded) //nolint:errcheck

	if string(decoded.Payload) != string(raw) {
		t.Errorf("Payload = %s, want %s", decoded.Payload, raw)
	}
}

func TestTopicConstants_AreStable(t *testing.T) {
	// Topic names are written to Kafka and consumed by multiple services.
	// Changing them silently would break running consumers — this test makes
	// any rename a deliberate, visible failure.
	cases := []struct {
		got  string
		want string
	}{
		{ticket.TopicCreated, "ticket.created"},
		{ticket.TopicUpdated, "ticket.updated"},
		{ticket.TopicStatusChanged, "ticket.status-changed"},
		{ticket.TopicCommented, "ticket.commented"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("topic constant = %q, want %q", tc.got, tc.want)
		}
	}
}

func TestEvent_JSONFieldNames(t *testing.T) {
	evt := ticket.Event{
		EventID:   "id-1",
		EventType: "test",
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Payload:   json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(evt)
	s := string(data)

	for _, key := range []string{`"event_id"`, `"event_type"`, `"timestamp"`, `"payload"`} {
		if !strings.Contains(s, key) {
			t.Errorf("JSON output missing key %s; got: %s", key, s)
		}
	}
}
