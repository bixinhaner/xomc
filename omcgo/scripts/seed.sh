#!/bin/bash
# seed.sh - Load seed data into the database
# Usage: ./scripts/seed.sh [--dsn DSN]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SEED_DIR="$PROJECT_DIR/datamodels/seed"

DSN="${OMCGO_DB_DSN:-postgres://omcgo:omcgo@localhost:5432/omcgo?sslmode=disable}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dsn) DSN="$2"; shift 2;;
        *) echo "Unknown option: $1"; exit 1;;
    esac
done

echo "=== OMC Go Seed Data Loader ==="
echo "DSN: ${DSN%%@*}@***"

# Load carrier default data models
echo ""
echo "--- Loading carrier default data models ---"
for file in "$SEED_DIR"/carrier_defaults/*.json; do
    if [ ! -f "$file" ]; then
        echo "No seed files found in $SEED_DIR/carrier_defaults/"
        break
    fi

    filename=$(basename "$file")
    echo "Loading: $filename"

    carrier=$(jq -r '.carrier' "$file")
    technology=$(jq -r '.technology' "$file")
    version=$(jq -r '.version' "$file")
    scope=$(jq -r '.scope' "$file")
    root_object=$(jq -r '.root_object' "$file")
    source_field=$(jq -r '.source' "$file")
    description=$(jq -r '.description' "$file")
    parameter_tree=$(jq -c '.parameter_tree' "$file")

    psql "$DSN" -c "
        INSERT INTO data_model_definitions (
            carrier, technology, version, scope, root_object,
            parameter_tree, source, imported_by, description,
            status, is_active
        ) VALUES (
            '$carrier', '$technology', '$version', '$scope', '$root_object',
            '$parameter_tree'::jsonb, '$source_field', 'seed.sh', '$description',
            'active', true
        )
        ON CONFLICT DO NOTHING;
    " 2>/dev/null && echo "  OK: $filename" || echo "  SKIP: $filename (may already exist)"
done

echo ""
echo "=== Seed data loading complete ==="
