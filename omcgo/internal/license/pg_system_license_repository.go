// pg_system_license_repository.go — F06 System License 重构 P1 Step 1 新增。
//
// PostgreSQL 实现，依赖 system_license + system_license_history 双表
// （migration 000121）。Replace 方法的事务保证 singleton 不变量。
package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// systemLicenseColumns — 与 system_license 表 schema 一一对齐（migration 000121）。
//
// 列顺序固定，用于 SELECT / Scan 双向序列化；新增列时同步 scanSystemLicense。
var systemLicenseColumns = []string{
	"id", "license_id", "license_type", "issuer", "licensee",
	"issued_at", "expiry_date", "devices_support", "feature_list",
	"raw_content", "signature", "signature_key_id", "signature_status",
	"uploaded_at", "uploaded_by_user_id", "is_current",
	"created_at", "updated_at",
}

var systemLicenseHistoryColumns = []string{
	"id", "license_id", "license_type", "issuer", "licensee",
	"issued_at", "expiry_date", "devices_support", "feature_list",
	"raw_content", "signature", "signature_key_id", "signature_status",
	"uploaded_at", "uploaded_by_user_id",
	"replaced_at", "replaced_by_id", "created_at",
}

// PgSystemLicenseRepository 是 SystemLicenseRepository 的 PostgreSQL 实现。
type PgSystemLicenseRepository struct {
	pool *pgxpool.Pool
}

// 编译期接口契约检查。
var _ SystemLicenseRepository = (*PgSystemLicenseRepository)(nil)

// NewPgSystemLicenseRepository 构造仓库。
func NewPgSystemLicenseRepository(pool *pgxpool.Pool) *PgSystemLicenseRepository {
	return &PgSystemLicenseRepository{pool: pool}
}

// GetCurrent 返回当前生效（is_current=true）的 system_license。
// 无 license 时返 (nil, ErrSystemLicenseNotFound)。
func (r *PgSystemLicenseRepository) GetCurrent(ctx context.Context) (*SystemLicense, error) {
	query, args, err := storage.Psql.Select(systemLicenseColumns...).
		From("system_license").
		Where(sq.Eq{"is_current": true}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get_current SQL: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	lic, err := scanSystemLicense(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSystemLicenseNotFound
		}
		return nil, fmt.Errorf("scan current system_license: %w", err)
	}
	return lic, nil
}

// Replace 在事务里完成 singleton 切换：
//  1. 锁 current（若有），UPDATE is_current=false
//  2. INSERT newLic（is_current=true，ID 由实现层 gen_random_uuid 兜底）
//  3. 把旧 current 整行 COPY 到 system_license_history（replaced_at=NOW, replaced_by_id=newLic.ID）
//
// 注意 Step 3 必须在 Step 2 之后：history.replaced_by_id 经 fk_replaced_by
// 引用 system_license.id，新 license 行必须先存在。
//
// 返回的 replacedHistory 是已被替换的 history 行（如有），调用方可用于审计 / 通知。
//
// 错误：
//   - newLic.LicenseID 与 system_license.license_id 撞 UNIQUE → ErrSystemLicenseIDExists
//   - 任何 SQL 错误整批回滚
func (r *PgSystemLicenseRepository) Replace(ctx context.Context, newLic *SystemLicense) (*SystemLicenseHistory, error) {
	if newLic == nil {
		return nil, errors.New("newLic is nil")
	}
	if newLic.ID == uuid.Nil {
		newLic.ID = uuid.New()
	}
	newLic.IsCurrent = true

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Step 1: 锁 + 读取 current（若有），同步 UPDATE is_current=false
	//   SELECT ... FOR UPDATE 防并发 Replace 撞 singleton 约束。
	curQuery, curArgs, qerr := storage.Psql.Select(systemLicenseColumns...).
		From("system_license").
		Where(sq.Eq{"is_current": true}).
		Suffix("FOR UPDATE").
		Limit(1).
		ToSql()
	if qerr != nil {
		return nil, fmt.Errorf("build lock_current SQL: %w", qerr)
	}
	curRow := tx.QueryRow(ctx, curQuery, curArgs...)
	current, scanErr := scanSystemLicense(curRow)
	hasCurrent := scanErr == nil
	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("scan current for replace: %w", scanErr)
	}

	var historyRow *SystemLicenseHistory
	if hasCurrent {
		// Step 2a: UPDATE is_current=false（让 partial unique index 释放）
		if _, err := tx.Exec(ctx,
			`UPDATE system_license SET is_current = false, updated_at = NOW() WHERE id = $1`,
			current.ID,
		); err != nil {
			return nil, fmt.Errorf("clear is_current: %w", err)
		}
	}

	// Step 3: INSERT newLic（is_current=true）
	//   必须在 insertHistory 之前：history.replaced_by_id 经 fk_replaced_by
	//   引用 system_license.id，新行需先存在，否则 FK 校验失败。
	if err := insertSystemLicense(ctx, tx, newLic); err != nil {
		// 命中 license_id UNIQUE → ErrSystemLicenseIDExists
		if isUniqueViolation(err, "system_license_license_id_key") {
			return nil, ErrSystemLicenseIDExists
		}
		return nil, fmt.Errorf("insert new system_license: %w", err)
	}

	// Step 2b: COPY current 到 history（replaced_by_id 指向已插入的 newLic.ID）
	if hasCurrent {
		historyRow = &SystemLicenseHistory{
			ID:               uuid.New(),
			LicenseID:        current.LicenseID,
			LicenseType:      current.LicenseType,
			Issuer:           current.Issuer,
			Licensee:         current.Licensee,
			IssuedAt:         current.IssuedAt,
			ExpiryDate:       current.ExpiryDate,
			DevicesSupport:   current.DevicesSupport,
			FeatureList:      current.FeatureList,
			RawContent:       current.RawContent,
			Signature:        current.Signature,
			SignatureKeyID:   current.SignatureKeyID,
			SignatureStatus:  current.SignatureStatus,
			UploadedAt:       current.UploadedAt,
			UploadedByUserID: current.UploadedByUserID,
			ReplacedByID:     &newLic.ID,
		}
		if err := insertHistory(ctx, tx, historyRow); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace tx: %w", err)
	}
	return historyRow, nil
}

