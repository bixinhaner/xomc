package metrics

import "regexp"

// object_ldn（小区/PLMN 维度键）解析——单一真值源。
//
// object_ldn 落库即完整 LDN 串，按制式三种格式：
//
//   - 4G / LTE：
//     `Cellid=N`（基础小区行，无 PLMN）
//     `Cellid=N,PLMN=M`（PLMN 行）
//
//   - 5G / NR：
//     `Type=Cell,Mode=SA,gNBID=N`（设备级）
//     `Type=Cell,Mode=SA,gNBID=N,NrCGI=N,CUID=N`（CU 小区级）
//     `Type=Cell,Mode=SA,gNBID=N,NrCGI=N,CUID=N,PLMNID=M`（PLMN 行）
//     `Type=Cell,Mode=SA,gNBID=N,NrCGI=N,CUID=N,NSSAI=N/N/N`（切片）
//     `Type=Cell,Mode=SA,gNBID=N,NrCGI=N,CUID=N,SCLICEGROUP=N/N/N`（切片组）
//     `Type=Cell,Mode=SA,gNBID=N,NrCGI=N,DUID=N`（DU 小区级）
//
//   - GSM：
//     `Uid=N-N`
//
// 设计取舍：
//   - 单一真值源：本包是 object_ldn 解析唯一入口，extractCellID 等不再各写一套正则。
//   - 5G/GSM 的 CellID/Plmn 字段刻意留空（不填 NrCGI/Uid）——
//     当前 aggregator 在 `CellID != "" && Plmn != ""` 才触发跨层级配对（仅 4G 行），
//     若把 NrCGI/Uid 塞进 CellID 会无意激活配对路径，破坏阶段 A "不动 aggregator 配对" 约束。
//     5G/GSM 的天然小区标识从 NrCGI/Uid 字段直接读，BaseCellID 方法封装该选择逻辑。
//   - 缺段全部留空字符串、不 panic；不认识的串所有字段皆空。

// Tech 标识 ParseObjectLDN 判定出的制式。
type Tech string

const (
	TechUnknown Tech = ""
	TechLTE     Tech = "lte" // 4G
	TechNR      Tech = "nr"  // 5G
	TechGSM     Tech = "gsm"
)

// ObjectLDNFields 是 object_ldn 串拆出的所有维度键，按制式补缺段空。
//
// 调用方按需取字段：
//   - 4G 聚合器读 CellID / Plmn（跨层级配对依据）。
//   - 5G/GSM 取 BaseCellID() 拿天然小区标识；按需取 GNBID / CUID / PLMNID / Uid 等。
//   - 前端 formatObjectLdn 读 Tech 选格式化分支。
type ObjectLDNFields struct {
	// 制式判定结果（按字段优先级 gNBID>Uid>Cellid 决定）。
	Tech Tech

	// 4G / LTE
	CellID string // Cellid=N（仅 LTE 行填；5G/GSM 留空）
	Plmn   string // PLMN=M（仅 LTE 行填；5G/GSM 留空，用 PLMNID）

	// 5G / NR
	GNBID      string // gNBID=N
	NrCGI      string // NrCGI=N
	CUID       string // CUID=N
	DUID       string // DUID=N
	PLMNID     string // PLMNID=N（5G 的 PLMN，注意与 LTE PLMN 字段名不同）
	NSSAI      string // NSSAI=N/N/N
	SliceGroup string // SCLICEGROUP=N/N/N

	// GSM
	Uid string // Uid=N-N（值含连字符）
}

// BaseCellID 返回按制式选定的"基础小区标识"——
//
//	4G: CellID（Cellid=）
//	5G: NrCGI（CU/DU 级唯一）
//	GSM: Uid
//	未知: ""
//
// extractCellID 等需要"单一小区标签"的下游用此；不返回 fallback 串，由调用方决定是否回退。
func (o ObjectLDNFields) BaseCellID() string {
	switch o.Tech {
	case TechLTE:
		return o.CellID
	case TechNR:
		return o.NrCGI
	case TechGSM:
		return o.Uid
	default:
		return ""
	}
}

