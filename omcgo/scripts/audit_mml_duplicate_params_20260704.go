package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultDSN = "postgres://omcgo:omcgo123@172.19.1.73:5432/omcgo?sslmode=disable"

type metric struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
	Note  string `json:"note,omitempty"`
}

type outputSummary struct {
	GeneratedAt string   `json:"generated_at"`
	DSN         string   `json:"dsn"`
	Metrics     []metric `json:"metrics"`
	Outputs     []string `json:"outputs"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		fatalf("ping db: %v", err)
	}

	outDir := defaultOutputDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir output: %v", err)
	}

	files := []string{}
	crossCommandCSV := filepath.Join(outDir, "mml-duplicate-standard-paths-cross-command-20260704.csv")
	if err := writeQueryCSV(ctx, db, crossCommandDuplicateSQL, crossCommandCSV); err != nil {
		fatalf("write cross-command duplicates: %v", err)
	}
	files = append(files, crossCommandCSV)

	sameOperationCSV := filepath.Join(outDir, "mml-duplicate-standard-paths-same-operation-20260704.csv")
	if err := writeQueryCSV(ctx, db, sameOperationDuplicateSQL, sameOperationCSV); err != nil {
		fatalf("write same-operation duplicates: %v", err)
	}
	files = append(files, sameOperationCSV)

	sameCommandCSV := filepath.Join(outDir, "mml-duplicate-standard-paths-same-command-20260704.csv")
	if err := writeQueryCSV(ctx, db, sameCommandDuplicateSQL, sameCommandCSV); err != nil {
		fatalf("write same-command duplicates: %v", err)
	}
	files = append(files, sameCommandCSV)

	mmlCodeCSV := filepath.Join(outDir, "mml-duplicate-mml-code-same-command-20260704.csv")
	if err := writeQueryCSV(ctx, db, duplicateMMLCodeSQL, mmlCodeCSV); err != nil {
		fatalf("write duplicate mml code: %v", err)
	}
	files = append(files, mmlCodeCSV)

	targetPathCSV := filepath.Join(outDir, "mml-duplicate-target-paths-json-20260704.csv")
	if err := writeQueryCSV(ctx, db, duplicateTargetPathsSQL, targetPathCSV); err != nil {
		fatalf("write target path duplicates: %v", err)
	}
	files = append(files, targetPathCSV)

	metrics, err := readMetrics(ctx, db)
	if err != nil {
		fatalf("read metrics: %v", err)
	}

	summary := outputSummary{
		GeneratedAt: time.Now().Format(time.RFC3339),
		DSN:         dsn,
		Metrics:     metrics,
		Outputs:     files,
	}
	summaryPath := filepath.Join(outDir, "mml-duplicate-params-audit-20260704.summary.json")
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fatalf("marshal summary: %v", err)
	}
	if err := os.WriteFile(summaryPath, append(data, '\n'), 0o644); err != nil {
		fatalf("write summary: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fatalf("encode stdout: %v", err)
	}
}

func readMetrics(ctx context.Context, db *sql.DB) ([]metric, error) {
	rows, err := db.QueryContext(ctx, metricsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []metric{}
	for rows.Next() {
		var item metric
		if err := rows.Scan(&item.Name, &item.Value, &item.Note); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func writeQueryCSV(ctx context.Context, db *sql.DB, sqlText, path string) error {
	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write(cols); err != nil {
		return err
	}

	raw := make([]sql.NullString, len(cols))
	dest := make([]any, len(cols))
	for i := range raw {
		dest[i] = &raw[i]
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		record := make([]string, len(cols))
		for i, value := range raw {
			if value.Valid {
				record[i] = value.String
			}
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return w.Error()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func defaultOutputDir() string {
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "omcgo" {
		return filepath.Join("data", "model-library")
	}
	return filepath.Join("omcgo", "data", "model-library")
}

const activeLinksCTE = `
WITH active_links AS (
    SELECT
        csf.id AS sub_field_id,
        c.id AS command_id,
        c.command_code,
        COALESCE(c.operation_type, '') AS operation_type,
        COALESCE(c.source, '') AS command_source,
        COALESCE(g.group_code, '') AS group_code,
        COALESCE(g.chapter_code, '') AS chapter_code,
        COALESCE(g.group_name_zh, '') AS group_name_zh,
        sp.id AS standard_path_id,
        sp.standard_path,
        COALESCE(sp.description, '') AS description,
        COALESCE(sp.entry_type, '') AS entry_type,
        COALESCE(sp.access, '') AS access,
        csf.mml_code,
        csf.sort_order
    FROM mml_command_sub_fields csf
    JOIN mml_commands c ON c.id = csf.command_id
    LEFT JOIN mml_command_groups g ON g.id = c.group_id
    JOIN standard_params sp ON sp.id = csf.standard_path_id
    WHERE c.deprecated_at IS NULL
      AND csf.deprecated_at IS NULL
)`

const metricsSQL = activeLinksCTE + `
, path_command_duplicates AS (
    SELECT standard_path_id
    FROM active_links
    GROUP BY standard_path_id
    HAVING COUNT(DISTINCT command_id) > 1
), same_command_duplicates AS (
    SELECT command_id, standard_path_id
    FROM active_links
    GROUP BY command_id, standard_path_id
    HAVING COUNT(*) > 1
), same_operation_duplicates AS (
    SELECT standard_path_id, operation_type
    FROM active_links
    GROUP BY standard_path_id, operation_type
    HAVING COUNT(DISTINCT command_id) > 1
), duplicate_mml_codes AS (
    SELECT command_id, mml_code
    FROM active_links
    GROUP BY command_id, mml_code
    HAVING COUNT(*) > 1
), duplicate_target_paths AS (
    SELECT c.id
    FROM mml_commands c
    CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(c.target_paths, '[]'::jsonb)) p(path)
    WHERE c.deprecated_at IS NULL
    GROUP BY c.id, p.path
    HAVING COUNT(*) > 1
)
SELECT metric, value, note
FROM (
    SELECT 1 AS ord, 'active_sub_field_links' AS metric, COUNT(*)::bigint AS value,
           'active mml_command_sub_fields rows joined to active commands' AS note
    FROM active_links
    UNION ALL
    SELECT 2, 'active_commands_with_params', COUNT(DISTINCT command_id)::bigint,
           'active commands that have at least one parameter binding'
    FROM active_links
    UNION ALL
    SELECT 3, 'unique_standard_paths_in_mml', COUNT(DISTINCT standard_path_id)::bigint,
           'unique standard_params referenced by MML config'
    FROM active_links
    UNION ALL
    SELECT 4, 'same_command_same_path_duplicate_groups', COUNT(*)::bigint,
           'hard duplicate; should be 0 because uq_command_standard_path exists'
    FROM same_command_duplicates
    UNION ALL
    SELECT 5, 'same_command_same_mml_code_duplicate_groups', COUNT(*)::bigint,
           'hard duplicate; should be 0 because uq_command_mml_code exists'
    FROM duplicate_mml_codes
    UNION ALL
    SELECT 6, 'standard_paths_used_by_multiple_commands', COUNT(*)::bigint,
           'cross-command reuse; not an insertion duplicate by itself'
    FROM path_command_duplicates
    UNION ALL
    SELECT 7, 'same_operation_cross_command_duplicate_groups', COUNT(*)::bigint,
           'same path appears in multiple commands with the same operation_type; review list'
    FROM same_operation_duplicates
    UNION ALL
    SELECT 8, 'commands_with_duplicate_target_paths_json', COUNT(*)::bigint,
           'duplicate path inside mml_commands.target_paths JSON array'
    FROM duplicate_target_paths
) s
ORDER BY ord`

const crossCommandDuplicateSQL = activeLinksCTE + `
SELECT
    standard_path,
    COUNT(*)::text AS link_count,
    COUNT(DISTINCT command_id)::text AS command_count,
    STRING_AGG(DISTINCT operation_type, ',' ORDER BY operation_type) AS operation_types,
    STRING_AGG(DISTINCT chapter_code, ',' ORDER BY chapter_code) AS chapter_codes,
    STRING_AGG(DISTINCT command_code, '; ' ORDER BY command_code) AS command_codes,
    STRING_AGG(DISTINCT group_code, '; ' ORDER BY group_code) AS group_codes,
    MIN(description) AS description
