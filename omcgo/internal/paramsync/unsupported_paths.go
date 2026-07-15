package paramsync

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGReadUnsupportedPathRepository is the read side of GPV capability learning.
// Records are exact for a product and firmware version; a fault learned on one
// firmware must never suppress probing after an upgrade or rollback.
type PGReadUnsupportedPathRepository struct{ pool *pgxpool.Pool }

func NewPGReadUnsupportedPathRepository(pool *pgxpool.Pool) *PGReadUnsupportedPathRepository {
	return &PGReadUnsupportedPathRepository{pool: pool}
}

func (r *PGReadUnsupportedPathRepository) ListReadUnsupportedPaths(
	ctx context.Context, productID uuid.UUID, firmwareVersion string,
) ([]string, error) {
	firmwareVersion = strings.TrimSpace(firmwareVersion)
	if r == nil || r.pool == nil || productID == uuid.Nil || firmwareVersion == "" {
		return nil, nil
	}
	query, args, err := sq.Select("standard_path").From("product_unsupported_paths").
		Where(sq.Eq{
			"product_id": productID, "firmware_version": firmwareVersion, "read_unsupported": true,
		}).OrderBy("standard_path").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list firmware read-unsupported paths: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list firmware read-unsupported paths: %w", err)
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan firmware read-unsupported path: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate firmware read-unsupported paths: %w", err)
	}
	return paths, nil
}
