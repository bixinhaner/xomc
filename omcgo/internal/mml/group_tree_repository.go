package mml

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================
// group_tree_repository.go — MML 命令树查询（spec v2.3 §R-1 chapter 顶层）
//
// 设计依据：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md
//
// 单 SQL JOIN 抓 mml_command_groups + mml_commands。BuildTree 仅返 group_code
// 带 "chapter:" 前缀的 18 个章节行（SA-SR）作顶层；子分组通过 LTREE path 深度
// 自然嵌套，命令通过 group_id 挂在所属节点的 commands[] 上。
//
// 历史 v1 视图（ltree path 自然层级 + wrapByChapter 合成章节层 + InferFamily 推断）
// 已下线，仅保留 chapter 顶层 + DB 真实嵌套。
// ============================================================

// GroupTreeNode 是命令树的一个节点（group + 该 group 下的 commands + 子 groups）。
type GroupTreeNode struct {
	ID           uuid.UUID         `json:"id"`
	GroupCode    string            `json:"code"`
	Name         string            `json:"name"`      // 当前 lang 派生
	NameI18n     map[string]string `json:"name_i18n"` // 完整 i18n
	Path         string            `json:"path"`      // LTREE path 字符串形式
	DisplayOrder int               `json:"display_order"`
	// ChapterCode：CMCC TD-LTE v2.3 章节码（SA/SB/SC/.../SR）。顶层 chapter 行的
	// group_code 是 "chapter:SA"；ChapterCode 列冗余持久化，便于子节点跨章节排序。
	ChapterCode      string             `json:"chapter_code,omitempty"`
	Source           string             `json:"source"`
	CatalogProtected bool               `json:"catalog_protected"`
	Commands         []GroupTreeCommand `json:"commands"`
	Children         []GroupTreeNode    `json:"children"`
}

// GroupTreeCommand 是树中一条命令叶子的精简视图（不含 sub_fields，sub_fields 通过
// `GET /mml/commands/:id/sub-fields` 单独懒加载）。
type GroupTreeCommand struct {
	ID               uuid.UUID         `json:"id"`
	CommandCode      string            `json:"command_code"`
	LogicalCode      string            `json:"logical_code"`
	LogicalName      string            `json:"logical_name"` // lang 派生
	LogicalNameI18n  map[string]string `json:"logical_name_i18n"`
	OperationType    string            `json:"operation_type"`
	DisplayName      string            `json:"display_name"` // "设备信息(LST DEVICE_INFO)"
	RPCMethod        string            `json:"rpc_method"`
	RequireConfirm   bool              `json:"require_confirm"`
	TargetObject     string            `json:"target_object,omitempty"`
	Source           string            `json:"source"`
	CatalogProtected bool              `json:"catalog_protected"`
	// InstanceRangeMeta 是 spec §R-4.1.1 每层 {i} 占位符的取值范围 metadata，
	// 由 catalog Loader 写入 mml_commands.instance_range_meta JSONB 列；
	// 前端 LST/MOD/ADD/RMV 操作面板用此校验 InstancePicker 输入。
	InstanceRangeMeta json.RawMessage `json:"instance_range_meta,omitempty"`

	// T-0172 catalog 按产品过滤的标注字段（仅 BuildGroupTree 调用方传入
	// product_class 时填充；否则全部 omit，老调用者无感）。
	// TotalPathCount: 该命令操作的总 path 数（LST/MOD = len(target_paths);
	//                 ADD/RMV = 1）。
	// UnsupportedPaths: 当前产品不支持的具体 path 列表；前端用作 tooltip / banner。
	// ProductResolved: 入参 product_class 是否成功路由到 product；false=孤儿。
	TotalPathCount   *int     `json:"total_path_count,omitempty"`
	UnsupportedPaths []string `json:"unsupported_paths,omitempty"`
	ProductResolved  *bool    `json:"product_resolved,omitempty"`
	// rawTargetPaths 是 mml_commands.target_paths JSONB raw bytes，仅在
	// repository → service 内部流转用于过滤；JSON 序列化时排除（- tag）。
	rawTargetPaths []byte `json:"-"`
}

// TargetPathsRaw 返回未导出的 JSONB raw bytes（供 service 层 AnnotateCommand 用）。
func (c *GroupTreeCommand) TargetPathsRaw() []byte { return c.rawTargetPaths }

