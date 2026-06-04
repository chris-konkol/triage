# Triage — Full Project Plan

## Overview

Build a self-hosted, event-driven ticket system called **Triage** using **Go**, **Kafka**, **gRPC**, and **React**. The system follows a microservices architecture where internal services communicate via gRPC and asynchronous events flow through Kafka. The React frontend communicates with an HTTP API gateway that translates requests to gRPC calls.

This is a learning project designed to build working proficiency in Go, Kafka, and gRPC.

---

## Architecture

```
React UI (Vite + TypeScript)
   │
   │ REST/JSON over HTTP
   ▼
┌─────────────────────────────────────┐
│  API Gateway (Go)                   │
│  - HTTP server (Chi router)         │
│  - Translates HTTP → gRPC calls     │
│  - Handles CORS, auth middleware    │
│  - Serves React static build in     │
│    production mode                  │
│  - Emits traces, metrics, logs      │
└──────────┬──────────────────────────┘
           │ gRPC (with OTel interceptors)
           ▼
┌─────────────────────────────────────┐
│  Ticket Service (Go, gRPC server)   │
│  - Core CRUD logic                  │
│  - Owns PostgreSQL database         │
│  - Publishes events to Kafka        │
│  - Emits traces, metrics, logs      │
└──────────┬──────────────────────────┘
           │ Kafka topics (trace context propagated in headers)
           ▼
┌──────────────────────────────────────────────────────┐
│                    Kafka (Redpanda)                   │
│                                                      │
│  Topics:                                             │
│  - ticket.created                                    │
│  - ticket.updated                                    │
│  - ticket.status-changed                             │
│  - ticket.assigned                                   │
│  - ticket.commented                                  │
└──────┬──────────────┬──────────────┬─────────────────┘
       │              │              │
       ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Notification │ │ Analytics    │ │ Audit        │
│ Service      │ │ Service      │ │ Service      │
│ (Go, gRPC)   │ │ (Go, gRPC)   │ │ (Go, gRPC)   │
│ Consumes     │ │ Consumes     │ │ Consumes     │
│ events,      │ │ events,      │ │ all events,  │
│ sends alerts │ │ aggregates   │ │ writes       │
│              │ │ stats, serves│ │ immutable    │
│              │ │ dashboard    │ │ log          │
│              │ │ data via gRPC│ │              │
│ Emits traces │ │ Emits traces │ │ Emits traces │
└──────────────┘ └──────────────┘ └──────────────┘

All services emit telemetry via OTLP ──►  OTel Collector ──┬──► Prometheus (metrics)
                                                           ├──► Grafana Tempo (traces)
                                                           └──► Grafana Loki (logs)
                                                                    │
                                                                    ▼
                                                              Grafana (dashboards,
                                                              trace viewer, log explorer)
```

---

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Frontend | React 18+ with Vite, TypeScript | Tanstack Query for data fetching, React Router v6 for navigation |
| UI Components | Mantine or Shadcn/UI | Pick one, avoid custom CSS for most things |
| API Gateway | Go, Chi router | Thin HTTP layer, translates to gRPC |
| Service Communication | gRPC with Protocol Buffers | All inter-service calls use gRPC |
| Message Broker | Redpanda (Kafka-compatible) | Easier to run locally than Kafka, single binary, no Zookeeper |
| Kafka Client | segmentio/kafka-go | Pure Go, no CGO dependency |
| Database | PostgreSQL 16 | Single database, owned by Ticket Service |
| Auth | JWT tokens | Simple implementation, middleware on gateway |
| Containerization | Docker + Docker Compose | Entire stack runs with one command |
| Proto Generation | buf or protoc with protoc-gen-go | buf is simpler to configure |
| Tracing | OpenTelemetry + Grafana Tempo | Distributed traces across gRPC and Kafka |
| Metrics | OpenTelemetry + Prometheus | Request latency, error rates, Kafka consumer lag |
| Logging | zerolog + Grafana Loki | Structured JSON logs with trace/span IDs |
| Dashboards | Grafana | Unified view of traces, metrics, and logs |
| Telemetry Pipeline | OpenTelemetry Collector | Receives OTLP from all services, exports to backends |

---

## Repository Structure

