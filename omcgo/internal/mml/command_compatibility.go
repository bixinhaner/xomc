package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/product"
)

// ============================================================
// command_compatibility.go — Spec §R-8.5 命令兼容性警告
//
// 设计文档：docs/design/r-8-5-command-compatibility-design-20260521.md
//
// 给定 product_class，返回"该产品类型不支持的命令 ID 列表"。判定逻辑：
//   command 的 tree_node_refs（v2 catalog） 或 target_paths（v1 catalog）
//   是否全部位于该 product 对应 ParamModel 的 default param_mappings 集合。
//
// 解析路径（T-0177）：复用 ProductRegistry.MatchProductClass 全局正则路由，
// 与 PR-C/E 其他 resolver（group_tree / device_supported_paths / result_aggregator）
// 模式统一；不再依赖 devices.product_id / param_model_id denorm 列填充状态。
//
//   - 命中正则 → 取 Product.ID / Product.ParamModelID
//   - product.ErrOrphan → supportedPaths 留空（与旧 SQL "no param_model linkage"
//     分支语义等价；UI 据 paramModelID==nil 提示"未配置参数模型"而非"全部不兼容"）
//   - 其他错误 → wrap 后返回
//
// 三处独立单元：
//   1. CommandPathRepository — 从 mml_commands 取每条命令的 path 集合
//      （tree_node_refs 优先，回退 target_paths）
//   2. ComputeCommandCompatibility — 纯函数，无外部依赖，可单测
//   3. CompatibilityService — 编排：productMatcher → paramRegistry → 计算
//
// 与现有 ConsoleService 解耦（独立 struct + 独立 deps），避免破坏
// NewConsoleService 签名影响 5 个测试文件。
// ============================================================

// CommandCompatibilityResult 是 GetCommandCompatibility 的返回 / API 响应载荷。
type CommandCompatibilityResult struct {
	ProductClass          string      `json:"product_class"`
	ProductID             uuid.UUID   `json:"product_id"`
	ParamModelID          uuid.UUID   `json:"param_model_id,omitempty"` // 空 UUID 表示设备/产品未配 ParamModel
	UnsupportedCommandIDs []uuid.UUID `json:"unsupported_command_ids"`
}

// CommandPathRow 是一条命令的路径集合视图（内部 + 测试可见，故 export）。
type CommandPathRow struct {
	ID    uuid.UUID
	Paths []string
}

// CommandPathRepository — 取 mml_commands 全表的 path 集合（消费侧定义的接口）。
type CommandPathRepository interface {
	ListAllCommandPaths(ctx context.Context) ([]CommandPathRow, error)
}

// pgCommandPathRepository — PG 实现。
type pgCommandPathRepository struct {
	pool *pgxpool.Pool
}

// NewPgCommandPathRepository 构造 PG 实现。
func NewPgCommandPathRepository(pool *pgxpool.Pool) CommandPathRepository {
	return &pgCommandPathRepository{pool: pool}
}

// ListAllCommandPaths SQL：tree_node_refs 非空数组时优先用，否则 fallback target_paths。
//
// 设计选择：用 COALESCE + NULLIF 在 SQL 侧完成 fallback，Go 侧只反序列化一次 JSON。
// target_paths 是 PG TEXT[]，用 to_jsonb 转为 JSON 数组形态便于统一反序列化。
func (r *pgCommandPathRepository) ListAllCommandPaths(ctx context.Context) ([]CommandPathRow, error) {
	const sql = `
SELECT
    id,
    COALESCE(NULLIF(tree_node_refs, '[]'::jsonb), to_jsonb(target_paths), '[]'::jsonb) AS paths
FROM mml_commands`

	rows, err := r.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query command paths: %w", err)
	}
	defer rows.Close()

	var out []CommandPathRow
	for rows.Next() {
		var row CommandPathRow
		var pathsJSON []byte
		if err := rows.Scan(&row.ID, &pathsJSON); err != nil {
			return nil, fmt.Errorf("scan command paths: %w", err)
		}
		if len(pathsJSON) > 0 && string(pathsJSON) != "null" {
			// JSON 解析失败 → 视为空 paths（容错：单条脏数据不影响整体）
			_ = json.Unmarshal(pathsJSON, &row.Paths)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate command paths: %w", err)
	}
	return out, nil
}

// ComputeCommandCompatibility 纯函数：对每条 command 检查 paths ⊆ supportedPaths。
//
// 边界：
//   - cmd.Paths 为空 → 视为支持（不警告，对应 ADD 命令无 path / 老 catalog 等场景）
//   - supportedPaths 为空 → 所有有 path 的 cmd 全部 unsupported
//   - 任一 path 不在 → 整条 cmd 标 unsupported（破环式短路）
//
// 性能：1183 commands × ~10 paths × O(1) hash lookup ≈ 12K ops，毫秒级。
func ComputeCommandCompatibility(
	commands []CommandPathRow,
	supportedPaths map[string]struct{},
) []uuid.UUID {
	var unsupported []uuid.UUID
	for _, cmd := range commands {
		if len(cmd.Paths) == 0 {
			continue
		}
		ok := true
		for _, p := range cmd.Paths {
			if _, exists := supportedPaths[p]; !exists {
				ok = false
				break
			}
		}
		if !ok {
			unsupported = append(unsupported, cmd.ID)
		}
	}
	return unsupported
}

