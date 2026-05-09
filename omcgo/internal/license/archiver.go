// Package license — log archive (T-0100-P4-B + P4-B1).
//
// 周级 cron 任务把 license_logs 表里 created_at < (now - retention_months) 的
// 行打包写到 MinIO，按月聚合 + 按 tick 唯一化键名：
//
//	license-logs/{YYYY-MM}/{tickTimestampUTC}-{count}.jsonl.gz
//
// 归档成功后从 DB 删除对应行。等保 2.0 三级 8.1.4.7 要求重要操作日志保留
// ≥ 6 个月（PRD §5.4.5）。
//
// **键名设计（T-0100-P4-B1 修复 CRITICAL）**：
//
// P4-B 初版用 `license-logs/{YYYY-MM}.jsonl.gz` 固定键，cutoff 漂移落在月内时
// 同月会被跨 tick 覆盖（review 报告 REVIEW_922d87a4_chenbo01_license.md
// CRITICAL #1）。P4-B1 把 tick UTC 时间戳 + 本批日志条数嵌入键名，确保：
//   - 同 tick 重复跑（重启 / 重试）覆盖自身（幂等）
//   - 不同 tick 即使覆盖到同一个月也写到不同 key（零数据丢失）
//   - 按月前缀 `license-logs/{YYYY-MM}/` 列举可拿到该月所有归档片段，
//     审计工具按 tickTimestamp 排序合并即可
//
// 单次 tick 的正确性保证：
//  1. 计算 cutoff = now - retention_months
//  2. ListBefore(cutoff, batchSize) 拉一批最早的日志
//  3. 按 YYYY-MM 分桶（同月聚合），月内按 created_at 升序
//  4. 对每个月的日志：写 PutObject 到唯一键 `{YYYY-MM}/{tickTS}-{count}.jsonl.gz`
//  5. 全部月归档成功后，DeleteByIDs(本批已归档行的 id 集合) 物理删 DB 行
//     —— **不**用 DeleteBefore(cutoff)，因为 ListBefore 受 batchSize 限制可能
//     只取了一部分；对超出 batchSize 的剩余行，DeleteBefore 会把没归档的也
//     一并删掉造成数据丢失（T-0100-P4-B2 修复 review 922d87a4 WARNING #3）
//  6. 任一步失败 → return error，不删 DB（下次 tick 重试，幂等）
package license

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// archiveObjectIO 是 LogArchiver 消费的 MinIO 窄接口。consumer-side 定义
// 让测试 mock 只需 ~20 行；*minio.Client 天然实现。
//
// 仅依赖 PutObject：键名走 tick 唯一化方案后不再需要 StatObject 探活
// （详见 package doc T-0100-P4-B1 修复说明）。
type archiveObjectIO interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

// LogArchiver 归档器。零值不可用；通过 NewLogArchiver 构造。
type LogArchiver struct {
	repo            LicenseLogRepository
	minio           archiveObjectIO
	bucket          string
	retentionMonths int
	logger          *zap.Logger
	batchSize       int
}