```
triage/
├── proto/                          # Protobuf definitions
│   ├── buf.yaml                    # buf configuration
│   ├── buf.gen.yaml                # Code generation config
│   ├── ticket/
│   │   └── v1/
│   │       └── ticket.proto        # Ticket service definition
│   ├── notification/
│   │   └── v1/
│   │       └── notification.proto  # Notification service definition
│   ├── analytics/
│   │   └── v1/
│   │       └── analytics.proto     # Analytics service definition
│   └── audit/
│       └── v1/
│           └── audit.proto         # Audit service definition
│
├── cmd/                            # Application entry points
│   ├── gateway/
│   │   └── main.go                 # API Gateway server
│   ├── ticket-svc/
│   │   └── main.go                 # Ticket Service server
│   ├── notification-svc/
│   │   └── main.go                 # Notification consumer
│   ├── analytics-svc/
│   │   └── main.go                 # Analytics consumer + gRPC server
│   └── audit-svc/
│       └── main.go                 # Audit consumer
│
├── internal/                       # Shared internal packages
│   ├── ticket/
│   │   ├── model.go                # Domain models
│   │   ├── service.go              # Business logic
│   │   └── repository.go          # PostgreSQL queries
│   ├── kafka/
│   │   ├── producer.go             # Kafka producer wrapper
│   │   └── consumer.go             # Kafka consumer wrapper
│   ├── db/
│   │   ├── postgres.go             # Connection setup
│   │   └── migrations/             # SQL migration files
│   │       ├── 001_create_tickets.up.sql
│   │       ├── 001_create_tickets.down.sql
│   │       ├── 002_create_comments.up.sql
│   │       ├── 002_create_comments.down.sql
│   │       ├── 003_create_audit_log.up.sql
│   │       └── 003_create_audit_log.down.sql
│   ├── auth/
│   │   ├── jwt.go                  # JWT generation and validation
│   │   └── middleware.go           # Auth middleware for gateway
│   ├── telemetry/
│   │   ├── otel.go                 # OTel SDK bootstrap (tracer, meter, logger providers)
│   │   ├── middleware.go           # HTTP middleware for trace context + request metrics
│   │   ├── grpc.go                 # gRPC interceptors (unary + stream) for tracing
│   │   ├── kafka.go                # Trace context propagation into/from Kafka headers
│   │   └── metrics.go              # Custom metric definitions (counters, histograms)
│   └── config/
│       └── config.go               # Environment-based configuration
│
├── deploy/                         # Observability configuration
│   ├── otel-collector.yaml         # OTel Collector pipeline config
│   ├── prometheus.yml              # Prometheus scrape config
│   ├── tempo.yaml                  # Grafana Tempo config
│   ├── loki.yaml                   # Grafana Loki config
│   └── grafana/
│       ├── datasources.yaml        # Auto-provision Prometheus, Tempo, Loki
│       └── dashboards/
│           ├── dashboard.yaml      # Dashboard provisioning config
│           ├── services.json       # Service health dashboard
│           └── kafka.json          # Kafka consumer dashboard
│
├── gen/                            # Generated protobuf Go code (gitignored or committed)
│   ├── ticket/v1/
│   ├── notification/v1/
│   ├── analytics/v1/
│   └── audit/v1/
│
├── frontend/                       # React application
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/                    # API client functions
│       │   └── tickets.ts
│       ├── components/
│       │   ├── layout/
│       │   │   ├── Header.tsx
│       │   │   ├── Sidebar.tsx
│       │   │   └── Layout.tsx
│       │   ├── tickets/
│       │   │   ├── TicketList.tsx
│       │   │   ├── TicketCard.tsx
│       │   │   ├── TicketDetail.tsx
│       │   │   ├── TicketForm.tsx
│       │   │   ├── TicketFilters.tsx
│       │   │   └── CommentSection.tsx
│       │   └── dashboard/
│       │       ├── Dashboard.tsx
│       │       ├── StatsCards.tsx
│       │       └── Charts.tsx
│       ├── hooks/
│       │   └── useTickets.ts       # Tanstack Query hooks
│       ├── pages/
│       │   ├── TicketsPage.tsx
│       │   ├── TicketDetailPage.tsx
│       │   ├── CreateTicketPage.tsx
│       │   ├── DashboardPage.tsx
│       │   └── LoginPage.tsx
│       ├── types/
│       │   └── ticket.ts           # TypeScript interfaces matching protobuf
│       └── utils/
│           └── auth.ts
│
├── docker-compose.yml              # Full stack: Postgres, Redpanda, observability, all Go services
├── Dockerfile                      # Multi-stage build for Go services
├── Makefile                        # Proto generation, build, run, migrate
├── go.mod
├── go.sum
└── README.md
```

---

## Protobuf Definitions

### ticket/v1/ticket.proto