// SetTargetPathsRaw 由 repository 调用时填入。导出 setter 让 repository（同 pkg）
// 在 BuildTree 时塞值；外部 pkg 不需要。
func (c *GroupTreeCommand) SetTargetPathsRaw(raw []byte) { c.rawTargetPaths = raw }

// GroupTreeRepository 提供命令树查询能力。
type GroupTreeRepository interface {
	// BuildTree 加载完整命令树。
	// - rootCode 为空时返回所有顶层 chapter 节点（SA-SR）顺序数组，每个含子树
	// - rootCode 非空时只返该 group 及其后代
	// - lang 选 zh-CN / en-US（默认 zh-CN）派生 name / logical_name 顶级字段
	BuildTree(ctx context.Context, rootCode, lang string) ([]GroupTreeNode, error)
}

// PgGroupTreeRepository PostgreSQL 实现。
type PgGroupTreeRepository struct {
	pool *pgxpool.Pool
}

// NewPgGroupTreeRepository 构造函数。
func NewPgGroupTreeRepository(pool *pgxpool.Pool) *PgGroupTreeRepository {
	return &PgGroupTreeRepository{pool: pool}
}

var _ GroupTreeRepository = (*PgGroupTreeRepository)(nil)

// BuildTree 单 SQL JOIN 抓 groups + commands，Go 侧按 path 分层组装。
//
// SQL 侧 WHERE 已过滤为 "chapter:" 前缀的章节行 + 子树命令；Go 侧仅做层级组装
// 与排序，不再做章节合成或 family 推断。
func (r *PgGroupTreeRepository) BuildTree(ctx context.Context, rootCode, lang string) ([]GroupTreeNode, error) {
	if lang == "" {
		lang = "zh-CN"
	}

	rows, err := r.queryGroupsAndCommands(ctx, rootCode)
	if err != nil {
		return nil, fmt.Errorf("query groups+commands: %w", err)
	}

	groupByID := make(map[uuid.UUID]*GroupTreeNode, len(rows))
	pathToGroupID := make(map[string]uuid.UUID, len(rows))
	commandSeen := make(map[uuid.UUID]struct{}, len(rows))

	for _, row := range rows {
		node, ok := groupByID[row.GroupID]
		if !ok {
			node = &GroupTreeNode{
				ID:               row.GroupID,
				GroupCode:        row.GroupCode,
				NameI18n:         row.GroupNameI18n,
				Name:             pickI18n(row.GroupNameI18n, lang, row.GroupNameZh, strOrEmpty(row.GroupNameEn), row.GroupCode),
				Path:             row.GroupPath,
				DisplayOrder:     row.GroupDisplayOrder,
				ChapterCode:      row.GroupChapterCode,
				Source:           row.GroupSource,
				CatalogProtected: row.GroupCatalogProtected,
				Commands:         []GroupTreeCommand{},
				Children:         []GroupTreeNode{},
			}
			groupByID[row.GroupID] = node
			pathToGroupID[row.GroupPath] = row.GroupID
		}

		if row.CommandID != nil {
			if _, dup := commandSeen[*row.CommandID]; !dup {
				cmd := GroupTreeCommand{
					ID:                *row.CommandID,
					CommandCode:       strOrEmpty(row.CommandCode),
					LogicalCode:       strOrEmpty(row.LogicalCode),
					LogicalNameI18n:   row.CommandLogicalNameI18n,
					LogicalName:       pickI18n(row.CommandLogicalNameI18n, lang, "", "", strOrEmpty(row.LogicalCode)),
					OperationType:     strOrEmpty(row.OperationType),
					RPCMethod:         strOrEmpty(row.RPCMethod),
					RequireConfirm:    row.RequireConfirm,
					TargetObject:      strOrEmpty(row.TargetObject),
					Source:            strOrEmpty(row.CommandSource),
					CatalogProtected:  row.CommandCatalogProtected,
					InstanceRangeMeta: rawJSONOrNil(row.CommandInstanceRangeMeta),
				}
				cmd.DisplayName = buildDisplayName(cmd.LogicalName, cmd.OperationType, cmd.LogicalCode, lang)
				// T-0172：把 target_paths raw bytes 塞入未导出字段，供 service 层
				// AnnotateCommand 在 product_class 过滤路径上读取。
				if len(row.CommandTargetPaths) > 0 {
					cmd.SetTargetPathsRaw(row.CommandTargetPaths)
				}
				node.Commands = append(node.Commands, cmd)
				commandSeen[*row.CommandID] = struct{}{}
			}
		}
	}

	nodes := assembleHierarchy(groupByID, pathToGroupID)
	sortNodesByDisplayOrder(nodes)
	return nodes, nil
}

