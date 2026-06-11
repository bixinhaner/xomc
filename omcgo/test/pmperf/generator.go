package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// cstZone 固定 +08:00 时区。不用 time.LoadLocation("Asia/Shanghai") 以免目标机
// 缺 tzdata 时失败；样本文件里的时间戳本就是 +08:00（PM 文件 3GPP 32.435 dateTime）。
var cstZone = time.FixedZone("CST", 8*3600)

// iso8601DurRe 解析 granPeriod duration="PT900S" → 秒。只支持秒粒度（PM 文件常见
// PT900S / PT300S / PT3600S），分钟/小时格式回退到默认 900s。
var iso8601DurRe = regexp.MustCompile(`^PT(\d+)S$`)

// localDnRe 抓第一处 localDn="..." 的属性值，用于推断 body 内的设备 SN。
var localDnRe = regexp.MustCompile(`localDn="([^"]+)"`)

// generator 把一个真机样本 XML 模板化：记下原始 beginTime / endTime / body SN，
// 之后按 (sn, 时间窗) 做廉价 strings.ReplaceAll 生成每一份压测文件。
//
// 为什么用字符串替换而不是重新序列化 XML：样本文件是运营商真机产物（120KB~260KB，
// 数百个 measType），保真度最高的做法就是原样透传、只改身份与时间字段。重新 marshal
// 反而会丢字段顺序 / 命名空间 / 缩进等真机特征，降低压测代表性。
type generator struct {
	name        string // 模板文件名（仅用于报告展示）
	raw         string // 模板原始内容
	origBegin   string // 模板里 fileHeader/measCollec@beginTime 原值
	origEnd     string // 模板里 granPeriod@endTime / fileFooter@endTime 原值
	origBodySN  string // 模板里 managedElement localDn 内的 SN（可能为空）
	oui         string // 文件名里用的厂商 OUI 段
	granSeconds int    // 粒度秒数（PT900S → 900）
	counters    int    // 模板单文件的 <r p=> 数据点数 = 入库 pm_metrics 行数（measType × 小区数；报告用）
	rewriteBody bool   // 是否改写 body 内 localDn 的 SN（默认 true，纯属保真，?sn= 才是权威源）
}

func (g *generator) label() string       { return g.name }
func (g *generator) counterCount() int   { return g.counters }
func (g *generator) granSecondsVal() int { return g.granSeconds }

// loadGenerator 读取模板并抽取需要被替换的锚点。
func loadGenerator(path, oui string, rewriteBody bool) (*generator, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}
	raw := string(b)

	origBegin := firstAttr(raw, "beginTime")
	if origBegin == "" {
		return nil, fmt.Errorf("%s: 找不到 beginTime= 锚点，不是合法的 32.435 measCollecFile？", filepath.Base(path))
	}
	origEnd := firstAttr(raw, "endTime")
	if origEnd == "" {
		return nil, fmt.Errorf("%s: 找不到 endTime= 锚点", filepath.Base(path))
	}

	gran := 900
	if dur := firstAttr(raw, "duration"); dur != "" {
		if m := iso8601DurRe.FindStringSubmatch(dur); m != nil {
			if n, e := strconv.Atoi(m[1]); e == nil && n > 0 {
				gran = n
			}
		}
	}

	g := &generator{
		name:        filepath.Base(path),
		raw:         raw,
		origBegin:   origBegin,
		origEnd:     origEnd,
		origBodySN:  extractLocalDnSN(raw),
		oui:         oui,
		granSeconds: gran,
		counters:    strings.Count(raw, "<r p=\""),
		rewriteBody: rewriteBody,
	}
	return g, nil
}

// generate 产出一份压测文件：filename（用于 URL ?filename= 与 MinIO 落地名）+ body 字节。
//   - sn   ：本次模拟设备 SN（同时进 ?sn= 权威源 / 文件名 / body localDn）
//   - begin/end：本次 KPI 时间窗（写进 beginTime / granPeriod endTime / fileFooter endTime）
//   - seq  ：全局序号，保证文件名全局唯一，避免同日 MinIO 路径互相覆盖
func (g *generator) generate(sn string, begin, end time.Time, seq int) (string, []byte) {
	body := g.raw
	// 时间窗：beginTime 一处、endTime 多处（每个 measInfo 的 granPeriod + fileFooter），
	// 模板内同一文件所有窗口一致，ReplaceAll 一次性覆盖。
	body = strings.ReplaceAll(body, g.origBegin, fmtTime(begin))
	body = strings.ReplaceAll(body, g.origEnd, fmtTime(end))
	if g.rewriteBody && g.origBodySN != "" {
		body = strings.ReplaceAll(body, g.origBodySN, sn)
	}

	// 文件名沿用真机 Baicells 方言 A{date}.{start}+0800-{end}+0800_{OUI}.{SN}，
	// 末尾再缀 __{seq} 保证唯一。注意：设备身份以 URL ?sn= 为准（applyPayloadIdentity
	// 覆盖 body 解析结果），文件名仅用于落地与可读性，不参与路由。
	filename := fmt.Sprintf("A%s.%s+0800-%s+0800_%s.%s__%06d.xml",
		begin.Format("20060102"),
		begin.Format("1504"),
		end.Format("1504"),
		g.oui, sn, seq)

	return filename, []byte(body)
}

// fmtTime 输出 3GPP dateTime（RFC3339，+08:00 偏移），与样本一致：2026-06-11T08:15:00+08:00。
func fmtTime(t time.Time) string {
	return t.In(cstZone).Format("2006-01-02T15:04:05-07:00")
}

// firstAttr 返回 `name="value"` 中第一处的 value（裸字符串扫描，避免引入 XML 解析开销）。
func firstAttr(s, name string) string {
	key := name + `="`
	i := strings.Index(s, key)
	if i < 0 {
		return ""
	}
	i += len(key)
	j := strings.IndexByte(s[i:], '"')
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}

// extractLocalDnSN 从第一处 localDn="..." 推断 SN：取最后一个 '=' 之后、再取最后一个
// '-' 之后的片段。覆盖样本两种方言：
//
//	Station=eNb-120200024118AA00001  → 120200024118AA00001
//	ManagedElement=1202000543236VB0028 → 1202000543236VB0028
func extractLocalDnSN(raw string) string {
	m := localDnRe.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	v := m[1]
	if k := strings.LastIndexByte(v, '='); k >= 0 {
		v = v[k+1:]
	}
	if k := strings.LastIndexByte(v, '-'); k >= 0 {
		v = v[k+1:]
	}
	return strings.TrimSpace(v)
}
