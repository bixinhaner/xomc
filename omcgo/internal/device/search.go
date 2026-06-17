// search.go — 设备列表多关键字搜索 helper（device list 页面 G07 升级）。
//
// device_info_pg_repository.ListDevicesWithInfo 之前只支持单个关键字 ILIKE；
// 前端 placeholder 承诺了 "SN/名称/IP/MAC/PCI"，而本文件提供的公共 helper 只负责
// 把传入字段集展开为多关键字 OR 条件，具体搜哪些字段由 caller 决定：
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
// 50 个关键字 × 9 字段 = 450 个 ILIKE 子句，PostgreSQL planner 可接受；超过此值
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

// SplitCSV 把逗号分隔的多值字符串拆成切片；TrimSpace + 去空。
// 主要供 device list 多选筛选字段（model_name / software_version /
// firmware_version / product_class）从 CSV 形式的 query string 还原成 []string，
// 配合 sq.Eq{"col": vals} 自动展开为 IN (...)。
//
// 与 BuildSearchOR 共享 MaxSearchKeywords 上限避免恶意大数组撑爆 SQL。
//
// 单值场景：splitCSV("foo") → ["foo"]，sq.Eq{"col": ["foo"]} 行为与
// sq.Eq{"col": "foo"} 完全等价（IN ("foo") == ("foo")），向后兼容。
func SplitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
		if len(out) >= MaxSearchKeywords {
			break
		}
	}
	return out
}
