package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	analyticsv1 "github.com/chris-konkol/triage/gen/analytics/v1"
	ticketv1 "github.com/chris-konkol/triage/gen/ticket/v1"
	"github.com/chris-konkol/triage/internal/auth"
	"github.com/chris-konkol/triage/internal/config"
	"github.com/chris-konkol/triage/internal/db"
	"github.com/chris-konkol/triage/internal/telemetry"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadGateway()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdown, err := telemetry.Init(ctx, cfg.ServiceName, cfg.OTELEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("init telemetry")
	}
	defer func() {
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer flushCancel()
		shutdown(flushCtx) //nolint:errcheck
	}()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer pool.Close()

	dialOpts := append(
		telemetry.DialOptions(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	conn, err := grpc.NewClient(cfg.TicketSvcAddr, dialOpts...)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to ticket-svc")
	}
	defer conn.Close()

	analyticsConn, err := grpc.NewClient(cfg.AnalyticsSvcAddr, dialOpts...)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to analytics-svc")
	}
	defer analyticsConn.Close()

	tickets := &ticketHandlers{client: ticketv1.NewTicketServiceClient(conn)}
	analytics := &analyticsHandlers{client: analyticsv1.NewAnalyticsServiceClient(analyticsConn)}
	authH := &authHandlers{db: pool, jwtSecret: cfg.JWTSecret}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	r.Post("/api/auth/register", authH.register)
	r.Post("/api/auth/login", authH.login)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))
		r.Get("/api/tickets", tickets.list)
		r.Post("/api/tickets", tickets.create)
		r.Get("/api/tickets/{id}", tickets.get)
		r.Put("/api/tickets/{id}", tickets.update)
		r.Delete("/api/tickets/{id}", tickets.delete)
		r.Post("/api/tickets/{id}/comments", tickets.addComment)
		r.Get("/api/dashboard", analytics.dashboard)
	})

	// Serve React SPA static build. In dev the Vite server handles the frontend
	// at :5173; in production `npm run build` populates frontend/dist and the
	// gateway serves everything from here.
	r.Get("/*", spaHandler("./frontend/dist"))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: otelhttp.NewHandler(r, "gateway"),
	}

	log.Info().Str("port", cfg.Port).Msg("gateway starting")

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("serve error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down gateway")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx) //nolint:errcheck
}

func spaHandler(distDir string) http.HandlerFunc {
	fs := http.Dir(distDir)
	fileServer := http.FileServer(fs)
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open(r.URL.Path)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	}
}