// ListHistory 分页列出 history。按 replaced_at DESC 排序。
func (r *PgSystemLicenseRepository) ListHistory(
	ctx context.Context, filter SystemLicenseHistoryFilter,
) (*model.ListResponse[SystemLicenseHistory], error) {
	// Limit() 会把 PageSize 归一到 [1, 1000]，Offset() 内部会顺手把 Page 归一到 ≥1；
	// 这两个方法是 ListRequest 上的指针接收者，filter 是值类型不能直接调，先取地址。
	limit := filter.ListRequest.Limit()
	offset := filter.ListRequest.Offset()
	page := filter.ListRequest.Page // Offset 已确保 ≥1

	base := storage.Psql.Select(systemLicenseHistoryColumns...).From("system_license_history")
	countBase := storage.Psql.Select("COUNT(*)").From("system_license_history")

	if filter.LicenseID != nil {
		base = base.Where(sq.Eq{"license_id": *filter.LicenseID})
		countBase = countBase.Where(sq.Eq{"license_id": *filter.LicenseID})
	}

	// 总数
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("query count: %w", err)
	}

	// 数据
	dataSQL, dataArgs, err := base.
		OrderBy("replaced_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	items := make([]SystemLicenseHistory, 0, limit)
	for rows.Next() {
		h, scanErr := scanSystemLicenseHistory(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan history row: %w", scanErr)
		}
		items = append(items, *h)
	}
	if rerr := rows.Err(); rerr != nil {
		return nil, fmt.Errorf("iterate history rows: %w", rerr)
	}

	return model.NewListResponse(items, total, page, limit), nil
}

// ExistsByLicenseID 判断 license_id 是否在 current 或 history 中出现过。
//
// 用于 Update 前置检查：同 license_id 不允许重复使用（防回滚后再上传旧文件造成混淆）。
func (r *PgSystemLicenseRepository) ExistsByLicenseID(ctx context.Context, licenseID string) (bool, error) {
	const q = `
SELECT EXISTS (
    SELECT 1 FROM system_license          WHERE license_id = $1
    UNION ALL
    SELECT 1 FROM system_license_history  WHERE license_id = $1
) AS exists`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, licenseID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check license_id exists: %w", err)
	}
	return exists, nil
}

// ---- internal helpers ----

// insertSystemLicense 通过现有 tx 插入一条 system_license。
func insertSystemLicense(ctx context.Context, tx pgx.Tx, lic *SystemLicense) error {
	devicesJSON, err := json.Marshal(lic.DevicesSupport)
	if err != nil {
		return fmt.Errorf("marshal devices_support: %w", err)
	}
	featuresJSON := []byte(lic.FeatureList)
	if len(featuresJSON) == 0 {
		featuresJSON = []byte("{}")
	}
	const q = `
INSERT INTO system_license
    (id, license_id, license_type, issuer, licensee,
     issued_at, expiry_date, devices_support, feature_list,
     raw_content, signature, signature_key_id, signature_status,
     uploaded_at, uploaded_by_user_id, is_current)
VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8::jsonb, $9::jsonb,
     $10, $11, $12, $13,
     COALESCE($14, NOW()), $15, $16)`
	_, err = tx.Exec(ctx, q,
		lic.ID, lic.LicenseID, lic.LicenseType, lic.Issuer, lic.Licensee,
		lic.IssuedAt, lic.ExpiryDate, devicesJSON, featuresJSON,
		lic.RawContent, lic.Signature, lic.SignatureKeyID, string(lic.SignatureStatus),
		nullableTime(lic.UploadedAt), lic.UploadedByUserID, lic.IsCurrent,
	)
	return err
}

