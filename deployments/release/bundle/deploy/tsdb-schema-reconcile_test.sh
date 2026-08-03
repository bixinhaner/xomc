#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL="$SCRIPT_DIR/install.sh"
SQL="$SCRIPT_DIR/tsdb-schema-reconcile.sql"

grep -Fq 'TSDB_RECONCILE_SQL="$DEPLOY_DIR/tsdb-schema-reconcile.sql"' "$INSTALL"
grep -Fq 'psql -v ON_ERROR_STOP=1' "$INSTALL"
grep -Fq 'ADD COLUMN IF NOT EXISTS measurement_start' "$SQL"
grep -Fq 'ADD COLUMN IF NOT EXISTS measurement_end' "$SQL"
grep -Fq 'CREATE TABLE IF NOT EXISTS public.pm_slot_health' "$SQL"
grep -Fq 'CREATE INDEX IF NOT EXISTS idx_pm_files_measurement_slot' "$SQL"
grep -Fq "WHERE status = 'bootstrap_ignored'" "$SQL"
grep -Fq 'AND coverage_ratio >= 0.98' "$SQL"

migrate_line="$(grep -n 'if \[ "$TSDB_MIGRATE_OK" = 1 \]' "$INSTALL" | head -1 | cut -d: -f1)"
reconcile_line="$(grep -n 'TSDB_RECONCILE_SQL=' "$INSTALL" | head -1 | cut -d: -f1)"
startup_line="$(grep -n 'Step 8. up 业务' "$INSTALL" | head -1 | cut -d: -f1)"
[ "$migrate_line" -lt "$reconcile_line" ]
[ "$reconcile_line" -lt "$startup_line" ]

echo "tsdb schema reconciliation contract: PASS"
