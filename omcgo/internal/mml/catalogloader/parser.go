package catalogloader

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
// 重复 groupCode 处理：CMCC v2.3 规范中，部分 object（例 FAPService）在多个章节
// 被引用，导致同 groupCode 多次出现。Parse 在 validate 前**先去重 first-wins**，
// 保留首次出现的 chapterCode，跳过后续副本，并 log.Printf 告警便于排查。
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json: %w", err)
	}
	dedupGroupsFirstWins(&c)
	if err := validate(&c); err != nil {
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

// validate 执行最小语义校验，不访问 DB。
//
// 关键约束：
//   - specVersion / carrier / tech 必填
//   - 每 group 内 (operationType) 唯一
//   - operationType ∈ {LST, MOD, ADD, RMV}
//   - accessType ∈ {RO, RW}
//   - groupCode 在文件内唯一
func validate(c *Catalog) error {
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

func isValidOp(op string) bool {
	switch op {
	case "LST", "MOD", "ADD", "RMV":
		return true
	}
	return false
}
