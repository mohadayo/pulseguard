.PHONY: test test-python test-go test-ts up down build lint clean

## Run all tests
test: test-python test-go test-ts

## Run Python tests
test-python:
	cd services/analyzer && pip install -q -r requirements.txt && pytest -v

## Run Go tests
test-go:
	cd services/collector && go test -v ./...

## Run TypeScript tests
test-ts:
	cd services/gateway && npm install --silent && npm test

## Start all services with Docker Compose
up:
	docker compose up --build -d

## Stop all services
down:
	docker compose down

## Build all Docker images
build:
	docker compose build

## Run linters
lint: lint-python lint-go lint-ts

## Lint Python
lint-python:
	cd services/analyzer && pip install -q flake8 && flake8 app.py tests/ --max-line-length=120

## Lint Go
lint-go:
	cd services/collector && go vet ./...

## Lint TypeScript
lint-ts:
	cd services/gateway && npm install --silent && npx eslint src/

## Clean build artifacts
clean:
	rm -rf services/gateway/node_modules services/gateway/dist
	rm -rf services/analyzer/__pycache__ services/analyzer/.pytest_cache
	rm -f services/collector/collector

## Show help
help:
	@echo "Available targets:"
	@echo "  make test         - Run all tests"
	@echo "  make test-python  - Run Python tests"
	@echo "  make test-go      - Run Go tests"
	@echo "  make test-ts      - Run TypeScript tests"
	@echo "  make up           - Start all services (Docker Compose)"
	@echo "  make down         - Stop all services"
	@echo "  make build        - Build Docker images"
	@echo "  make lint         - Run all linters"
	@echo "  make clean        - Clean build artifacts"
