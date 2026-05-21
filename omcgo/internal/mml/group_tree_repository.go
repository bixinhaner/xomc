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
// group_tree_repository.go — T-0123-P1 命令树查询
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.2.1
//
// 单 SQL JOIN 抓 mml_param_groups + mml_commands；Go 侧按 LTREE path 长度分层组装树。
//
// 老 OMC 实测命令树（playwright BSC Configuration）：
//   MML List
//   ├── BSC Configuration          ← 一级 group (path=BSC_CONFIGURATION)
//   │   ├── Basic Info             ← 二级 group (path=BSC_CONFIGURATION.BASIC_INFO)
//   │   │   ├── Device info(LST DEVICE_INFO)   ← 命令叶子
//   │   │   └── Device info(MOD DEVICE_INFO)
//   │   ├── BTS
//   │   │   ├── BTS Info(LST BTS_INFO) ...
//   ...
// ============================================================

// GroupTreeNode 是命令树的一个节点（group + 该 group 下的 commands + 子 groups）。
type GroupTreeNode struct {
	ID           uuid.UUID         `json:"id"`
	GroupCode    string            `json:"code"`
	Name         string            `json:"name"`      // 当前 lang 派生
	NameI18n     map[string]string `json:"name_i18n"` // 完整 i18n
	Path         string            `json:"path"`      // LTREE path 字符串形式
	DisplayOrder int               `json:"display_order"`
	// ChapterCode：CMCC TD-LTE v2.3 章节码（SA/SB/SC/.../SR）。空字符串表示老 catalog
	// 行不带章节信息。前端不渲染章节为节点，但用作主排序键（让对象级 group 按
	// SA→SB→SC 顺序排列，避免跨章节顺序错乱）。详见 plan §6.6 "按 SA-SR 顺序排列"。
	ChapterCode string `json:"chapter_code,omitempty"`
	// FamilyCode：v2.4 D32+D38 path-prefix-family 聚合键。空字符串表示该 group 自成
	// family。前端按此聚合：相同 FamilyCode 的多个 group 在 CommandTree 顶层折叠为
	// 一个"家族节点"。映射规则见 §15.4 + omcgo/internal/mml/family.go InferFamily。
	FamilyCode string `json:"family_code,omitempty"`
	// FamilyNameZh：family 在 UI 上展示的中文名（如"设备信息"/"告警实例"）。
	// 与 FamilyCode 同步出现，空字符串表示 fallback 到 Name。
	FamilyNameZh     string             `json:"family_name_zh,omitempty"`
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
}

