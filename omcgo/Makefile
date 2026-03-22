.PHONY: build build-acs build-app build-worker build-migrate build-omcctl \
       test test-coverage test-integration test-integration-up test-integration-down \
       lint vet generate \
       migrate-up migrate-down docker-build docker-up docker-down clean

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

# Test
test:
	go test ./... -v -race -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	bash scripts/integration_test.sh

test-integration-up:
	docker compose -f deployments/docker/docker-compose.test.yml up -d --wait

test-integration-down:
	docker compose -f deployments/docker/docker-compose.test.yml down -v

# Quality
lint:
	golangci-lint run ./...

vet:
	go vet ./...

# Generate
generate:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto
	@echo "Generating swagger docs..."
	swag init -g cmd/app/main.go -o api/openapi

# Database
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(name)

# Docker
docker-build:
	docker build -f deployments/docker/Dockerfile.acs -t omcgo-acs:latest .
	docker build -f deployments/docker/Dockerfile.app -t omcgo-app:latest .
	docker build -f deployments/docker/Dockerfile.worker -t omcgo-worker:latest .

docker-up:
	docker-compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker/docker-compose.yml down

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
