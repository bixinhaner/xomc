package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// indicatorTagRe 抓指标库 XML 里每个 <indicator .../> 自闭合标签。
var indicatorTagRe = regexp.MustCompile(`<indicator\b[^>]*/>`)

// attrReportKeyRe / attrIsCounterRe 从单个 indicator 标签里取属性值。
var (
	attrReportKeyRe = regexp.MustCompile(`reportKey="([^"]*)"`)
	attrIsCounterRe = regexp.MustCompile(`isCounter="([^"]*)"`)
)

// libGen 是「指标库驱动」的合成生成器：读某平台指标库 XML，取出全部 isCounter=1 的
// reportKey（去重），每份压测文件把这些 reportKey 作为 measType 各发一个值。这样：
//   - 覆盖该平台全部内置源指标 → KPI 反算能把所有派生指标都算到（“内置指标都压到”）。
//   - reportKey 在库内唯一 → 单文件无重名 counter → 不会撞 pm_metrics 自然键（避开 21000）。
//   - 不依赖真机样本 → GSM 这类没有现成样本的制式也能压。
type libGen struct {
	prof       ratProfile
	oui        string
	reportKeys []string // 去重后的 isCounter=1 reportKey，按库内顺序
	gran       int      // 粒度秒数
}

func (g *libGen) label() string       { return g.prof.platform }
func (g *libGen) counterCount() int   { return len(g.reportKeys) }
func (g *libGen) granSecondsVal() int { return g.gran }

// loadLibGen 解析指标库 XML，构建合成生成器。granSeconds 为时间窗粒度（一般 900）。
func loadLibGen(prof ratProfile, indicatorPath, oui string, granSeconds int) (*libGen, error) {
	b, err := os.ReadFile(indicatorPath)
	if err != nil {
		return nil, fmt.Errorf("read indicator library %s: %w", indicatorPath, err)
	}
	content := string(b)

	seen := make(map[string]struct{})
	var keys []string
	for _, tag := range indicatorTagRe.FindAllString(content, -1) {
		ic := attrIsCounterRe.FindStringSubmatch(tag)
		if ic == nil || ic[1] != "1" {
			continue // 只发源 counter；派生 KPI（isCounter=0）由 worker 反算
		}
		rk := attrReportKeyRe.FindStringSubmatch(tag)
		if rk == nil || rk[1] == "" {
			continue
		}
		if _, dup := seen[rk[1]]; dup {
			continue
		}
		seen[rk[1]] = struct{}{}
		keys = append(keys, rk[1])
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%s: 未解析到任何 isCounter=1 的 reportKey", indicatorPath)
	}
	if granSeconds <= 0 {
		granSeconds = 900
	}
	return &libGen{prof: prof, oui: oui, reportKeys: keys, gran: granSeconds}, nil
}

// generate 合成一份 measCollecFile：单 measInfo、单 measValue（单小区），把全部 reportKey
// 作为 measType 各发一个值。值按 (seq,i) 变化，避免全 0 / 全同。
func (g *libGen) generate(sn string, begin, end time.Time, seq int) (string, []byte) {
	beginS, endS := fmtTime(begin), fmtTime(end)
	var b strings.Builder
	b.Grow(len(g.reportKeys) * 64)
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<measCollecFile xmlns="http://www.3gpp.org/ftp/specs/archive/32_series/32.435#measCollec">` + "\n")
	b.WriteString(`  <fileHeader fileFormatVersion="32.435 v9.0" vendorName="KPILOADTEST">` + "\n")
	fmt.Fprintf(&b, "    <fileSender elementType=%q/>\n", g.prof.elementType)
	fmt.Fprintf(&b, "    <measCollec beginTime=%q/>\n", beginS)
	b.WriteString("  </fileHeader>\n  <measData>\n")
	fmt.Fprintf(&b, "    <managedElement localDn=%q swVersion=\"KPILT_1.0\"/>\n", fmt.Sprintf(g.prof.meLdnFmt, sn))
	b.WriteString(`    <measInfo measInfoId="1">` + "\n")
	fmt.Fprintf(&b, "      <granPeriod duration=\"PT%dS\" endTime=%q/>\n", g.gran, endS)
	fmt.Fprintf(&b, "      <repPeriod duration=\"PT%dS\"/>\n", g.gran)
	for i, rk := range g.reportKeys {
		fmt.Fprintf(&b, "      <measType p=\"%d\">%s</measType>\n", i+1, rk)
	}
	fmt.Fprintf(&b, "      <measValue measObjLdn=%q>\n", g.prof.cellLdn)
	for i := range g.reportKeys {
		v := 1 + ((seq*131 + i*7919) % 100000) // 确定性、随 (seq,i) 变化的正整数
		fmt.Fprintf(&b, "        <r p=\"%d\">%d</r>\n", i+1, v)
	}
	b.WriteString("      </measValue>\n    </measInfo>\n  </measData>\n")
	fmt.Fprintf(&b, "  <fileFooter><measCollec endTime=%q/></fileFooter>\n", endS)
	b.WriteString("</measCollecFile>\n")

	filename := fmt.Sprintf("A%s.%s+0800-%s+0800_%s.%s__%06d.xml",
		begin.Format("20060102"), begin.Format("1504"), end.Format("1504"), g.oui, sn, seq)
	return filename, []byte(b.String())
}
