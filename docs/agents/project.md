# Project Overview

Legero is a restaurant order-management backend written in Go. It manages users, workspaces, and orders, and pushes order updates to clients in real time over WebSocket. The binary is a single Cobra CLI; running it without a subcommand starts the HTTP server.

## Tech Stack

* Go 1.25 — `go.mod` is the source of truth; CI uses `go-version-file`
* HTTP: `gin-gonic/gin`, with CORS, request logging, and recovery middleware
* CLI: `spf13/cobra` — subcommands `serve`, `create-user`, `version`
* Database: PostgreSQL via `jackc/pgx/v5` connection pool + `uptrace/bun` ORM (pgdialect)
* Migrations: `golang-migrate/migrate/v4` over embedded SQL files (`migrations/*.sql`, `//go:embed`); applied automatically at startup
* Auth: phone + password; Argon2id password hashing; PASETO v4 symmetric tokens with rotating refresh tokens
* Realtime: `gorilla/websocket` broker + session manager with heartbeats
* Config: `spf13/viper` — YAML file plus environment-variable overrides
* Logging: `rs/zerolog` (console writer, RFC3339 timestamps)
* DI: `google/wire` — providers in `internal/app/provider.go`, generated injectors in `internal/app/wire_gen.go`
* Tests: `stretchr/testify` unit tests + `testcontainers-go` integration tests (postgres:18)

## Repository Layout

```
cmd/                  cobra root command and subcommands (serve, create-user, version)
config/               config.example.yaml (config/config.yaml is gitignored)
internal/
  app/                composition root: wire providers/injectors, router, lifecycle
  apperr/             transport-agnostic application errors (AppError/Kind, constructors)
  handler/            router + middleware (auth, cors, logger) + versioned handlers (v1), request/response DTOs, and httpresp response helpers (Kind -> HTTP mapping)
  domain/             pure domain models and business rules
  repo/               data access (bun queries)
    schema/           bun persistence mappings, repo-private by convention
  service/            business logic: interfaces + request/result types in `service`, implementations in `service/v1`
  infra/
    config/           viper config loading; build info (Version/Commit/BuildTime/GoVersion)
    crypto/           Argon2id password hashing; PASETO token issue/verify
    database/         pgx pool + bun setup
    identity/         authenticated identity context
    logger/           zerolog setup
    realtime/         WebSocket broker and session manager
    shutdown/         graceful-shutdown helper
    timex/            time formatting helpers
migrations/           embedded SQL migrations (up/down)
scripts/              build_android.sh / build_android.ps1
bin/                  local build output (gitignored)
.github/workflows/    ci.yml
```

## Architecture

Requests flow through the standard layered structure:

`handler → service → repo → domain` (gin middleware lives inside the handler package)

* `internal/app` is the composition root. Google wire (`wire.go` → generated `wire_gen.go`) bootstraps config, logger, timezone, migrations, DB, the realtime broker/session manager, services, handlers, and the gin router; `Application.Run` serves until a shutdown signal and then drains gracefully (30s timeout). The `create-user` CLI uses its own injector (`InitializeUserCreator`).
* Handlers (and the gin middleware in `internal/handler/middleware`) translate HTTP/WS into service calls and render JSON via `internal/handler/httpresp` (mapping `apperr.Kind` to HTTP status); they contain no business logic. Services and infra return `*apperr.AppError` and never import the handler layer.
* Services hold business rules and orchestrate repos and the realtime broker.
* Repos encapsulate all bun queries; they take a `bun.IDB` so callers can pass a transaction, and they operate on bun-mapped structs from the private `internal/repo/schema` subpackage, converting to and from domain types internally.
* Domain models live in `internal/domain` as pure structs with no ORM mapping; business rules such as step toggling and price computation live there too and are unit-testable without a database.

## Domain Model

* `users` — phone, Argon2id password hash, active flag
* `workspaces` + `workspace_members` — restaurant workspaces; membership roles `owner` | `staff`
* `orders` — order form (staple/meat/greens/condiments/packaging codes), step status codes, price in cents, audit fields (`created_by`, `updated_by`, `completed_at`), per-workspace `display_no` sequenced by `workspace_daily_counters`
* `refresh_tokens` — rotating refresh tokens (hash stored, revoked/rotated chain)
* `workspace_daily_counters` — per-workspace per-business-day sequence for order numbers

Order form values are encoded as smallint codes (`stapleTypeCode`, `selectedMeatCodes`, …), with cooking step statuses `unrequired` | `not-started` | `completed` (`domain.StepStatus*`). An order can be marked served only when all required steps are completed (`Order.CanServe`).

## Version & optimistic concurrency

Every primary entity carries a server-maintained monotonic `version` (bigint, starting at 1). Orders advance it atomically on every successful state-changing mutation (`UPDATE ... SET version = version + 1 RETURNING version`), so commits within the same second remain distinguishable. Users, workspaces, and refresh tokens are seeded at version 1; refresh-token rotation is the only existing mutation path there and also advances it.

`OrderDTO` (and therefore order list/snapshot responses, mutation responses, and realtime `order.upsert` events) exposes `version`. Order mutations accept an optional `expectedVersion`; a mismatch rejects the write with a `409` `order_conflict`. The deprecated `expectedUpdatedAt` token remains supported as a fallback when `expectedVersion` is absent, so existing clients keep working during migration.

## API Surface

Public:

* `GET /healthz` — liveness
* `POST /api/auth/login`, `POST /api/auth/refresh` — PASETO auth
* `GET /api/ws?ticket=...` — WebSocket upgrade, authenticated by a one-time ticket

Authenticated (Bearer access token):

* `GET /api/bootstrap` — current user/workspace context
* `GET /api/orders`, `POST /api/orders`
* `PUT /api/orders/:id`, `DELETE /api/orders/:id`
* `POST /api/orders/:id/actions/toggle-step`
* `POST /api/orders/:id/actions/toggle-served`
* `POST /api/orders/actions/clear`
* `POST /api/realtime/session` — issue a one-time WS ticket
* `GET /api/stats/daily` — daily stats for the authenticated workspace

## Realtime WebSocket

A client first calls `POST /api/realtime/session` (authenticated) to obtain a short-lived one-time ticket, then connects to `GET /api/ws?ticket=...`. The broker fans out order events (`internal/domain/order_events.go`) to connected sessions. Heartbeat interval, session TTL, read/write timeouts, and allowed origins are configured under `realtime:` and `ws:`.

## Configuration

Configuration is loaded from `config/config.yaml` by default (see `config/config.example.yaml`), overridable via `-c` and via environment variables (bindings in `internal/infra/config/config.go`, e.g. `DATABASE_URL`, `HTTP_ADDR`, `PASETO_SYMMETRIC_KEY`, `BIZ_TIMEZONE`). Build metadata (`Version`/`Commit`/`BuildTime`/`GoVersion`) lives in `internal/infra/config` and is injected via ldflags; see `docs/agents/commands.md`.