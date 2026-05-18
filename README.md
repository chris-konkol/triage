# Triage

A self-hosted, event-driven ticket system built with Go microservices, Kafka, gRPC, and React.

## Architecture

```
Browser (React + Vite)
    │
    ▼
Gateway (:8080) ── gRPC ──► ticket-svc (:50051) ── Postgres
    │                   └── gRPC ──► analytics-svc (:50052) ── Postgres
    │
    ▼ Kafka topics
 notification-svc   audit-svc   analytics-svc
```

All services export traces, metrics, and logs via OpenTelemetry → Grafana stack.

## Services

| Service | Port | Role |
|---|---|---|
| gateway | 8080 | HTTP API + React static serving |
| ticket-svc | 50051 | gRPC ticket CRUD |
| analytics-svc | 50052 | gRPC analytics + Kafka consumer |
| notification-svc | — | Kafka consumer (logging, future push) |
| audit-svc | — | Kafka consumer, writes to audit_log |

## Observability

| Tool | URL |
|---|---|
| Grafana | http://localhost:3001 (admin / admin) |
| Prometheus | http://localhost:9090 |
| Redpanda Console | http://localhost:8081 |

## Prerequisites

- Go 1.22+
- Docker + Docker Compose
- [buf](https://buf.build/docs/installation) — Protobuf code generator
- [golang-migrate](https://github.com/golang-migrate/migrate) — DB migrations
- Node.js 20+ / npm — Frontend

## Quick Start

### 1. Start infrastructure

```bash
docker compose up -d
```

### 2. Run migrations

```bash
export DATABASE_URL="postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable"
make migrate-up
```

### 3. Build protobuf

```bash
make proto
```

### 4. Start services (separate terminals)

```bash
# ticket-svc
DATABASE_URL="postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable" \
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/ticket-svc

# analytics-svc
DATABASE_URL="postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable" \
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/analytics-svc

# gateway
DATABASE_URL="postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable" \
go run ./cmd/gateway

# audit-svc
OTEL_SERVICE_NAME=audit-svc \
KAFKA_GROUP=audit-svc \
DATABASE_URL="postgres://tickets:tickets_dev@localhost:5432/tickets?sslmode=disable" \
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/audit-svc

# notification-svc
OTEL_SERVICE_NAME=notification-svc \
KAFKA_GROUP=notification-svc \
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/notification-svc
```

### 5. Start frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

App is at http://localhost:5173

### 6. Register and log in

```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

## Production Build

Build the React frontend and let the gateway serve it:

```bash
cd frontend && npm run build
cd ..
go build -o bin/gateway ./cmd/gateway
./bin/gateway  # serves frontend/dist at http://localhost:8080
```

## Health Check

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## Makefile Targets

| Command | Description |
|---|---|
| `make proto` | Regenerate gRPC code from .proto files |
| `make build` | Build all Go binaries to `bin/` |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back last migration |
| `make test` | Run Go tests |
| `make infra` | `docker compose up -d` |

## Environment Variables

### Gateway
| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | required | Postgres connection string |
| `TICKET_SVC_ADDR` | `localhost:50051` | ticket-svc gRPC address |
| `ANALYTICS_SVC_ADDR` | `localhost:50052` | analytics-svc gRPC address |
| `JWT_SECRET` | `dev-secret-change-in-prod` | JWT signing key |

### ticket-svc
| Variable | Default | Description |
|---|---|---|
| `GRPC_PORT` | `50051` | gRPC listen port |
| `DATABASE_URL` | required | Postgres connection string |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |

### Consumer services (audit-svc, notification-svc)
| Variable | Default | Description |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `KAFKA_GROUP` | `consumer-svc` | Consumer group ID |
| `DATABASE_URL` | required (audit-svc only) | Postgres connection string |
| `OTEL_SERVICE_NAME` | `consumer-svc` | Service name in traces |
