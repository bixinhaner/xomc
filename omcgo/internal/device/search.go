// search.go — 设备列表多关键字搜索 helper（device list 页面 G07 升级）。
//
// 之前 device_info_pg_repository.ListDevicesWithInfo 只支持单个关键字 ILIKE，
// 而前端搜索框 placeholder 承诺 "SN/名称/IP/MAC/PCI" 多字段且用户期望能一次
// 搜多个值（如批量粘贴 SN 列表）。本文件提供一个公共 helper：
//
//   - 输入字符串按英文逗号分隔成多个关键字
//   - 每个关键字 TrimSpace，跳过空
//   - 最多 50 个关键字（防止用户粘贴超大列表把 SQL 撑爆）
//   - 每个关键字 vs 每个字段都生成 ILIKE 子句，全部 OR
//
// 语义：任一关键字在任一字段命中即匹配（设备级 OR）。
//
// 单值场景完全向后兼容：split 后只 1 个关键字，与老行为等价。
//
// 字段集合由 caller 传入，让不同列表（含 join / 不含 join / 不同字段集）共用本 helper。
package device

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// MaxSearchKeywords 限制单次搜索的关键字数量上限。
//
// 50 个 SN × 6 字段 = 300 个 ILIKE 子句，PostgreSQL planner 可接受；超过此值
// 直接截断，避免用户误粘 10k 行把 SQL 体积撑爆。
const MaxSearchKeywords = 50

// BuildSearchOR 把逗号分隔的多关键字字符串 + 字段列表展开为 sq.Or 条件。
//
// 入参：
//   - search：用户输入，例如 "SN1, SN2,SN3"；空白 / 空字符串返 nil
//   - fields：要参与匹配的列名（含表别名），例如
//     {"d.serial_number", "d.site_name", "di.device_name"}
//
// 返回：
//   - 至少 1 个有效关键字 + 至少 1 个字段 → 非空 sq.Or（caller 直接 .Where(cond)）
//   - 否则返 nil（caller 应短路跳过 Where 调用）
func BuildSearchOR(search string, fields []string) sq.Or {
	if strings.TrimSpace(search) == "" || len(fields) == 0 {
		return nil
	}
	parts := strings.Split(search, ",")
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		k := strings.TrimSpace(p)
		if k == "" {
			continue
		}
		keywords = append(keywords, k)
		if len(keywords) >= MaxSearchKeywords {
			break
		}
	}
	if len(keywords) == 0 {
		return nil
	}
	cond := make(sq.Or, 0, len(keywords)*len(fields))
	for _, kw := range keywords {
		pattern := "%" + kw + "%"
		for _, f := range fields {
			cond = append(cond, sq.ILike{f: pattern})
		}
	}
	return cond
}
