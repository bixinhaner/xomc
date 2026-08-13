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

// ErrSystemLicenseIDExists — 并发兜底 sentinel：Replace 内 INSERT 撞 system_license
// license_id UNIQUE 时返回。singleton 删除语义下理论不会触发（SELECT FOR UPDATE 序列化
// + 旧行先 DELETE），保留作防御。
var ErrSystemLicenseIDExists = errors.New("system license_id already exists")

// SystemLicenseRepository 是 system_license + history 的数据访问接口。
//
// 设计：接口小（3 个方法）；singleton 业务规则在 Replace 内部用事务保证。
type SystemLicenseRepository interface {
	// GetCurrent 返回当前生效的 license（is_current=true）。
	// 无 license 时返回 (nil, ErrSystemLicenseNotFound)，调用方据此显示空状态。
	GetCurrent(ctx context.Context) (*SystemLicense, error)

	// Replace 把当前 current 归档进 history，并把新 license INSERT 为唯一 current。
	//
	// 事务原子保证（singleton 删除语义）：
	//   1. DELETE system_license 全部旧行（old current + 残留 is_current=false 行）
	//   2. INSERT 新行 is_current=true
	//   3. 若有 old current：COPY 整行到 history（replaced_by_id=new.id, replaced_at=NOW()）
	//   - 任一步失败整批回滚
	//   - 调用方传入的 newLic.ID / IsCurrent / CreatedAt / UpdatedAt 由实现层设置
	//   - 允许重传任意 license_id（含历史用过的）：旧行先删，不撞 license_id UNIQUE
	Replace(ctx context.Context, newLic *SystemLicense) (replacedHistory *SystemLicenseHistory, err error)

	// ListHistory 分页列出 history 记录。
	// 按 replaced_at DESC 排序（最近替换在前）。
	ListHistory(ctx context.Context, filter SystemLicenseHistoryFilter) (*model.ListResponse[SystemLicenseHistory], error)
}
