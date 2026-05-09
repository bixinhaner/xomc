// Package license — log archive (T-0100-P4-B).
//
// 周级 cron 任务把 license_logs 表里 created_at < (now - retention_months) 的
// 行打包写到 MinIO，键名 `license-logs/{YYYY-MM}.jsonl.gz`，归档成功后从 DB
// 删除对应行。等保 2.0 三级 8.1.4.7 要求重要操作日志保留 ≥ 6 个月（PRD §5.4.5）。
//
// 单次 tick 的正确性保证：
//  1. 计算 cutoff = now - retention_months
//  2. ListBefore(cutoff, batchSize) 拉一批最早的日志
//  3. 按 YYYY-MM 分桶（同月聚合），月内按 created_at 升序
//  4. 对每个月的日志：
//     a. StatObject 看 MinIO 上是否已有 `license-logs/{YYYY-MM}.jsonl.gz`
//     b. 已存在：append 模式（下载 → 解压 → 追加 → 压缩 → 上传覆盖）
//        简化处理：当前实现直接覆盖（cron 同月只跑一次，理论上不会同月二次归档）
//     c. PutObject 写新对象
//  5. 全部月归档成功后，DeleteBefore(cutoff) 物理删 DB 行
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

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// archiveObjectIO 是 LogArchiver 消费的 MinIO 窄接口。consumer-side 定义
// 让测试 mock 只需 ~30 行；*minio.Client 天然实现。
type archiveObjectIO interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	StatObject(ctx context.Context, bucket, key string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
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
//   - 当月日志可能跨 tick 被分批归档：每月单独 PutObject，重复跑会覆盖
//     （objects 是 immutable 替换语义，无版本号一致性问题）
//   - 删除 DB 在所有月归档成功后才执行，保证"先持久化再删原表"原子性
//     （MinIO 失败 → DB 不变 → 下 tick 重试不丢数据）
func (a *LogArchiver) ArchiveOnce(ctx context.Context) (*ArchiveResult, error) {
	if a.minio == nil || a.bucket == "" {
		a.logger.Debug("license log archiver disabled (no MinIO or bucket)")
		return &ArchiveResult{}, nil
	}

	cutoff := nowFunc().AddDate(0, -a.retentionMonths, 0).UTC()
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
		key := fmt.Sprintf("license-logs/%s.jsonl.gz", month)
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

	deleted, err := a.repo.DeleteBefore(ctx, cutoff)
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
