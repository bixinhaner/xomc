// gen_seed_sql 把 cmcc_tdlte_v23.json seed 派生命令树并输出 Goose seed SQL
// 到 stdout，由调用方重定向到 migrations/seed/000152_cmcc_tdlte_v23_mml_commands.sql。
//
// 使用：
//
//	cd omcgo
//	go run ./cmd/tools/gen_seed_sql/ > migrations/seed/000152_cmcc_tdlte_v23_mml_commands.sql
//
// 复用：
//   - mmlstandardloader.ParseSeed / DeriveCommandsFromSeed / CategorizePath
//
// 与运行期 seed_importer 字段语义保持一致；ON CONFLICT 保证迁移幂等。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
)

const (
	defaultSeedPath = "internal/config/parammodel/mmlstandardloader/seeds/cmcc_tdlte_v23.json"
	sourceTag       = "standard"
)

func main() {
	seedPath := flag.String("seed", defaultSeedPath, "JSON seed 文件路径（相对 omcgo/ 目录）")
	flag.Parse()

	if err := generate(*seedPath, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "gen_seed_sql failed:", err)
		os.Exit(1)
	}
}

func generate(seedPath string, w io.Writer) error {
	raw, err := os.ReadFile(seedPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("read seed file %s: %w", seedPath, err)
	}
	seed, err := mmlstandardloader.ParseSeed(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("parse seed: %w", err)
	}
	derived := mmlstandardloader.DeriveCommandsFromSeed(seed)

	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	versionCode := seed.VersionCode()
	versionName := seed.VersionName()

	// ---------- 头部注释 + Up ----------
	fmt.Fprintln(w, "-- +goose Up")
	fmt.Fprintln(w, "-- 中国移动 TD-LTE 南向数据模型 v2.3 — MML 命令树种子数据")
	fmt.Fprintln(w, "--")
	fmt.Fprintln(w, "-- 数据来源：omcgo/internal/config/parammodel/mmlstandardloader/seeds/cmcc_tdlte_v23.json")
	fmt.Fprintln(w, "-- 生成方式：go run ./cmd/tools/gen_seed_sql/ > migrations/seed/000152_cmcc_tdlte_v23_mml_commands.sql")
	fmt.Fprintf(w, "-- 数据规模：%d 个一级分组 / %d 条派生命令\n", len(seed.Groups), len(derived))
	fmt.Fprintf(w, "-- seed sha256：%s\n", hash)
	fmt.Fprintln(w, "--")
	fmt.Fprintln(w, "-- 涉及表：mml_param_versions / mml_command_groups / mml_commands /")
	fmt.Fprintln(w, "--          mml_command_sub_fields（standard_params 为只读引用）")
	fmt.Fprintln(w, "-- 幂等：所有 INSERT 使用 ON CONFLICT 保护，可重复执行。")
	fmt.Fprintln(w, "")

	// ---------- Step 1: mml_param_versions ----------
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintln(w, "-- Step 1: mml_param_versions — 版本锚点")
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintf(w, `INSERT INTO mml_param_versions (
    id, version_code, version_name, description,
    source, is_active, is_deprecated, content_hash
) VALUES (
    gen_random_uuid(), %s, %s, %s,
    %s, true, false, %s
)
ON CONFLICT (version_code) DO UPDATE
SET version_name  = EXCLUDED.version_name,
    description   = EXCLUDED.description,
    source        = EXCLUDED.source,
    is_active     = true,
    is_deprecated = false,
    content_hash  = COALESCE(EXCLUDED.content_hash, mml_param_versions.content_hash),
    updated_at    = NOW();
`,
		sqlStr(versionCode),
		sqlStr(versionName),
		sqlStr(fmt.Sprintf("由 cmcc_tdlte_v%s.json 派生（goose seed 000152）", seed.Version)),
		sqlStr(sourceTag),
		sqlStr(hash),
	)
	fmt.Fprintln(w, "")

	// ---------- Step 2: mml_command_groups ----------
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintf(w, "-- Step 2: mml_command_groups — %d 个一级分组（SA…SR）\n", len(seed.Groups))
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintln(w, `INSERT INTO mml_command_groups (
    id, group_code, group_name_zh, group_name_en,
    path, name_i18n, param_version,
    display_order, is_active,
    source, catalog_protected, chapter_code
) VALUES`)
	for i, g := range seed.Groups {
		nameJSON := jsonI18n(g.Name, g.Name)
		// spec §R-1：一级分组稳定标识 group_code=chapter:<SA-SR>；
		// ltree path 不允许冒号，path 形式 chapter_<X>。chapter_code 列仍保留
		// 裸 SA/SB/.../SR（机器排序键 + admin 视图）。
		groupCode := "chapter:" + g.Code
		pathLtree := "chapter_" + g.Code
		comma := ","
		if i == len(seed.Groups)-1 {
			comma = ""
		}
		fmt.Fprintf(w,
			"    (gen_random_uuid(), %s, %s, %s, %s::ltree, %s::jsonb, %s, %d, true, %s, true, %s)%s\n",
			sqlStr(groupCode), sqlStr(g.Name), sqlStr(g.Name),
			sqlStr(pathLtree), sqlStr(string(nameJSON)), sqlStr(versionCode),
			i,
			sqlStr(sourceTag), sqlStr(g.Code), comma,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (param_version, group_code) DO UPDATE
SET group_name_zh     = EXCLUDED.group_name_zh,
    group_name_en     = EXCLUDED.group_name_en,
    path              = EXCLUDED.path,
    name_i18n         = EXCLUDED.name_i18n,
    display_order     = EXCLUDED.display_order,
    is_active         = true,
    source            = EXCLUDED.source,
    catalog_protected = EXCLUDED.catalog_protected,
    chapter_code      = EXCLUDED.chapter_code,
    deprecated_at     = NULL,
    updated_at        = NOW();`)
	fmt.Fprintln(w, "")

	// ---------- Step 3: mml_commands ----------
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintf(w, "-- Step 3: mml_commands — %d 条派生命令\n", len(derived))
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintln(w, `INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method,
    operation_type, target_paths, target_object,
    group_id, command_name_i18n,
    logical_code, logical_name_i18n,
    source, catalog_protected, help_doc
) VALUES`)
	for i, c := range derived {
		paths := make([]string, 0, len(c.SubFields))
		for _, p := range c.SubFields {
			paths = append(paths, p.Path)
		}
		targetPathsJSON, _ := json.Marshal(paths)

		var targetObjectExpr string
		if c.TargetObject != "" {
			targetObjectExpr = sqlStr(c.TargetObject)
		} else {
			targetObjectExpr = "NULL"
		}

		category := mmlstandardloader.CategorizePath(c.ObjectPath)
		nameJSON := jsonI18n(c.NameZh, c.NameEn)
		logicalNameJSON := jsonI18n(c.LogicalNameZh, c.LogicalNameEn)
		// commands.group_id FK 走 chapter: 前缀的 group_code（spec §R-1）。
		groupIDExpr := fmt.Sprintf(
			"(SELECT id FROM mml_command_groups WHERE param_version=%s AND group_code=%s)",
			sqlStr(versionCode), sqlStr("chapter:"+c.GroupCode),
		)

		comma := ","
		if i == len(derived)-1 {
			comma = ""
		}
		fmt.Fprintf(w,
			"    (gen_random_uuid(), %s, %s, %s, %s, %s, %s, %s::jsonb, %s, %s, %s::jsonb, %s, %s::jsonb, %s, true, '')%s\n",
			sqlStr(c.NameZh), sqlStr(c.CommandCode), sqlStr(category),
			sqlStr(c.NameZh), sqlStr(c.RPCMethod),
			sqlStr(c.OperationType), sqlStr(string(targetPathsJSON)), targetObjectExpr,
			groupIDExpr, sqlStr(string(nameJSON)),
			sqlStr(c.LogicalCode), sqlStr(string(logicalNameJSON)),
			sqlStr(sourceTag),
			comma,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (command_code) DO UPDATE
SET command_name      = EXCLUDED.command_name,
    category          = EXCLUDED.category,
    description       = EXCLUDED.description,
    rpc_method        = EXCLUDED.rpc_method,
    operation_type    = EXCLUDED.operation_type,
    target_paths      = EXCLUDED.target_paths,
    target_object     = EXCLUDED.target_object,
    group_id          = EXCLUDED.group_id,
    command_name_i18n = EXCLUDED.command_name_i18n,
    logical_code      = EXCLUDED.logical_code,
    logical_name_i18n = EXCLUDED.logical_name_i18n,
    source            = EXCLUDED.source,
    catalog_protected = true,
    deprecated_at     = NULL,
    updated_at        = NOW();`)
	fmt.Fprintln(w, "")

	// ---------- Step 4: mml_command_sub_fields ----------
	fmt.Fprintln(w, "-- ============================================================")
	fmt.Fprintln(w, "-- Step 4: mml_command_sub_fields — 命令 ↔ standard_params 关联")
	fmt.Fprintln(w, "-- LST 写全部字段，MOD/ADD 写 RW 字段，RMV 不写 sub_field。")
	fmt.Fprintln(w, "-- 通过 standard_path 在 standard_params 中查 id；缺失 path 自然跳过。")
	fmt.Fprintln(w, "-- ============================================================")
	subFieldRows := 0
	for _, c := range derived {
		if !c.HasSubFields() || len(c.SubFields) == 0 {
			continue
		}
		mmlCodeSeen := make(map[string]int, len(c.SubFields))
		for sortOrder, p := range c.SubFields {
			mmlBase := lastSegmentForCode(p.Path)
			mmlCode := mmlBase
			if mmlCodeSeen[mmlBase] > 0 {
				mmlCode = fmt.Sprintf("%s_%d", mmlBase, mmlCodeSeen[mmlBase]+1)
			}
			mmlCodeSeen[mmlBase]++

			labelJSON := jsonI18n(p.Name, p.Name)
			access := subFieldAccessType(p.Access)
			isRequired := p.IsWritable()

			fmt.Fprintf(w, `INSERT INTO mml_command_sub_fields (
    id, command_id, standard_path_id, mml_code, label_i18n,
    default_selected, is_required, sort_order, access_type
)
SELECT gen_random_uuid(),
       (SELECT id FROM mml_commands WHERE command_code = %s),
       sp.id,
       %s, %s::jsonb,
       true, %s, %d, %s
FROM standard_params sp
WHERE sp.standard_path = %s
ON CONFLICT (command_id, standard_path_id) DO UPDATE
SET mml_code      = EXCLUDED.mml_code,
    label_i18n    = EXCLUDED.label_i18n,
    is_required   = EXCLUDED.is_required,
    sort_order    = EXCLUDED.sort_order,
    access_type   = EXCLUDED.access_type,
    deprecated_at = NULL,
    updated_at    = NOW();
`,
				sqlStr(c.CommandCode),
				sqlStr(mmlCode), sqlStr(string(labelJSON)),
				sqlBool(isRequired), sortOrder, sqlStr(access),
				sqlStr(p.Path),
			)
			subFieldRows++
		}
	}
	fmt.Fprintf(w, "-- mml_command_sub_fields 候选行数：%d（实际入库取决于 standard_params 命中数）\n", subFieldRows)
	fmt.Fprintln(w, "")

	// ---------- Down ----------
	fmt.Fprintln(w, "-- +goose Down")
	fmt.Fprintln(w, "-- 仅清理 standard 来源的命令树；standard_params 行为只读引用，不在此处删除。")
	fmt.Fprintln(w, "DELETE FROM mml_command_sub_fields WHERE command_id IN (")
	fmt.Fprintln(w, "    SELECT id FROM mml_commands WHERE source = 'standard'")
	fmt.Fprintln(w, ");")
	fmt.Fprintln(w, "DELETE FROM mml_commands WHERE source = 'standard';")
	fmt.Fprintln(w, "DELETE FROM mml_command_groups WHERE source = 'standard';")
	return nil
}

