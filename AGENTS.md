# AGENTS.md

## Project Overview

Legero is a restaurant order-management backend written in Go: a JSON HTTP API (Gin) for users, workspaces, and orders, a WebSocket channel for real-time order updates, and a Cobra CLI (`serve`, `create-user`, `version`). Read [`docs/agents/project.md`](docs/agents/project.md) before working on unfamiliar areas — it covers the tech stack, repository layout, architecture, domain model, and API surface.

## Common Commands

* Build / vet: `go build ./...`, `go vet ./...`
* Run the server: `go run . serve -c config/config.yaml`
* Test: `go test ./... -count=1` — integration tests need Docker (testcontainers, postgres:18)
* Create a user: `go run . create-user --phone <phone> --password <password> [--workspace <name> | --workspace-id <uuid>] [--role owner|staff]`

See [`docs/agents/commands.md`](docs/agents/commands.md) for the full command reference (config setup, CLI flags, Android builds, build-info ldflags, CI).

## Conventions

Before writing or modifying code, read and follow [`docs/agents/conventions.md`](docs/agents/conventions.md): package layout and dependency direction, model and error conventions, time/money handling, logging, config, and testing rules.

## Skills

Read and follow [`docs/agents/skills.md`](docs/agents/skills.md) for skill discovery, selection, and loading rules.

## Sub-agents

Read and follow [`docs/agents/subagents.md`](docs/agents/subagents.md) for sub-agent delegation, orchestration, and model-selection rules.