```protobuf
syntax = "proto3";

package ticket.v1;

option go_package = "github.com/<your-github-username>/triage/gen/ticket/v1;ticketv1";

import "google/protobuf/timestamp.proto";

// Enums

enum Priority {
  PRIORITY_UNSPECIFIED = 0;
  PRIORITY_LOW = 1;
  PRIORITY_MEDIUM = 2;
  PRIORITY_HIGH = 3;
  PRIORITY_CRITICAL = 4;
}

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_OPEN = 1;
  STATUS_IN_PROGRESS = 2;
  STATUS_WAITING = 3;
  STATUS_RESOLVED = 4;
  STATUS_CLOSED = 5;
}

enum Category {
  CATEGORY_UNSPECIFIED = 0;
  CATEGORY_BUG = 1;
  CATEGORY_FEATURE_REQUEST = 2;
  CATEGORY_SUPPORT = 3;
  CATEGORY_DOCUMENTATION = 4;
  CATEGORY_INFRASTRUCTURE = 5;
}

// Messages

message Ticket {
  string id = 1;
  string title = 2;
  string description = 3;
  Priority priority = 4;
  Status status = 5;
  Category category = 6;
  string created_by = 7;
  string assigned_to = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
  google.protobuf.Timestamp resolved_at = 11;
  repeated string tags = 12;
}

message Comment {
  string id = 1;
  string ticket_id = 2;
  string author = 3;
  string body = 4;
  google.protobuf.Timestamp created_at = 5;
}

// Requests and Responses

message CreateTicketRequest {
  string title = 1;
  string description = 2;
  Priority priority = 3;
  Category category = 4;
  string assigned_to = 5;
  repeated string tags = 6;
}

message CreateTicketResponse {
  Ticket ticket = 1;
}

message GetTicketRequest {
  string id = 1;
}

message GetTicketResponse {
  Ticket ticket = 1;
  repeated Comment comments = 2;
}

message ListTicketsRequest {
  Status status_filter = 1;
  Priority priority_filter = 2;
  Category category_filter = 3;
  string assigned_to_filter = 4;
  int32 page = 5;
  int32 page_size = 6;
  string search_query = 7;
}

message ListTicketsResponse {
  repeated Ticket tickets = 1;
  int32 total_count = 2;
}

message UpdateTicketRequest {
  string id = 1;
  optional string title = 2;
  optional string description = 3;
  optional Priority priority = 4;
  optional Status status = 5;
  optional Category category = 6;
  optional string assigned_to = 7;
  repeated string tags = 8;
}

message UpdateTicketResponse {
  Ticket ticket = 1;
}

message DeleteTicketRequest {
  string id = 1;
}

message DeleteTicketResponse {}

message AddCommentRequest {
  string ticket_id = 1;
  string body = 2;
}

message AddCommentResponse {
  Comment comment = 1;
}

// Service

service TicketService {
  rpc CreateTicket(CreateTicketRequest) returns (CreateTicketResponse);
  rpc GetTicket(GetTicketRequest) returns (GetTicketResponse);
  rpc ListTickets(ListTicketsRequest) returns (ListTicketsResponse);
  rpc UpdateTicket(UpdateTicketRequest) returns (UpdateTicketResponse);
  rpc DeleteTicket(DeleteTicketRequest) returns (DeleteTicketResponse);
  rpc AddComment(AddCommentRequest) returns (AddCommentResponse);
}
```

### analytics/v1/analytics.proto

```protobuf
syntax = "proto3";

package analytics.v1;

option go_package = "github.com/<your-github-username>/triage/gen/analytics/v1;analyticsv1";

message GetDashboardStatsRequest {}

message GetDashboardStatsResponse {
  int32 total_open = 1;
  int32 total_in_progress = 2;
  int32 total_resolved = 3;
  int32 total_closed = 4;
  double avg_resolution_hours = 5;
  repeated CategoryCount tickets_by_category = 6;
  repeated PriorityCount tickets_by_priority = 7;
  repeated DailyCount tickets_per_day = 8;
}

message CategoryCount {
  string category = 1;
  int32 count = 2;
}

message PriorityCount {
  string priority = 1;
  int32 count = 2;
}

message DailyCount {
  string date = 1;
  int32 created = 2;
  int32 resolved = 3;
}

service AnalyticsService {
  rpc GetDashboardStats(GetDashboardStatsRequest) returns (GetDashboardStatsResponse);
}
```

---

## Kafka Event Schemas

All Kafka messages use JSON serialization with this envelope:

```json
{
  "event_id": "uuid-v4",
  "event_type": "ticket.created",
  "timestamp": "2026-05-13T10:30:00Z",
  "payload": { ... }
}
```

### Topics and Events

| Topic | Event Type | Payload | Published When |
|-------|-----------|---------|---------------|
| `ticket.created` | `ticket.created` | Full ticket object | New ticket created |
| `ticket.updated` | `ticket.updated` | Ticket ID + changed fields (old and new values) | Any field updated |
| `ticket.status-changed` | `ticket.status-changed` | Ticket ID, old status, new status, changed_by | Status transitions |
| `ticket.assigned` | `ticket.assigned` | Ticket ID, old assignee, new assignee | Assignment changes |
| `ticket.commented` | `ticket.commented` | Ticket ID, comment object | New comment added |

### Consumer Groups

