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
//   - ADD/RMV：supported set 中至少 1 条 path 以 target_object 为前缀 → 可见
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
		// 非孤儿：set 中有 path 以 target_object 为前缀 → 可见 + supported=1
		// （user Q5/Option A 决定）
		if supported.HasPathWithPrefix(targetObject) {
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
