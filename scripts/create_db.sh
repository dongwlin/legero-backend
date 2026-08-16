#!/usr/bin/env bash
# Create the Legero PostgreSQL database if it does not already exist.
#
# The application only runs migrations against an existing database and never
# creates one, so run this script once before the first `go run . serve`.
#
# Usage:
#   scripts/create_db.sh
#
# Honors the same DATABASE_URL the app uses (defaults match config/config.yaml):
#   DATABASE_URL=postgres://user:pass@host:5432/dbname scripts/create_db.sh
set -euo pipefail

cd "$(dirname "$0")/.."

DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/legero?sslmode=disable}"

if command -v podman >/dev/null 2>&1; then
  runtime=podman
elif command -v docker >/dev/null 2>&1; then
  runtime=docker
else
  echo "error: neither podman nor docker found" >&2
  exit 1
fi

# Find a running postgres container.
container="$("$runtime" ps --format '{{.Names}} {{.Image}}' | awk '$2 ~ /postgres/ {print $1; exit}')"
if [[ -z "$container" ]]; then
  echo "error: no running postgres container found (start one first)" >&2
  exit 1
fi

# Parse postgres://user:password@host:port/dbname?opts
if [[ "$DATABASE_URL" =~ ^postgres(ql)?://([^:]+):([^@]*)@([^:/]+)(:([0-9]+))?/([^?]+) ]]; then
  pg_user="${BASH_REMATCH[2]}"
  pg_password="${BASH_REMATCH[3]}"
  pg_host="${BASH_REMATCH[4]}"
  pg_port="${BASH_REMATCH[6]:-5432}"
  pg_db="${BASH_REMATCH[7]}"
else
  echo "error: cannot parse DATABASE_URL: $DATABASE_URL" >&2
  exit 1
fi

if "$runtime" exec -e PGPASSWORD="$pg_password" "$container" psql     -h "$pg_host" -p "$pg_port" -U "$pg_user" -d postgres -tAc     "SELECT 1 FROM pg_database WHERE datname = '$pg_db'" | grep -q 1; then
  echo "database '$pg_db' already exists"
else
  "$runtime" exec -e PGPASSWORD="$pg_password" "$container" psql     -h "$pg_host" -p "$pg_port" -U "$pg_user" -d postgres     -c "CREATE DATABASE \"$pg_db\";"
  echo "created database '$pg_db'"
fi
