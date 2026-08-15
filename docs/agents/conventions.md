# Code Conventions

## Package Layout & Dependency Direction

* `main.go` at the repo root is thin: it only calls `cmd.Execute()`.
* `cmd/` holds cobra commands only; all bootstrap goes through the wire injectors in `internal/app` (`InitializeApplication`, `InitializeUserCreator`), which return a cleanup function for closing resources.
* `internal/app` is the composition root. Providers live in `internal/app/provider.go`, injectors in `internal/app/wire.go`, and generated code in `internal/app/wire_gen.go` (regenerate with `wire ./internal/app` or `go generate`). Services, handlers, and the router are wired here and nowhere else.
* Dependencies flow one way: `handler → service → repo`, with `model` shared by all layers; gin middleware lives in `internal/handler/middleware` and only depends on `service`/`infra`. Never import `handler` from `service`, `service` from `repo`, or `internal/app` from `internal/infra`.
* Cross-cutting concerns (config, crypto, database, httpx, identity, logger, realtime, shutdown, timex) live in `internal/infra` and must not import domain packages.

## Models & Domain Logic

* `internal/model` holds unified domain + ORM models: bun struct tags plus domain methods on the same type (e.g. `Order.CanServe`, `Order.ToggleStep`, `OrderFormInput.InitialStepStatuses`).
* Encode enumerated option values as smallint codes and expose named constants (`StepStatus*`, `RoleOwner`, `RoleStaff`); validate codes in the model.
* Domain validation errors are sentinel `errors.New` values grouped in `internal/model/errors.go`.
* All monetary values are integer cents.
* Keep pure domain rules testable without a database (see `internal/model/*_test.go`).

## Errors

* Surface failures as `*httpx.AppError`: HTTP status + stable machine code + user-facing message + optional cause.
* Use the constructors in `internal/infra/httpx`: `NewError`/`WrapError` and the helpers `ValidationError`, `UnauthorizedError`, `ForbiddenError`, `NotFoundError`, `ConflictError`, `InternalError`.
* Render errors in handlers with `httpx.AbortError(c, err)`. Non-`AppError` errors become a generic `internal_error` 500, so wrap expected failures explicitly.
* Translate sentinels from `model/errors.go` into an `AppError` at the service boundary rather than leaking them raw to the API.

## Time, Money & Formatting

* Store times as `timestamptz`; the test harness pins `timezone=UTC`.
* Services/handlers receive `*time.Location` (from `biz.timezone`, provided by `internal/app.ProvideLocation`); format timestamps with `timex.FormatTime` and never hardcode a zone.
* Keep prices as integer cents end to end; convert only in DTO/display layers.

## Logging

* Use `zerolog` with structured fields (`log.Info().Str("addr", ...).Msg(...)`); the global logger is initialized by `internal/infra/logger.New()` in `internal/app.ProvideLogger`.

## Configuration

* Every new setting needs: an entry in `config/config.example.yaml`, a field on `config.Config` (or a nested struct), and a default + env binding in `internal/infra/config` (`setDefaults`, `bindEnv`).
* Env var names are uppercase snake (e.g. `DATABASE_URL`, `WS_ALLOWED_ORIGINS`).

## Cobra Commands

* New subcommands live in `cmd/` with a `flagXxx` constant per flag; require mandatory flags via `MarkFlagRequired`.
* Commands that need the DB bootstrap through a wire injector (`app.InitializeApplication`, `app.InitializeUserCreator`) and close resources with `defer cleanup()`.
* Keep the root command as the default `serve` behavior; do not add side effects to `init()` beyond flag/subcommand registration.

## Testing

* Unit tests sit next to the code they test (`foo_test.go`) and use `testify`.
* DB integration tests (repo layer, services touching the DB) use testcontainers `postgres:18`; the shared harness lives in `internal/repo/main_test.go` (`TestMain` starts one container, runs migrations, exposes `testDB`).
* Isolate integration tests per transaction: `testDB.BeginTx` + `t.Cleanup(func() { _ = tx.Rollback() })`, passing the tx as `bun.IDB` to the repo under test.
* Run the full suite with `go test ./... -count=1` (Docker required). CI runs `go vet ./...` before tests and fails the build on any failure.

## Build Info

* `internal/infra/config` build variables are set via `-ldflags "-X ..."` (see `docs/agents/commands.md`); keep the `Version`/`Commit`/`BuildTime`/`GoVersion` contract stable — CI and `scripts/build_android.*` rely on it.