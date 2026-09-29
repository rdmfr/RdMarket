.DEFAULT_GOAL := help

.PHONY: help setup dev build test lint migrate-up migrate-down seed-dev e2e compose-up compose-down contract-check

help:
	@echo "Targets: setup dev build test lint migrate-up migrate-down seed-dev e2e compose-up compose-down contract-check"

setup:
	cd frontend && npm ci
	cd backend && go mod download

dev:
	docker compose up --build

build:
	cd backend && go build -o /dev/null ./...
	cd frontend && npm run build

test:
	cd backend && go test ./...
	cd frontend && npm run test

lint:
	cd backend && gofmt -l . && go vet ./...
	cd frontend && npm run lint

migrate-up:
	cd backend && go run ./cmd/server migrate-up

migrate-down:
	cd backend && go run ./cmd/server migrate-down

seed-dev:
	cd backend && go run ./cmd/server seed-dev

e2e:
	cd e2e && npx playwright test

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

contract-check:
	cd backend && go test -v -run TestOpenAPIContract ./...
	@test -f docs/openapi.yaml
	@echo "Contract check passed: docs/openapi.yaml"
