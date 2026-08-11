package admin

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	legacyStorageProtectionMenuID         = "aaaa0008-1000-0000-0000-000000000010"
	legacyStorageProtectionRoutePath      = "/system/storage-protection"
	legacyStorageProtectionComponentPath  = "system/StorageProtection"
	retiredStorageProtectionRoutePath     = "/system/config?tab=retention_bp"
	retiredStorageProtectionComponentPath = "system/SystemConfig"
)

func EnsureLegacyStorageProtectionMenuRetired(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("ensure legacy storage protection menu retired: postgres pool is nil")
	}
	query, args, err := buildLegacyStorageProtectionMenuRetireSQL()
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ensure legacy storage protection menu retired: %w", err)
	}
	return nil
}

func buildLegacyStorageProtectionMenuRetireSQL() (string, []interface{}, error) {
	menuID, err := uuid.Parse(legacyStorageProtectionMenuID)
	if err != nil {
		return "", nil, fmt.Errorf("parse legacy storage protection menu id: %w", err)
	}
	query, args, err := storage.Psql.Update("menus").
		Set("route_path", retiredStorageProtectionRoutePath).
		Set("component_path", retiredStorageProtectionComponentPath).
		Set("show_status", string(MenuHide)).
		Set("status", string(MenuStatusDisabled)).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Or{
			sq.Eq{"id": menuID},
			sq.Eq{"route_path": legacyStorageProtectionRoutePath},
			sq.Eq{"component_path": legacyStorageProtectionComponentPath},
		}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build legacy storage protection menu retire SQL: %w", err)
	}
	return query, args, nil
}
