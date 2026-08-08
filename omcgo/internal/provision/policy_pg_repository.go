package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var policyColumns = []string{"id", "name", "enabled", "product_class", "product_classes", "execute_type",
	"priority", "upgrade_enabled", "target_version", "license_enabled", "self_config_enabled",
	"config", "created_at", "updated_at"}

type PgPlugAndPlayRepository struct{ pool *pgxpool.Pool }

func NewPgPlugAndPlayRepository(pool *pgxpool.Pool) *PgPlugAndPlayRepository {
	return &PgPlugAndPlayRepository{pool: pool}
}

func (r *PgPlugAndPlayRepository) CreatePolicy(ctx context.Context, p *PlugAndPlayPolicy) error {
	normalizePolicyProductClasses(p)
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Priority == 0 {
		p.Priority = 100
	}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	if len(p.Config) == 0 {
		p.Config = []byte(`{}`)
	}
	q, args, err := storage.Psql.Insert("plug_and_play_policies").Columns(policyColumns...).Values(
		p.ID, p.Name, p.Enabled, p.ProductClass, p.ProductClasses, p.ExecuteType, p.Priority,
		p.UpgradeEnabled, nullableString(p.TargetVersion), p.LicenseEnabled,
		p.SelfConfigEnabled, p.Config, p.CreatedAt, p.UpdatedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build create plug and play policy SQL: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create plug and play policy: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = ensureEnabledPolicyProductsAvailable(ctx, tx, p); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("create plug and play policy: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create plug and play policy: %w", err)
	}
	return nil
}

func (r *PgPlugAndPlayRepository) GetPolicy(ctx context.Context, id uuid.UUID) (*PlugAndPlayPolicy, error) {
	q, args, err := storage.Psql.Select(policyColumns...).From("plug_and_play_policies").
		Where(sq.Eq{"id": id, "deleted_at": nil}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get plug and play policy SQL: %w", err)
	}
	return scanPolicy(r.pool.QueryRow(ctx, q, args...))
}

func (r *PgPlugAndPlayRepository) ListPolicies(ctx context.Context, f PolicyFilter) ([]PlugAndPlayPolicy, int64, error) {
	pred := sq.And{sq.Eq{"deleted_at": nil}}
	if f.ProductClass != "" || f.ProductName != "" {
		productPredicates := sq.Or{}
		if f.ProductClass != "" {
			productPredicates = append(productPredicates,
				sq.Eq{"product_class": f.ProductClass},
				sq.Expr("? = ANY(product_classes)", f.ProductClass),
			)
		}
		if f.ProductName != "" {
			productPredicates = append(productPredicates,
				sq.Eq{"product_class": f.ProductName},
				sq.Expr("? = ANY(product_classes)", f.ProductName),
				sq.Expr(`EXISTS (
					SELECT 1 FROM product_class_patterns pcp
					JOIN products prod ON prod.id = pcp.product_id
					WHERE prod.product_name = ?
					  AND (pcp.product_class = plug_and_play_policies.product_class
					       OR pcp.product_class = ANY(plug_and_play_policies.product_classes))
				)`, f.ProductName),
			)
		}
		pred = append(pred, productPredicates)
	}
	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		pred = append(pred, sq.Or{
			sq.ILike{"name": pattern},
			sq.ILike{"product_class": pattern},
			sq.Expr("array_to_string(product_classes, ',') ILIKE ?", pattern),
			sq.Expr(`EXISTS (
				SELECT 1 FROM product_class_patterns pcp
				JOIN products prod ON prod.id = pcp.product_id
				WHERE prod.product_name ILIKE ?
				  AND (pcp.product_class = plug_and_play_policies.product_class
				       OR pcp.product_class = ANY(plug_and_play_policies.product_classes))
			)`, pattern),
		})
	}
	count := storage.Psql.Select("COUNT(*)").From("plug_and_play_policies")
	if len(pred) > 0 {
		count = count.Where(pred)
	}
	q, args, err := count.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count plug and play policies SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count plug and play policies: %w", err)
	}
	size, page := f.PageSize, f.Page
	if size < 1 || size > 100 {
		size = 20
	}
	if page < 1 {
		page = 1
	}
	b := storage.Psql.Select(policyColumns...).From("plug_and_play_policies").
		OrderBy("priority ASC", "created_at DESC").Limit(uint64(size)).Offset(uint64((page - 1) * size))
	if len(pred) > 0 {
		b = b.Where(pred)
	}
	q, args, err = b.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list plug and play policies SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list plug and play policies: %w", err)
	}
	defer rows.Close()
	items := make([]PlugAndPlayPolicy, 0)
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
	}
	return items, total, rows.Err()
}

