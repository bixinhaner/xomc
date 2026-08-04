package specparser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerateSQL 把 DiffReport + SpecCatalog 渲染为完整 goose seed SQL 文件内容。
//
// 输出形态（与 CLAUDE.md §5.5 迁移规范完全对齐）：
//
//	-- +goose Up
//	BEGIN;
//	  Section A: INSERT standard_params ... ON CONFLICT (standard_path) DO UPDATE SET ...
//	  Section B: INSERT mml_commands     ... ON CONFLICT (command_code)  DO UPDATE SET ...
//	  Section C: INSERT mml_command_sub_fields ... ON CONFLICT (command_id, standard_path_id) DO UPDATE SET ...
//	COMMIT;
//
//	-- +goose Down
//	BEGIN;
//	  DELETE FROM mml_command_sub_fields WHERE command_id IN (SELECT id FROM mml_commands WHERE command_code IN (...));
//	  DELETE FROM mml_commands WHERE command_code IN (...);
//	  -- standard_params 不删（多消费者共享）
//	COMMIT;
//
// 约束：
//   - 全部 UUID 用 gen_random_uuid() 或 (SELECT id FROM ...) 子查询，不硬编码 UUID 字面量
//   - JSON 字段（i18n）经 json.Marshal 后再 SQL 字符串转义（双重单引号）
//   - 所有 INSERT 含 ON CONFLICT ... DO UPDATE SET（重跑幂等）
//   - 列名顺序与 migrations/000058+000095 schema 严格对齐
func GenerateSQL(rep *DiffReport, cat *SpecCatalog) string {
	var b strings.Builder
	writeHeader(&b, cat, rep)
	b.WriteString("-- +goose Up\n")
	b.WriteString("BEGIN;\n\n")

	writeSectionStandardParams(&b, rep)
	writeSectionCommands(&b, rep)
	writeSectionSubFields(&b, rep)

	b.WriteString("COMMIT;\n\n")
	b.WriteString("-- +goose Down\n")
	writeDownSection(&b, rep)
	return b.String()
}

