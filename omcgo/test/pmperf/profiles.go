package main

import "time"

// fileGen 是 PM 文件生成器抽象：模板法（generator）与指标库合成法（libGen）都实现它。
//
//	generate(sn, begin, end, seq) → (filename, body)
type fileGen interface {
	generate(sn string, begin, end time.Time, seq int) (string, []byte)
	label() string       // 报告里展示的生成器名（平台名 / 模板名）
	counterCount() int   // 该生成器单文件携带的 counter 数（用于报告）
	granSecondsVal() int // 粒度秒数
}

// ratProfile 描述一个制式（RAT）压测所需的全部约定：
//   - productClass：必须能被 products.xml 的 pattern 命中 → 路由到 indicator platform，
//     否则 worker 既算不出 KPI，又因白名单为空放过重名 counter 撞自然键（SQLSTATE 21000）。
//   - platform/deviceType：决定读哪张 perf_indicators_* / rela_platform_indicator_formula_*。
//   - indicatorRel：指标库 XML（相对 omcgo 目录），合成法据此取全部 isCounter=1 的 reportKey。
//   - cellLdn：measValue 的 measObjLdn（单小区），KPI 反算按它聚合。
//
// 三个 productClass 取值已用 PG 正则校验命中（见 product_class_patterns）：
//
//	FAP/BAIBLQ/SC → ^FAP/BAIBLQ/SC$        (BLQ, enb)
//	FAP/BSCNR     → FAP/\w*BSC\w+          (BaiBNQ, gnb)
//	FAP/PGSM      → ^FAP/PGSM$             (BSC, gsm)
type ratProfile struct {
	rat          string // lte | nr | gsm
	productClass string
	platform     string
	deviceType   string // enb | gnb | gsm
	tech         string // devices.technology 列
	elementType  string // fileSender@elementType
	indicatorRel string // 指标库 XML 路径（相对 omcgo 目录）
	cellLdn      string // measValue@measObjLdn（单小区）
	meLdnFmt     string // managedElement@localDn 格式串，%s = SN
}

// builtinProfiles 内置三制式画像。productClass 取各平台 globalOrder 最靠前、唯一命中的值。
var builtinProfiles = map[string]ratProfile{
	"lte": {
		rat: "lte", productClass: "FAP/BAIBLQ/SC", platform: "BLQ", deviceType: "enb", tech: "lte",
		elementType: "eNodeB", indicatorRel: "data/indicator-library/enb/BLQ.xml",
		cellLdn: "Cellid=1", meLdnFmt: "Station=eNb-%s",
	},
	"nr": {
		rat: "nr", productClass: "FAP/BSCNR", platform: "BaiBNQ", deviceType: "gnb", tech: "nr",
		elementType: "gNB", indicatorRel: "data/indicator-library/GNB.xml",
		cellLdn: "Type=Cell,Mode=SA,gNBID=1,NrCGI=1,CUID=1", meLdnFmt: "ManagedElement=%s",
	},
	"gsm": {
		rat: "gsm", productClass: "FAP/PGSM", platform: "BSC", deviceType: "gsm", tech: "gsm",
		elementType: "BSC", indicatorRel: "data/indicator-library/GSM.xml",
		cellLdn: "BtsSiteManager=1,Bts=1", meLdnFmt: "BscFunction=%s",
	},
}