func (r *PgPlugAndPlayRepository) UpdatePolicy(ctx context.Context, p *PlugAndPlayPolicy) error {
	normalizePolicyProductClasses(p)
	p.UpdatedAt = time.Now()
	q, args, err := storage.Psql.Update("plug_and_play_policies").
		Set("name", p.Name).Set("enabled", p.Enabled).Set("product_class", p.ProductClass).
		Set("product_classes", p.ProductClasses).
		Set("execute_type", p.ExecuteType).Set("priority", p.Priority).
		Set("upgrade_enabled", p.UpgradeEnabled).Set("target_version", nullableString(p.TargetVersion)).
		Set("license_enabled", p.LicenseEnabled).Set("self_config_enabled", p.SelfConfigEnabled).
		Set("config", p.Config).Set("updated_at", p.UpdatedAt).Where(sq.Eq{"id": p.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update plug and play policy SQL: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update plug and play policy: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = ensureEnabledPolicyProductsAvailable(ctx, tx, p); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update plug and play policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update plug and play policy: %w", err)
	}
	return nil
}

const enabledPolicyProductLock = "plug_and_play_policies:enabled_product"

func ensureEnabledPolicyProductsAvailable(ctx context.Context, tx pgx.Tx, policy *PlugAndPlayPolicy) error {
	if policy == nil || !policy.Enabled {
		return nil
	}
	productKeys := make([]string, 0, len(policy.ProductClasses)+1)
	seen := make(map[string]struct{}, len(policy.ProductClasses)+1)
	for _, productName := range append([]string{policy.ProductClass}, policy.ProductClasses...) {
		key := strings.ToLower(strings.TrimSpace(productName))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		productKeys = append(productKeys, key)
	}
	if len(productKeys) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", enabledPolicyProductLock); err != nil {
		return fmt.Errorf("lock enabled plug and play policies: %w", err)
	}
	var existingName string
	err := tx.QueryRow(ctx, `
		SELECT name
		FROM plug_and_play_policies
		WHERE enabled = true
		  AND deleted_at IS NULL
		  AND id <> $1
		  AND (
		    lower(product_class) = ANY($2::text[])
		    OR EXISTS (
		      SELECT 1 FROM unnest(product_classes) AS configured_product
		      WHERE lower(configured_product) = ANY($2::text[])
		    )
		  )
		LIMIT 1`, policy.ID, productKeys).Scan(&existingName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check enabled plug and play policy products: %w", err)
	}
	return fmt.Errorf("%w: %s", ErrEnabledPolicyProductConflict, existingName)
}

func (r *PgPlugAndPlayRepository) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	q, args, err := storage.Psql.Update("plug_and_play_policies").
		Set("enabled", false).Set("deleted_at", time.Now()).Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id, "deleted_at": nil}).ToSql()
	if err != nil {
		return fmt.Errorf("build soft delete plug and play policy SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("soft delete plug and play policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgPlugAndPlayRepository) CreateXML(ctx context.Context, f *ProvisioningXMLFile) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if f.DownloadToken == uuid.Nil {
		f.DownloadToken = uuid.New()
	}
	f.CreatedAt = time.Now()
	q, args, err := storage.Psql.Insert("provisioning_xml_files").
		Columns("id", "policy_id", "device_id", "file_name", "content", "checksum", "download_token", "created_at").
		Values(f.ID, f.PolicyID, f.DeviceID, f.FileName, f.Content, f.Checksum, f.DownloadToken, f.CreatedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build create provisioning XML SQL: %w", err)
	}
	if _, err = r.pool.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("create provisioning XML: %w", err)
	}
	return nil
}

func (r *PgPlugAndPlayRepository) GetXML(ctx context.Context, id uuid.UUID) (*ProvisioningXMLFile, error) {
	q, args, err := storage.Psql.Select("id", "policy_id", "device_id", "file_name", "content", "checksum", "download_token", "created_at").
		From("provisioning_xml_files").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get provisioning XML SQL: %w", err)
	}
	var f ProvisioningXMLFile
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&f.ID, &f.PolicyID, &f.DeviceID, &f.FileName, &f.Content, &f.Checksum, &f.DownloadToken, &f.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get provisioning XML: %w", err)
	}
	return &f, nil
}

func (r *PgPlugAndPlayRepository) RotateXMLDownloadToken(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	token := uuid.New()
	q, args, err := storage.Psql.Update("provisioning_xml_files").
		Set("download_token", token).
		Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build rotate provisioning XML download token SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("rotate provisioning XML download token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return uuid.Nil, commonerrors.ErrNotFound
	}
	return token, nil
}

type policyScanner interface{ Scan(...any) error }

func scanPolicy(row policyScanner) (*PlugAndPlayPolicy, error) {
	var p PlugAndPlayPolicy
	var target sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &p.Enabled, &p.ProductClass, &p.ProductClasses, &p.ExecuteType,
		&p.Priority, &p.UpgradeEnabled, &target, &p.LicenseEnabled, &p.SelfConfigEnabled,
		&p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan plug and play policy: %w", err)
	}
	if target.Valid {
		p.TargetVersion = target.String
	}
	normalizePolicyProductClasses(&p)
	return &p, nil
}

var _ PlugAndPlayPolicyRepository = (*PgPlugAndPlayRepository)(nil)
var _ ProvisioningXMLRepository = (*PgPlugAndPlayRepository)(nil)