func writeHeader(b *strings.Builder, cat *SpecCatalog, rep *DiffReport) {
	fmt.Fprintf(b, "-- ============================================================\n")
	fmt.Fprintf(b, "-- MML catalog 增量补齐种子（T-0169 自动生成）\n")
	fmt.Fprintf(b, "--\n")
	fmt.Fprintf(b, "-- 源规范: %s\n", cat.SourceMD)
	fmt.Fprintf(b, "-- 规范 hash: %s\n", cat.SourceHash)
	fmt.Fprintf(b, "-- 版本: %s\n", cat.Version)
	fmt.Fprintf(b, "-- 生成时间: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(b, "--\n")
	fmt.Fprintf(b, "-- Diff Summary:\n")
	fmt.Fprintf(b, "--   🔴 新增 standard_params:        %d\n", rep.Summary.NewStandardParams)
	fmt.Fprintf(b, "--   🔴 新增 mml_commands:           %d\n", rep.Summary.NewCommands)
	fmt.Fprintf(b, "--   🟡 修改 mml_commands:           %d\n", rep.Summary.UpdatedCommands)
	fmt.Fprintf(b, "--   🔴 新增 mml_command_sub_fields: %d\n", rep.Summary.NewSubFieldLinks)
	fmt.Fprintf(b, "--   ⚪ DB 多余 (orphan) commands:   %d  (仅标记不删)\n", rep.Summary.OrphanCommands)
	fmt.Fprintf(b, "--   ⚠ CrossCheck warnings:         %d\n", rep.Summary.CrossWarnings)
	fmt.Fprintf(b, "--\n")
	fmt.Fprintf(b, "-- 工具: omcctl mml import-spec-md (internal/mml/specparser)\n")
	fmt.Fprintf(b, "-- 设计: docs/project/prd/F06-mml-catalog-spec-parser.md\n")
	fmt.Fprintf(b, "-- 由 T-0169 落地；不要手工修改，请重跑 omcctl 重新生成。\n")
	fmt.Fprintf(b, "-- ============================================================\n\n")
}

// =============================================================================
// Section A: standard_params
// =============================================================================

func writeSectionStandardParams(b *strings.Builder, rep *DiffReport) {
	if len(rep.NewStandardParams) == 0 {
		b.WriteString("-- Section A: standard_params — 0 条新增（全部已存在）\n\n")
		return
	}
	fmt.Fprintf(b, "-- Section A: standard_params — 新增 %d 条\n", len(rep.NewStandardParams))
	b.WriteString("INSERT INTO standard_params (standard_path, entry_type, access, data_type, change_applies, min_value, max_value) VALUES\n")
	for i, p := range rep.NewStandardParams {
		entryType := EntryTypeParameter
		if strings.HasSuffix(p.StandardPath, ".") {
			entryType = EntryTypeObject
		}
		fmt.Fprintf(b, "    (%s, %s, %s, %s, %s, %s, %s)",
			sqlStr(p.StandardPath),
			sqlStr(entryType),
			sqlStr(nzAccess(p.Access)),
			sqlStr(nzDataType(p.DataType)),
			sqlStr(ChangeAppliesImmediate),
			sqlNullableInt64(p.MinValue),
			sqlNullableInt64(p.MaxValue),
		)
		if i < len(rep.NewStandardParams)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString(`ON CONFLICT (standard_path) DO UPDATE SET
    entry_type     = EXCLUDED.entry_type,
    access         = EXCLUDED.access,
    data_type      = EXCLUDED.data_type,
    change_applies = EXCLUDED.change_applies,
    min_value      = EXCLUDED.min_value,
    max_value      = EXCLUDED.max_value,
    updated_at     = NOW();

`)
}

// =============================================================================
// Section B: mml_commands
// =============================================================================

func writeSectionCommands(b *strings.Builder, rep *DiffReport) {
	all := append([]*SpecCommand{}, rep.NewCommands...)
	all = append(all, rep.UpdatedCommands...)
	if len(all) == 0 {
		b.WriteString("-- Section B: mml_commands — 0 条新增/修改\n\n")
		return
	}
	fmt.Fprintf(b, "-- Section B: mml_commands — 新增 %d 条 + 修改 %d 条\n",
		len(rep.NewCommands), len(rep.UpdatedCommands))

	// 用 CTE 缓存 chapter 分组 id，避免每行子查询
	b.WriteString("WITH chapter_groups AS (\n")
	b.WriteString("    SELECT group_code, id FROM mml_command_groups\n")
	b.WriteString("     WHERE param_version = 'cmcc-td-lte-v2.3'\n")
	b.WriteString("       AND group_code LIKE 'chapter:%'\n")
	b.WriteString(")\n")
	// logical_code 列已 DROP（读时从 command_code 派生），seed 生成不再写该列。
	b.WriteString("INSERT INTO mml_commands (\n")
	b.WriteString("    command_name, command_code, category, description,\n")
	b.WriteString("    rpc_method, operation_type, target_paths, target_object,\n")
	b.WriteString("    group_id, command_name_i18n,\n")
	b.WriteString("    logical_name_i18n,\n")
	b.WriteString("    source, catalog_protected, help_doc\n")
	b.WriteString(") VALUES\n")

	for i, c := range all {
		writeCommandValueRow(b, c)
		if i < len(all)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString(`ON CONFLICT (command_code) DO UPDATE SET
    target_paths      = EXCLUDED.target_paths,
    target_object     = EXCLUDED.target_object,
    command_name_i18n = EXCLUDED.command_name_i18n,
    logical_name_i18n = EXCLUDED.logical_name_i18n,
    description       = EXCLUDED.description,
    source            = 'standard',
    catalog_protected = true;

`)
}

func writeCommandValueRow(b *strings.Builder, c *SpecCommand) {
	zhName := opZhPrefix(c.OperationType) + " " + c.CommandZhName
	enLogicalName := commandEnglishLogicalName(c)
	enName := opEnPrefix(c.OperationType) + " " + enLogicalName
	// issue #67 §5：i18n 键统一长码 zh-CN/en-US（与 seed/000039、pickI18n 对齐），
	// 不再写短键 zh/en，避免 catalog 再导入时回灌短键。
	cmdNameI18n, _ := json.Marshal(map[string]string{"en-US": enName, "zh-CN": zhName})
	logicalI18n, _ := json.Marshal(map[string]string{"en-US": enLogicalName, "zh-CN": c.CommandZhName})

	targetPathsJSON, _ := json.Marshal(c.TargetPaths)
	targetObj := "NULL"
	if c.TargetObject != "" {
		targetObj = sqlStr(c.TargetObject)
	}

	fmt.Fprintf(b, "    (%s, %s, %s, %s, %s, %s, %s::jsonb, %s,\n",
		sqlStr(zhName),                     // command_name
		sqlStr(c.CommandCode),              // command_code
		sqlStr(chapterCategory(c.Chapter)), // category
		sqlStr(zhName),                     // description
		sqlStr(c.RPCMethod),                // rpc_method
		sqlStr(c.OperationType),            // operation_type
		sqlStr(string(targetPathsJSON)),    // target_paths::jsonb
		targetObj,                          // target_object
	)
	fmt.Fprintf(b, "     (SELECT id FROM chapter_groups WHERE group_code = %s), %s::jsonb,\n",
		sqlStr("chapter:"+c.Chapter),
		sqlStr(string(cmdNameI18n)),
	)
	fmt.Fprintf(b, "     %s::jsonb,\n",
		sqlStr(string(logicalI18n)),
	)
	fmt.Fprintf(b, "     'standard', true, '')")
}

// =============================================================================
// Section C: mml_command_sub_fields
// =============================================================================

func writeSectionSubFields(b *strings.Builder, rep *DiffReport) {
	if len(rep.NewSubFieldLinks) == 0 {
		b.WriteString("-- Section C: mml_command_sub_fields — 0 条新增关联\n\n")
		return
	}
	fmt.Fprintf(b, "-- Section C: mml_command_sub_fields — 新增 %d 条关联\n",
		len(rep.NewSubFieldLinks))
	// 按 CommandCode + SortOrder 排序，便于阅读
	links := append([]*SubFieldLink{}, rep.NewSubFieldLinks...)
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].CommandCode != links[j].CommandCode {
			return links[i].CommandCode < links[j].CommandCode
		}
		return links[i].SortOrder < links[j].SortOrder
	})

	for _, l := range links {
		// issue #67 §2/§5：长码键，且把 en 用 ParamName(英文优先)，缺失时不再硬塞中文——
		// 由 seed/000038 的 standard_path 叶子兜底逻辑负责回填英文标签。
		labelI18n, _ := json.Marshal(map[string]string{"en-US": l.ParamName, "zh-CN": l.ChineseName})
		fmt.Fprintf(b, "INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)\n")
		fmt.Fprintf(b, "SELECT c.id, p.id, %s, %s::jsonb, true, false, %d\n",
			sqlStr(l.MmlCode), sqlStr(string(labelI18n)), l.SortOrder)
		fmt.Fprintf(b, "  FROM mml_commands  c\n")
		fmt.Fprintf(b, "  JOIN standard_params p ON p.standard_path = %s\n", sqlStr(l.StandardPath))
		fmt.Fprintf(b, " WHERE c.command_code = %s\n", sqlStr(l.CommandCode))
		fmt.Fprintf(b, "ON CONFLICT (command_id, standard_path_id) DO UPDATE SET\n")
		fmt.Fprintf(b, "    mml_code   = EXCLUDED.mml_code,\n")
		fmt.Fprintf(b, "    label_i18n = EXCLUDED.label_i18n,\n")
		fmt.Fprintf(b, "    sort_order = EXCLUDED.sort_order;\n\n")
	}
}