// GroupTreeRepository 提供命令树查询能力。
type GroupTreeRepository interface {
	// BuildTree 加载完整命令树。
	// - rootCode 为空时返回所有 path 深度 = 1 的根 group 顺序数组（每个根含子树）
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
func (r *PgGroupTreeRepository) BuildTree(ctx context.Context, rootCode, lang string) ([]GroupTreeNode, error) {
	if lang == "" {
		lang = "zh-CN"
	}

	rows, err := r.queryGroupsAndCommands(ctx, rootCode)
	if err != nil {
		return nil, fmt.Errorf("query groups+commands: %w", err)
	}

	// 按 group_id 聚合行
	groupByID := make(map[uuid.UUID]*GroupTreeNode, len(rows))
	pathToGroupID := make(map[string]uuid.UUID, len(rows))
	commandSeen := make(map[uuid.UUID]struct{}, len(rows))

	for _, row := range rows {
		// group 不在缓存则创建
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
				// v2.4 P3：family 直接从 DB 读（migration 000141 持久化）。
				// 空字符串语义 = "self-family"，attachFamily 会作 fallback 尝试。
				FamilyCode:       row.GroupFamilyCode,
				FamilyNameZh:     row.GroupFamilyNameZh,
				Source:           row.GroupSource,
				CatalogProtected: row.GroupCatalogProtected,
				Commands:         []GroupTreeCommand{},
				Children:         []GroupTreeNode{},
			}
			groupByID[row.GroupID] = node
			pathToGroupID[row.GroupPath] = row.GroupID
		}

		// command 若存在则填入 node.Commands（去重）
		if row.CommandID != nil {
			if _, dup := commandSeen[*row.CommandID]; !dup {
				cmd := GroupTreeCommand{
					ID:               *row.CommandID,
					CommandCode:      strOrEmpty(row.CommandCode),
					LogicalCode:      strOrEmpty(row.LogicalCode),
					LogicalNameI18n:  row.CommandLogicalNameI18n,
					LogicalName:      pickI18n(row.CommandLogicalNameI18n, lang, "", "", strOrEmpty(row.LogicalCode)),
					OperationType:    strOrEmpty(row.OperationType),
					RPCMethod:        strOrEmpty(row.RPCMethod),
					RequireConfirm:   row.RequireConfirm,
					TargetObject:     strOrEmpty(row.TargetObject),
					Source:           strOrEmpty(row.CommandSource),
					CatalogProtected: row.CommandCatalogProtected,
				}
				cmd.DisplayName = buildDisplayName(cmd.LogicalName, cmd.OperationType, cmd.LogicalCode, lang)
				node.Commands = append(node.Commands, cmd)
				commandSeen[*row.CommandID] = struct{}{}
			}
		}
	}

	// 按 path 深度组装父子关系
	nodes := assembleHierarchy(groupByID, pathToGroupID)
	// v2.4 P3：family 已通过 migration 000141 + catalogloader.upsertGroup 持久化到 DB，
	// 上面 SQL 已直接读出。这里 attachFamily 退化为防御性 fallback —— 仅当 DB 返回空
	// （历史/损坏数据 / 迁移未跑）时调用 InferFamily 兜底，避免 UI 缺少 family 节点。
	// 详见 docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §15.6 + §15.8
	attachFamily(nodes)
	return nodes, nil
}

// attachFamily 递归遍历所有节点，**防御性 fallback**：仅当 DB 返回空 FamilyCode 时
// 才调用 InferFamily 推断并填充。正常情况下 family_code / family_name_zh 已由
// catalogloader.upsertGroup 写入 DB（migration 000141）。
//
// 设计：v2.4 P3 之前是无条件覆盖（不入库时的唯一推断时机），P3 之后入库为主，
// 这里只兜底历史 / 异常数据，避免 UI 缺少 family 节点。
func attachFamily(nodes []GroupTreeNode) {
	for i := range nodes {
		if nodes[i].FamilyCode == "" {
			nodes[i].FamilyCode, nodes[i].FamilyNameZh = InferFamily(nodes[i].GroupCode)
		}
		attachFamily(nodes[i].Children)
	}
}

// ============================================================
// SQL 查询
// ============================================================

