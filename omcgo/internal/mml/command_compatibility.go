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
// 三处独立单元：
//   1. CommandPathRepository — 从 mml_commands 取每条命令的 path 集合
//      （tree_node_refs 优先，回退 target_paths）
//   2. ComputeCommandCompatibility — 纯函数，无外部依赖，可单测
//   3. CompatibilityService — 编排：ProductRegistry → ParamRegistry → 计算
//
// 与现有 ConsoleService 解耦（独立 struct + 独立 deps），避免破坏
// NewConsoleService 签名影响 5 个测试文件。
// ============================================================

// ErrProductClassNotFound 是 product_class 无任何 product 匹配时的 sentinel error，
// 用于 handler 区分 404 vs 500。
var ErrProductClassNotFound = errors.New("product_class not matched to any product")

// CommandCompatibilityResult 是 GetCommandCompatibility 的返回 / API 响应载荷。
type CommandCompatibilityResult struct {
	ProductClass          string      `json:"product_class"`
	ProductID             uuid.UUID   `json:"product_id"`
	ParamModelID          uuid.UUID   `json:"param_model_id,omitempty"` // 空 UUID 表示 product 未配 ParamModel
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

// CompatibilityService 编排 ProductRegistry → ParamRegistry → CommandPathRepository →
// ComputeCommandCompatibility。
//
// 独立于 ConsoleService 以最小化对现有签名 / 测试的影响（spec §R-8.5 可选 P2）。
type CompatibilityService struct {
	productRegistry *product.Registry
	paramRegistry   *parammodel.Registry
	cmdPathRepo     CommandPathRepository
	logger          *zap.Logger
}

// NewCompatibilityService 构造。所有 deps 非 nil 必需。
func NewCompatibilityService(
	productRegistry *product.Registry,
	paramRegistry *parammodel.Registry,
	cmdPathRepo CommandPathRepository,
	logger *zap.Logger,
) *CompatibilityService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CompatibilityService{
		productRegistry: productRegistry,
		paramRegistry:   paramRegistry,
		cmdPathRepo:     cmdPathRepo,
		logger:          logger.Named("mml-compatibility"),
	}
}

// GetCommandCompatibility 计算指定 product_class 下的命令兼容性。
//
// 错误约定：
//   - productClass 无匹配 product → 返回 ErrProductClassNotFound（handler 404）
//   - ParamModel ID 为 nil 或映射加载失败 → supportedPaths=空集 → 所有有 path 的命令 unsupported
//   - 其他底层错误 → wrap 后返回
func (s *CompatibilityService) GetCommandCompatibility(
	ctx context.Context,
	productClass string,
) (*CommandCompatibilityResult, error) {
	if s.productRegistry == nil || s.paramRegistry == nil || s.cmdPathRepo == nil {
		return nil, fmt.Errorf("compatibility service not properly initialized")
	}

	// 1. resolve product_class → product
	match, err := s.productRegistry.MatchProductClass(ctx, productClass)
	if err != nil {
		// ProductRegistry.MatchProductClass 用 ErrOrphan 表示未命中，外部 wrap 一致
		s.logger.Debug("product_class no match",
			zap.String("product_class", productClass), zap.Error(err))
		return nil, ErrProductClassNotFound
	}
	if match == nil || match.Product == nil {
		return nil, ErrProductClassNotFound
	}
	prod := match.Product

	// 2. get default param_mappings (per design §3.3 — 不查 discovered)
	supportedPaths := make(map[string]struct{})
	if prod.ParamModelID != nil {
		ms, err := s.paramRegistry.GetByParamModel(ctx, *prod.ParamModelID)
		if err != nil {
			// 不视为致命 — 缺映射相当于"全部 unsupported"，由 spec §R-8.5 不阻塞语义兜底
			s.logger.Warn("load param_model mappings",
				zap.String("product_id", prod.ID.String()),
				zap.String("param_model_id", prod.ParamModelID.String()),
				zap.Error(err))
		} else if ms != nil {
			for _, m := range ms.Mappings {
				supportedPaths[m.StandardPath] = struct{}{}
			}
		}
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
		ProductID:             prod.ID,
		UnsupportedCommandIDs: unsupported,
	}
	if prod.ParamModelID != nil {
		result.ParamModelID = *prod.ParamModelID
	}
	return result, nil
}
