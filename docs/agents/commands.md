# Common Commands

## Prerequisites

* Go 1.25+
* Docker — required for tests: integration tests spin up a `postgres:18` container via testcontainers
* A PostgreSQL instance for local runs; config comes from `config/config.yaml`

## Setup

Copy the example config and adjust it:

    cp config/config.example.yaml config/config.yaml

`config/config.yaml` is gitignored. Every YAML key can also be set via environment variables; bindings live in `internal/infra/config/config.go` (e.g. `DATABASE_URL`, `HTTP_ADDR`, `PASETO_SYMMETRIC_KEY`, `BIZ_TIMEZONE`).

## Build & Vet

    go build ./...
    go vet ./...

## Tests

    go test ./... -count=1 -timeout 15m   # CI equivalent
    go test ./internal/repo/... -count=1  # DB integration tests only

Integration tests require Docker. Prefer `-count=1` to avoid cached results where DB state matters.

## Run the Server

    go run .                     # no subcommand ⇒ serve
    go run . serve               # explicit
    go run . serve -c /path/to/config.yaml

On startup the server runs DB migrations, then listens on the configured address (default `:8080`) and shuts down gracefully on SIGINT/SIGTERM (30s drain).

## CLI Reference

    go run . version                    # print version
    go run . version --build-info       # version + commit + build time + go version
    go run . create-user --phone 13800000000 --password secret
    go run . create-user --phone 13800000000 --password secret --workspace "My Restaurant"
    go run . create-user --phone 13800000000 --password secret --workspace-id <uuid> --role staff

`create-user` requires `--phone` and `--password`. With `--workspace-id` the user is attached to an existing workspace; otherwise a new workspace is created from `--workspace`. `--role` is `owner` (default) or `staff`.

## Android Build

    scripts/build_android.sh     # Linux/macOS
    scripts/build_android.ps1    # Windows

Output: `bin/android/legero` (android/arm64, CGO disabled). The scripts derive version/commit from git and inject build info via ldflags.

## Build Info Injection

`internal/infra/config` exposes `Version`, `Commit`, `BuildTime`, `GoVersion` (defaults `dev`/`none`/`unknown`). Inject at build time:

    go build -trimpath -ldflags "-s -w \
      -X github.com/dongwlin/legero-backend/internal/infra/config.Version=v1.0.0 \
      -X github.com/dongwlin/legero-backend/internal/infra/config.Commit=abc123 \
      -X github.com/dongwlin/legero-backend/internal/infra/config.BuildTime=2026-01-01T00:00:00Z" .

## CI

`.github/workflows/ci.yml` runs on push/PR to `main` and on tags:

1. `go vet ./...`
2. Regenerate `internal/app/wire_gen.go` with the go.mod-pinned wire and fail if `git diff` is non-empty (generated-code freshness check)
3. `go test ./... -count=1 -timeout 15m` (testcontainers)
4. Cross-build `linux-amd64`, `linux-arm64`, `android-arm64` (`CGO_ENABLED=0`) and upload artifacts
5. On `v*` tags: publish a GitHub release with the artifacts