// 解析正则：每段是 `Key=Value`，逗号分隔；Value 取到下一个逗号或串尾（[^,]+）。
// Uid 值含 `-`、NSSAI/SCLICEGROUP 含 `/`，故不能限定 [0-9]+。
// 同时兼容 Cellid 大小写（真实样本 `Cellid=`，旧 extractCellID 测试用 `CellId=`）。
var (
	reCellID     = regexp.MustCompile(`(?i)\bCellid=([^,]+)`)
	rePlmn       = regexp.MustCompile(`\bPLMN=([^,]+)`)        // 注意：与 PLMNID 区分（下面用 \b 边界）
	reGNBID      = regexp.MustCompile(`\bgNBID=([^,]+)`)
	reNrCGI      = regexp.MustCompile(`\bNrCGI=([^,]+)`)
	reCUID       = regexp.MustCompile(`\bCUID=([^,]+)`)
	reDUID       = regexp.MustCompile(`\bDUID=([^,]+)`)
	rePLMNID     = regexp.MustCompile(`\bPLMNID=([^,]+)`)
	reNSSAI      = regexp.MustCompile(`\bNSSAI=([^,]+)`)
	reSliceGroup = regexp.MustCompile(`\bSCLICEGROUP=([^,]+)`)
	reUid        = regexp.MustCompile(`\bUid=([^,]+)`)
)

// ParseObjectLDN 从原始 object_ldn 拆出全部维度键并判定制式。
// 缺段留空字符串；完全无法识别 → Tech=TechUnknown 且所有字段空（不 panic）。
//
// 兼容性：旧两返回值签名 (cellID, plmn string) 已下线；调用方改用结构体字段。
func ParseObjectLDN(ldn string) ObjectLDNFields {
	var out ObjectLDNFields

	// 5G 字段
	if m := reGNBID.FindStringSubmatch(ldn); len(m) == 2 {
		out.GNBID = m[1]
	}
	if m := reNrCGI.FindStringSubmatch(ldn); len(m) == 2 {
		out.NrCGI = m[1]
	}
	if m := reCUID.FindStringSubmatch(ldn); len(m) == 2 {
		out.CUID = m[1]
	}
	if m := reDUID.FindStringSubmatch(ldn); len(m) == 2 {
		out.DUID = m[1]
	}
	if m := rePLMNID.FindStringSubmatch(ldn); len(m) == 2 {
		out.PLMNID = m[1]
	}
	if m := reNSSAI.FindStringSubmatch(ldn); len(m) == 2 {
		out.NSSAI = m[1]
	}
	if m := reSliceGroup.FindStringSubmatch(ldn); len(m) == 2 {
		out.SliceGroup = m[1]
	}
	// GSM
	if m := reUid.FindStringSubmatch(ldn); len(m) == 2 {
		out.Uid = m[1]
	}
	// 4G（最后处理，便于制式判定的优先级控制）
	if m := reCellID.FindStringSubmatch(ldn); len(m) == 2 {
		out.CellID = m[1]
	}
	if m := rePlmn.FindStringSubmatch(ldn); len(m) == 2 {
		out.Plmn = m[1]
	}

	// 制式判定：以特征字段为准（一票通过）。
	//   - gNBID 出现 → 5G（即便同串里恰好有 Cellid 也以 gNBID 为准；真实样本互斥）
	//   - Uid 出现 → GSM
	//   - Cellid 出现 → 4G
	switch {
	case out.GNBID != "" || out.NrCGI != "":
		out.Tech = TechNR
		// 5G 不复用 LTE 的 CellID/Plmn 字段（避免误触 aggregator 跨层级配对）。
		out.CellID = ""
		out.Plmn = ""
	case out.Uid != "":
		out.Tech = TechGSM
		out.CellID = ""
		out.Plmn = ""
	case out.CellID != "":
		out.Tech = TechLTE
	default:
		out.Tech = TechUnknown
	}

	return out
}
