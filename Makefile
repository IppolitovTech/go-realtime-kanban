.PHONY: up down migrate-up migrate-down sqlc-generate db-sync run test test-integration

up:
	docker compose up -d postgres

down:
	docker compose down

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

# Requires `make up` (Postgres reachable on localhost:5432) — see ADR 004
# and roadmap.md, Stage 1, for what these cover.
test-integration:
	go test -race -tags=integration ./internal/service/...
