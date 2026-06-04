package metrics

import "regexp"

// object_ldn（小区/PLMN 维度键）解析——单一真值源。
//
// object_ldn 落库即完整 LDN 串：
//   - 基础小区行：`Cellid=N`（只有小区，无 PLMN）
//   - PLMN 行：`Cellid=N,PLMN=M`（小区 + PLMN）
//
// 两类靠 cellID = N 关联到同一物理小区，供 KPI 跨层级配对（小区级计数器配 PLMN 级计数器）。
//
// ⚠ 当前仅认 `Cellid=`/`PLMN=`（LTE，与 T-0193 一致）；NR 的 `NRCellDU=` 不被解析（提小区得空）。
// 真机为 LTE；NR/空小区实体由调用方按"plmn=='' 且 cellID==''"安全降级（不 panic）。
var (
	objectLDNCellIDRe = regexp.MustCompile(`Cellid=([0-9]+)`)
	objectLDNPlmnRe   = regexp.MustCompile(`PLMN=([0-9]+)`)
)

// ParseObjectLDN 从原始 object_ldn 拆出 cell_id / plmn（缺段则留空）。
//
// 判定：
//   - plmn == ""  → 基础小区行（小区级）
//   - plmn != ""  → PLMN 行（PLMN 级）；cellID 用于跨层级配对找同小区基础行
func ParseObjectLDN(ldn string) (cellID, plmn string) {
	if m := objectLDNCellIDRe.FindStringSubmatch(ldn); len(m) == 2 {
		cellID = m[1]
	}
	if m := objectLDNPlmnRe.FindStringSubmatch(ldn); len(m) == 2 {
		plmn = m[1]
	}
	return cellID, plmn
}
