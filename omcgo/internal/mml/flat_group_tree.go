package mml

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================
// flat_group_tree.go — Task #4 扁平命令树（GET /api/v1/mml/group-tree?format=flat）
//
// 设计目标：一次性返回 18 个 SA-SR 章节 × 命令 × 参数路径(含约束)，前端无需再
// 调 /mml/commands/:id/sub-fields 二次加载。
//
// 响应格式：
//
//	{
//	  "groups": [
//	    {
//	      "code": "SA",
//	      "name": "设备信息参数管理",
//	      "commands": [
//	        {"id":"...", "name":"LST 设备基本信息",
//	         "object_path":["Device.DeviceInfo.UserLabel", ...]},        // LST: string[]
//	        {"id":"...", "name":"MOD 设备基本信息",
//	         "object_path":[{"path":"...","type":"string","max_length":64}, ...]}, // MOD: object[]
//	        {"id":"...", "name":"ADD FAP载波",
//	         "object_path":"Device.Services.FAPService.{i}."}            // ADD/RMV: string
//	      ]
//	    }, ...
//	  ]
//	}
//
// 不破坏现有 GroupTree 行为：原 GET /mml/group-tree（无 format / format=tree）保持
// 树形递归结构；仅在 format=flat 时走本文件 BuildFlatGroupTree。
// ============================================================

// FlatGroupTreeResponse 是 ?format=flat 的 HTTP 响应根对象。
type FlatGroupTreeResponse struct {
	Groups []FlatGroup `json:"groups"`
}

// FlatGroup 是 18 个 SA-SR 章节中的一个，下挂该章节的全部 commands。
//
// 一级分组严格扁平，无 children 嵌套；老 catalog（chapter_code 为空）的命令
// 不出现在响应中。
type FlatGroup struct {
	Code     string        `json:"code"`     // SA / SB / ... / SR
	Name     string        `json:"name"`     // 纯中文名（如"设备信息参数管理"）
	Commands []FlatCommand `json:"commands"`
}

// FlatCommand 是单条 MML 命令的扁平视图。
//
// ObjectPath 字段三态：
//   - LST: []string         所有 sub_field 关联的 standard_path
//   - MOD: []ModParamPath    带 type / 约束的对象数组
//   - ADD/RMV: string        目标对象路径（mml_command_groups.object_path_template）
type FlatCommand struct {
	ID         uuid.UUID   `json:"id"`
	Name       string      `json:"name"`
	ObjectPath interface{} `json:"object_path"`
}