| Service | Consumer Group | Subscribed Topics | What It Does |
|---------|---------------|-------------------|-------------|
| Notification | `notification-svc` | `ticket.created`, `ticket.assigned`, `ticket.status-changed`, `ticket.commented` | Logs notifications to console (stretch: sends email via SMTP or webhook) |
| Analytics | `analytics-svc` | `ticket.created`, `ticket.updated`, `ticket.status-changed` | Aggregates counts and stats into an analytics table, serves via gRPC |
| Audit | `audit-svc` | ALL topics | Writes every event to an immutable audit_log table with full payload |

---

## Database Schema

```sql
-- 001_create_tickets.up.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority SMALLINT NOT NULL DEFAULT 2,      -- 1=low, 2=medium, 3=high, 4=critical
    status SMALLINT NOT NULL DEFAULT 1,         -- 1=open, 2=in_progress, 3=waiting, 4=resolved, 5=closed
    category SMALLINT NOT NULL DEFAULT 3,       -- 1=bug, 2=feature, 3=support, 4=docs, 5=infra
    created_by VARCHAR(100) NOT NULL,
    assigned_to VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);

-- 002_create_comments.up.sql
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author VARCHAR(100) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_ticket_id ON comments(ticket_id);

-- 003_create_audit_log.up.sql
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_event_type ON audit_log(event_type);
CREATE INDEX idx_audit_log_received_at ON audit_log(received_at);

-- 004_create_users.up.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'submitter', -- submitter, agent, admin
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 005_create_analytics.up.sql
CREATE TABLE analytics_snapshots (
    id BIGSERIAL PRIMARY KEY,
    snapshot_date DATE NOT NULL,
    tickets_created INT DEFAULT 0,
    tickets_resolved INT DEFAULT 0,
    tickets_by_priority JSONB DEFAULT '{}',
    tickets_by_category JSONB DEFAULT '{}',
    avg_resolution_hours DOUBLE PRECISION,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_analytics_date ON analytics_snapshots(snapshot_date);
```

---

## Docker Compose

```yaml
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: tickets
      POSTGRES_PASSWORD: tickets_dev
      POSTGRES_DB: tickets
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tickets"]
      interval: 5s
      timeout: 3s
      retries: 5

  redpanda:
    image: redpandadata/redpanda:latest
    command:
      - redpanda start
      - --smp 1
      - --memory 512M
      - --reserve-memory 0M
      - --overprovisioned
      - --node-id 0
      - --kafka-addr PLAINTEXT://0.0.0.0:9092
      - --advertise-kafka-addr PLAINTEXT://redpanda:9092
      - --set redpanda.auto_create_topics_enabled=true
    ports:
      - "9092:9092"    # Kafka API
      - "9644:9644"    # Admin API
    volumes:
      - rpdata:/var/lib/redpanda/data
    healthcheck:
      test: ["CMD", "rpk", "cluster", "health"]
      interval: 10s
      timeout: 5s
      retries: 5

  redpanda-console:
    image: redpandadata/console:latest
    ports:
      - "8080:8080"
    environment:
      KAFKA_BROKERS: redpanda:9092
    depends_on:
      redpanda:
        condition: service_healthy

  gateway:
    build:
      context: .
      args:
        SERVICE: gateway
    ports:
      - "3000:3000"
    environment:
      PORT: "3000"
      TICKET_SVC_ADDR: "ticket-svc:50051"
      ANALYTICS_SVC_ADDR: "analytics-svc:50053"
      JWT_SECRET: "dev-secret-change-in-prod"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "gateway"
    depends_on:
      - ticket-svc
      - analytics-svc

  ticket-svc:
    build:
      context: .
      args:
        SERVICE: ticket-svc
    ports:
      - "50051:50051"
    environment:
      GRPC_PORT: "50051"
      DATABASE_URL: "postgres://tickets:tickets_dev@postgres:5432/tickets?sslmode=disable"
      KAFKA_BROKERS: "redpanda:9092"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "ticket-svc"
    depends_on:
      postgres:
        condition: service_healthy
      redpanda:
        condition: service_healthy

  notification-svc:
    build:
      context: .
      args:
        SERVICE: notification-svc
    environment:
      KAFKA_BROKERS: "redpanda:9092"
      KAFKA_GROUP: "notification-svc"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "notification-svc"
    depends_on:
      redpanda:
        condition: service_healthy

  analytics-svc:
    build:
      context: .
      args:
        SERVICE: analytics-svc
    ports:
      - "50053:50053"
    environment:
      GRPC_PORT: "50053"
      DATABASE_URL: "postgres://tickets:tickets_dev@postgres:5432/tickets?sslmode=disable"
      KAFKA_BROKERS: "redpanda:9092"
      KAFKA_GROUP: "analytics-svc"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "analytics-svc"
    depends_on:
      postgres:
        condition: service_healthy
      redpanda:
        condition: service_healthy

  audit-svc:
    build:
      context: .
      args:
        SERVICE: audit-svc
    environment:
      DATABASE_URL: "postgres://tickets:tickets_dev@postgres:5432/tickets?sslmode=disable"
      KAFKA_BROKERS: "redpanda:9092"
      KAFKA_GROUP: "audit-svc"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "audit-svc"
    depends_on:
      postgres:
        condition: service_healthy
      redpanda:
        condition: service_healthy

  # --- Observability Stack ---

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector.yaml"]
    volumes:
      - ./deploy/otel-collector.yaml:/etc/otel-collector.yaml
    ports:
      - "4317:4317"    # OTLP gRPC receiver
      - "4318:4318"    # OTLP HTTP receiver
      - "8888:8888"    # Collector metrics
    depends_on:
      - prometheus
      - tempo
      - loki

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./deploy/prometheus.yml:/etc/prometheus/prometheus.yml
      - promdata:/prometheus
    ports:
      - "9090:9090"
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=7d
      - --web.enable-remote-write-receiver

  tempo:
    image: grafana/tempo:latest
    command: ["-config.file=/etc/tempo.yaml"]
    volumes:
      - ./deploy/tempo.yaml:/etc/tempo.yaml
      - tempodata:/tmp/tempo
    ports:
      - "3200:3200"    # Tempo query API

  loki:
    image: grafana/loki:latest
    command: ["-config.file=/etc/loki/loki.yaml"]
    volumes:
      - ./deploy/loki.yaml:/etc/loki/loki.yaml
      - lokidata:/loki
    ports:
      - "3100:3100"

  grafana:
    image: grafana/grafana:latest
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_AUTH_ANONYMOUS_ORG_ROLE: Viewer
    volumes:
      - ./deploy/grafana/datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml
      - ./deploy/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - grafanadata:/var/lib/grafana
    ports:
      - "3001:3000"
    depends_on:
      - prometheus
      - tempo
      - loki

volumes:
  pgdata:
  rpdata:
  promdata:
  tempodata:
  lokidata:
  grafanadata:
```

