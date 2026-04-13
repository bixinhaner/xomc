.PHONY: build build-acs build-app build-worker build-migrate build-seed build-omcctl build-rpctool \
       test test-race test-coverage test-integration test-integration-up test-integration-down test-e2e \
       test-cover cover-report cover-summary cover-check \
       lint vet generate generate-mocks sqlc-generate \
       migrate-up migrate-down migrate-reset migrate-schema-up migrate-schema-down migrate-seed-up migrate-seed-down \
       seed-apply seed-rollback seed-list \
       docker-build docker-up docker-down clean

# Build
build: build-acs build-app build-worker build-migrate build-seed build-omcctl

build-acs:
	go build -o bin/omcgo-acs ./cmd/acs

build-app:
	go build -o bin/omcgo-app ./cmd/app

build-worker:
	go build -o bin/omcgo-worker ./cmd/worker

build-migrate:
	go build -o bin/omcgo-migrate ./cmd/migrate

build-seed:
	go build -o bin/omcgo-seed ./cmd/seed

build-omcctl:
	go build -o bin/omcctl ./cmd/omcctl

build-rpctool:
	go build -o bin/rpctool ./scripts/rpctool

# Test
test:
	go test ./... -v -count=1

test-race:  ## Run tests with race detector (slower, use for CI or pre-release)
	go test -race -count=1 ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-cover:  ## Run tests with coverage profile
	go test -coverprofile=coverage.out ./...

cover-report: test-cover  ## Generate HTML coverage report
	go tool cover -html=coverage.out -o coverage.html

cover-summary: test-cover  ## Print per-package coverage summary
	go tool cover -func=coverage.out

COVER_THRESHOLD ?= 40
cover-check: test-cover  ## Check coverage meets minimum threshold (default 40%)
	@total=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	echo "Total coverage: $${total}%  (threshold: $(COVER_THRESHOLD)%)"; \
	if [ $$(echo "$${total} < $(COVER_THRESHOLD)" | bc) -eq 1 ]; then \
		echo "FAIL: coverage $${total}% is below threshold $(COVER_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "OK: coverage $${total}% meets threshold $(COVER_THRESHOLD)%"; \
	fi

test-integration:
	bash scripts/integration_test.sh

test-integration-up:
	docker compose -f ../deployments/docker/docker-compose.test.yml up -d --wait

test-integration-down:
	docker compose -f ../deployments/docker/docker-compose.test.yml down -v

test-e2e:  ## Run end-to-end tests (requires Docker infrastructure)
	bash scripts/e2e_test.sh

# Quality
lint:
	golangci-lint run ./...

vet:
	go vet ./...

# Generate
generate: generate-mocks
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto
	@echo "Generating swagger docs..."
	swag init -g cmd/app/main.go -o api/openapi

generate-mocks:  ## Generate mock implementations for key interfaces
	go generate ./internal/core/storage/... ./internal/admin/... ./internal/device/... ./internal/topology/...

sqlc-generate:  ## Generate type-safe Go code from SQL queries
	sqlc generate

sqlc-validate:  ## Validate sqlc queries against the database schema
	sqlc vet

# Database
migrate-up:
	go run ./cmd/migrate up --paths migrations,migrations/seed

migrate-down:
	go run ./cmd/migrate down --path migrations

migrate-reset:
	go run ./cmd/migrate reset --path migrations

migrate-schema-up:  ## Run schema migrations only
	go run ./cmd/migrate up --path migrations

migrate-schema-down:  ## Rollback schema migrations only
	go run ./cmd/migrate down --path migrations

migrate-seed-up:  ## Run seed data migrations only
	go run ./cmd/migrate up --path migrations/seed

migrate-seed-down:  ## Rollback seed data migrations only
	go run ./cmd/migrate down --path migrations/seed

# Seed data management
seed-apply:
	go run ./cmd/seed apply

seed-rollback:
	go run ./cmd/seed rollback

seed-list:
	go run ./cmd/seed list

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	@next=$$(ls migrations/*.sql 2>/dev/null | wc -l | tr -d ' '); \
		next=$$((next + 1)); \
		fname=$$(printf "migrations/%06d_%s.sql" $$next "$(name)"); \
		printf "-- +goose Up\n\n-- +goose Down\n" > $$fname; \
		echo "Created $$fname"

# Docker
docker-build:
	docker build -f ../deployments/docker/Dockerfile.acs -t omcgo-acs:latest .
	docker build -f ../deployments/docker/Dockerfile.app -t omcgo-app:latest .
	docker build -f ../deployments/docker/Dockerfile.worker -t omcgo-worker:latest .

docker-up:
	docker-compose -f ../deployments/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f ../deployments/docker/docker-compose.yml down

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
