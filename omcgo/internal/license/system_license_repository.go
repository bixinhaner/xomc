// system_license_repository.go — F06 System License 重构 P1 Step 1 新增。
//
// 接口契约。pg 实现见 pg_system_license_repository.go。
package license

import (
	"context"
	"errors"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ErrSystemLicenseNotFound — 当前没有任何 is_current=true 的 system_license 行。
//
// 业务约定：fresh install / 上传第一张前可能没有 license；调用方应处理此 sentinel
// （前端显示空状态 + Update 按钮可用）。
var ErrSystemLicenseNotFound = errors.New("system license not found")

// ErrSystemLicenseIDExists — 上传的 license_id 已存在（违反 UNIQUE）。
// 同 license_id 不允许重复 Import，必须先 Rollback 或换 ID。
var ErrSystemLicenseIDExists = errors.New("system license_id already exists")

// SystemLicenseRepository 是 system_license + history 的数据访问接口。
//
// 设计：接口小（4 个方法）；singleton 业务规则在 Replace 内部用事务保证。
type SystemLicenseRepository interface {
	// GetCurrent 返回当前生效的 license（is_current=true）。
	// 无 license 时返回 (nil, ErrSystemLicenseNotFound)，调用方据此显示空状态。
	GetCurrent(ctx context.Context) (*SystemLicense, error)

	// Replace 把当前 current 移到 history，并把新 license INSERT 为新 current。
	//
	// 事务原子保证：
	//   1. 若有 current 行：UPDATE is_current=false, COPY 到 history（replaced_by_id=new.id, replaced_at=NOW()）
	//   2. INSERT 新行 is_current=true
	//   - 任一步失败整批回滚
	//   - 调用方传入的 newLic.ID / IsCurrent / CreatedAt / UpdatedAt 由实现层设置
	//   - 若 newLic.LicenseID 与 license_id UNIQUE 撞 → 返 ErrSystemLicenseIDExists
	Replace(ctx context.Context, newLic *SystemLicense) (replacedHistory *SystemLicenseHistory, err error)

	// ListHistory 分页列出 history 记录。
	// 按 replaced_at DESC 排序（最近替换在前）。
	ListHistory(ctx context.Context, filter SystemLicenseHistoryFilter) (*model.ListResponse[SystemLicenseHistory], error)

	// ExistsByLicenseID 判断 license_id 是否在 current 表或 history 表中存在。
	// 用于 Update 前置检查（防止同 license_id 重复使用）。
	ExistsByLicenseID(ctx context.Context, licenseID string) (bool, error)
}