---

## Multi-stage Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

ARG SERVICE

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/service ./cmd/${SERVICE}

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/service /service
ENTRYPOINT ["/service"]
```

---

## Makefile

```makefile
.PHONY: proto build run dev migrate test

# Generate protobuf code
proto:
	buf generate

# Build all services
build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/ticket-svc ./cmd/ticket-svc
	go build -o bin/notification-svc ./cmd/notification-svc
	go build -o bin/analytics-svc ./cmd/analytics-svc
	go build -o bin/audit-svc ./cmd/audit-svc

# Run infrastructure only
infra:
	docker compose up postgres redpanda redpanda-console otel-collector prometheus tempo loki grafana -d

# Run everything
up:
	docker compose up --build -d

down:
	docker compose down

# Run database migrations
migrate-up:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" down

# Run Go tests
test:
	go test ./...

# Frontend dev
frontend-dev:
	cd frontend && npm run dev

# Create Kafka topics manually (optional, auto-create is enabled)
topics:
	rpk topic create ticket.created ticket.updated ticket.status-changed ticket.assigned ticket.commented --brokers localhost:9092

# Lint protobuf
lint-proto:
	buf lint
```

---

## Observability

Observability is built into every service from Phase 1, not bolted on later. Every Go service initializes an OpenTelemetry SDK on startup that exports traces, metrics, and logs to the OTel Collector via OTLP. The Collector fans out to Prometheus (metrics), Grafana Tempo (traces), and Grafana Loki (logs). Grafana provides a unified UI to view all three.

### What Gets Instrumented

**Traces** — A single trace follows the full lifecycle of a request:
1. HTTP request hits gateway → span created
2. Gateway makes gRPC call to Ticket Service → child span (gRPC interceptor)
3. Ticket Service queries Postgres → child span
4. Ticket Service publishes to Kafka → child span, trace context injected into Kafka message headers
5. Consumer reads message → new span linked to the producer span via extracted trace context
6. Consumer writes to DB or sends notification → child span

This means you can open Grafana Tempo, find a single trace for "create ticket," and see the entire journey from HTTP to Kafka consumer — across multiple services.

**Metrics** — Each service exposes:
- `http_request_duration_seconds` (histogram) — Gateway request latency by route, method, status code
- `grpc_request_duration_seconds` (histogram) — gRPC call latency by method and status
- `grpc_requests_total` (counter) — gRPC calls by method and status
- `kafka_messages_produced_total` (counter) — Events published by topic
- `kafka_messages_consumed_total` (counter) — Events consumed by topic and consumer group
- `kafka_consumer_lag` (gauge) — How far behind each consumer is
- `kafka_consume_duration_seconds` (histogram) — Time to process each consumed message
- `tickets_created_total` (counter) — Business metric
- `tickets_resolved_total` (counter) — Business metric
- `db_query_duration_seconds` (histogram) — Database query latency

**Structured Logs** — All services use zerolog to emit JSON logs with these fields automatically injected:
- `trace_id` — Links the log line to a distributed trace in Tempo
- `span_id` — Links to the specific span
- `service` — Which service emitted it
- `level` — info, warn, error
- `timestamp` — ISO 8601

This lets you click a trace in Grafana Tempo, jump to the correlated logs in Loki, and see exactly what happened.

### Trace Context Propagation

The critical piece is propagating trace context across boundaries:

- **HTTP → gRPC:** The gateway's OTel HTTP middleware creates a span, then the gRPC client interceptor propagates the context via gRPC metadata automatically.
- **gRPC → Kafka:** When the Ticket Service publishes to Kafka, inject the trace context into Kafka message headers using `propagation.TraceContext{}`. The `internal/telemetry/kafka.go` helper handles this.
- **Kafka → Consumer:** When a consumer reads a message, extract the trace context from Kafka headers and create a new span linked to the producer's span. This connects the async processing to the original request.

### OTel Collector Pipeline Config (deploy/otel-collector.yaml)

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 1024

exporters:
  prometheusremotewrite:
    endpoint: "http://prometheus:9090/api/v1/write"
  otlp/tempo:
    endpoint: "http://tempo:4317"
    tls:
      insecure: true
  loki:
    endpoint: "http://loki:3100/loki/api/v1/push"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/tempo]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheusremotewrite]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [loki]
```

