.PHONY: dev migrate test test-backend test-frontend build-backend lint-backend lint-frontend

dev:
	docker compose up --build

migrate:
	docker compose --profile migrate run --rm migrate

test: test-backend test-frontend

test-backend:
	cd backend && go test ./... -v

test-frontend:
	cd frontend && npm test -- --run

build-backend:
	cd backend && go build -o bin/server ./cmd/server

lint-backend:
	cd backend && go vet ./...

lint-frontend:
	cd frontend && npm run lint
