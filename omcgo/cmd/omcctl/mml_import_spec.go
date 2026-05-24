package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/omcgo/omcgo/internal/mml/specparser"
)

// newMMLImportSpecMDCmd 注册 `omcctl mml import-spec-md` 子命令（T-0169）。
//
// 解析 cmcc-tdlte-southbound-data-model-v2.3.md → 与 DB diff → 生成 goose seed SQL
// + 同步生成 data/mml-catalog/cmcc-tdlte-v2.3.json。
//
// 完整设计文档：docs/project/prd/F06-mml-catalog-spec-parser.md
func newMMLImportSpecMDCmd() *cobra.Command {
	var (
		flagSpec    string
		flagDiffDB  string
		flagCarrier string
		flagVersion string
		flagOut     string
		flagOutJSON string
		flagDryRun  bool
		flagVerbose bool
	)

	cmd := &cobra.Command{
		Use:   "import-spec-md",
		Short: "Parse southbound spec markdown → generate incremental MML catalog seed SQL + JSON",
		Long: `Parse a southbound data model spec markdown (e.g. cmcc-tdlte-southbound-data-model-v2.3.md),
diff against the current DB (mml_commands / mml_command_sub_fields / standard_params),
and emit:
  1. An idempotent goose seed SQL (--out)
  2. A catalog JSON file (--out-json, optional)

The spec MD is the AUTHORITATIVE source for the 71 group_codes (§R-2.4) and their
path sets (§SA-SR detail tables). Existing source='admin' rows in DB are never
modified or deleted.

Examples:

  # Dry-run: just print diff summary, no files written
  omcctl mml import-spec-md \
    --spec=omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md \
    --diff-db=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
    --dry-run

  # Generate seed SQL + JSON
  omcctl mml import-spec-md \
    --spec=omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md \
    --diff-db=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
    --out=omcgo/migrations/seed/000172_mml_catalog_v23_incremental.sql \
    --out-json=omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagSpec == "" {
				return fmt.Errorf("--spec is required")
			}
			if flagVersion == "" {
				flagVersion = "cmcc-td-lte-" + flagCarrier
			}

			// 1. Parse spec md
			logf(flagVerbose, "→ Parsing spec md: %s", flagSpec)
			cat, err := specparser.ParseMarkdown(flagSpec, flagVersion)
			if err != nil {
				return fmt.Errorf("parse spec md: %w", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Parsed %d groups (§R-2.4), %d blacklist entries (§R-3.2), spec hash=%s\n",
				len(cat.Groups), len(cat.Blacklist), cat.SourceHash)

			// 2. Load DB snapshot if --diff-db provided
			var snap *specparser.DBSnapshot
			if flagDiffDB != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				pool, err := openPgPool(ctx, flagDiffDB)
				if err != nil {
					return fmt.Errorf("open db: %w", err)
				}
				defer pool.Close()
				logf(flagVerbose, "→ Loading DB snapshot")
				snap, err = specparser.LoadDBSnapshot(ctx, pool, flagVersion)
				if err != nil {
					return fmt.Errorf("load db snapshot: %w", err)
				}
				fmt.Fprintf(os.Stderr, "✓ DB snapshot: %d standard_params, %d standard commands, %d sub_field groups\n",
					len(snap.StandardParams), len(snap.Commands), len(snap.CommandSubFields))
			} else {
				// 空快照 → 全部当作"全缺"输出（适用于初次 bootstrap）
				snap = &specparser.DBSnapshot{
					StandardParams:   map[string]*specparser.StandardParamRow{},
					Commands:         map[string]*specparser.MMLCommandRow{},
					CommandSubFields: map[string]map[string]*specparser.SubFieldRow{},
				}
				fmt.Fprintln(os.Stderr, "⚠ No --diff-db: assuming empty DB (all rows will be NEW)")
			}

			// 3. Diff
			rep := specparser.Diff(cat, snap)
			fmt.Fprintf(os.Stderr, "✓ Diff summary:\n")
			fmt.Fprintf(os.Stderr, "    🔴 NewStandardParams: %d\n", rep.Summary.NewStandardParams)
			fmt.Fprintf(os.Stderr, "    🔴 NewCommands:       %d\n", rep.Summary.NewCommands)
			fmt.Fprintf(os.Stderr, "    🟡 UpdatedCommands:   %d\n", rep.Summary.UpdatedCommands)
			fmt.Fprintf(os.Stderr, "    🔴 NewSubFieldLinks:  %d\n", rep.Summary.NewSubFieldLinks)
			fmt.Fprintf(os.Stderr, "    ⚪ OrphanCommands:    %d  (marked, not deleted)\n", rep.Summary.OrphanCommands)

			if flagVerbose && len(rep.OrphanCommands) > 0 {
				fmt.Fprintf(os.Stderr, "    Orphan list:\n")
				for _, c := range rep.OrphanCommands {
					fmt.Fprintf(os.Stderr, "      - %s\n", c)
				}
			}

			if flagDryRun {
				fmt.Fprintln(os.Stderr, "✓ Dry-run mode: no files written.")
				return nil
			}

			// 4. Write seed SQL
			if flagOut != "" {
				sql := specparser.GenerateSQL(rep, cat)
				if err := os.WriteFile(flagOut, []byte(sql), 0o644); err != nil {
					return fmt.Errorf("write seed SQL: %w", err)
				}
				fmt.Fprintf(os.Stderr, "✓ Seed SQL written: %s (%d bytes)\n", flagOut, len(sql))
			}

			// 5. Write JSON catalog
			if flagOutJSON != "" {
				js, err := specparser.GenerateJSON(cat)
				if err != nil {
					return fmt.Errorf("generate JSON: %w", err)
				}
				if err := os.WriteFile(flagOutJSON, js, 0o644); err != nil {
					return fmt.Errorf("write JSON catalog: %w", err)
				}
				fmt.Fprintf(os.Stderr, "✓ JSON catalog written: %s (%d bytes)\n", flagOutJSON, len(js))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flagSpec, "spec", "", "Path to spec markdown (required)")
	cmd.Flags().StringVar(&flagDiffDB, "diff-db", os.Getenv("OMCGO_DB_DSN"), "PG DSN for diff; empty=assume empty DB")
	cmd.Flags().StringVar(&flagCarrier, "carrier", "v2.3", "Spec version suffix (cmcc default, future: ctcc/cucc)")
	cmd.Flags().StringVar(&flagVersion, "version", "", "Override mml_param_versions.version_code (default: cmcc-td-lte-<carrier>)")
	cmd.Flags().StringVar(&flagOut, "out", "", "Output seed SQL path (required for non-dry-run)")
	cmd.Flags().StringVar(&flagOutJSON, "out-json", "", "Output catalog JSON path (optional)")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Print summary only, don't write files")
	cmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")
	return cmd
}

func logf(verbose bool, format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
