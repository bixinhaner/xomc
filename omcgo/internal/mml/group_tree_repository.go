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
	ID               uuid.UUID         `json:"id"`
	GroupCode        string            `json:"code"`
	Name             string            `json:"name"`           // 当前 lang 派生
	NameI18n         map[string]string `json:"name_i18n"`      // 完整 i18n
	Path             string            `json:"path"`           // LTREE path 字符串形式
	DisplayOrder     int               `json:"display_order"`
	Source           string            `json:"source"`
	CatalogProtected bool              `json:"catalog_protected"`
	Commands         []GroupTreeCommand `json:"commands"`
	Children         []GroupTreeNode    `json:"children"`
}

// GroupTreeCommand 是树中一条命令叶子的精简视图（不含 sub_fields，sub_fields 通过
// `GET /mml/commands/:id/sub-fields` 单独懒加载）。
type GroupTreeCommand struct {
	ID               uuid.UUID         `json:"id"`
	CommandCode      string            `json:"command_code"`
	LogicalCode      string            `json:"logical_code"`
	LogicalName      string            `json:"logical_name"`       // lang 派生
	LogicalNameI18n  map[string]string `json:"logical_name_i18n"`
	OperationType    string            `json:"operation_type"`
	DisplayName      string            `json:"display_name"`       // "设备信息(LST DEVICE_INFO)"
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
				cmd.DisplayName = buildDisplayName(cmd.LogicalName, cmd.OperationType, cmd.LogicalCode)
				node.Commands = append(node.Commands, cmd)
				commandSeen[*row.CommandID] = struct{}{}
			}
		}
	}

	// 按 path 深度组装父子关系
	nodes := assembleHierarchy(groupByID, pathToGroupID)
	return nodes, nil
}

// ============================================================
// SQL 查询
// ============================================================

// queryGroupsAndCommands 单 SQL JOIN 抓 groups + commands。
// rootCode 为空时拉全表；非空时按 LTREE @> 拉子树。
func (r *PgGroupTreeRepository) queryGroupsAndCommands(ctx context.Context, rootCode string) ([]groupTreeRow, error) {
	const baseSQL = `
SELECT
    g.id, g.group_code, g.group_name_zh, g.group_name_en, g.name_i18n,
    g.path::text AS path_text, g.display_order, g.source, g.catalog_protected,
    c.id, c.command_code, c.logical_code, c.logical_name_i18n,
    c.operation_type, c.rpc_method, c.require_confirm, c.target_object,
    c.source AS cmd_source, c.catalog_protected AS cmd_catalog_protected
FROM mml_param_groups g
LEFT JOIN mml_commands c ON c.group_id = g.id
%s
ORDER BY g.path, g.display_order, c.operation_type, c.logical_code, c.command_code`

	var whereClause string
	var args []any
	if rootCode != "" {
		// LTREE @> $1 匹配 path 等于 rootCode 或以 rootCode 为前缀
		whereClause = "WHERE g.path <@ $1::ltree OR g.path ~ ($1::text || '.*')::lquery"
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
			&row.GroupPath, &row.GroupDisplayOrder, &row.GroupSource, &row.GroupCatalogProtected,
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
	GroupID                uuid.UUID
	GroupCode              string
	GroupNameZh            string
	GroupNameEn            *string
	GroupNameI18n          map[string]string
	GroupPath              string
	GroupDisplayOrder      int
	GroupSource            string
	GroupCatalogProtected  bool

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
//   "A" → ""
//   "A.B" → "A"
//   "A.B.C" → "A.B"
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

func sortNodesByDisplayOrder(nodes []GroupTreeNode) {
	sortSlice(nodes, func(a, b GroupTreeNode) bool {
		if a.DisplayOrder != b.DisplayOrder {
			return a.DisplayOrder < b.DisplayOrder
		}
		return a.GroupCode < b.GroupCode
	})
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

// buildDisplayName 派生命令树叶子显示名（老 OMC 格式）：
//   "设备信息(LST DEVICE_INFO)"
func buildDisplayName(logicalName, op, logicalCode string) string {
	if logicalName == "" {
		logicalName = logicalCode
	}
	return fmt.Sprintf("%s(%s %s)", logicalName, op, logicalCode)
}
