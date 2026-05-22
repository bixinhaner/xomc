package catalogloader

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseFile reads a JSON catalog file from disk and returns the parsed Catalog.
//
// 解析失败属于 fatal error（fail-fast）；上层 Load 收到错误后应阻止 app 启动。
func ParseFile(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes raw JSON bytes into a Catalog. Performs lightweight semantic
// validation (required fields, allowed enums) that does not require DB access.
//
// 完整的引用完整性校验（如 TreeNodeRefs ↔ standard_params 一致性）
// 在 upsert 阶段进行，以便利用 DB 唯一约束。
//
// 仅校验 v2 schema（spec §R-1/§R-2/§R-2.4/§R-3）。
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json: %w", err)
	}
	if err := validateV2(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

// validateV2 执行 v2 schema 校验，对应 spec §R-1 / §R-2 / §R-2.4 / §R-3。
//
// 关键约束：
//   - specVersion / carrier / tech 必填
//   - 每个 group.groupCode 必须带 "chapter:" 前缀（§R-1 一级分组 = 18 章节）
//   - 每条 command 必须有 commandCode + operationType + logicalNameI18n["zh-CN"]
//     + groupCodeObject
//   - 同 group 内 (groupCodeObject, operationType) 唯一
//   - §R-2.4 跨 group：同 (zh-CN, op) 不允许对应不同 groupCodeObject
//   - LST/MOD 必须有 treeNodeRefs；ADD/RMV 的 treeNodeRefs 应是父级实例路径
//   - operationType ∈ {LST, MOD, ADD, RMV}
func validateV2(c *Catalog) error {
	if c.SpecVersion == "" {
		return fmt.Errorf("v2 catalog: specVersion is required")
	}
	if c.Carrier == "" {
		return fmt.Errorf("v2 catalog: carrier is required")
	}
	if c.Tech == "" {
		return fmt.Errorf("v2 catalog: tech is required")
	}
	if len(c.Groups) == 0 {
		return fmt.Errorf("v2 catalog: groups is empty")
	}

	groupCodes := make(map[string]struct{}, len(c.Groups))
	// §R-2.4 跨章节 (zh-CN, op) 唯一性：value 记录首次出现的 groupCodeObject，
	// 用于碰撞时报告冲突位置。
	type nameOpOrigin struct {
		groupCode       string
		groupCodeObject string
	}
	globalNameOp := make(map[string]nameOpOrigin, len(c.Groups)*4)

	for gi, g := range c.Groups {
		if g.GroupCode == "" {
			return fmt.Errorf("v2 catalog: groups[%d].groupCode is empty", gi)
		}
		if !strings.HasPrefix(g.GroupCode, "chapter:") {
			return fmt.Errorf("v2 catalog: groups[%d].groupCode %q must have 'chapter:' prefix (§R-1 一级分组 = 18 章节)",
				gi, g.GroupCode)
		}
		if _, dup := groupCodes[g.GroupCode]; dup {
			return fmt.Errorf("v2 catalog: groups[%d].groupCode %q duplicated", gi, g.GroupCode)
		}
		groupCodes[g.GroupCode] = struct{}{}

		if g.NameI18n == nil || g.NameI18n["zh-CN"] == "" {
			return fmt.Errorf("v2 catalog: groups[%d] (%s) missing nameI18n.zh-CN", gi, g.GroupCode)
		}

		if len(g.Commands) == 0 {
			return fmt.Errorf("v2 catalog: groups[%d] (%s) has no commands", gi, g.GroupCode)
		}

		objOpSeen := make(map[string]int, len(g.Commands))
		for ci, cmd := range g.Commands {
			if !isValidOp(cmd.OperationType) {
				return fmt.Errorf("v2 catalog: groups[%d].commands[%d].operationType %q invalid (allowed: LST/MOD/ADD/RMV)",
					gi, ci, cmd.OperationType)
			}
			if cmd.CommandCode == "" {
				return fmt.Errorf("v2 catalog: groups[%d].commands[%d] missing commandCode (期望 '<OP>:<groupCodeObject>')",
					gi, ci)
			}
			if cmd.GroupCodeObject == "" {
				return fmt.Errorf("v2 catalog: groups[%d].commands[%d] (%s) missing groupCodeObject",
					gi, ci, cmd.CommandCode)
			}
			zh := ""
			if cmd.LogicalNameI18n != nil {
				zh = cmd.LogicalNameI18n["zh-CN"]
			}
			if zh == "" {
				return fmt.Errorf("v2 catalog: groups[%d].commands[%d] (%s) missing logicalNameI18n.zh-CN (§R-2.4)",
					gi, ci, cmd.CommandCode)
			}
			if len(cmd.TreeNodeRefs) == 0 {
				return fmt.Errorf("v2 catalog: groups[%d].commands[%d] (%s) has empty treeNodeRefs",
					gi, ci, cmd.CommandCode)
			}
			localKey := cmd.GroupCodeObject + "::" + cmd.OperationType
			if prevIdx, dup := objOpSeen[localKey]; dup {
				return fmt.Errorf("v2 catalog: groups[%d] commands[%d] (%s) duplicates (object=%s, op=%s) with index %d",
					gi, ci, cmd.CommandCode, cmd.GroupCodeObject, cmd.OperationType, prevIdx)
			}
			objOpSeen[localKey] = ci

			// §R-2.4 跨章节同名禁令：同 (zh-CN, op) 不能映射到不同 groupCodeObject
			globalKey := zh + "::" + cmd.OperationType
			if prev, dup := globalNameOp[globalKey]; dup {
				if prev.groupCodeObject != cmd.GroupCodeObject {
					return fmt.Errorf("v2 catalog: §R-2.4 跨章节同名禁令违反 — (zh=%q, op=%q) 同时映射到 groupCodeObject=%q 和 %q (groups %s vs %s)",
						zh, cmd.OperationType,
						prev.groupCodeObject, cmd.GroupCodeObject,
						prev.groupCode, g.GroupCode)
				}
			} else {
				globalNameOp[globalKey] = nameOpOrigin{
					groupCode:       g.GroupCode,
					groupCodeObject: cmd.GroupCodeObject,
				}
			}
		}
	}
	return nil
}

func isValidOp(op string) bool {
	switch op {
	case "LST", "MOD", "ADD", "RMV":
		return true
	}
	return false
}