// insertHistory 通过现有 tx 插入一条 system_license_history。
func insertHistory(ctx context.Context, tx pgx.Tx, h *SystemLicenseHistory) error {
	devicesJSON, err := json.Marshal(h.DevicesSupport)
	if err != nil {
		return fmt.Errorf("marshal devices_support for history: %w", err)
	}
	featuresJSON := []byte(h.FeatureList)
	if len(featuresJSON) == 0 {
		featuresJSON = []byte("{}")
	}
	const q = `
INSERT INTO system_license_history
    (id, license_id, license_type, issuer, licensee,
     issued_at, expiry_date, devices_support, feature_list,
     raw_content, signature, signature_key_id, signature_status,
     uploaded_at, uploaded_by_user_id,
     replaced_at, replaced_by_id)
VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8::jsonb, $9::jsonb,
     $10, $11, $12, $13,
     $14, $15,
     NOW(), $16)`
	_, err = tx.Exec(ctx, q,
		h.ID, h.LicenseID, h.LicenseType, h.Issuer, h.Licensee,
		h.IssuedAt, h.ExpiryDate, devicesJSON, featuresJSON,
		h.RawContent, h.Signature, h.SignatureKeyID, string(h.SignatureStatus),
		h.UploadedAt, h.UploadedByUserID,
		h.ReplacedByID,
	)
	return err
}

// scanSystemLicense 把 pgx Row 解到 SystemLicense。
// 列顺序必须与 systemLicenseColumns 完全一致。
func scanSystemLicense(row pgx.Row) (*SystemLicense, error) {
	var lic SystemLicense
	var devicesRaw, featuresRaw []byte
	var sigStatus string
	if err := row.Scan(
		&lic.ID, &lic.LicenseID, &lic.LicenseType, &lic.Issuer, &lic.Licensee,
		&lic.IssuedAt, &lic.ExpiryDate, &devicesRaw, &featuresRaw,
		&lic.RawContent, &lic.Signature, &lic.SignatureKeyID, &sigStatus,
		&lic.UploadedAt, &lic.UploadedByUserID, &lic.IsCurrent,
		&lic.CreatedAt, &lic.UpdatedAt,
	); err != nil {
		return nil, err
	}
	lic.SignatureStatus = SignatureStatus(sigStatus)
	if len(devicesRaw) > 0 {
		if err := json.Unmarshal(devicesRaw, &lic.DevicesSupport); err != nil {
			return nil, fmt.Errorf("unmarshal devices_support: %w", err)
		}
	} else {
		lic.DevicesSupport = DevicesSupport{}
	}
	lic.FeatureList = FeatureList(featuresRaw)
	return &lic, nil
}

// scanSystemLicenseHistory 把 pgx Row 解到 SystemLicenseHistory。
func scanSystemLicenseHistory(row pgx.Row) (*SystemLicenseHistory, error) {
	var h SystemLicenseHistory
	var devicesRaw, featuresRaw []byte
	var sigStatus string
	if err := row.Scan(
		&h.ID, &h.LicenseID, &h.LicenseType, &h.Issuer, &h.Licensee,
		&h.IssuedAt, &h.ExpiryDate, &devicesRaw, &featuresRaw,
		&h.RawContent, &h.Signature, &h.SignatureKeyID, &sigStatus,
		&h.UploadedAt, &h.UploadedByUserID,
		&h.ReplacedAt, &h.ReplacedByID, &h.CreatedAt,
	); err != nil {
		return nil, err
	}
	h.SignatureStatus = SignatureStatus(sigStatus)
	if len(devicesRaw) > 0 {
		if err := json.Unmarshal(devicesRaw, &h.DevicesSupport); err != nil {
			return nil, fmt.Errorf("unmarshal devices_support history: %w", err)
		}
	} else {
		h.DevicesSupport = DevicesSupport{}
	}
	h.FeatureList = FeatureList(featuresRaw)
	return &h, nil
}

// isUniqueViolation 判断 pg error 是否是指定 unique constraint 违反。
//
// 用于把"license_id 撞 UNIQUE"翻译成业务 sentinel ErrSystemLicenseIDExists。
// 实际 constraint 名取决于 DDL 中的列名 + _key 后缀（PostgreSQL 默认命名规则）。
func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" { // unique_violation
		return false
	}
	return pgErr.ConstraintName == constraintName
}

// nullableTime — UploadedAt 零值时让 PG 走 DEFAULT NOW()；非零透传。
func nullableTime(t any) any {
	if v, ok := t.(interface{ IsZero() bool }); ok && v.IsZero() {
		return nil
	}
	return t
}