// ModParamPath 是 MOD 命令 object_path 数组项；展开 standard_params 的约束。
//
// 字段映射来源（迁移 000058 schema）：
//   - Path        ← standard_params.standard_path
//   - Type        ← standard_params.data_type
//   - Min/Max     ← standard_params.min_value / max_value（BIGINT）
//
// MaxLength / Enum 当前 schema 未持久化，预留字段保持响应契约稳定，
// 待 standard_params 扩列后再填入（任务 #4 spec §约束字段映射）。
type ModParamPath struct {
	Path      string   `json:"path"`
	Type      string   `json:"type,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Min       *int64   `json:"min,omitempty"`
	Max       *int64   `json:"max,omitempty"`
	Enum      []string `json:"enum,omitempty"`
}

// FlatGroupTreeRepository 提供扁平命令树的查询能力。
type FlatGroupTreeRepository interface {
	BuildFlatTree(ctx context.Context) ([]FlatGroup, error)
}

// PgFlatGroupTreeRepository 是 PostgreSQL 实现。
//
// 单 SQL JOIN：mml_command_groups + mml_commands + mml_command_sub_fields +
// standard_params；Go 侧按 (chapter_code, command_id) 聚合。
type PgFlatGroupTreeRepository struct {
	pool *pgxpool.Pool
}

// NewPgFlatGroupTreeRepository 构造。
func NewPgFlatGroupTreeRepository(pool *pgxpool.Pool) *PgFlatGroupTreeRepository {
	return &PgFlatGroupTreeRepository{pool: pool}
}

var _ FlatGroupTreeRepository = (*PgFlatGroupTreeRepository)(nil)

// flatRow 单 SQL 扫描结果，一行 = (chapter, group, command, sub_field/standard_param) 笛卡尔积。
type flatRow struct {
	ChapterCode        string
	GroupDisplayOrder  int
	GroupObjectPath    *string // mml_command_groups.object_path_template
	CommandID          uuid.UUID
	OperationType      string
	CommandCode        string
	CommandTargetObj   *string
	CommandLogicalName []byte // logical_name_i18n JSONB
	CommandNameI18n    []byte // command_name_i18n JSONB
	SubFieldOrder      *int
	StandardPath       *string
	DataType           *string
	MinValue           *int64
	MaxValue           *int64
}

// BuildFlatTree 加载 18 SA-SR 章节的扁平命令视图。
func (r *PgFlatGroupTreeRepository) BuildFlatTree(ctx context.Context) ([]FlatGroup, error) {
	const sqlText = `
SELECT
    g.chapter_code,
    g.display_order                       AS group_display_order,
    g.object_path_template                AS group_object_path,
    c.id                                  AS command_id,
    c.operation_type,
    c.command_code,
    c.target_object,
    c.logical_name_i18n,
    c.command_name_i18n,
    csf.sort_order                        AS sub_field_order,
    sp.standard_path,
    sp.data_type,
    sp.min_value,
    sp.max_value
FROM mml_command_groups g
JOIN mml_commands c        ON c.group_id = g.id
LEFT JOIN mml_command_sub_fields csf
       ON csf.command_id = c.id
      AND csf.deprecated_at IS NULL
LEFT JOIN standard_params sp ON sp.id = csf.standard_path_id
WHERE g.chapter_code IS NOT NULL
  AND g.chapter_code <> ''
  AND g.source = 'standard'
  AND g.deprecated_at IS NULL
  AND g.deleted_at IS NULL
  AND c.deprecated_at IS NULL
  AND c.source = 'standard'
ORDER BY g.chapter_code,
         g.display_order,
         c.operation_type,
         c.command_code,
         csf.sort_order NULLS LAST
`

	rows, err := r.pool.Query(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("query flat group tree: %w", err)
	}
	defer rows.Close()

	// 章节 → 命令 顺序保留：order 切片 + map 双视图
	chapterOrder := make([]string, 0, 18)
	chapters := make(map[string]*flatChapterAcc)

	for rows.Next() {
		var row flatRow
		if err := rows.Scan(
			&row.ChapterCode,
			&row.GroupDisplayOrder,
			&row.GroupObjectPath,
			&row.CommandID,
			&row.OperationType,
			&row.CommandCode,
			&row.CommandTargetObj,
			&row.CommandLogicalName,
			&row.CommandNameI18n,
			&row.SubFieldOrder,
			&row.StandardPath,
			&row.DataType,
			&row.MinValue,
			&row.MaxValue,
		); err != nil {
			return nil, fmt.Errorf("scan flat row: %w", err)
		}

		ch, ok := chapters[row.ChapterCode]
		if !ok {
			ch = &flatChapterAcc{code: row.ChapterCode, cmds: make(map[uuid.UUID]*flatCmdAcc)}
			chapters[row.ChapterCode] = ch
			chapterOrder = append(chapterOrder, row.ChapterCode)
		}

		cmd, ok := ch.cmds[row.CommandID]
		if !ok {
			nameZh := pickI18nFromBytes(row.CommandLogicalName, "zh-CN")
			if nameZh == "" {
				// 兜底：command_name_i18n 通常是 "<动词> <对象>" 形式，去 verb 前缀后保留对象名
				nameZh = stripVerbPrefix(pickI18nFromBytes(row.CommandNameI18n, "zh-CN"))
			}
			if nameZh == "" {
				nameZh = strOrEmpty(row.CommandTargetObj)
			}
			cmd = &flatCmdAcc{
				id:           row.CommandID,
				op:           row.OperationType,
				commandCode:  row.CommandCode,
				nameZh:       nameZh,
				groupObjPath: strOrEmpty(row.GroupObjectPath),
				targetObject: strOrEmpty(row.CommandTargetObj),
				seenPath:     make(map[string]struct{}),
				seenMod:      make(map[string]struct{}),
			}
			ch.cmds[row.CommandID] = cmd
			ch.cmdOrder = append(ch.cmdOrder, row.CommandID)
		}

		// 累积 sub_field 路径（仅 LST / MOD 用）
		if row.StandardPath != nil && *row.StandardPath != "" {
			path := *row.StandardPath
			switch cmd.op {
			case "LST":
				if _, dup := cmd.seenPath[path]; !dup {
					cmd.seenPath[path] = struct{}{}
					cmd.lstPaths = append(cmd.lstPaths, path)
				}
			case "MOD":
				if _, dup := cmd.seenMod[path]; !dup {
					cmd.seenMod[path] = struct{}{}
					cmd.modPaths = append(cmd.modPaths, ModParamPath{
						Path: path,
						Type: strOrEmpty(row.DataType),
						Min:  row.MinValue,
						Max:  row.MaxValue,
					})
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate flat rows: %w", err)
	}

	// 章节 → FlatGroup（按 SA-SR 字典序）
	sort.SliceStable(chapterOrder, func(i, j int) bool {
		return chapterOrder[i] < chapterOrder[j]
	})

	out := make([]FlatGroup, 0, len(chapterOrder))
	for _, code := range chapterOrder {
		ch := chapters[code]
		fg := FlatGroup{
			Code:     code,
			Name:     resolveChapterName(code),
			Commands: make([]FlatCommand, 0, len(ch.cmdOrder)),
		}
		for _, cid := range ch.cmdOrder {
			c := ch.cmds[cid]
			fg.Commands = append(fg.Commands, FlatCommand{
				ID:         c.id,
				Name:       buildFlatCommandName(c.op, c.nameZh),
				ObjectPath: buildObjectPath(c),
			})
		}
		out = append(out, fg)
	}
	return out, nil
}

// resolveChapterName 把章节码翻译为纯中文名。优先取 chapterMetadata（与 v2.3
// spec 主表一致）；表外 code 退化返回 code 自身。
func resolveChapterName(code string) string {
	if meta, ok := chapterMetadata[code]; ok {
		return meta.NameZH
	}
	return code
}

// buildFlatCommandName 派生 "<OP> <对象中文名>"（如 "LST 设备基本信息"）。
//
// 不带翻译动词前缀（与 buildDisplayName 区分）—— Task #4 spec 要求保留 OP code，
// 让前端按 OP 决定 object_path 的解析格式（数组/对象数组/字符串）。
func buildFlatCommandName(op, nameZh string) string {
	if nameZh == "" {
		return op
	}
	return fmt.Sprintf("%s %s", op, nameZh)
}

// flatChapterAcc 是 BuildFlatTree 内部按章节聚合的累加器。
type flatChapterAcc struct {
	code     string
	cmdOrder []uuid.UUID
	cmds     map[uuid.UUID]*flatCmdAcc
}

// flatCmdAcc 是 BuildFlatTree 内部按命令聚合的累加器（同 command 多 sub_field 行合并）。
type flatCmdAcc struct {
	id           uuid.UUID
	op           string
	commandCode  string
	nameZh       string
	groupObjPath string
	targetObject string
	seenPath     map[string]struct{}
	seenMod      map[string]struct{}
	lstPaths     []string
	modPaths     []ModParamPath
}

// buildObjectPath 按 op 类型派生 object_path 字段值。
//
// 行为对照：
//   - LST: []string（全部 sub_field standard_path，去重保序）
//   - MOD: []ModParamPath（path + type + min/max；可能为空数组）
//   - ADD/RMV: string（取 group.object_path_template；空则退化 target_object）
//   - 其他 op：兜底返回 ""，避免 NULL 让前端反序列化困难
func buildObjectPath(c *flatCmdAcc) interface{} {
	switch c.op {
	case "LST":
		if c.lstPaths == nil {
			return []string{}
		}
		return c.lstPaths
	case "MOD":
		if c.modPaths == nil {
			return []ModParamPath{}
		}
		return c.modPaths
	case "ADD", "RMV":
		if c.groupObjPath != "" {
			return c.groupObjPath
		}
		return c.targetObject
	default:
		return ""
	}
}

// pickI18nFromBytes 解析 JSONB i18n 列并按 lang 取值（带 zh-CN/en-US fallback）。
func pickI18nFromBytes(b []byte, lang string) string {
	if len(b) == 0 {
		return ""
	}
	m := map[string]string{}
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	if v, ok := m[lang]; ok && v != "" {
		return v
	}
	if v, ok := m["zh-CN"]; ok && v != "" {
		return v
	}
	if v, ok := m["en-US"]; ok && v != "" {
		return v
	}
	return ""
}

// stripVerbPrefix 去掉 command_name_i18n 中文值的动词前缀。
//
// 老 catalog command_name_i18n 形如 "查询 内存使用"/"修改 NTP"/"添加 PLMN"/"删除 X"，
// 这里去掉首词得到对象名（"内存使用"/"NTP"/"PLMN"/"X"）；不匹配时原样返回。
func stripVerbPrefix(s string) string {
	for _, verb := range []string{"查询 ", "修改 ", "添加 ", "删除 "} {
		if strings.HasPrefix(s, verb) {
			return strings.TrimPrefix(s, verb)
		}
	}
	return s
}