// NewLogArchiver 构造归档器。retentionMonths ≤ 0 时使用默认 6（PRD §5.4.5）。
// minio 可空（dev / 单测时禁用归档），传 nil 则 ArchiveOnce 短路返 nil。
func NewLogArchiver(
	repo LicenseLogRepository,
	mc archiveObjectIO,
	bucket string,
	retentionMonths int,
	logger *zap.Logger,
) *LogArchiver {
	if retentionMonths <= 0 {
		retentionMonths = 6
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LogArchiver{
		repo:            repo,
		minio:           mc,
		bucket:          bucket,
		retentionMonths: retentionMonths,
		logger:          logger.Named("license-log-archiver"),
		batchSize:       10000, // 单 tick 上限；超出则下次 tick 接力
	}
}

// SetBatchSize 调整 ListBefore 单次拉取上限（测试用）。
func (a *LogArchiver) SetBatchSize(n int) {
	if n > 0 {
		a.batchSize = n
	}
}

// ArchiveResult 归档结果（监控 / 日志摘要）。
type ArchiveResult struct {
	Cutoff       time.Time
	MonthsBucket int   // 涉及月数
	LogsArchived int   // 写入 MinIO 的日志条数
	BytesWritten int64 // 写入 MinIO 的总字节数（gzip 后）
	LogsDeleted  int64 // 从 DB 删除的行数（应等于 LogsArchived）
}

// ArchiveOnce 跑一次归档：list → group → upload → delete。幂等。
//
// 设计取舍：
//   - minio 为 nil 或 bucket 为空 → 短路返 nil（dev 配置友好）
//   - 键名按 tick 唯一化（YYYY-MM/{tickTS}-{count}.jsonl.gz）：同月跨 tick
//     归档不会互相覆盖（T-0100-P4-B1 修复 review CRITICAL #1）
//   - 删除 DB 在所有月归档成功后才执行，保证"先持久化再删原表"原子性
//     （MinIO 失败 → DB 不变 → 下 tick 重试不丢数据）
func (a *LogArchiver) ArchiveOnce(ctx context.Context) (*ArchiveResult, error) {
	if a.minio == nil || a.bucket == "" {
		a.logger.Debug("license log archiver disabled (no MinIO or bucket)")
		return &ArchiveResult{}, nil
	}

	now := nowFunc()
	cutoff := now.AddDate(0, -a.retentionMonths, 0).UTC()
	tickTS := now.UTC().Format("20060102T150405Z") // T-0100-P4-B1：tick 唯一标识

	logs, err := a.repo.ListBefore(ctx, cutoff, a.batchSize)
	if err != nil {
		return nil, fmt.Errorf("list logs before cutoff: %w", err)
	}
	if len(logs) == 0 {
		a.logger.Debug("no license logs to archive",
			zap.Time("cutoff", cutoff))
		return &ArchiveResult{Cutoff: cutoff}, nil
	}

	// 按 YYYY-MM 分桶
	groups := groupLogsByMonth(logs)

	var totalBytes int64
	for month, monthLogs := range groups {
		// T-0100-P4-B1：键名 = `license-logs/{YYYY-MM}/{tickTS}-{count}.jsonl.gz`
		// {tickTS} 防同月跨 tick 覆盖；{count} 让运维直接从 key 读出条数。
		key := fmt.Sprintf("license-logs/%s/%s-%d.jsonl.gz", month, tickTS, len(monthLogs))
		body, err := encodeLogsJSONLGz(monthLogs)
		if err != nil {
			return nil, fmt.Errorf("encode logs for %s: %w", month, err)
		}
		info, err := a.minio.PutObject(ctx, a.bucket, key,
			bytes.NewReader(body), int64(len(body)),
			minio.PutObjectOptions{ContentType: "application/gzip"})
		if err != nil {
			return nil, fmt.Errorf("put archive object %s: %w", key, err)
		}
		totalBytes += info.Size
		a.logger.Info("license logs archived to MinIO",
			zap.String("month", month),
			zap.String("bucket", a.bucket),
			zap.String("key", key),
			zap.Int("log_count", len(monthLogs)),
			zap.Int64("bytes_written", info.Size))
	}

	// T-0100-P4-B2：仅删本 tick 实际归档的行（按 id），不要按 cutoff 一刀切，
	// 防止 ListBefore 受 batchSize 限制只读了一部分时把剩余未归档行也误删。
	archivedIDs := make([]uuid.UUID, 0, len(logs))
	for i := range logs {
		archivedIDs = append(archivedIDs, logs[i].ID)
	}
	deleted, err := a.repo.DeleteByIDs(ctx, archivedIDs)
	if err != nil {
		return nil, fmt.Errorf("delete archived logs from DB: %w", err)
	}

	res := &ArchiveResult{
		Cutoff:       cutoff,
		MonthsBucket: len(groups),
		LogsArchived: len(logs),
		BytesWritten: totalBytes,
		LogsDeleted:  deleted,
	}
	a.logger.Info("license log archive run complete",
		zap.Time("cutoff", res.Cutoff),
		zap.Int("months", res.MonthsBucket),
		zap.Int("logs_archived", res.LogsArchived),
		zap.Int64("bytes_written", res.BytesWritten),
		zap.Int64("logs_deleted", res.LogsDeleted))
	return res, nil
}

// groupLogsByMonth 按 created_at 的 UTC YYYY-MM 分组；每组按 created_at 升序。
func groupLogsByMonth(logs []LicenseLog) map[string][]LicenseLog {
	groups := map[string][]LicenseLog{}
	for _, l := range logs {
		key := l.CreatedAt.UTC().Format("2006-01")
		groups[key] = append(groups[key], l)
	}
	for k := range groups {
		sort.Slice(groups[k], func(i, j int) bool {
			return groups[k][i].CreatedAt.Before(groups[k][j].CreatedAt)
		})
	}
	return groups
}

// encodeLogsJSONLGz 把 []LicenseLog 渲染为 gzip(JSONL)：每行一条 JSON 记录。
//
// JSONL 选型理由：
//   - 流式追加友好（单行 atomic，便于审计工具按行 grep）
//   - vs 单大 JSON array：append 不需要先解压再 splice 再压缩
//   - vs CSV：保留 details JSONB 嵌套结构
func encodeLogsJSONLGz(logs []LicenseLog) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gw)
	for i := range logs {
		if err := enc.Encode(&logs[i]); err != nil {
			_ = gw.Close()
			return nil, fmt.Errorf("encode log %d: %w", i, err)
		}
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}
	return buf.Bytes(), nil
}