### Grafana Dashboards

Build two pre-provisioned dashboards:

**Service Health Dashboard (deploy/grafana/dashboards/services.json)**
- Request rate (RPM) per service
- Error rate (%) per service
- P50/P95/P99 latency per service
- Active gRPC connections

**Kafka Dashboard (deploy/grafana/dashboards/kafka.json)**
- Messages produced per topic per minute
- Consumer lag per consumer group
- Message processing latency per consumer
- Failed message count (dead letter candidates)

Grafana datasource provisioning (deploy/grafana/datasources.yaml) auto-configures Prometheus, Tempo, and Loki with trace-to-logs and trace-to-metrics correlations enabled, so clicking a trace ID in Tempo jumps directly to the matching logs in Loki.

---

## API Gateway HTTP Endpoints

The gateway exposes a REST API that the React frontend consumes. Each endpoint translates to a gRPC call.

| Method | Path | gRPC Call | Auth Required |
|--------|------|-----------|--------------|
| POST | `/api/auth/login` | Direct DB | No |
| GET | `/api/tickets` | `TicketService.ListTickets` | Yes |
| POST | `/api/tickets` | `TicketService.CreateTicket` | Yes |
| GET | `/api/tickets/:id` | `TicketService.GetTicket` | Yes |
| PUT | `/api/tickets/:id` | `TicketService.UpdateTicket` | Yes |
| DELETE | `/api/tickets/:id` | `TicketService.DeleteTicket` | Yes (admin) |
| POST | `/api/tickets/:id/comments` | `TicketService.AddComment` | Yes |
| GET | `/api/dashboard` | `AnalyticsService.GetDashboardStats` | Yes |

---

## React Frontend Pages

### Pages and Components

1. **Login / Register Page** — Simple form, stores JWT in memory (not localStorage for security, use httpOnly cookie or in-memory with refresh).

2. **Ticket List Page** (`/tickets`)
   - Table or card view of all tickets
   - Filters: status, priority, category, assignee
   - Search bar with debounced input
   - Sort by created date, priority, status
   - Pagination
   - "Create Ticket" button

3. **Ticket Detail Page** (`/tickets/:id`)
   - Full ticket info with editable fields (inline or modal)
   - Status transition buttons (Open → In Progress → Resolved → Closed)
   - Assignment dropdown
   - Comment thread with add comment form
   - Activity/audit history sidebar

4. **Create Ticket Page** (`/tickets/new`)
   - Form with title, description, priority, category, assignee, tags
   - Validation
   - Redirects to detail page on success

5. **Dashboard Page** (`/dashboard`)
   - Stats cards: open, in progress, resolved, closed counts
   - Chart: tickets created vs resolved per day (line chart, use Recharts)
   - Chart: tickets by priority (bar chart)
   - Chart: tickets by category (pie or donut chart)
   - Average resolution time display

### React Tech Decisions

- **Vite** for build tooling
- **TypeScript** throughout
- **React Router v6** for routing
- **Tanstack Query (React Query)** for all data fetching, caching, and mutations
- **Mantine** for UI components (or Shadcn/UI — pick one)
- **Recharts** for dashboard charts
- **Zod** for form validation schemas

---

## Build Order (Phase by Phase)

### ✅ Phase 1 — Foundation

