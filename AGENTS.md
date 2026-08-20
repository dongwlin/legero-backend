# AGENTS.md

## Mandatory Pre-task Checklist

Complete every step below before starting task work. Skipping any step is a
process violation: stop, complete the step, then continue.

A delegated instruction reference is a reference through which a section
defers its detailed instructions to another document. Delegated references
are mandatory reading; ordinary documentation references remain on-demand.

1. Read this file (`AGENTS.md`) in full.
2. Resolve every delegated instruction reference in this file and read each
   applicable referenced document in full.
3. Only then begin task work.

## Project Overview

Legero is a restaurant order-management backend written in Go: a JSON HTTP API
(Gin) for users, workspaces, and orders, a WebSocket channel for real-time
order updates, and a Cobra CLI (`serve`, `create-user`, `version`).

For unfamiliar areas, see `docs/agents/project.md` for the tech stack,
repository layout, architecture, domain model, and API surface.

## Common Commands

* Build / vet: `go build ./...`, `go vet ./...`
* Run the server: `go run . serve -c config/config.yaml`
* Test: `go test ./... -count=1` — integration tests need Docker (testcontainers, postgres:18)
* Create a user: `go run . create-user --phone <phone> --password <password> [--workspace <name> | --workspace-id <uuid>] [--role owner|staff]`

See `docs/agents/commands.md` when additional command details are needed,
including config setup, CLI flags, Android builds, build-info ldflags, and CI.

## Specs

Feature behavior specs are delegated to the `docs/specs/` directory; read every
spec applicable to the task before starting task work. Specs are normative —
when code conflicts with a spec, fix the code or update the spec deliberately.

* `docs/specs/api-etag.md` — ETag generation and `If-None-Match` revalidation
* `docs/specs/api-versioning.md` — response structure changes require new route versions
* `docs/specs/api-healthz.md` — liveness endpoint excluded from ETag and cache handling

## Conventions

Instructions are delegated to `docs/agents/conventions.md`.

## Skills

Instructions are delegated to `docs/agents/skills.md`.

## Sub-agents
Instructions are delegated to `docs/agents/subagents.md`.

## Local Environment

If `AGENTS.local.md` exists in the repository root, machine-local instructions
and context are delegated to it.

`AGENTS.local.md` supplements this file and must not override repository-wide
requirements unless explicitly permitted here.