.PHONY: up down migrate-up migrate-down sqlc-generate db-sync run test

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
