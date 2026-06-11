package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// pmProm 是 worker 进程 /metrics 里与 PM 解析入库相关的指标快照。
// 取测试前后两次快照做差，得到本轮的解析吞吐 / 时延 / 失败 / 迟到数据。
type pmProm struct {
	filesSuccess     float64 // omc_pm_files_processed_total{status="success"}
	filesFailed      float64 // omc_pm_files_processed_total{status="failed"}
	filesLate        float64 // omc_pm_files_processed_total{status="late_arrival"}
	lateArrivalTotal float64 // omc_pm_late_arrival_files_total（全标签求和）
	droppedCounters  float64 // omc_pm_dropped_counters_total（全标签求和）
	procDurSum       float64 // omc_pm_processing_duration_seconds_sum
	procDurCount     float64 // omc_pm_processing_duration_seconds_count
	delaySum         float64 // omc_pm_report_delay_seconds_sum（全标签求和）
	delayCount       float64 // omc_pm_report_delay_seconds_count（全标签求和）
	scraped          bool    // 是否成功抓取（抓取失败时整组按 0 处理并提示）
}

// avgProcMillis 单文件平均处理耗时（毫秒）。
func (p pmProm) avgProcMillis() float64 {
	if p.procDurCount <= 0 {
		return 0
	}
	return p.procDurSum / p.procDurCount * 1000.0
}

// avgDelaySec 平均上报延迟（ingest_time - end_time，秒）。
func (p pmProm) avgDelaySec() float64 {
	if p.delayCount <= 0 {
		return 0
	}
	return p.delaySum / p.delayCount
}

// sub 计算两次快照差（after - before），用于报告本轮增量。
func (p pmProm) sub(b pmProm) pmProm {
	return pmProm{
		filesSuccess:     p.filesSuccess - b.filesSuccess,
		filesFailed:      p.filesFailed - b.filesFailed,
		filesLate:        p.filesLate - b.filesLate,
		lateArrivalTotal: p.lateArrivalTotal - b.lateArrivalTotal,
		droppedCounters:  p.droppedCounters - b.droppedCounters,
		procDurSum:       p.procDurSum - b.procDurSum,
		procDurCount:     p.procDurCount - b.procDurCount,
		delaySum:         p.delaySum - b.delaySum,
		delayCount:       p.delayCount - b.delayCount,
		scraped:          p.scraped && b.scraped,
	}
}

// scrapeProm 抓取 worker /metrics 并解析 PM 指标。失败时返回 scraped=false（不报错，
// 仅让调用方在报告里标注"未抓到 worker 指标"——DB 行数仍能独立验证入库能力）。
func scrapeProm(ctx context.Context, url string, timeout time.Duration) pmProm {
	var out pmProm
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return out
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return out
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		name, labels, val, ok := parsePromLine(line)
		if !ok {
			continue
		}
		switch name {
		case "omc_pm_files_processed_total":
			switch labels["status"] {
			case "success":
				out.filesSuccess += val
			case "failed":
				out.filesFailed += val
			case "late_arrival":
				out.filesLate += val
			}
		case "omc_pm_late_arrival_files_total":
			out.lateArrivalTotal += val
		case "omc_pm_dropped_counters_total":
			out.droppedCounters += val
		case "omc_pm_processing_duration_seconds_sum":
			out.procDurSum += val
		case "omc_pm_processing_duration_seconds_count":
			out.procDurCount += val
		case "omc_pm_report_delay_seconds_sum":
			out.delaySum += val
		case "omc_pm_report_delay_seconds_count":
			out.delayCount += val
		}
	}
	out.scraped = true
	return out
}

// parsePromLine 解析一行 Prometheus 文本格式：`name{l1="v1",l2="v2"} value [ts]`。
// 返回 metric 名、标签 map、值。无标签时 labels 为空 map。
func parsePromLine(line string) (name string, labels map[string]string, val float64, ok bool) {
	labels = map[string]string{}
	brace := strings.IndexByte(line, '{')
	var rest string
	if brace >= 0 {
		name = line[:brace]
		end := strings.IndexByte(line, '}')
		if end < 0 || end < brace {
			return "", nil, 0, false
		}
		parseLabels(line[brace+1:end], labels)
		rest = strings.TrimSpace(line[end+1:])
	} else {
		sp := strings.IndexByte(line, ' ')
		if sp < 0 {
			return "", nil, 0, false
		}
		name = line[:sp]
		rest = strings.TrimSpace(line[sp+1:])
	}
	// rest 可能是 "value" 或 "value timestamp"，取第一个字段。
	if i := strings.IndexByte(rest, ' '); i >= 0 {
		rest = rest[:i]
	}
	v, err := strconv.ParseFloat(rest, 64)
	if err != nil {
		return "", nil, 0, false
	}
	return name, labels, v, true
}

func parseLabels(s string, out map[string]string) {
	for _, kv := range splitLabels(s) {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(kv[:eq])
		v := strings.Trim(strings.TrimSpace(kv[eq+1:]), `"`)
		out[k] = v
	}
}

// splitLabels 按逗号切分标签，跳过引号内的逗号。
func splitLabels(s string) []string {
	var parts []string
	var b strings.Builder
	inQ := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQ = !inQ
			b.WriteByte(c)
		case c == ',' && !inQ:
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}

// promHint 给报告里抓取失败的提示串。
func promHint(url string) string {
	return fmt.Sprintf("(未能抓取 worker 指标 %s —— 入库吞吐以 DB 行数增量为准；如需 worker 指标，确认 -worker-metrics 可达)", url)
}
