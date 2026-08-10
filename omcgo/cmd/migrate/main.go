package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-migrate",
		Short: "OMC Database Migration Tool",
		Long:  "Database schema migration tool for PostgreSQL/TimescaleDB (powered by Goose)",
	}

	rootCmd.PersistentFlags().String("dsn", "", "database connection string (e.g. postgres://user:pass@localhost:5432/omcgo?sslmode=disable)")
	rootCmd.PersistentFlags().String("path", "migrations", "migrations directory path")
	rootCmd.PersistentFlags().String("paths", "", "comma-separated migration directories (applied in order)")
	rootCmd.PersistentFlags().String("table", "", "custom goose version table name (default: goose_db_version)")

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Run all pending migrations",
			RunE:  runMigrateUp,
		},
		&cobra.Command{
			Use:   "down",
			Short: "Rollback the last migration",
			RunE:  runMigrateDown,
		},
		&cobra.Command{
			Use:   "down-to [version]",
			Short: "Rollback migrations down to the specified version (exclusive)",
			Args:  cobra.ExactArgs(1),
			RunE:  runMigrateDownTo,
		},
		&cobra.Command{
			Use:   "version",
			Short: "Show current migration version",
			RunE:  runMigrateVersion,
		},
		&cobra.Command{
			Use:   "force [version]",
			Short: "Force set migration version (use to fix dirty state)",
			Args:  cobra.ExactArgs(1),
			RunE:  runMigrateForce,
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Rollback all migrations (drop all tables)",
			RunE:  runMigrateReset,
		},
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// openDB opens a database connection using the DSN from flag or env var.
func openDB(cmd *cobra.Command) (*sql.DB, error) {
	dsn, _ := cmd.Flags().GetString("dsn")
	if dsn == "" {
		dsn = os.Getenv("OMCGO_DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("--dsn flag or OMCGO_DB_DSN env var is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 等待数据库可接受连接再继续。整栈一起重启时，本一次性容器可能早于 postgres
	// 进程开始接受连接的瞬间启动（即便 depends_on: service_healthy +
	// network_mode: service:postgres，仍存在 healthy 与端口 listen 之间的竞态窗口），
	// 不重试就会首连 "connection refused"：轻则刷错误噪声，重则迁移硬失败、阻塞
	// app/acs/worker 的 depends_on 启动链。退避重试令其优雅等待 DB 就绪。
	if err := pingWithRetry(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// pingWithRetry 在固定间隔内重试 Ping，直到数据库可连接或耗尽尝试次数。
func pingWithRetry(db *sql.DB) error {
	const attempts = 30
	const interval = time.Second

	var lastErr error
	for i := 1; i <= attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		fmt.Fprintf(os.Stderr, "等待数据库就绪（%d/%d）：%v\n", i, attempts, err)
		time.Sleep(interval)
	}
	return fmt.Errorf("数据库在 %d 次尝试后仍不可连接: %w", attempts, lastErr)
}

// setupGooseTable configures the goose version table from --table flag or GOOSE_TABLE env var.
func setupGooseTable(cmd *cobra.Command) {
	table, _ := cmd.Flags().GetString("table")
	if table == "" {
		table = os.Getenv("GOOSE_TABLE")
	}
	if table != "" {
		goose.SetTableName(table)
	}
}

// migrateDir returns the migrations directory from flag.
func migrateDir(cmd *cobra.Command) string {
	path, _ := cmd.Flags().GetString("path")
	return path
}

// migratePaths returns the comma-separated migration directories from flag.
// Returns nil if --paths is not set.
func migratePaths(cmd *cobra.Command) []string {
	paths, _ := cmd.Flags().GetString("paths")
	if paths == "" {
		return nil
	}
	parts := strings.Split(paths, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	// If --paths is set, run goose.Up on each directory in sequence.
	if dirs := migratePaths(cmd); dirs != nil {
		for _, dir := range dirs {
			fmt.Printf("Migrating directory: %s\n", dir)
			if err := goose.Up(db, dir, goose.WithAllowMissing()); err != nil {
				return fmt.Errorf("migrate up %s: %w", dir, err)
			}
		}
		version, err := goose.GetDBVersion(db)
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}
		fmt.Printf("Migration complete. Version: %d\n", version)
		return nil
	}

	migrationDir := migrateDir(cmd)
	reconcileMain := isMainSeedMigrationDir(migrationDir)
	if reconcileMain {
		if err := reconcileMainBaselineSchema(db, migrationDir); err != nil {
			return fmt.Errorf("reconcile main baseline schema: %w", err)
		}
	}

	if err := goose.Up(db, migrationDir, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if reconcileMain {
		if err := reconcileMainBaselineSeed(db, migrationDir); err != nil {
			return fmt.Errorf("reconcile main baseline seed: %w", err)
		}
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	fmt.Printf("Migration complete. Version: %d\n", version)
	return nil
}

const (
	mainReconcileBegin = "-- +omcgo MainReconcileBegin"
	mainReconcileEnd   = "-- +omcgo MainReconcileEnd"
)

func isMainSeedMigrationDir(migrationDir string) bool {
	return filepath.Base(filepath.Clean(migrationDir)) == "seed"
}

func mainBaselineRootDir(migrationDir string) string {
	rootDir := filepath.Clean(migrationDir)
	if isMainSeedMigrationDir(rootDir) {
		return filepath.Dir(rootDir)
	}
	return rootDir
}

// Before the first release, the project intentionally keeps only 000001 files;
// Goose therefore cannot detect additive changes inside an applied baseline.
// The seed flow replays explicitly marked schema sections before Goose seed,
// then marked seed sections afterwards, without a forbidden 000002 migration.
func reconcileMainBaselineSchema(db *sql.DB, migrationDir string) error {
	rootDir := mainBaselineRootDir(migrationDir)
	schemaSQL, err := readMainReconcileSQL(filepath.Join(rootDir, "000001_init_schema.sql"))
	if err != nil {
		return fmt.Errorf("read schema reconcile sections: %w", err)
	}
	if err := applyMainReconcileSQL(db, schemaSQL); err != nil {
		return fmt.Errorf("apply schema reconcile sections: %w", err)
	}
	return nil
}

func reconcileMainBaselineSeed(db *sql.DB, migrationDir string) error {
	rootDir := mainBaselineRootDir(migrationDir)
	seedSQL, err := readMainReconcileSQL(filepath.Join(rootDir, "seed", "000001_init_seed.sql"))
	if err != nil {
		return fmt.Errorf("read seed reconcile sections: %w", err)
	}
	if err := applyMainReconcileSQL(db, seedSQL); err != nil {
		return fmt.Errorf("apply seed reconcile sections: %w", err)
	}
	fmt.Println("Main baseline reconciliation complete.")
	return nil
}

func applyMainReconcileSQL(db *sql.DB, sections ...string) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin main baseline reconcile: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended('omcgo-main-baseline-reconcile', 0))",
	); err != nil {
		return fmt.Errorf("lock main baseline reconcile: %w", err)
	}
	for i, section := range sections {
		if _, err := tx.ExecContext(ctx, section); err != nil {
			return fmt.Errorf("apply main baseline reconcile section %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit main baseline reconcile: %w", err)
	}

	return nil
}

func readMainReconcileSQL(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return extractMainReconcileSQL(string(contents))
}

func extractMainReconcileSQL(contents string) (string, error) {
	var sections []string
	remainder := contents
	for {
		begin := strings.Index(remainder, mainReconcileBegin)
		orphanEnd := strings.Index(remainder, mainReconcileEnd)
		if orphanEnd >= 0 && (begin < 0 || orphanEnd < begin) {
			return "", fmt.Errorf("reconcile end marker has no matching begin marker")
		}
		if begin < 0 {
			break
		}
		remainder = remainder[begin+len(mainReconcileBegin):]
		end := strings.Index(remainder, mainReconcileEnd)
		if end < 0 {
			return "", fmt.Errorf("reconcile section is missing end marker")
		}
		section := strings.TrimSpace(remainder[:end])
		if section == "" {
			return "", fmt.Errorf("reconcile section is empty")
		}
		sections = append(sections, section)
		remainder = remainder[end+len(mainReconcileEnd):]
	}
	if len(sections) == 0 {
		return "", fmt.Errorf("no reconcile sections found")
	}
	return strings.Join(sections, "\n\n"), nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Down(db, migrateDir(cmd)); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		fmt.Println("Migration rolled back. No version set.")
		return nil
	}
	fmt.Printf("Migration rolled back. Version: %d\n", version)
	return nil
}

func runMigrateDownTo(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	version, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version number: %w", err)
	}

	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.DownTo(db, migrateDir(cmd), version); err != nil {
		return fmt.Errorf("migrate down-to %d: %w", version, err)
	}

	current, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	fmt.Printf("Rolled back to version: %d\n", current)
	return nil
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	fmt.Printf("Version: %d\n", version)
	return nil
}

func runMigrateForce(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	version, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version number: %w", err)
	}

	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	// Ensure version table exists, then set the version directly.
	tableName := goose.TableName()
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", tableName)); err != nil {
		return fmt.Errorf("clear version table: %w", err)
	}
	if _, err := db.Exec(
		fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, true)", tableName),
		version,
	); err != nil {
		return fmt.Errorf("set version %d: %w", version, err)
	}

	fmt.Printf("Forced version to: %d\n", version)
	return nil
}

func runMigrateReset(cmd *cobra.Command, args []string) error {
	setupGooseTable(cmd)
	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Reset(db, migrateDir(cmd)); err != nil {
		return fmt.Errorf("migrate reset: %w", err)
	}

	fmt.Println("All migrations rolled back. Database is empty.")
	return nil
}
