.PHONY: up down dev dev-backend migrate-up migrate-down sqlc-generate db-sync run test test-integration test-docker

up:
	docker compose up -d postgres

down:
	docker compose down

# Whole stack in one command: Postgres, migrations, API, frontend.
dev:
	docker compose up --build

# Same, minus the frontend container — run `npm run dev` in web/ yourself.
dev-backend:
	docker compose up --build postgres api migrate

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down 1

sqlc-generate:
	docker compose run --rm sqlc generate

db-sync: migrate-up sqlc-generate

run:
	go run ./cmd/api

test:
	go test -race ./...

# Fallback for machines without a local C compiler (`-race` needs cgo,
# which needs gcc/clang) — runs the same command inside golang:1.26.
test-docker:
	MSYS_NO_PATHCONV=1 docker run --rm -v "$(CURDIR)":/app -w /app -e GOCACHE=/tmp/gocache golang:1.26 go test -race ./...

# Requires `make up` (Postgres reachable on localhost:5432) — see ADR 004
# and roadmap.md, Stage 1, for what these cover.
test-integration:
	go test -race -tags=integration ./internal/service/...