// ============================================================
// SQL 查询
// ============================================================

// queryGroupsAndCommands 单 SQL JOIN 抓 groups + commands。
// rootCode 为空时拉所有 chapter 子树；非空时按 LTREE @> 拉指定子树。
//
// 始终通过 group_code LIKE 'chapter:%' 过滤为 chapter 顶层视图（spec v2.3 §R-1）。
func (r *PgGroupTreeRepository) queryGroupsAndCommands(ctx context.Context, rootCode string) ([]groupTreeRow, error) {
	// LEFT JOIN：父容器 group（无 commands）扫出的 c.* 列全为 NULL。Scan 到
	// 非指针 bool（RequireConfirm / CommandCatalogProtected）会报
	// "cannot scan NULL into *bool"。用 COALESCE 在 SQL 侧兜底 false。
	// COALESCE(g.chapter_code, '') 把 NULL 兜底为空串。
	const baseSQL = `
SELECT
    g.id, g.group_code, g.group_name_zh, g.group_name_en, g.name_i18n,
    COALESCE(g.path::text, '') AS path_text, g.display_order,
    COALESCE(g.chapter_code, '') AS chapter_code,
    g.source, g.catalog_protected,
    c.id, c.command_code, c.logical_code, c.logical_name_i18n,
    c.operation_type, c.rpc_method,
    COALESCE(c.require_confirm, false) AS require_confirm,
    c.target_object,
    c.source AS cmd_source,
    COALESCE(c.catalog_protected, false) AS cmd_catalog_protected,
    -- §R-4.1.1：每层 {i} 占位符的取值范围 metadata，无 metadata → 兜底 '[]'
    COALESCE(c.instance_range_meta, '[]'::jsonb) AS instance_range_meta,
    -- T-0172：操作的标准路径列表，无值兜底 '[]'
    COALESCE(c.target_paths, '[]'::jsonb) AS target_paths
FROM mml_command_groups g
LEFT JOIN mml_commands c ON c.group_id = g.id
WHERE g.path IS NOT NULL
  AND g.group_code LIKE 'chapter:%%'
%s
ORDER BY g.path, g.display_order, c.operation_type, c.logical_code, c.command_code`

	var whereParts []string
	var args []any
	if rootCode != "" {
		// LTREE @> $1 匹配 path 等于 rootCode 或以 rootCode 为前缀
		args = append(args, rootCode)
		whereParts = append(whereParts, "AND (g.path <@ $1::ltree OR g.path ~ ($1::text || '.*')::lquery)")
	}

	whereClause := strings.Join(whereParts, " ")
	sqlText := fmt.Sprintf(baseSQL, whereClause)
	dbRows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	var out []groupTreeRow
	for dbRows.Next() {
		var row groupTreeRow
		var nameI18nBytes, cmdLogicalNameI18nBytes []byte
		if err := dbRows.Scan(
			&row.GroupID, &row.GroupCode, &row.GroupNameZh, &row.GroupNameEn, &nameI18nBytes,
			&row.GroupPath, &row.GroupDisplayOrder, &row.GroupChapterCode,
			&row.GroupSource, &row.GroupCatalogProtected,
			&row.CommandID, &row.CommandCode, &row.LogicalCode, &cmdLogicalNameI18nBytes,
			&row.OperationType, &row.RPCMethod, &row.RequireConfirm, &row.TargetObject,
			&row.CommandSource, &row.CommandCatalogProtected,
			&row.CommandInstanceRangeMeta,
			&row.CommandTargetPaths,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		row.GroupNameI18n = parseI18nJSON(nameI18nBytes)
		row.CommandLogicalNameI18n = parseI18nJSON(cmdLogicalNameI18nBytes)
		out = append(out, row)
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return out, nil
}

// groupTreeRow 是单 SQL 的扫描结果（一行 = (group, command) 笛卡尔行；
// command 列全 NULL 表示该 group 无任何命令）。
type groupTreeRow struct {
	GroupID           uuid.UUID
	GroupCode         string
	GroupNameZh       string
	GroupNameEn       *string
	GroupNameI18n     map[string]string
	GroupPath         string
	GroupDisplayOrder int
	// 章节码（CMCC TD-LTE v2.3 SA/SB/SC/.../SR）；SQL COALESCE 兜底空字串
	GroupChapterCode      string
	GroupSource           string
	GroupCatalogProtected bool

	CommandID               *uuid.UUID
	CommandCode             *string
	LogicalCode             *string
	CommandLogicalNameI18n  map[string]string
	OperationType           *string
	RPCMethod               *string
	RequireConfirm          bool
	TargetObject            *string
	CommandSource           *string
	CommandCatalogProtected bool
	// §R-4.1.1：每层 {i} 占位符取值范围 metadata（JSONB raw bytes）。
	CommandInstanceRangeMeta []byte
	// T-0172：命令操作的标准路径列表（LST/MOD），用于 product_class 过滤。
	// ADD/RMV 始终为空数组（这两种用 TargetObject）。JSONB raw bytes。
	CommandTargetPaths []byte
}

// ============================================================
// 树形组装（path 长度分层）
// ============================================================

// assembleHierarchy 把扁平的 groupByID 按 LTREE path 深度组装成嵌套树。
// 顶级（depth=1）作为返回切片；非顶级 group 挂到父 group.Children。
//
// 算法关键：**深度 DESC**（先深后浅）—— 在 append 子节点到父节点之前，子节点的
// Children 必须已经被它更深的后代填充完毕。如果先处理浅层（深度 ASC），父节点会
// 在子节点 Children 仍为空时拷贝子节点，后续修改子节点对拷贝不可见 → bug。
//
// 排序约束：同层按 (display_order, group_code) 字典序稳定。
func assembleHierarchy(groupByID map[uuid.UUID]*GroupTreeNode, pathToGroupID map[string]uuid.UUID) []GroupTreeNode {
	allPaths := make([]string, 0, len(groupByID))
	for p := range pathToGroupID {
		allPaths = append(allPaths, p)
	}
	// 深度 DESC（先深后浅）：保证子先于父被处理
	sortSlice(allPaths, func(a, b string) bool {
		da := strings.Count(a, ".")
		db := strings.Count(b, ".")
		if da != db {
			return da > db
		}
		return a < b
	})

	for _, n := range groupByID {
		sortCommandsByLogicalCode(n.Commands)
	}

	for _, p := range allPaths {
		gid, ok := pathToGroupID[p]
		if !ok {
			continue
		}
		child := groupByID[gid]
		if child == nil {
			continue
		}
		sortNodesByDisplayOrder(child.Children)

		parentPath := parentLTreePath(p)
		if parentPath == "" {
			continue
		}
		parentID, ok := pathToGroupID[parentPath]
		if !ok {
			continue
		}
		parent := groupByID[parentID]
		if parent == nil {
			continue
		}
		parent.Children = append(parent.Children, *child)
	}

	var roots []GroupTreeNode
	for p, gid := range pathToGroupID {
		parentPath := parentLTreePath(p)
		if parentPath == "" {
			roots = append(roots, *groupByID[gid])
			continue
		}
		if _, ok := pathToGroupID[parentPath]; !ok {
			roots = append(roots, *groupByID[gid])
		}
	}
	sortNodesByDisplayOrder(roots)
	return roots
}

// parentLTreePath 返回 LTREE 字符串 path 的父节点；顶级返回 ""。
//
//	"A" → ""
//	"A.B" → "A"
//	"A.B.C" → "A.B"
func parentLTreePath(p string) string {
	idx := strings.LastIndex(p, ".")
	if idx < 0 {
		return ""
	}
	return p[:idx]
}

// sortByDepthThenLex 按 LTREE 深度升序（点数）后字典序排序。
// 注：assembleHierarchy 现用反向（深度 DESC）；本函数保留供测试 / 未来其他需求使用。
func sortByDepthThenLex(paths []string) {
	sortSlice(paths, func(a, b string) bool {
		da := strings.Count(a, ".")
		db := strings.Count(b, ".")
		if da != db {
			return da < db
		}
		return a < b
	})
}

// sortNodesByDisplayOrder 同层节点排序。
// 主键 ChapterCode（SA/SB/SC.../SR；空串排末位），副键 DisplayOrder，再副键 GroupCode。
func sortNodesByDisplayOrder(nodes []GroupTreeNode) {
	sortSlice(nodes, func(a, b GroupTreeNode) bool {
		ac, bc := chapterSortKey(a.ChapterCode), chapterSortKey(b.ChapterCode)
		if ac != bc {
			return ac < bc
		}
		if a.DisplayOrder != b.DisplayOrder {
			return a.DisplayOrder < b.DisplayOrder
		}
		return a.GroupCode < b.GroupCode
	})
}

// chapterSortKey 把章节码归一化为可排序字符串。
// 空字符串映射为高位 sentinel "~"（ASCII 126），确保排末位。
func chapterSortKey(chapter string) string {
	if chapter == "" {
		return "~~~~~"
	}
	return chapter
}

func sortCommandsByLogicalCode(cmds []GroupTreeCommand) {
	sortSlice(cmds, func(a, b GroupTreeCommand) bool {
		if a.LogicalCode != b.LogicalCode {
			return a.LogicalCode < b.LogicalCode
		}
		return a.OperationType < b.OperationType
	})
}

// sortSlice generic helper using sort.SliceStable（避免 generics import 复杂度）。
// 用 Go 1.21+ 范型实现保证类型安全。
func sortSlice[T any](s []T, less func(a, b T) bool) {
	stableSort(s, less)
}

// stableSort 内置稳定排序（简单插入排序，适用于树节点数量 <= 1000 的场景；
// 真实 catalog 数据 group 数量 ~30，commands ~1000，性能 O(n²) 可接受 O(n^2)）。
func stableSort[T any](s []T, less func(a, b T) bool) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && less(s[j], s[j-1]) {
			s[j-1], s[j] = s[j], s[j-1]
			j--
		}
	}
}

