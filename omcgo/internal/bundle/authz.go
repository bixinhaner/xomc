package bundle

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// SNVisibilityReader 把一批设备序列号按「调用者可见设备组」过滤，是 #63 设备组
// 可见性强制层在 bundle 批量下载链路的归属判定器。遵循 authz 包的三态契约：
//
//	visibleGroups == nil   → 超管：返回全部 SN（不过滤）。
//	len(visibleGroups)==0  → 无任何分组权限：返回空集（fail-closed）。
//	[g1, g2, ...]          → 仅返回归属这些分组的设备 SN。
//
// 返回的 map 以可见 SN 为键，便于 O(1) 命中判断。
type SNVisibilityReader interface {
	VisibleSerialNumbers(ctx context.Context, visibleGroups []uuid.UUID, sns []string) (map[string]struct{}, error)
}

// PgSNVisibilityReader 用 devices JOIN device_group_members 实现 SNVisibilityReader。
// bundle 的下载源都以 serial_number 为键（设备物理文件存档），没有 device_id UUID 列，
// 因此不能直接复用 authz.ApplyDeviceVisibilityFilter（那是按 device_id 列的子查询）；
// 这里按 SN 关联 devices 表把可见性翻译到 SN 维度。
type PgSNVisibilityReader struct {
	pool *pgxpool.Pool
}

// NewPgSNVisibilityReader 构造一个 PgSNVisibilityReader。
func NewPgSNVisibilityReader(pool *pgxpool.Pool) *PgSNVisibilityReader {
	return &PgSNVisibilityReader{pool: pool}
}

// VisibleSerialNumbers 见 SNVisibilityReader 契约。
func (r *PgSNVisibilityReader) VisibleSerialNumbers(ctx context.Context, visibleGroups []uuid.UUID, sns []string) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(sns))
	if len(sns) == 0 {
		return out, nil
	}
	// 超管（nil）：全部放行。
	if visibleGroups == nil {
		for _, sn := range sns {
			out[sn] = struct{}{}
		}
		return out, nil
	}
	// 无任何分组权限：fail-closed 空集。
	if len(visibleGroups) == 0 {
		return out, nil
	}

	query, args, err := storage.Psql.
		Select("DISTINCT d.serial_number").
		From("devices d").
		Join("device_group_members dgm ON dgm.device_id = d.id").
		Where(sq.Eq{"d.serial_number": sns}).
		Where(sq.Eq{"dgm.group_id": visibleGroups}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build visible serial numbers query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query visible serial numbers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sn string
		if scanErr := rows.Scan(&sn); scanErr != nil {
			return nil, fmt.Errorf("scan visible serial number: %w", scanErr)
		}
		out[sn] = struct{}{}
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate visible serial numbers: %w", rows.Err())
	}
	return out, nil
}

// filterVisibleSNs 返回 sns 中归属调用者可见设备组的子集（保持入参顺序、去重）。
// reader == nil（dev/test 退化）或 visibleGroups == nil（超管）→ 原样返回（不过滤）。
func filterVisibleSNs(ctx context.Context, reader SNVisibilityReader, visibleGroups []uuid.UUID, sns []string) ([]string, error) {
	if reader == nil || visibleGroups == nil {
		return sns, nil
	}
	visible, err := reader.VisibleSerialNumbers(ctx, visibleGroups, sns)
	if err != nil {
		return nil, err
	}
	kept := make([]string, 0, len(sns))
	for _, sn := range sns {
		if _, ok := visible[sn]; ok {
			kept = append(kept, sn)
		}
	}
	return kept, nil
}
