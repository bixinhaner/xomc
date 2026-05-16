package trace

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Exporter worker 进程订阅 trace.export.requested，生成单个 XML 文件写 MinIO exchange bucket。
//
// 流程：
//  1. 读 export_job（确认状态 queued）+ 读 task（拿 device_sn）
//  2. 分页拉 task 下所有 messages（外置报文走 BulkStore.Get 回拉到 inline）
//  3. 拼 XML（与 handler.ExportXML 同款，但写到 MinIO exchange 而非 HTTP 流）
//  4. 更新 export_job：status=done / object_key / size_bytes / completed_at
//  5. 任一步失败：status=failed + error_message
//
// 单 worker 实例顺序处理；多 worker 实例由 WorkQueuePolicy 自动 fan-out。
type Exporter struct {
	repo     Repository
	bulk     *BulkStore // 用于回读外置报文（可 nil — 任务里若有 external 报文则该 job 失败）
	minio    *minio.Client
	bucket   string // exchange bucket 名（M2 默认 omc-exchange）
	logger   *zap.Logger
	queueName string

	pageSize int
	maxRows  int // 单文件硬上限，超过 → failed（M3 切流式分片导出）

	completed uint64
	failed    uint64
}

// ExporterConfig 配置。
type ExporterConfig struct {
	Bucket    string // exchange bucket，必填
	PageSize  int    // 默认 500
	MaxRows   int    // 单文件硬上限；默认 200000（约 100MB）
	QueueName string // 默认 "trace-export"
}

// DefaultExporterConfig 默认配置。
func DefaultExporterConfig(bucket string) ExporterConfig {
	return ExporterConfig{
		Bucket:    bucket,
		PageSize:  500,
		MaxRows:   200000,
		QueueName: "trace-export",
	}
}

// NewExporter 构造函数。
func NewExporter(repo Repository, bulk *BulkStore, minioClient *minio.Client, cfg ExporterConfig, logger *zap.Logger) *Exporter {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 500
	}
	if cfg.MaxRows <= 0 {
		cfg.MaxRows = 200000
	}
	if cfg.QueueName == "" {
		cfg.QueueName = "trace-export"
	}
	return &Exporter{
		repo:      repo,
		bulk:      bulk,
		minio:     minioClient,
		bucket:    cfg.Bucket,
		logger:    logger.Named("trace-exporter"),
		queueName: cfg.QueueName,
		pageSize:  cfg.PageSize,
		maxRows:   cfg.MaxRows,
	}
}

// Subscribe 订阅 trace.export.requested。
func (e *Exporter) Subscribe(bus event.EventBus) (event.Subscription, error) {
	return bus.QueueSubscribe(event.SubjectTraceExportRequested, e.queueName, e.handle)
}

// CompletedCount 累计成功导出。
func (e *Exporter) CompletedCount() uint64 { return atomic.LoadUint64(&e.completed) }

// FailedCount 累计失败导出。
func (e *Exporter) FailedCount() uint64 { return atomic.LoadUint64(&e.failed) }

func (e *Exporter) handle(_ context.Context, evt event.Event) error {
	var payload ExportRequestedEvent
	if err := evt.DecodePayload(&payload); err != nil {
		e.logger.Warn("trace exporter: decode event failed", zap.Error(err))
		return nil
	}
	if payload.JobID == uuid.Nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	e.run(ctx, payload.JobID)
	return nil
}