// queryGroupsAndCommands 单 SQL JOIN 抓 groups + commands。
// rootCode 为空时拉全表；非空时按 LTREE @> 拉子树。
func (r *PgGroupTreeRepository) queryGroupsAndCommands(ctx context.Context, rootCode string) ([]groupTreeRow, error) {
	// LEFT JOIN：父容器 group（无 commands）扫出的 c.* 列全为 NULL。Scan 到
	// 非指针 bool（RequireConfirm / CommandCatalogProtected）会报
	// "cannot scan NULL into *bool"。用 COALESCE 在 SQL 侧兜底 false，避免
	// 改 Go 层 Scan 字段为 *bool 引发的连锁改造。
	// COALESCE(g.chapter_code, '') 把 NULL 兜底为空串，方便 Scan 进 string；
	// 老 catalog 行没有章节码，前端渲染时按 "" 视为"未分章"统一末位排序。
	// v2.4 P3：family_code / family_name_zh 持久化在 mml_param_groups（migration 000141）。
	// SELECT 加 COALESCE 兜底，处理迁移未跑或历史 NULL 行（理论上 NOT NULL DEFAULT ''，
	// 但 COALESCE 双保险）。BuildTree post-process 的 attachFamily 仍对空值做 fallback。
	const baseSQL = `
SELECT
    g.id, g.group_code, g.group_name_zh, g.group_name_en, g.name_i18n,
    COALESCE(g.path::text, '') AS path_text, g.display_order,
    COALESCE(g.chapter_code, '') AS chapter_code,
    COALESCE(g.family_code, '') AS family_code,
    COALESCE(g.family_name_zh, '') AS family_name_zh,
    g.source, g.catalog_protected,
    c.id, c.command_code, c.logical_code, c.logical_name_i18n,
    c.operation_type, c.rpc_method,
    COALESCE(c.require_confirm, false) AS require_confirm,
    c.target_object,
    c.source AS cmd_source,
    COALESCE(c.catalog_protected, false) AS cmd_catalog_protected
FROM mml_param_groups g
LEFT JOIN mml_commands c ON c.group_id = g.id
WHERE g.path IS NOT NULL
%s
ORDER BY g.path, g.display_order, c.operation_type, c.logical_code, c.command_code`

	var whereClause string
	var args []any
	if rootCode != "" {
		// LTREE @> $1 匹配 path 等于 rootCode 或以 rootCode 为前缀
		whereClause = "AND (g.path <@ $1::ltree OR g.path ~ ($1::text || '.*')::lquery)"
		args = []any{rootCode}
	}

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
			&row.GroupFamilyCode, &row.GroupFamilyNameZh,
			&row.GroupSource, &row.GroupCatalogProtected,
			&row.CommandID, &row.CommandCode, &row.LogicalCode, &cmdLogicalNameI18nBytes,
			&row.OperationType, &row.RPCMethod, &row.RequireConfirm, &row.TargetObject,
			&row.CommandSource, &row.CommandCatalogProtected,
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
	// 章节码（CMCC TD-LTE v2.3 SA/SB/SC/.../SR）；老 catalog NULL → SQL COALESCE 兜底空字串
	GroupChapterCode string
	// v2.4 P3：family 持久化字段（migration 000141）。COALESCE 兜底空字串 → 上层 attachFamily
	// 对空值做 InferFamily fallback。
	GroupFamilyCode       string
	GroupFamilyNameZh     string
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
			return da > db // 深度大的先
		}
		return a < b
	})

	// 排序每个节点的 commands（一次性）
	for _, n := range groupByID {
		sortCommandsByLogicalCode(n.Commands)
	}

	// 深度 DESC 处理：先 sort 自身 Children，再 append 到父节点
	for _, p := range allPaths {
		gid, ok := pathToGroupID[p]
		if !ok {
			continue
		}
		child := groupByID[gid]
		if child == nil {
			continue
		}
		// 此时 child.Children 已被所有更深路径填充完毕（深度 DESC 处理顺序）
		sortNodesByDisplayOrder(child.Children)

		parentPath := parentLTreePath(p)
		if parentPath == "" {
			continue // 顶级稍后收集
		}
		parentID, ok := pathToGroupID[parentPath]
		if !ok {
			continue // 孤儿（partial subtree）— 稍后作顶级收集
		}
		parent := groupByID[parentID]
		if parent == nil {
			continue
		}
		parent.Children = append(parent.Children, *child)
	}

	// 收集顶级 + 孤儿（父 path 不在结果集的）作 roots
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
// 这让对象级 group 跨章节按"SA章节内所有对象 → SB章节内所有对象 → ..."顺序排列，
// 而前端 CommandTree 不渲染章节为节点（plan §6.6 "object 一级 + 按 SA-SR 顺序"）。
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
// 空字符串（老 catalog 未分章）映射为高位 sentinel "~"（ASCII 126），确保排末位。
// 非空原样返回（"SA" < "SB" < ... < "SR" 字典序自然正确）。
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
	if v, ok := m["zh-CN"]; ok && v != "" {
		return v
	}
	if v, ok := m["en-US"]; ok && v != "" {
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

// buildDisplayName 派生命令树叶子显示名（业务化命名，T-0123 v3）：
//
//	"查询 设备信息"  /  "Query Device Info"
//
// 替代老格式 "设备信息(LST DEVICE_INFO)" — 用户决策 2026-05-16：去掉 path 风格
// 的 logical_code 后缀，让命令叶子直接呈现"做什么+对哪个对象"。
func buildDisplayName(logicalName, op, logicalCode, lang string) string {
	if logicalName == "" {
		logicalName = logicalCode
	}
	return fmt.Sprintf("%s %s", verbLabel(op, lang), logicalName)
}
