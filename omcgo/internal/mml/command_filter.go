package mml

import (
	"encoding/json"
)

// ============================================================
// command_filter.go — T-0172 命令级 catalog 过滤与计数标注
//
// 输入：单条 GroupTreeCommand + 该命令的 target_paths（来自 SQL JOIN，
// 见 group_tree_repository.go CommandTargetPaths 列）+ SupportedSet。
//
// 输出：是否对该 productClass 设备可见 + 计数 + 不支持 path 列表。
//
// 显示规则（user 确认）：
//   - LST/MOD：target_paths 中至少 1 条在 supported set → 可见
//   - ADD/RMV：supported set 显式包含 target_object 下的实例对象映射 → 可见
//   - 孤儿设备：visible=true，全部 path 标 unsupported（仅给信息提示）
//   - supported set 为 nil（未启用过滤）：visible=true，不做标注
// ============================================================

// CommandFilterAnnotation 是命令的过滤+计数结果。
// 字段语义对应 GroupTreeCommand 上待补的同名字段。
type CommandFilterAnnotation struct {
	Visible            bool     // 是否在 UI 上显示该命令
	SupportedPathCount int      // 当前 productClass 下「is_supported=true 且 is_active=true」的 path 数；ADD/RMV 取值 0 或 1
	UnsupportedPaths   []string // 不支持的具体 path 列表
	ProductResolved    bool     // 该 productClass 是否成功匹配到 product
}

// AnnotateCommand 对单条命令做过滤+计数决策。
//
// 参数：
//   - opType: 命令的 operation_type ("LST" / "MOD" / "ADD" / "RMV")
//   - targetPathsJSON: 命令的 target_paths jsonb raw bytes（来自 mml_commands 表）
//   - targetObject: 命令的 target_object（ADD/RMV 用）
//   - supported: SupportedSet；nil → 不过滤、不标注
func AnnotateCommand(opType string, targetPathsJSON []byte, targetObject string, supported *SupportedSet) CommandFilterAnnotation {
	// supported 为 nil → 不启用过滤，直接可见，无标注
	if supported == nil {
		return CommandFilterAnnotation{Visible: true}
	}

	annotation := CommandFilterAnnotation{ProductResolved: supported.ProductResolved}

	switch opType {
	case "LST", "MOD":
		paths := parseTargetPathsJSON(targetPathsJSON)
		unsupported := make([]string, 0)
		supportedCount := 0
		for _, p := range paths {
			if supported.Contains(p) {
				supportedCount++
			} else {
				unsupported = append(unsupported, p)
			}
		}
		annotation.SupportedPathCount = supportedCount
		annotation.UnsupportedPaths = unsupported
		// 孤儿设备：visible=true（user Q4 决定）
		// 非孤儿且 supportedCount=0 → 隐藏（user Q1 决定）
		if !supported.ProductResolved {
			annotation.Visible = true
		} else {
			annotation.Visible = supportedCount > 0
		}

	case "ADD", "RMV":
		// 孤儿：visible=true，标 unsupported，supported=0
		if !supported.ProductResolved {
			annotation.Visible = true
			annotation.SupportedPathCount = 0
			if targetObject != "" {
				annotation.UnsupportedPaths = []string{targetObject}
			}
			break
		}
		// 非孤儿：只有显式对象映射才表示支持 AddObject/DeleteObject；子参数不算。
		if supported.SupportsObjectCollection(targetObject) {
			annotation.Visible = true
			annotation.SupportedPathCount = 1
		} else {
			annotation.Visible = false
			annotation.SupportedPathCount = 0
			if targetObject != "" {
				annotation.UnsupportedPaths = []string{targetObject}
			}
		}

	default:
		// 未知 op_type：保留可见（兼容性）
		annotation.Visible = true
	}

	return annotation
}

// parseTargetPathsJSON 把 mml_commands.target_paths JSONB raw bytes 解为 []string。
// 解析失败返回空 slice（不报错；上层用空 supported 处理）。
func parseTargetPathsJSON(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var paths []string
	if err := json.Unmarshal(raw, &paths); err != nil {
		return nil
	}
	return paths
}

// unsupportedPathFilter 是执行过程中按产品记录的运行时不支持 path 集合。
// 它与 ParamModel 的静态支持集合叠加使用，保证命令树与参数列表使用同一口径。
type unsupportedPathFilter struct {
	read  map[string]struct{}
	write map[string]struct{}
}

func newUnsupportedPathFilter(paths []UnsupportedPath) *unsupportedPathFilter {
	if len(paths) == 0 {
		return nil
	}
	filter := &unsupportedPathFilter{
		read:  make(map[string]struct{}),
		write: make(map[string]struct{}),
	}
	for _, path := range paths {
		if path.Path == "" {
			continue
		}
		if path.ReadUnsupported {
			filter.read[path.Path] = struct{}{}
		}
		if path.WriteUnsupported {
			filter.write[path.Path] = struct{}{}
		}
	}
	if len(filter.read) == 0 && len(filter.write) == 0 {
		return nil
	}
	return filter
}

func (f *unsupportedPathFilter) blocks(operationType, path string) bool {
	if f == nil || path == "" {
		return false
	}
	var blocked map[string]struct{}
	if operationType == "LST" {
		blocked = f.read
	} else {
		blocked = f.write
	}
	_, ok := blocked[path]
	return ok
}

// countAvailableCommandPaths 计算命令在两套支持集合叠加后的最终可用 path 数。
// ADD/RMV 的 target_object 没有逐 path 列表，必须由显式实例对象映射授权。
func countAvailableCommandPaths(cmd GroupTreeCommand, supported *SupportedSet, blocked *unsupportedPathFilter) int {
	switch cmd.OperationType {
	case "LST", "MOD":
		count := 0
		for _, path := range parseTargetPathsJSON(cmd.TargetPathsRaw()) {
			if supported.Contains(path) && !blocked.blocks(cmd.OperationType, path) {
				count++
			}
		}
		return count
	case "ADD", "RMV":
		if cmd.TargetObject == "" || !supported.SupportsObjectCollection(cmd.TargetObject) {
			return 0
		}
		objectPath := cmd.TargetObject + "{i}."
		if blocked.blocks(cmd.OperationType, objectPath) {
			return 0
		}
		return 1
	default:
		// mml_commands 当前约束为 LST/MOD/ADD/RMV；未知类型沿用兼容可见语义。
		return 1
	}
}
