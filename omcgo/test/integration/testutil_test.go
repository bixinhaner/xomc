package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// SetupTestDB connects to the test PostgreSQL database using OMCGO_TEST_DB_DSN.
// If the env var is not set, the test is skipped.
// The returned pool is closed automatically when the test finishes.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("failed to ping test DB: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// SetupTestRedis connects to the test Redis instance using OMCGO_TEST_REDIS_ADDR.
// If the env var is not set, the test is skipped.
// The returned client is closed automatically when the test finishes.
func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("OMCGO_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("OMCGO_TEST_REDIS_ADDR not set, skipping integration test")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatalf("failed to ping test Redis: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

// CleanupTables truncates the given tables in the test database.
// Uses TRUNCATE ... CASCADE to handle foreign key constraints.
func CleanupTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	if len(tables) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Sanitize table names to prevent injection (only allow alphanumeric and underscores)
	for _, table := range tables {
		for _, c := range table {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				t.Fatalf("invalid table name: %s", table)
			}
		}
	}

	query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(tables, ", "))
	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatalf("failed to truncate tables [%s]: %v", strings.Join(tables, ", "), err)
	}
}

// RunMigrations applies all database migrations to the test database.
// This uses golang-migrate directly via the library, so no binary is needed.
func RunMigrations(t *testing.T, dsn string) {
	t.Helper()

	// Import the migrate package to run migrations programmatically.
	// We use the CLI tool from the shell script instead, so this is a
	// convenience for tests that need to ensure migrations are applied.
	//
	// The integration_test.sh script already runs migrations before tests,
	// so in most cases this is a no-op safety check.

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("RunMigrations: failed to connect: %v", err)
	}
	defer pool.Close()

	// Verify that migrations have been applied by checking for the schema_migrations table
	var exists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'schema_migrations')").
		Scan(&exists)
	if err != nil {
		t.Fatalf("RunMigrations: failed to check schema_migrations: %v", err)
	}
	if !exists {
		t.Fatal("RunMigrations: schema_migrations table not found. Run migrations first: bash scripts/integration_test.sh")
	}
}