// ============================================================
// SQL 字面量 / 字段格式化 helpers（与 seed_importer 等价语义）
// ============================================================

// sqlStr 把 Go 字符串包装为 PostgreSQL 单引号字符串字面量；
// 内部 ' 双写为 ”。standard_conforming_strings=on 默认开启，无需额外转义 \。
func sqlStr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// sqlBool PostgreSQL 布尔字面量 true/false。
func sqlBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// jsonI18n {"zh": zh, "en": en} JSONB 字符串。与 seed_importer.jsonI18n 对齐。
func jsonI18n(zh, en string) []byte {
	m := map[string]string{"zh": zh, "en": en}
	b, _ := json.Marshal(m)
	return b
}

// subFieldAccessType seed access → mml_command_sub_fields.access_type CHECK 词表。
// 与 seed_importer.subFieldAccessType 对齐：RW → "RW"，其它 → "RO"。
func subFieldAccessType(a string) string {
	if a == mmlstandardloader.AccessRW {
		return "RW"
	}
	return "RO"
}

// lastSegmentForCode 取 path 末段（去 {i}）作为 mml_code。
// 与 seed_importer.lastSegmentForCode 对齐。
//
//	"Device.DeviceInfo.UserLabel" → "UserLabel"
//	"Device.X.{i}.Y"              → "Y"
func lastSegmentForCode(p string) string {
	clean := strings.ReplaceAll(p, ".{i}", "")
	clean = strings.TrimSuffix(clean, ".")
	if clean == "" {
		return clean
	}
	idx := strings.LastIndex(clean, ".")
	if idx == -1 {
		return clean
	}
	return clean[idx+1:]
}
