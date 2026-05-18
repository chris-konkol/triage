package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	ticketv1 "github.com/chris-konkol/triage/gen/ticket/v1"
	"github.com/chris-konkol/triage/internal/config"
	"github.com/chris-konkol/triage/internal/db"
	"github.com/chris-konkol/triage/internal/telemetry"
	"github.com/chris-konkol/triage/internal/ticket"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadTicketSvc()
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

	producer := ticket.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	repo := ticket.NewRepository(pool)
	svc := ticket.NewService(repo, producer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatal().Err(err).Msg("listen")
	}

	srv := grpc.NewServer(telemetry.ServerOptions()...)
	ticketv1.RegisterTicketServiceServer(srv, svc)
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(srv)

	log.Info().Str("port", cfg.GRPCPort).Msg("ticket-svc starting")

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("serve error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully")
	srv.GracefulStop()
}
