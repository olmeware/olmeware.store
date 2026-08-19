#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

database_url="postgres://olmeware:olmeware@localhost:5432/olmeware?sslmode=disable"
export DATABASE_URL="$database_url"
export JWT_SECRET="${JWT_SECRET:-local-development-secret}"

docker compose up -d --wait postgres

if ! docker compose exec -T postgres psql -U olmeware -d olmeware -tAc \
    "select to_regclass('public.users') is not null" | grep -q t; then
    docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U olmeware -d olmeware \
        < database/scheme.sql
fi
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U olmeware -d olmeware \
    < database/exec.sql

go run . -seed

go run . &
server_pid=$!
cleanup() {
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

attempt=0
until curl --fail --silent http://localhost:8000/api/v1/health >/dev/null; do
    if ! kill -0 "$server_pid" 2>/dev/null; then
        echo "backend exited before becoming healthy" >&2
        exit 1
    fi
    attempt=$((attempt + 1))
    if [ "$attempt" -ge 30 ]; then
        echo "backend did not become healthy on port 8000" >&2
        exit 1
    fi
    sleep 1
done

go test ./...