func (e *Exporter) run(ctx context.Context, jobID uuid.UUID) {
	job, err := e.repo.GetExportJob(ctx, jobID)
	if err != nil {
		e.logger.Warn("trace exporter: load job failed",
			zap.String("job_id", jobID.String()), zap.Error(err))
		return
	}
	if job.Status != ExportJobQueued {
		// 已被其他 worker 处理过 — 防重复
		return
	}
	now := time.Now()
	job.Status = ExportJobRunning
	job.StartedAt = &now
	if err := e.repo.UpdateExportJob(ctx, job); err != nil {
		e.logger.Warn("trace exporter: mark running failed",
			zap.String("job_id", jobID.String()), zap.Error(err))
		return
	}

	task, err := e.repo.GetTask(ctx, job.TaskID)
	if err != nil {
		e.failJob(ctx, job, fmt.Sprintf("load task: %v", err))
		return
	}

	xml, count, err := e.buildXML(ctx, task)
	if err != nil {
		e.failJob(ctx, job, err.Error())
		return
	}

	// 写 MinIO exchange/trace-export/{job_id}.xml。
	// Content-Type 故意用 application/octet-stream（而非 application/xml）：
	// 浏览器对 XML MIME 会忽略 <a download> 属性、改在新 tab inline 渲染，
	// 导致用户找不到下载文件。在预签名 URL 上覆盖 response-content-type 会
	// 破坏 SigV4 签名（实测 403 SignatureDoesNotMatch），所以在上传源对象时
	// 就定 octet-stream 一劳永逸。
	key := fmt.Sprintf("trace-export/%s.xml", job.ID.String())
	if _, err := e.minio.PutObject(ctx, e.bucket, key,
		bytes.NewReader(xml), int64(len(xml)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"}); err != nil {
		e.failJob(ctx, job, fmt.Sprintf("put object: %v", err))
		return
	}

	completed := time.Now()
	job.Status = ExportJobDone
	job.ObjectBucket = e.bucket
	job.ObjectKey = key
	job.MessageCount = count
	job.SizeBytes = int64(len(xml))
	job.CompletedAt = &completed
	if err := e.repo.UpdateExportJob(ctx, job); err != nil {
		e.logger.Warn("trace exporter: mark done failed",
			zap.String("job_id", jobID.String()), zap.Error(err))
		return
	}
	atomic.AddUint64(&e.completed, 1)
	e.logger.Info("trace export job done",
		zap.String("job_id", jobID.String()),
		zap.String("task_id", job.TaskID.String()),
		zap.Int("count", count),
		zap.Int64("size_bytes", int64(len(xml))))
}

func (e *Exporter) failJob(ctx context.Context, job *ExportJob, msg string) {
	now := time.Now()
	job.Status = ExportJobFailed
	job.ErrorMessage = msg
	job.CompletedAt = &now
	if err := e.repo.UpdateExportJob(ctx, job); err != nil {
		e.logger.Warn("trace exporter: mark failed failed",
			zap.String("job_id", job.ID.String()), zap.Error(err))
	}
	atomic.AddUint64(&e.failed, 1)
	e.logger.Warn("trace export job failed",
		zap.String("job_id", job.ID.String()), zap.String("reason", msg))
}

// buildXML 分页拉所有 messages → 拼 XML（CDATA 包裹原文）。
// external 报文走 BulkStore 回读；BulkStore 未注入时 external 报文输出 placeholder 注释。
func (e *Exporter) buildXML(ctx context.Context, task *Task) ([]byte, int, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<TraceExport task_id="` + task.ID.String() + `" device_sn="` + task.DeviceSN + `">` + "\n")

	count := 0
	page := 1
	for {
		listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		listResp, err := e.repo.ListMessages(listCtx, MessageFilter{
			TaskID: task.ID,
			ListRequest: model.ListRequest{
				Page:     page,
				PageSize: e.pageSize,
				SortDir:  "asc",
			},
		})
		cancel()
		if err != nil {
			return nil, count, fmt.Errorf("list messages page %d: %w", page, err)
		}
		if len(listResp.Items) == 0 {
			break
		}
		for i := range listResp.Items {
			msg := &listResp.Items[i]
			payload, fetchErr := e.fetchPayload(ctx, msg)
			if fetchErr != nil {
				e.logger.Warn("trace exporter: fetch external payload failed",
					zap.String("msg_id", msg.ID.String()), zap.Error(fetchErr))
			}
			writeMessageXML(&buf, msg, payload)
			count++
			if count > e.maxRows {
				return nil, count, fmt.Errorf("message count exceeds limit %d (use M3 streaming export)", e.maxRows)
			}
		}
		if len(listResp.Items) < e.pageSize {
			break
		}
		page++
	}
	buf.WriteString("</TraceExport>\n")
	return buf.Bytes(), count, nil
}

func (e *Exporter) fetchPayload(ctx context.Context, m *Message) (string, error) {
	if m.PayloadInline != "" {
		return m.PayloadInline, nil
	}
	if m.PayloadObjectKey == "" {
		return "", nil
	}
	if e.bulk == nil || !e.bulk.Enabled() {
		return "<!-- external payload unavailable: bulk store disabled -->", nil
	}
	return e.bulk.Get(ctx, m.PayloadObjectKey)
}

func writeMessageXML(buf *bytes.Buffer, m *Message, payload string) {
	buf.WriteString(`<Message captured_at="` + m.CapturedAt.Format(time.RFC3339Nano) +
		`" direction="` + string(m.Direction) + `"`)
	if m.RPCMethod != "" {
		buf.WriteString(` rpc_method="` + m.RPCMethod + `"`)
	}
	if m.CwmpID != "" {
		buf.WriteString(` cwmp_id="` + m.CwmpID + `"`)
	}
	buf.WriteString(">\n")
	buf.WriteString("<![CDATA[")
	// CDATA 内禁止出现 "]]>"；做一次安全替换
	buf.WriteString(strings.ReplaceAll(payload, "]]>", "]]]]><![CDATA[>"))
	buf.WriteString("]]>\n</Message>\n")
}

