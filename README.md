# Go Real-Time Kanban Board

Multiplayer Kanban board with real-time synchronization of changes between participants.

## Status

Stage 0 done (project skeleton: HTTP server, DB connectivity, migrations, sqlc wired up). No domain features yet — see the roadmap below.

## Docs

- [`docs/ru/vision.md`](docs/ru/vision.md) — scope, domain model, use cases
- [`docs/ru/architecture.md`](docs/ru/architecture.md) — internal design notes
- [`docs/ru/roadmap.md`](docs/ru/roadmap.md) — stage-by-stage implementation plan
- [`docs/ru/adr/`](docs/ru/adr/) — architecture decision records
- [`docs/ru/openapi.yaml`](docs/ru/openapi.yaml) — REST API spec

## Running the project

Requires Docker (with Compose) and Go 1.26+.

```sh
make up             # start Postgres
make db-sync        # run migrations + sqlc generate
make run            # start the API server on :8080
curl localhost:8080/health
```

Or run the whole stack (Postgres + migrations + API + frontend) in containers:

```sh
make dev             # same as: docker compose up --build
```

This starts the API on `localhost:8080` (or `$API_PORT`) and the frontend
on `localhost:5173` (or `$WEB_PORT`), with migrations applied automatically
before the API starts.

Run tests:

```sh
make test
```