// =============================================================================
// Down 段
// =============================================================================

func writeDownSection(b *strings.Builder, rep *DiffReport) {
	if len(rep.NewCommands) == 0 {
		b.WriteString("-- 本次 seed 仅 UPDATE 现有命令 / 增加 sub_field 关联，down 段无需 DELETE\n")
		b.WriteString("-- standard_params 不删（多消费者共享）。\n")
		return
	}
	b.WriteString("BEGIN;\n\n")
	b.WriteString("-- 精确 DELETE 本 seed INSERT 的命令 (source='standard') 的 sub_field 关联\n")
	b.WriteString("DELETE FROM mml_command_sub_fields\n")
	b.WriteString(" WHERE command_id IN (\n")
	b.WriteString("   SELECT id FROM mml_commands WHERE command_code IN (\n")
	for i, c := range rep.NewCommands {
		fmt.Fprintf(b, "     %s", sqlStr(c.CommandCode))
		if i < len(rep.NewCommands)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("   )\n")
	b.WriteString(" );\n\n")

	b.WriteString("DELETE FROM mml_commands\n")
	b.WriteString(" WHERE command_code IN (\n")
	for i, c := range rep.NewCommands {
		fmt.Fprintf(b, "     %s", sqlStr(c.CommandCode))
		if i < len(rep.NewCommands)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString(" );\n\n")
	b.WriteString("-- standard_params 不删（多消费者共享，可能被其他 catalog 引用）\n")
	b.WriteString("COMMIT;\n")
}

// =============================================================================
// SQL 转义与映射辅助
// =============================================================================

// sqlStr 把 Go 字符串转为 SQL 单引号字符串（双重单引号转义）。
func sqlStr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func sqlNullableInt64(v *int64) string {
	if v == nil {
		return "NULL"
	}
	return fmt.Sprintf("%d", *v)
}

// nzAccess 把空字符串归一为 READ_ONLY（standard_params.access 列允许 NULL 但建议非空）。
func nzAccess(s string) string {
	if s == "" {
		return AccessReadOnly
	}
	return s
}

// nzDataType 把空字符串归一为 "string"。
func nzDataType(s string) string {
	if s == "" {
		return "string"
	}
	return s
}

// opZhPrefix / opEnPrefix 是命令叶子的中文 / 英文前缀。
func opZhPrefix(op string) string {
	switch op {
	case OpLST:
		return "列出"
	case OpMOD:
		return "修改"
	case OpADD:
		return "增加"
	case OpRMV:
		return "删除"
	}
	return op
}

func opEnPrefix(op string) string {
	switch op {
	case OpLST:
		return "List"
	case OpMOD:
		return "Modify"
	case OpADD:
		return "Add"
	case OpRMV:
		return "Remove"
	}
	return op
}

func commandEnglishLogicalName(c *SpecCommand) string {
	if c == nil {
		return "Command"
	}
	if name := strings.TrimSpace(c.CommandEnName); name != "" {
		return name
	}
	if name := logicalCodeToEnglishName(c.LogicalCode); name != "" {
		return name
	}
	if name := logicalCodeToEnglishName(deriveLogicalCodeFromCommandCode(c.CommandCode, c.OperationType)); name != "" {
		return name
	}
	return "Command"
}

func logicalCodeToEnglishName(logicalCode string) string {
	logicalCode = strings.Trim(logicalCode, "_ ")
	if logicalCode == "" {
		return ""
	}
	if logicalCode == "I_PSEC" {
		return "IPsec"
	}
	parts := strings.Split(logicalCode, "_")
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		words = append(words, englishTokenTitle(part))
	}
	return strings.Join(words, " ")
}

func englishTokenTitle(token string) string {
	switch token {
	case "ASSOC":
		return "Association"
	case "CONFIG":
		return "Configuration"
	case "CONN":
		return "Connection"
	case "CTRL":
		return "Control"
	case "FREQ":
		return "Frequency"
	case "INFO":
		return "Information"
	case "MGMT":
		return "Management"
	case "PARAM":
		return "Parameter"
	case "SW":
		return "Software"
	case "SYNC":
		return "Synchronization"
	case "BTS", "DNS", "EPC", "EUTRA", "FAP", "GERAN", "GPS", "GSM", "HALOD",
		"IP", "IRAT", "LTE", "LMT", "MAC", "MME", "MR", "NR", "NTP", "PHY",
		"PLMN", "PM", "RAN", "S1U", "SCTP", "SON", "TR069", "UTRA", "VRRP", "X2":
		return token
	case "IPSEC", "PSEC":
		return "IPsec"
	}
	lower := strings.ToLower(token)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// chapterCategory 把 chapter 转为 mml_commands.category 数字字符串。
// 与既有 seed/000152 的约定对齐（SA=3, SB=7, SC=6, SD=4 等业务字典分类）。
// 未对齐章节默认 '99' 表示"其他"。
func chapterCategory(chapter string) string {
	switch chapter {
	case "SA":
		return "3"
	case "SB":
		return "7"
	case "SC":
		return "6"
	case "SD":
		return "4"
	case "SE":
		return "8"
	case "SF", "SG", "SH", "SI", "SJ", "SK":
		return "1"
	case "SL", "SM":
		return "6"
	case "SN":
		return "6"
	case "SO":
		return "5"
	case "SP", "SQ":
		return "2"
	case "SR":
		return "3"
	}
	return "99"
}
