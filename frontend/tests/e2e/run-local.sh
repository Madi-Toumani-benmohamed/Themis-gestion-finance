#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
COMPOSE_FILE="$ROOT/frontend/tests/e2e/docker-compose.yml"
cd "$ROOT"

docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
docker compose -f "$COMPOSE_FILE" up -d --wait
cleanup() { docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null; }
trap cleanup EXIT

export THEMIS_E2E_DATABASE_URL='postgres://themis_e2e:e2e_only_password@127.0.0.1:55432/themis_e2e?sslmode=disable'
cd "$ROOT/frontend"
npx playwright install chromium
npm run test:e2e
