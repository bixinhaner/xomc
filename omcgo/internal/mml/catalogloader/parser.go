package catalogloader

import (
	"encoding/json"
	"fmt"
	"log"
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
// 完整的引用完整性校验（如 SubField.standardPath ↔ Command.targetPaths 一致性）
// 在 upsert 阶段进行，以便利用 DB 唯一约束。
//
// Schema 自动分派：
//   - JSON 根 `schemaVersion="v2"` → 走 validateV2（chapter:* groupCode、TreeNodeRefs 必填、
//     §R-2.4 (zh-CN, op) 跨章节唯一）
//   - 缺失或其它值 → 走 v1 路径（dedupGroupsFirstWins + validateV1）
//
// 重复 groupCode 处理（v1 only）：CMCC v2.3 规范中部分 object（例 FAPService）在多个章节
// 被引用，导致同 groupCode 多次出现。Parse 在 validate 前**先去重 first-wins**，
// 保留首次出现的 chapterCode，跳过后续副本，并 log.Printf 告警便于排查。
// v2 schema 已在 offline parser 阶段按 §R-3.1 合并，DB 行无重复 → 不需要 dedup。
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json: %w", err)
	}
	if c.IsV2() {
		if err := validateV2(&c); err != nil {
			return nil, err
		}
		return &c, nil
	}
	// v1 path
	dedupGroupsFirstWins(&c)
	if err := validateV1(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

// dedupGroupsFirstWins 把 groups 切片按 groupCode 去重，保留首次出现的实例。
// 规范源里部分 object 在多个 chapter 被引用 → 此处 first-wins 让 chapterCode
// 取首次出现位置；后续副本（即便 commands 形态相同）直接丢弃。
//
// 行为可观测：每丢一条 log.Printf 一次警告（启动期一次性，量极小）。
func dedupGroupsFirstWins(c *Catalog) {
	if len(c.Groups) <= 1 {
		return
	}
	seen := make(map[string]int, len(c.Groups))
	kept := c.Groups[:0]
	for _, g := range c.Groups {
		if firstIdx, ok := seen[g.GroupCode]; ok {
			log.Printf("catalogloader: dropped duplicate groupCode %q (chapter=%q kept first at index=%d, dup chapter=%q)",
				g.GroupCode, c.Groups[firstIdx].ChapterCode, firstIdx, g.ChapterCode)
			continue
		}
		seen[g.GroupCode] = len(kept)
		kept = append(kept, g)
	}
	c.Groups = kept
}

// validateV1 执行 v1 schema 最小语义校验，不访问 DB。
//
// 关键约束：
//   - specVersion / carrier / tech 必填
//   - 每 group 内 (operationType) 唯一
//   - operationType ∈ {LST, MOD, ADD, RMV}
//   - accessType ∈ {RO, RW}
//   - groupCode 在文件内唯一
//   - 每条 command 的 targetPaths 非空（v1 把 path 字符串数组直接存在 command 里）
func validateV1(c *Catalog) error {
	if c.SpecVersion == "" {
		return fmt.Errorf("catalog: specVersion is required")
	}
	if c.Carrier == "" {
		return fmt.Errorf("catalog: carrier is required")
	}
	if c.Tech == "" {
		return fmt.Errorf("catalog: tech is required")
	}
	if len(c.Groups) == 0 {
		return fmt.Errorf("catalog: groups is empty")
	}

	groupCodes := make(map[string]struct{}, len(c.Groups))
	for gi, g := range c.Groups {
		if g.GroupCode == "" {
			return fmt.Errorf("catalog: groups[%d].groupCode is empty", gi)
		}
		if _, dup := groupCodes[g.GroupCode]; dup {
			return fmt.Errorf("catalog: groups[%d].groupCode %q duplicated", gi, g.GroupCode)
		}
		groupCodes[g.GroupCode] = struct{}{}

		if len(g.Commands) == 0 {
			return fmt.Errorf("catalog: groups[%d] (%s) has no commands", gi, g.GroupCode)
		}
		opSeen := make(map[string]struct{}, len(g.Commands))
		for ci, cmd := range g.Commands {
			if !isValidOp(cmd.OperationType) {
				return fmt.Errorf("catalog: groups[%d].commands[%d].operationType %q invalid (allowed: LST/MOD/ADD/RMV)",
					gi, ci, cmd.OperationType)
			}
			if _, dup := opSeen[cmd.OperationType]; dup {
				return fmt.Errorf("catalog: groups[%d] (%s) has duplicate operationType %q",
					gi, g.GroupCode, cmd.OperationType)
			}
			opSeen[cmd.OperationType] = struct{}{}
			if len(cmd.TargetPaths) == 0 {
				return fmt.Errorf("catalog: groups[%d].commands[%d] (%s) has empty targetPaths",
					gi, ci, cmd.OperationType)
			}
		}

		mmlCodeSeen := make(map[string]struct{}, len(g.SubFields))
		for si, sf := range g.SubFields {
			if sf.MMLCode == "" {
				return fmt.Errorf("catalog: groups[%d].subFields[%d].mmlCode is empty", gi, si)
			}
			if _, dup := mmlCodeSeen[sf.MMLCode]; dup {
				return fmt.Errorf("catalog: groups[%d] (%s) has duplicate mmlCode %q",
					gi, g.GroupCode, sf.MMLCode)
			}
			mmlCodeSeen[sf.MMLCode] = struct{}{}

			if sf.AccessType != "" && sf.AccessType != "RO" && sf.AccessType != "RW" {
				return fmt.Errorf("catalog: groups[%d].subFields[%d].accessType %q invalid",
					gi, si, sf.AccessType)
			}
		}
	}
	return nil
}

// validateV2 执行 v2 schema 校验，对应 spec §R-1 / §R-2 / §R-2.4 / §R-3。
//
// 关键约束：
//   - specVersion / carrier / tech / schemaVersion="v2" 必填
//   - 每个 group.groupCode 必须带 "chapter:" 前缀（§R-1 一级分组 = 18 章节）
//   - 每条 command 必须有 commandCode + operationType + logicalNameI18n["zh-CN"]
//     + groupCodeObject
//   - 同 group 内 (groupCodeObject, operationType) 唯一
//   - §R-2.4 跨 group：同 (zh-CN, op) 不允许对应不同 groupCodeObject
//   - LST/MOD 必须有 treeNodeRefs；ADD/RMV 的 treeNodeRefs 应是父级实例路径
//     （非空但通常只 1 条）
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

		// 同 group 内 (groupCodeObject, operationType) 唯一
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