// ============================================================
// helpers
// ============================================================

// pickI18n 按 lang 选 i18n 值；未命中时按 fallback 顺序回退。
func pickI18n(m map[string]string, lang, fallbackZh, fallbackEn, fallbackCode string) string {
	if v, ok := m[lang]; ok && v != "" {
		return v
	}
	// seed/000152 注入的 i18n key 是 'zh' / 'en'（短码），而 API/前端约定用 'zh-CN' / 'en-US'。
	// 两套兼容：lang 命中失败时短码 + 长码各回落一次，避免按命令中文名搜索/展示全空白。
	if v, ok := m["zh-CN"]; ok && v != "" {
		return v
	}
	if v, ok := m["zh"]; ok && v != "" {
		return v
	}
	if v, ok := m["en-US"]; ok && v != "" {
		return v
	}
	if v, ok := m["en"]; ok && v != "" {
		return v
	}
	if fallbackZh != "" {
		return fallbackZh
	}
	if fallbackEn != "" {
		return fallbackEn
	}
	return fallbackCode
}

// rawJSONOrNil 把 JSONB 字节切片转成 json.RawMessage，**空数组 / null / 空切片**返回
// nil，使外层结构 `omitempty` 标记能彻底隐藏该字段。
func rawJSONOrNil(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" || s == "[]" {
		return nil
	}
	return json.RawMessage(b)
}

