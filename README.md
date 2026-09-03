# Go Real-Time Kanban Board

Multiplayer Kanban board with real-time synchronization of changes between participants.

## Quick start

```sh
docker compose up -d
```

Opens the API on `localhost:8080` and the frontend on `localhost:5173`
(migrations run automatically before the API starts — see "Running the
project" below for the local, non-Docker dev workflow). If a port is
already taken, copy [`.env.example`](.env.example) to `.env` and override
`POSTGRES_PORT`/`API_PORT`/`WEB_PORT` there.

No need to register — log in with the seeded demo account
(`internal/repository/postgres/migrations/000003_demo_user.up.sql`):

```
email:    demo@example.local
password: demo12345
```

## Status

Stages 0-3 done: CRUD for boards/columns/cards end-to-end (API + frontend),
JWT auth, and real-time sync over WebSocket. See
[`docs/ru/roadmap.md`](docs/ru/roadmap.md) for the full stage-by-stage
breakdown and what's next.

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
