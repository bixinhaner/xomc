package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-seed",
		Short: "OMC Seed Data Management Tool",
		Long:  "Manage seed data for the OMC system (roles, users, permissions, KPIs, etc.)",
	}

	rootCmd.PersistentFlags().String("dsn", "", "database connection string")
	rootCmd.PersistentFlags().String("dir", "datamodels/seed", "seed data directory")
	rootCmd.PersistentFlags().String("env", "dev", "environment: dev, test, prod")

	rootCmd.AddCommand(
		&cobra.Command{Use: "apply", Short: "Apply all seed data to the database", RunE: runSeedApply},
		&cobra.Command{Use: "rollback", Short: "Rollback seed data (truncate seeded tables)", RunE: runSeedRollback},
		&cobra.Command{Use: "list", Short: "List available seed data files", RunE: runSeedList},
		&cobra.Command{Use: "validate", Short: "Validate seed data JSON files", RunE: runSeedValidate},
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func openPool(cmd *cobra.Command) (*pgxpool.Pool, error) {
	dsn, _ := cmd.Flags().GetString("dsn")
	if dsn == "" {
		dsn = os.Getenv("OMCGO_DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("--dsn flag or OMCGO_DB_DSN env var is required")
	}
	ctx := context.Background()
	return pgxpool.New(ctx, dsn)
}

func seedDir(cmd *cobra.Command) string {
	dir, _ := cmd.Flags().GetString("dir")
	return dir
}

func runSeedApply(cmd *cobra.Command, args []string) error {
	dir := seedDir(cmd)
	env, _ := cmd.Flags().GetString("env")
	_ = env

	files, err := loadSeedFiles(dir)
	if err != nil {
		return err
	}

	pool, err := openPool(cmd)
	if err != nil {
		return err
	}
	defer pool.Close()

	ctx := context.Background()
	for _, sf := range files {
		if sf.Env != "" && sf.Env != env {
			fmt.Printf("Skipping seed (env mismatch): %s (requires %s, current %s)\n", sf.Name, sf.Env, env)
			continue
		}
		fmt.Printf("Applying seed: %s (%d records)...\n", sf.Name, len(sf.Records))
		if err := applySeed(ctx, pool, sf); err != nil {
			return fmt.Errorf("apply seed %s: %w", sf.Name, err)
		}
	}
	fmt.Println("Seed data applied successfully.")
	return nil
}

func runSeedRollback(cmd *cobra.Command, args []string) error {
	dir := seedDir(cmd)
	files, err := loadSeedFiles(dir)
	if err != nil {
		return err
	}

	pool, err := openPool(cmd)
	if err != nil {
		return err
	}
	defer pool.Close()

	ctx := context.Background()
	// Rollback in reverse order
	for i := len(files) - 1; i >= 0; i-- {
		sf := files[i]
		fmt.Printf("Rolling back seed: %s...\n", sf.Name)
		if err := rollbackSeed(ctx, pool, sf); err != nil {
			return fmt.Errorf("rollback seed %s: %w", sf.Name, err)
		}
	}
	fmt.Println("Seed data rolled back successfully.")
	return nil
}

func runSeedList(cmd *cobra.Command, args []string) error {
	dir := seedDir(cmd)
	files, err := loadSeedFiles(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Seed data files in %s:\n", dir)
	fmt.Printf("%-25s %-15s %s\n", "NAME", "TABLE", "RECORDS")
	separator := strings.Repeat("=", 25) + " " + strings.Repeat("=", 15) + " " + strings.Repeat("=", 7)
	fmt.Println(separator)
	for _, sf := range files {
		fmt.Printf("%-25s %-15s %d\n", sf.Name+".json", sf.Table, len(sf.Records))
	}
	return nil
}

func runSeedValidate(cmd *cobra.Command, args []string) error {
	dir := seedDir(cmd)
	files, err := loadSeedFiles(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Validating %d seed files...\n", len(files))
	for _, sf := range files {
		if sf.Table == "" {
			return fmt.Errorf("%s: missing table field", sf.Name)
		}
		if len(sf.Records) == 0 {
			fmt.Printf("  WARNING: %s has no records\n", sf.Name)
		} else {
			fmt.Printf("  OK: %s (%d records -> %s)\n", sf.Name, len(sf.Records), sf.Table)
		}
	}
	fmt.Println("All seed files valid.")
	return nil
}

// SeedFile represents a parsed seed data file.
type SeedFile struct {
	Name    string                   `json:"name"`
	Table   string                   `json:"table"`
	Columns []string                 `json:"columns"`
	Records []map[string]interface{} `json:"records"`
	Env     string                   `json:"env,omitempty"` // environment restriction
}

func loadSeedFiles(dir string) ([]SeedFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read seed directory %s: %w", dir, err)
	}

	var files []SeedFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var sf SeedFile
		if err := json.Unmarshal(data, &sf); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		sf.Name = entry.Name()[:len(entry.Name())-5] // strip .json
		files = append(files, sf)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func applySeed(ctx context.Context, pool *pgxpool.Pool, sf SeedFile) error {
	if len(sf.Records) == 0 {
		return nil
	}

	for _, record := range sf.Records {
		cols := make([]string, 0, len(record))
		vals := make([]interface{}, 0, len(record))
		for k, v := range record {
			cols = append(cols, k)
			vals = append(vals, v)
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
			sf.Table,
			joinCols(cols),
			placeholderStr(len(cols)),
		)

		_, err := pool.Exec(ctx, query, vals...)
		if err != nil {
			return fmt.Errorf("insert into %s: %w", sf.Table, err)
		}
	}
	return nil
}

func rollbackSeed(ctx context.Context, pool *pgxpool.Pool, sf SeedFile) error {
	// Truncate the table (use with caution in production)
	_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", sf.Table))
	return err
}

func joinCols(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}

func placeholderStr(n int) string {
	result := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			result += ", "
		}
		result += fmt.Sprintf("$%d", i)
	}
	return result
}
