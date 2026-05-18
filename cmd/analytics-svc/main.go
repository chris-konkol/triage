package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	analyticsv1 "github.com/chris-konkol/triage/gen/analytics/v1"
	"github.com/chris-konkol/triage/internal/config"
	"github.com/chris-konkol/triage/internal/consumer"
	"github.com/chris-konkol/triage/internal/db"
	"github.com/chris-konkol/triage/internal/telemetry"
	"github.com/chris-konkol/triage/internal/ticket"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadAnalyticsSvc()
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

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer pool.Close()

	// Start gRPC server with health check
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatal().Err(err).Msg("listen")
	}
	srv := grpc.NewServer(telemetry.ServerOptions()...)
	analyticsv1.RegisterAnalyticsServiceServer(srv, &analyticsServer{pool: pool})
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(srv)

	go func() {
		log.Info().Str("port", cfg.GRPCPort).Msg("analytics-svc gRPC starting")
		if err := srv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("grpc serve error")
		}
	}()

	// Start Kafka consumer
	topics := []string{
		ticket.TopicCreated,
		ticket.TopicUpdated,
		ticket.TopicStatusChanged,
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.KafkaBrokers},
		GroupID:     cfg.KafkaGroup,
		GroupTopics: topics,
	})
	defer r.Close()

	dlq := consumer.NewDLQWriter(cfg.KafkaBrokers)
	defer dlq.Close()

	log.Info().Strs("topics", topics).Msg("analytics-svc consumer starting")

	tracer := otel.Tracer("analytics-svc")

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
		msgCtx, span := tracer.Start(msgCtx, "analytics.process "+msg.Topic)
		span.SetAttributes(attribute.String("messaging.source", msg.Topic))

		err = consumer.ProcessWithRetry(msgCtx, dlq, msg, 3, func() error {
			return processEvent(msgCtx, pool, msg)
		})
		if err != nil {
			log.Error().Err(err).Str("topic", msg.Topic).Msg("process event")
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

	srv.GracefulStop()
	log.Info().Msg("analytics-svc stopped")
}

func processEvent(ctx context.Context, pool *pgxpool.Pool, msg kafka.Message) error {
	today := time.Now().UTC().Format("2006-01-02")

	var evt ticket.Event
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}

	switch msg.Topic {
	case ticket.TopicCreated:
		_, err := pool.Exec(ctx, `
			INSERT INTO analytics_snapshots (snapshot_date, tickets_created)
			VALUES ($1, 1)
			ON CONFLICT (snapshot_date) DO UPDATE
			SET tickets_created = analytics_snapshots.tickets_created + 1,
			    computed_at = NOW()
		`, today)
		return err

	case ticket.TopicStatusChanged, ticket.TopicUpdated:
		var payload struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			return nil
		}
		if payload.Status == "STATUS_RESOLVED" {
			_, err := pool.Exec(ctx, `
				INSERT INTO analytics_snapshots (snapshot_date, tickets_resolved)
				VALUES ($1, 1)
				ON CONFLICT (snapshot_date) DO UPDATE
				SET tickets_resolved = analytics_snapshots.tickets_resolved + 1,
				    computed_at = NOW()
			`, today)
			return err
		}
	}
	return nil
}