// productClassMatcher 是 ProductRegistry 的窄接口（只用 MatchProductClass）。
//
// 不直接耦合 *product.Registry，避免测试需要构造完整 Registry — 测试只需
// stub MatchProductClass 即可覆盖 happy / orphan / err 三条路径。
type productClassMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// paramMappingLookup 是 parammodel.Registry 的窄接口（只用 GetByParamModel）。
//
// 同 productClassMatcher，存在的目的是给单测一个稳定的契约，避免构造完整
// parammodel.Registry（依赖 PG / Redis / metrics 等基础设施）。
type paramMappingLookup interface {
	GetByParamModel(ctx context.Context, paramModelID uuid.UUID) (*parammodel.MappingSet, error)
}

// CompatibilityService 编排 productMatcher → ParamRegistry → CommandPathRepository →
// ComputeCommandCompatibility。
//
// 独立于 ConsoleService 以最小化对现有签名 / 测试的影响（spec §R-8.5 可选 P2）。
//
// T-0177：用 ProductRegistry.MatchProductClass 取代旧 raw SQL 反查 devices —
// 不再依赖 devices.product_id / param_model_id denorm 列是否填充，享受 PR-B
// (T-0173) productClass L1+L2 缓存。孤儿 productClass（ErrOrphan）→ supportedPaths
// 留空，与旧 "no param_model linkage" 分支语义等价。
type CompatibilityService struct {
	paramRegistry  paramMappingLookup
	productMatcher productClassMatcher
	cmdPathRepo    CommandPathRepository
	logger         *zap.Logger
}

// NewCompatibilityService 构造。所有 deps 非 nil 必需。
//
// productMatcher 一般传 *product.Registry（实现了 MatchProductClass），
// paramRegistry 一般传 *parammodel.Registry（实现了 GetByParamModel）。
// 用窄接口承接以便测试 stub。
func NewCompatibilityService(
	productMatcher productClassMatcher,
	paramRegistry paramMappingLookup,
	cmdPathRepo CommandPathRepository,
	logger *zap.Logger,
) *CompatibilityService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CompatibilityService{
		paramRegistry:  paramRegistry,
		productMatcher: productMatcher,
		cmdPathRepo:    cmdPathRepo,
		logger:         logger.Named("mml-compatibility"),
	}
}

// GetCommandCompatibility 计算指定 product_class 下的命令兼容性。
//
// 错误约定：
//   - productClass 是孤儿（product.ErrOrphan）→ 返回 200 + 空 supportedPaths
//     → 所有有 path 的命令 unsupported；UI 据 paramModelID 是否零值区分
//     "未配置参数模型" vs "全部不兼容"
//   - 设备/产品未关联 ParamModel → 同上，supportedPaths=空集
//   - 其他底层错误 → wrap 后返回
func (s *CompatibilityService) GetCommandCompatibility(
	ctx context.Context,
	productClass string,
) (*CommandCompatibilityResult, error) {
	if s.productMatcher == nil || s.paramRegistry == nil || s.cmdPathRepo == nil {
		return nil, fmt.Errorf("compatibility service not properly initialized")
	}

	// 1. 走 ProductRegistry 正则路由（T-0177）
	var (
		productID    *uuid.UUID
		paramModelID *uuid.UUID
	)
	mr, err := s.productMatcher.MatchProductClass(ctx, productClass)
	switch {
	case errors.Is(err, product.ErrOrphan):
		// 孤儿：productClass 不在 patterns 里（可能在 seed/000154 字典）
		// 与旧 SQL "no param_model linkage" 分支语义等价 — supportedPaths
		// 留空 → 所有有 path 的命令标 unsupported。UI 据 paramModelID 为
		// 零值提示"未配置参数模型"。
		s.logger.Debug("product_class orphan; treat as no-paramModel-linkage",
			zap.String("product_class", productClass))
	case err != nil:
		return nil, fmt.Errorf("match product_class %q: %w", productClass, err)
	default:
		if mr != nil && mr.Product != nil {
			pid := mr.Product.ID
			productID = &pid
			paramModelID = mr.Product.ParamModelID
		}
	}

	// 2. 取 default param_mappings（按 spec §3.3 — 不查 discovered）
	supportedPaths := make(map[string]struct{})
	if paramModelID != nil {
		ms, err := s.paramRegistry.GetByParamModel(ctx, *paramModelID)
		if err != nil {
			// 不视为致命 — 缺映射相当于"全部 unsupported"，由 spec §R-8.5 不阻塞语义兜底
			s.logger.Warn("load param_model mappings",
				zap.String("param_model_id", paramModelID.String()),
				zap.Error(err))
		} else if ms != nil {
			for _, m := range ms.Mappings {
				supportedPaths[m.StandardPath] = struct{}{}
			}
		}
	} else {
		s.logger.Debug("product_class resolved but no param_model linkage",
			zap.String("product_class", productClass))
	}

	// 3. list all commands + their paths
	commands, err := s.cmdPathRepo.ListAllCommandPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("list command paths: %w", err)
	}

	// 4. compute
	unsupported := ComputeCommandCompatibility(commands, supportedPaths)
	if unsupported == nil {
		unsupported = []uuid.UUID{} // 序列化为 [] 而非 null
	}

	result := &CommandCompatibilityResult{
		ProductClass:          productClass,
		UnsupportedCommandIDs: unsupported,
	}
	if productID != nil {
		result.ProductID = *productID
	}
	if paramModelID != nil {
		result.ParamModelID = *paramModelID
	}
	return result, nil
}
