package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	return db, nil
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
	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	// If --paths is set, run goose.Up on each directory in sequence.
	if dirs := migratePaths(cmd); dirs != nil {
		for _, dir := range dirs {
			fmt.Printf("Migrating directory: %s\n", dir)
			if err := goose.Up(db, dir); err != nil {
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

	if err := goose.Up(db, migrateDir(cmd)); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	fmt.Printf("Migration complete. Version: %d\n", version)
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
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
	version, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version number: %w", err)
	}

	db, err := openDB(cmd)
	if err != nil {
		return err
	}
	defer db.Close()

	// Ensure goose_db_version table exists, then set the version directly.
	if _, err := goose.EnsureDBVersion(db); err != nil {
		return fmt.Errorf("ensure version table: %w", err)
	}

	// Delete all existing version rows and insert the target version.
	if _, err := db.Exec("DELETE FROM goose_db_version"); err != nil {
		return fmt.Errorf("clear version table: %w", err)
	}
	if _, err := db.Exec(
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)",
		version,
	); err != nil {
		return fmt.Errorf("set version %d: %w", version, err)
	}

	fmt.Printf("Forced version to: %d\n", version)
	return nil
}

func runMigrateReset(cmd *cobra.Command, args []string) error {
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