func parseI18nJSON(b []byte) map[string]string {
	if len(b) == 0 || string(b) == "null" {
		return map[string]string{}
	}
	m := map[string]string{}
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]string{}
	}
	return m
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// verbLabel 把 op 翻译为业务化中文/英文动词。lang 走 i18n fallback：
//   - zh-CN → 查询/修改/添加/删除
//   - en-US → Query/Modify/Add/Delete
//   - 其他 / 未识别 op → 原 op 字串
func verbLabel(op, lang string) string {
	switch lang {
	case "en-US":
		switch op {
		case "LST":
			return "Query"
		case "MOD":
			return "Modify"
		case "ADD":
			return "Add"
		case "RMV":
			return "Delete"
		}
	default:
		switch op {
		case "LST":
			return "查询"
		case "MOD":
			return "修改"
		case "ADD":
			return "添加"
		case "RMV":
			return "删除"
		}
	}
	return op
}

// buildDisplayName 派生命令树叶子显示名（业务化命名）：
//
//	"查询 设备信息"  /  "Query Device Info"
func buildDisplayName(logicalName, op, logicalCode, lang string) string {
	if logicalName == "" {
		logicalName = logicalCode
	}
	return fmt.Sprintf("%s %s", verbLabel(op, lang), logicalName)
}
