package main

import (
	"sort"
	"time"
)

// workerStat 是单个上传 worker 的本地统计，避免并发写共享结构的锁竞争；
// 全部 worker 结束后由 mergeStats 汇总。
type workerStat struct {
	latencies []time.Duration // 每次成功/失败请求的端到端耗时
	bytes     int64           // 累计上传字节（仅成功请求计）
	ok        int             // HTTP 2xx 数
	failed    int             // HTTP 非 2xx 数
	errs      int             // 传输层错误（连接/超时等，无 HTTP 状态码）
	status    map[int]int     // 状态码分布
}

func newWorkerStat(hint int) *workerStat {
	return &workerStat{
		latencies: make([]time.Duration, 0, hint),
		status:    make(map[int]int, 8),
	}
}

// aggStat 是全局汇总结果。
type aggStat struct {
	total    int
	ok       int
	failed   int
	errs     int
	bytes    int64
	status   map[int]int
	lat      []time.Duration // 已排序
	wallTime time.Duration   // 上传阶段墙钟耗时
}

func mergeStats(ws []*workerStat, wall time.Duration) *aggStat {
	a := &aggStat{status: make(map[int]int, 16), wallTime: wall}
	for _, w := range ws {
		if w == nil {
			continue
		}
		a.ok += w.ok
		a.failed += w.failed
		a.errs += w.errs
		a.bytes += w.bytes
		for code, n := range w.status {
			a.status[code] += n
		}
		a.lat = append(a.lat, w.latencies...)
	}
	a.total = a.ok + a.failed + a.errs
	sort.Slice(a.lat, func(i, j int) bool { return a.lat[i] < a.lat[j] })
	return a
}

// pct 返回第 p 百分位耗时（p ∈ [0,100]）。空集合返回 0。
func (a *aggStat) pct(p float64) time.Duration {
	n := len(a.lat)
	if n == 0 {
		return 0
	}
	idx := int(float64(n-1) * p / 100.0)
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return a.lat[idx]
}

func (a *aggStat) maxLat() time.Duration {
	if len(a.lat) == 0 {
		return 0
	}
	return a.lat[len(a.lat)-1]
}

// throughput 上传速率（成功+失败有响应的请求 / 秒）。
func (a *aggStat) throughput() float64 {
	s := a.wallTime.Seconds()
	if s <= 0 {
		return 0
	}
	return float64(a.total) / s
}

// mbPerSec 上传带宽 MB/s。
func (a *aggStat) mbPerSec() float64 {
	s := a.wallTime.Seconds()
	if s <= 0 {
		return 0
	}
	return float64(a.bytes) / 1024.0 / 1024.0 / s
}

func msf(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }
