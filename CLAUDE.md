# Kanban Board — context for Claude

Pet project to demonstrate Go skills ahead of a technical interview
(not a commercial MVP). Full description of the goal, domain, and
engineering principles is in [`docs/ru/vision.md`](docs/ru/vision.md),
not duplicated here.

## Where to find things

- **What we're building and why** → `docs/ru/vision.md`
- **How it's built internally** → `docs/ru/architecture.md`
- **Stage-by-stage plan, what's already done** → `docs/ru/roadmap.md`
  (`[x]` checkboxes are the source of truth for progress, don't rely
  on git log)
- **Why technology X was chosen over Y** → `docs/ru/adr/*.md`
- **REST API** → `docs/ru/openapi.yaml`
- **WebSocket event format** → `docs/ru/websocket-events.md`
- **Out-of-scope ideas** → `docs/ru/future-ideas.md`

## Language

- All code — identifiers, comments, commit messages, error messages,
  log strings — must be written in English.
- The only exception is documentation under `docs/ru/` (vision,
  architecture, roadmap, ADRs), which stays in Russian.

## Stack

Go (`net/http` + chi, see ADR 001) · PostgreSQL + sqlc + pgx (ADR 003)
· `coder/websocket` for real-time (ADR 002) · React + dnd-kit + Tailwind
CSS on the frontend · Docker Compose.

## How to run / verify

```sh
make up          # Postgres in Docker
make db-sync     # migrations + sqlc generate
make run         # API on :8080
make test        # go test -race ./...
```

@CLAUDE.local.md
