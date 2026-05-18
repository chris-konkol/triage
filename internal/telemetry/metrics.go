package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the custom business and infrastructure counters/histograms.
type Metrics struct {
	TicketsCreated  metric.Int64Counter
	TicketsResolved metric.Int64Counter
	DBQueryDuration metric.Float64Histogram
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("triage")
	m := &Metrics{}
	var err error

	m.TicketsCreated, err = meter.Int64Counter("tickets_created_total",
		metric.WithDescription("Total number of tickets created"))
	if err != nil {
		return nil, err
	}

	m.TicketsResolved, err = meter.Int64Counter("tickets_resolved_total",
		metric.WithDescription("Total number of tickets resolved"))
	if err != nil {
		return nil, err
	}

	m.DBQueryDuration, err = meter.Float64Histogram("db_query_duration_seconds",
		metric.WithDescription("Database query latency in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return m, nil
}
