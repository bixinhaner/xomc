package collector

import (
	"errors"
	"fmt"
	"time"
)

// T-0164-P4 / G4 PM 时间窗校验。
//
// 三时间字段语义（设计文档 §4.4）：
//   - start_time  — 基站采集窗口起（基站时钟）
//   - end_time    — 基站采集窗口止（基站时钟）
//   - ingest_time — OMC 入库时刻（OMC 时钟）
//
// 校验规则：
//   1. start_time < end_time （单调性）
//   2. end_time - start_time ∈ [granPeriod.duration ± Tolerance] （容忍时钟漂移）
//   3. ingest_time - end_time < MaxIngestLag （拒绝过老文件）
//   4. 都不允许 zero（解析 fallback 已耗尽时报错）

// TimeWindowToleranceSeconds 是窗口长度与 granPeriod.duration 的最大允许偏差（秒）。
// 60s 容忍基站重启 / NTP 漂移 / 上报抖动。
const TimeWindowToleranceSeconds = 60

// MaxIngestLag 入库延迟上限：end_time 距 ingest_time 超过此值视为过老文件（默认 24h）。
// 项目运维可在 sys_configs 调整（留 G3/G5 阶段实施，G4 阶段先 hard-code）。
var MaxIngestLag = 24 * time.Hour

// ErrTimeWindowMissing 三时间字段中有 zero 值，parser fallback 已耗尽仍无法推断。
var ErrTimeWindowMissing = errors.New("start_time / end_time missing (parser fallback exhausted)")

// ErrTimeWindowReversed start_time >= end_time。
var ErrTimeWindowReversed = errors.New("start_time >= end_time")

// ErrTimeWindowSpanDeviates 窗口长度与 granPeriod.duration 偏差超过 Tolerance。
var ErrTimeWindowSpanDeviates = errors.New("window span deviates from granPeriod.duration > tolerance")

// ErrTimeWindowStale ingest_time - end_time > MaxIngestLag（过老文件）。
var ErrTimeWindowStale = errors.New("ingest_time - end_time exceeds MaxIngestLag (stale file)")

// ValidateTimeWindow 校验 PM 文件解析出的三时间字段是否在合理范围。
//
// 参数：
//   - start, end       — 基站时钟（fileHeader/fileFooter 或 fallback）
//   - granDuration     — granPeriod.duration（来自 measInfo），用于校验窗口长度
//   - ingest           — Parse 完成时刻（PMFileContent.IngestTime）
//
// 返回 nil 表示通过；返回错误时由调用方决定 reject 整个文件还是 log warn 继续。
func ValidateTimeWindow(start, end time.Time, granDuration time.Duration, ingest time.Time) error {
	if start.IsZero() || end.IsZero() {
		return ErrTimeWindowMissing
	}
	if !start.Before(end) {
		return fmt.Errorf("%w: start=%s end=%s", ErrTimeWindowReversed, start.Format(time.RFC3339), end.Format(time.RFC3339))
	}
	span := end.Sub(start)
	tolerance := time.Duration(TimeWindowToleranceSeconds) * time.Second
	if granDuration > 0 {
		diff := span - granDuration
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			return fmt.Errorf("%w: span=%v duration=%v diff=%v", ErrTimeWindowSpanDeviates, span, granDuration, diff)
		}
	}
	if !ingest.IsZero() && ingest.Sub(end) > MaxIngestLag {
		return fmt.Errorf("%w: ingest=%s end=%s lag=%v", ErrTimeWindowStale, ingest.Format(time.RFC3339), end.Format(time.RFC3339), ingest.Sub(end))
	}
	return nil
}
