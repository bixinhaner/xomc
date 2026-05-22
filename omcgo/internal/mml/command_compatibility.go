package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
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
// 解析路径（绕过 ProductRegistry.MatchProductClass 正则）：
//   SELECT d.product_id, COALESCE(d.param_model_id, p.param_model_id)
//     FROM devices d LEFT JOIN products p ON p.id = d.product_id
//    WHERE d.product_class = $1
//    ORDER BY (effective_param_model_id IS NULL) LIMIT 1
//
//   字典 type='product_class' 已对齐 devices.product_class DISTINCT 真实值
//   （seed/000154），因此前端传来的 product_class 一定能命中至少一台设备。
//   product_class_patterns 全局正则只在设备 Bootstrap 路径用，命令兼容性
//   计算不再依赖它，避免「字典里有但 patterns 没有」导致的 404。
//
// 三处独立单元：
//   1. CommandPathRepository — 从 mml_commands 取每条命令的 path 集合
//      （tree_node_refs 优先，回退 target_paths）
//   2. ComputeCommandCompatibility — 纯函数，无外部依赖，可单测
//   3. CompatibilityService — 编排：devices 反查 → ParamRegistry → 计算
//
// 与现有 ConsoleService 解耦（独立 struct + 独立 deps），避免破坏
// NewConsoleService 签名影响 5 个测试文件。
// ============================================================

// ErrProductClassNotFound 是 product_class 在 devices 表里完全无匹配时的 sentinel error，
// 用于 handler 区分 404 vs 500。字典与 devices 对齐后，正常路径不会触发。
var ErrProductClassNotFound = errors.New("product_class not matched to any device")

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

// CompatibilityService 编排 devices 反查 → ParamRegistry → CommandPathRepository →
// ComputeCommandCompatibility。
//
// 独立于 ConsoleService 以最小化对现有签名 / 测试的影响（spec §R-8.5 可选 P2）。
//
// 不再依赖 ProductRegistry：product_class 的来源（字典 type='product_class'）已
// 与 devices.product_class DISTINCT 对齐（seed/000154），直接走 devices LEFT JOIN
// products 拿 param_model_id，跳过 product_class_patterns 全局正则。
type CompatibilityService struct {
	pool          *pgxpool.Pool
	paramRegistry *parammodel.Registry
	cmdPathRepo   CommandPathRepository
	logger        *zap.Logger
}

// NewCompatibilityService 构造。所有 deps 非 nil 必需。
func NewCompatibilityService(
	pool *pgxpool.Pool,
	paramRegistry *parammodel.Registry,
	cmdPathRepo CommandPathRepository,
	logger *zap.Logger,
) *CompatibilityService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CompatibilityService{
		pool:          pool,
		paramRegistry: paramRegistry,
		cmdPathRepo:   cmdPathRepo,
		logger:        logger.Named("mml-compatibility"),
	}
}

// resolveProductClassSQL 直接从 devices 反查 product_id + 有效 param_model_id。
//
// 选择策略：优先返回"有 param_model 链路"的行（device.param_model_id 或
// product.param_model_id 非空），否则任选一台同 product_class 设备 — 这样
// 即使大多数设备未发现私网映射，只要有一台跑过 FileType=11 就能命中。
const resolveProductClassSQL = `
SELECT d.product_id,
       COALESCE(d.param_model_id, p.param_model_id) AS effective_param_model_id
  FROM devices d
  LEFT JOIN products p ON p.id = d.product_id
 WHERE d.product_class = $1
 ORDER BY (COALESCE(d.param_model_id, p.param_model_id) IS NULL)
 LIMIT 1`

// GetCommandCompatibility 计算指定 product_class 下的命令兼容性。
//
// 错误约定：
//   - productClass 在 devices 表完全无匹配 → 返回 ErrProductClassNotFound（handler 404）
//   - 设备/产品未关联 ParamModel → supportedPaths=空集 → 所有有 path 的命令 unsupported
//     （UI 应以"未配置参数模型"提示用户而非"全部不兼容"，由前端区分 paramModelID 是否为零 UUID）
//   - 其他底层错误 → wrap 后返回
func (s *CompatibilityService) GetCommandCompatibility(
	ctx context.Context,
	productClass string,
) (*CommandCompatibilityResult, error) {
	if s.pool == nil || s.paramRegistry == nil || s.cmdPathRepo == nil {
		return nil, fmt.Errorf("compatibility service not properly initialized")
	}

	// 1. 直接走 devices 反查（绕过 ProductRegistry.MatchProductClass 正则）
	var (
		productID    *uuid.UUID
		paramModelID *uuid.UUID
	)
	if err := s.pool.QueryRow(ctx, resolveProductClassSQL, productClass).
		Scan(&productID, &paramModelID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Debug("product_class has no device row",
				zap.String("product_class", productClass))
			return nil, ErrProductClassNotFound
		}
		return nil, fmt.Errorf("resolve product_class via devices: %w", err)
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
