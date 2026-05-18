.PHONY: proto build infra up down migrate-up migrate-down test frontend-dev topics lint-proto

# Generate protobuf Go code
proto:
	buf generate

# Build all services
build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/ticket-svc ./cmd/ticket-svc

# Start infrastructure only (no Go services)
infra:
	docker compose up postgres otel-collector prometheus tempo loki grafana -d

# Run everything
up:
	docker compose up --build -d

down:
	docker compose down

# Run database migrations (requires DATABASE_URL env var)
migrate-up:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DATABASE_URL)" down

# Run Go tests
test:
	go test ./...

# Frontend dev server
frontend-dev:
	cd frontend && npm run dev

# Lint protobuf definitions
lint-proto:
	buf lint
