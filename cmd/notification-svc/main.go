package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/chris-konkol/triage/internal/config"
	"github.com/chris-konkol/triage/internal/consumer"
	"github.com/chris-konkol/triage/internal/telemetry"
	"github.com/chris-konkol/triage/internal/ticket"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadConsumerSvc()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdown, err := telemetry.Init(ctx, cfg.ServiceName, cfg.OTELEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("init telemetry")
	}
	defer shutdown(context.Background()) //nolint:errcheck

	topics := []string{
		ticket.TopicCreated,
		ticket.TopicStatusChanged,
		ticket.TopicCommented,
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.KafkaBrokers},
		GroupID:     cfg.KafkaGroup,
		GroupTopics: topics,
	})
	defer r.Close()

	dlq := consumer.NewDLQWriter(cfg.KafkaBrokers)
	defer dlq.Close()

	log.Info().Strs("topics", topics).Msg("notification-svc starting")

	tracer := otel.Tracer("notification-svc")

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Error().Err(err).Msg("fetch message")
			continue
		}

		msgCtx := telemetry.ExtractKafka(ctx, msg)
		msgCtx, span := tracer.Start(msgCtx, "notification.process "+msg.Topic)
		span.SetAttributes(
			attribute.String("messaging.source", msg.Topic),
			attribute.String("messaging.consumer_group", cfg.KafkaGroup),
		)

		err = consumer.ProcessWithRetry(msgCtx, dlq, msg, 3, func() error {
			var evt ticket.Event
			if err := json.Unmarshal(msg.Value, &evt); err != nil {
				return err
			}
			log.Info().
				Str("event_id", evt.EventID).
				Str("event_type", evt.EventType).
				RawJSON("payload", evt.Payload).
				Msg("notification")
			return nil
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()

		if err := r.CommitMessages(msgCtx, msg); err != nil {
			log.Error().Err(err).Msg("commit message")
		}
	}

	log.Info().Msg("notification-svc stopped")
}