**Goal:** Ticket Service works standalone, testable with grpcurl. Telemetry pipeline is running.

1. Initialize Go module, install dependencies (`google.golang.org/grpc`, `segmentio/kafka-go`, `jackc/pgx/v5`, `golang-migrate/migrate`, OTel SDK packages)
2. Write protobuf definitions for TicketService
3. Generate Go code with `buf generate`
4. Set up Docker Compose with PostgreSQL + full observability stack (OTel Collector, Prometheus, Tempo, Loki, Grafana)
5. Write and run database migrations
6. Build `internal/telemetry` package: OTel SDK bootstrap (`otel.go`), gRPC interceptors (`grpc.go`), HTTP middleware (`middleware.go`), structured logging with trace ID injection
7. Implement Ticket Service: repository layer (Postgres queries), service layer (business logic), gRPC server — with OTel gRPC interceptors and DB query tracing from the start
8. Verify traces appear in Grafana Tempo, metrics in Prometheus
9. Test with `grpcurl` or Evans CLI

### ✅ Phase 2 — API Gateway + React Skeleton

**Goal:** React can create and list tickets through the gateway. End-to-end traces visible in Grafana.

1. Build API Gateway: HTTP server with Chi, gRPC client to Ticket Service, OTel HTTP middleware and gRPC client interceptors
2. Add CORS middleware
3. Scaffold React app with Vite + TypeScript
4. Build TicketList and CreateTicket pages
5. Wire up Tanstack Query hooks to call the gateway
6. Add basic auth (login, JWT middleware) — registration removed; accounts created directly in DB
7. Verify that a request from React creates a trace spanning gateway → ticket service → Postgres in Tempo

### ✅ Phase 3 — Kafka Integration

**Goal:** Events flow through Kafka, consumers process them. Traces span from HTTP through Kafka to consumers.

1. Add Redpanda to Docker Compose
2. Build `internal/telemetry/kafka.go` — inject/extract trace context into Kafka message headers using W3C TraceContext propagator
3. Add Kafka producer to Ticket Service — publish events after successful DB writes with trace context in headers
4. Build Notification consumer (logs to stdout initially) with trace context extraction so consumer spans link to producer spans
5. Build Audit consumer (writes to audit_log table)
6. Add Kafka metrics: messages produced/consumed counters, consumer lag gauge, processing duration histogram
7. Verify in Grafana Tempo: a single trace shows HTTP → gRPC → Kafka publish → consumer processing

### ✅ Phase 4 — Analytics + Dashboard

**Goal:** Dashboard shows live stats derived from Kafka events.

1. Build Analytics consumer — reads events, updates analytics_snapshots table
2. Write analytics.proto and implement AnalyticsService gRPC server in the same process
3. Add `/api/dashboard` route to gateway
4. Build React Dashboard page with Recharts

### ✅ Phase 5 — Polish and Production Readiness

**Goal:** System is robust, observable, and self-hostable.

1. Ticket detail page with comments, status transitions, assignment
2. Filtering, search, and pagination on ticket list
3. Error handling: Kafka dead letter topics, consumer retries with backoff, idempotent processing (use event_id to deduplicate)
4. Graceful shutdown for all services (handle SIGTERM, drain Kafka consumers, close gRPC connections, flush OTel spans)
5. Health check endpoints for each service
6. Build Grafana dashboards: Service Health (request rate, error rate, latency percentiles) and Kafka (consumer lag, message throughput, processing latency)
7. Configure Grafana datasource provisioning with Tempo → Loki trace-to-logs correlation
8. Docker Compose production profile with resource limits
9. Configure gateway to serve React static build in production
10. README with setup instructions

### ✅ Phase 5.5 — Homeserver Deployment

**Goal:** Full stack running on homeserver, accessible via domain over HTTPS.

1. `Dockerfile.gateway` — multi-stage build (Node.js frontend + Go gateway in one image)
2. `docker-compose.prod.yml` — all services wired together, external `proxy` network for Nginx Proxy Manager
3. `.env.example` — secret template (POSTGRES_PASSWORD, JWT_SECRET, GF_ADMIN_PASSWORD, DOMAIN)
4. `deploy.sh` — one-command setup: builds images, starts stack, waits for Postgres, runs migrations
5. Configure Nginx Proxy Manager proxy hosts: `triage.ckonkol.net → gateway:8080`, `grafana.ckonkol.net → grafana:3000`
6. DNS CNAME via No-IP pointing `triage.ckonkol.net` at homeserver public IP
7. Let's Encrypt SSL certs via NPM

> **Note on telemetry stack:** The observability containers (otel-collector, prometheus, tempo, loki, grafana) are currently bundled with Triage. If other homeserver apps need shared observability in the future, extract them into a standalone `homelab-observability` repo and have each app reference that stack's network as external.

### Phase 6 — MCP Server