FROM active_links
GROUP BY standard_path_id, standard_path
HAVING COUNT(DISTINCT command_id) > 1
ORDER BY COUNT(DISTINCT command_id) DESC, standard_path`

const sameOperationDuplicateSQL = activeLinksCTE + `
SELECT
    standard_path,
    operation_type,
    COUNT(*)::text AS link_count,
    COUNT(DISTINCT command_id)::text AS command_count,
    STRING_AGG(DISTINCT chapter_code, ',' ORDER BY chapter_code) AS chapter_codes,
    STRING_AGG(DISTINCT command_code, '; ' ORDER BY command_code) AS command_codes,
    STRING_AGG(DISTINCT group_code, '; ' ORDER BY group_code) AS group_codes,
    MIN(description) AS description
FROM active_links
GROUP BY standard_path_id, standard_path, operation_type
HAVING COUNT(DISTINCT command_id) > 1
ORDER BY COUNT(DISTINCT command_id) DESC, standard_path, operation_type`

const sameCommandDuplicateSQL = activeLinksCTE + `
SELECT
    command_code,
    operation_type,
    standard_path,
    COUNT(*)::text AS duplicate_count,
    STRING_AGG(sub_field_id::text, '; ' ORDER BY sub_field_id::text) AS sub_field_ids
FROM active_links
GROUP BY command_id, command_code, operation_type, standard_path_id, standard_path
HAVING COUNT(*) > 1
ORDER BY command_code, standard_path`

const duplicateMMLCodeSQL = activeLinksCTE + `
SELECT
    command_code,
    operation_type,
    mml_code,
    COUNT(*)::text AS duplicate_count,
    STRING_AGG(standard_path, '; ' ORDER BY standard_path) AS standard_paths
FROM active_links
GROUP BY command_id, command_code, operation_type, mml_code
HAVING COUNT(*) > 1
ORDER BY command_code, mml_code`

const duplicateTargetPathsSQL = `
SELECT
    c.command_code,
    COALESCE(c.operation_type, '') AS operation_type,
    p.path AS standard_path,
    COUNT(*)::text AS duplicate_count
FROM mml_commands c
CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(c.target_paths, '[]'::jsonb)) p(path)
WHERE c.deprecated_at IS NULL
GROUP BY c.id, c.command_code, c.operation_type, p.path
HAVING COUNT(*) > 1
ORDER BY c.command_code, p.path`
