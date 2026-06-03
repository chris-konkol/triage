#!/usr/bin/env bash
# deploy.sh — first-time setup and subsequent updates for the Triage homeserver stack.
# Run from the repo root on the server: bash deploy.sh
set -euo pipefail

COMPOSE="docker compose -f docker-compose.prod.yml -p triage"

# ── Preflight ──────────────────────────────────────────────────────────────────

if [ ! -f .env ]; then
  echo "ERROR: .env not found."
  echo "Copy .env.example to .env and fill in all values, then re-run."
  exit 1
fi

set -a
# shellcheck source=.env
source .env
set +a

# ── Proxy network ──────────────────────────────────────────────────────────────

if ! docker network ls --format '{{.Name}}' | grep -q '^proxy$'; then
  echo "Creating external 'proxy' Docker network..."
  docker network create proxy
else
  echo "'proxy' network already exists."
fi

# ── Build & start ──────────────────────────────────────────────────────────────

echo ""
echo "Building images and starting services..."
$COMPOSE up --build -d

# ── Wait for Postgres ──────────────────────────────────────────────────────────

echo ""
echo "Waiting for Postgres to be ready..."
for i in $(seq 1 30); do
  if $COMPOSE exec -T postgres pg_isready -U tickets > /dev/null 2>&1; then
    echo "Postgres is ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: Postgres did not become ready in time."
    exit 1
  fi
  sleep 2
done

# ── Migrations ─────────────────────────────────────────────────────────────────

echo ""
echo "Running database migrations..."
docker run --rm \
  --network triage_internal \
  -v "$(pwd)/internal/db/migrations:/migrations" \
  migrate/migrate \
  -path=/migrations \
  -database="postgres://tickets:${POSTGRES_PASSWORD}@postgres:5432/tickets?sslmode=disable" \
  up

echo ""
echo "Migrations complete."

# ── Done ───────────────────────────────────────────────────────────────────────

echo ""
echo "================================================================"
echo " Triage is running."
echo ""
echo " Next: configure Nginx Proxy Manager with two proxy hosts:"
echo ""
echo "   triage.${DOMAIN}  →  gateway:8080      (HTTP, no SSL termination here)"
echo "   grafana.${DOMAIN} →  grafana:3000      (optional)"
echo ""
echo " Both containers are on the 'proxy' Docker network."
echo " In NPM, use the container name as the Forward Hostname."
echo "================================================================"