**Goal:** Claude Code can query and manage Triage tickets directly from any conversation.

1. Build `cmd/mcp-svc` — a Go MCP server exposing tools over stdio transport:
   - `list_tickets` — filter by status, priority, search query
   - `get_ticket` — full ticket detail including comments
   - `create_ticket` — title, description, priority, category
   - `update_ticket` — change status, priority, assignment
   - `get_dashboard_stats` — summary counts and averages
2. Auth via bearer token — server reads `TRIAGE_API_URL` and `TRIAGE_TOKEN` from environment, calls the existing REST API
3. Configure Claude Code to use the server: add to `.claude/settings.json` `mcpServers` block so it's available in every session
4. (Optional) SSE transport variant for remote access via `mcp.ckonkol.net` through NPM

### Phase 7 — PWA + Web Push Notifications

**Goal:** App is installable on Android and sends real push notifications when tickets are created or changed.

1. Add `vite-plugin-pwa` — generate web manifest and service worker so Chrome on Android offers "Add to Home Screen"
2. Add icons (192x192, 512x512) for home screen and splash screen
3. Add `push_subscriptions` table (migration) — stores user ID, endpoint URL, and encryption keys per device
4. Add `POST /api/push/subscribe` endpoint on the gateway — saves a user's push subscription after they grant permission
5. Add permission request to the React app on login — calls the subscribe endpoint and registers the service worker
6. Upgrade notification-svc to send Web Push using `github.com/SherClockHolmes/webpush-go` — look up the affected user's subscription and deliver a push for `ticket.created`, `ticket.assigned`, and `ticket.status-changed` events
7. Handle the push in the service worker — call `showNotification()` with the ticket title and a deep link back into the app
8. Test end to end: create a ticket on desktop, receive a push notification on Android

### Phase 8 — Stretch Goals (Optional)

- WebSocket support for real-time ticket updates in the UI
- File attachments (upload to local disk or S3-compatible storage like MinIO)
- Role-based access control (submitter sees own tickets, agent sees assigned, admin sees all)
- SLA monitoring consumer (alert if critical ticket isn't resolved within N hours)
- Alerting rules in Grafana (e.g., alert when error rate > 5% or consumer lag > 1000)
- gRPC streaming for live notification feed
- CI pipeline with GitHub Actions (lint, test, build Docker images)

---

## Key Go Dependencies

```
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/jackc/pgx/v5
go get github.com/segmentio/kafka-go
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-jwt/jwt/v5
go get github.com/google/uuid
go get github.com/rs/zerolog          # Structured logging
go get github.com/kelseyhightower/envconfig  # Config from env vars

# OpenTelemetry
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc    # gRPC interceptors
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp                   # HTTP middleware
go get go.opentelemetry.io/otel/propagation                                            # Trace context for Kafka
```

---

## Environment Variables

Each service reads config from environment variables:

```env
# Ticket Service
GRPC_PORT=50051
DATABASE_URL=postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable
KAFKA_BROKERS=localhost:9092
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=ticket-svc

# API Gateway
PORT=3000
TICKET_SVC_ADDR=localhost:50051
ANALYTICS_SVC_ADDR=localhost:50053
JWT_SECRET=dev-secret-change-in-prod
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=gateway

# Consumers
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP=<service-name>
DATABASE_URL=postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=<service-name>
```

---

## Self-Hosting Requirements

**Minimum server:** 2 vCPU, 4GB RAM, 40GB SSD (tight but works)

**Comfortable server:** 4 vCPU, 8GB RAM, 60GB SSD (recommended — room for Grafana stack and data retention)

| Component | RAM Usage |
|-----------|----------|
| PostgreSQL 16 | 256–512 MB |
| Redpanda (Kafka-compatible) | 512 MB–1 GB |
| Go API Gateway | ~15–30 MB |
| Go Ticket Service | ~15–30 MB |
| 3x Go Consumers | ~45–90 MB |
| React (static files) | Negligible |
| Redpanda Console (Web UI) | ~100 MB |
| OTel Collector | ~50–100 MB |
| Prometheus | ~200–300 MB |
| Grafana Tempo | ~100–200 MB |
| Grafana Loki | ~100–200 MB |
| Grafana | ~150–250 MB |
| **Total** | **~1.5–2.7 GB** |

**Deployment:** Run `docker compose up -d` on the server. For production, add a reverse proxy (Caddy or nginx) in front of the gateway for TLS termination.

---

## Testing Strategy

- **Unit tests:** Service layer logic, Kafka event serialization/deserialization, JWT handling
- **Integration tests:** Repository layer against a test PostgreSQL container (use testcontainers-go)
- **gRPC tests:** Start the server in a goroutine, use a gRPC client in tests
- **End-to-end:** Docker Compose up, hit the HTTP API, verify events in Redpanda Console and audit_log table
