package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-migrate",
		Short: "OMC Database Migration Tool",
		Long:  "Database schema migration tool for PostgreSQL/TimescaleDB",
	}

	rootCmd.PersistentFlags().String("dsn", "", "database connection string (e.g. postgres://user:pass@localhost:5432/omcgo?sslmode=disable)")
	rootCmd.PersistentFlags().String("path", "migrations", "migrations directory path")

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
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newMigrate(cmd *cobra.Command) (*migrate.Migrate, error) {
	dsn, _ := cmd.Flags().GetString("dsn")
	if dsn == "" {
		dsn = os.Getenv("OMCGO_DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("--dsn flag or OMCGO_DB_DSN env var is required")
	}

	path, _ := cmd.Flags().GetString("path")
	sourceURL := "file://" + path

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}
	return m, nil
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	m, err := newMigrate(cmd)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ := m.Version()
	fmt.Printf("Migration complete. Version: %d, Dirty: %v\n", version, dirty)
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	m, err := newMigrate(cmd)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}

	version, dirty, verr := m.Version()
	if verr != nil {
		fmt.Println("Migration rolled back. No version set.")
	} else {
		fmt.Printf("Migration rolled back. Version: %d, Dirty: %v\n", version, dirty)
	}
	return nil
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	m, err := newMigrate(cmd)
	if err != nil {
		return err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	fmt.Printf("Version: %d, Dirty: %v\n", version, dirty)
	return nil
}

func runMigrateForce(cmd *cobra.Command, args []string) error {
	m, err := newMigrate(cmd)
	if err != nil {
		return err
	}
	defer m.Close()

	version, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid version number: %w", err)
	}

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version: %w", err)
	}

	fmt.Printf("Forced version to: %d\n", version)
	return nil
}
