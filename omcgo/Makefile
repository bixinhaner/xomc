.PHONY: build build-acs build-app build-worker build-migrate build-omcctl build-rpctool \
       test test-race test-coverage test-integration test-integration-up test-integration-down test-e2e \
       test-cover cover-report cover-summary cover-check \
       lint vet generate generate-mocks agent-handbook-generate agent-handbook-check sqlc-generate \
       migrate-up migrate-down migrate-reset migrate-schema-up migrate-schema-down migrate-seed-up migrate-seed-down \
       migrate-tsdb-up migrate-tsdb-down \
       docker-build docker-up docker-down clean

# Build
build: build-acs build-app build-worker build-migrate build-omcctl

build-acs:
	go build -o bin/omcgo-acs ./cmd/acs

build-app:
	go build -o bin/omcgo-app ./cmd/app

build-worker:
	go build -o bin/omcgo-worker ./cmd/worker

build-migrate:
	go build -o bin/omcgo-migrate ./cmd/migrate

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
	bash ../deployments/docker/dc-test.sh up -d --wait

test-integration-down:
	bash ../deployments/docker/dc-test.sh down -v

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

agent-handbook-generate:  ## Refresh source-derived Agent handbook contracts and embedded package
	go run ./cmd/agent-handbook -contracts-only

agent-handbook-check:  ## Fail when Agent handbook contracts or embedded package are stale
	go run ./cmd/agent-handbook -check-contracts

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

# 时序库（TsPool / postgres-tsdb 第二实例）schema 迁移 —— 独立 goose 版本表
# goose_db_version_tsdb，独立 DSN。本机裸跑：时序库映射 localhost:5433
# （docker compose 里则由 migrate-tsdb-schema 服务对 postgres-tsdb:5432 跑）。
# 覆盖 DSN：make migrate-tsdb-up TSDB_DSN=postgres://...
TSDB_DSN ?= postgres://omcgo:omcgo123@localhost:5433/omcgo?sslmode=disable

migrate-tsdb-up:  ## Run time-series DB schema migrations (postgres-tsdb)
	go run ./cmd/migrate up --path migrations/tsdb --table goose_db_version_tsdb --dsn "$(TSDB_DSN)"

migrate-tsdb-down:  ## Rollback time-series DB schema migrations (postgres-tsdb)
	go run ./cmd/migrate down --path migrations/tsdb --table goose_db_version_tsdb --dsn "$(TSDB_DSN)"

# 注：原 omcgo-seed (cmd/seed) JSON 种子工具已下线，种子数据全部走 goose
# migrations/seed/ — 用 make migrate-seed-up / migrate-seed-down 操作。

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
