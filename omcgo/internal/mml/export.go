package mml

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/pkg/soap"
)

// ErrExporterNotConfigured 当 MinIO 导出器未注入时返回（dev/单测场景），handler 映射 503。
var ErrExporterNotConfigured = errors.New("mml: result exporter not configured")

// Exporter 把 MML 任务结果 CSV 上传到 MinIO 并签发下载 URL。
//   - put：内部客户端（走 docker 内网 endpoint），用于 PutObject
//   - sign：预签名客户端（走 public_endpoint），用于浏览器可达的 PresignedGetObject
//   - bucket：reports bucket；对象统一落在 mml-results/ 独立目录下
type Exporter struct {
	put    *minio.Client
	sign   *minio.Client
	bucket string
	logger *zap.Logger
}

// NewExporter 构造导出器。put/sign 任一为 nil 视为未配置（ExportXxx 返回 503）。
func NewExporter(put, sign *minio.Client, bucket string, logger *zap.Logger) *Exporter {
	return &Exporter{put: put, sign: sign, bucket: bucket, logger: logger}
}

func (e *Exporter) ready() bool {
	return e != nil && e.put != nil && e.sign != nil && e.bucket != ""
}

func (e *Exporter) upload(ctx context.Context, key string, data []byte) error {
	_, err := e.put.PutObject(ctx, e.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "text/csv; charset=utf-8"})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (e *Exporter) presign(ctx context.Context, key string) (string, error) {
	u, err := e.sign.PresignedGetObject(ctx, e.bucket, key, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return u.String(), nil
}

// objectKeyPrefix 生成任务专属的独立目录前缀：mml-results/{YYYY}/{MM}/{DD}/{taskID}/。
func objectKeyPrefix(taskID uuid.UUID, t time.Time) string {
	return fmt.Sprintf("mml-results/%04d/%02d/%02d/%s/", t.Year(), int(t.Month()), t.Day(), taskID.String())
}

func aggregateObjectKey(taskID uuid.UUID, t time.Time) string {
	return objectKeyPrefix(taskID, t) + fmt.Sprintf("aggregate-%s.csv", short8(taskID))
}

func deviceObjectKey(taskID uuid.UUID, deviceSN string, t time.Time) string {
	return objectKeyPrefix(taskID, t) + fmt.Sprintf("device-%s-%s.csv", sanitizeSN(deviceSN), short8(taskID))
}

func short8(id uuid.UUID) string {
	s := id.String()
	if len(s) >= 8 {
		return s[:8]
	}
	return s
}

// sanitizeSN 把 SN 中可能的路径分隔符等替换为 '_'，避免破坏 object key 层级。
func sanitizeSN(sn string) string {
	return strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(sn)
}

// ── CSV 生成（完整参数矩阵；列 = 命令查询的 standardPath） ──────────────────────

// exportColumn 一列：standardPath（表头）+ privatePath（GPV 回值的实际 key）。
type exportColumn struct {
	standard string
	private  string
}

// exportColumnsFromTask 取命令查询的参数列。优先用 param_refs 的 standard/private 对，
// 这样 GPV 回的 privatePath 能精确对齐到 standardPath 列；缺 param_refs 时回退 param_paths。
func exportColumnsFromTask(commands []map[string]interface{}) []exportColumn {
	views := extractPathTranslations(commands) // 已按 standardPath 排序去重
	if len(views) > 0 {
		cols := make([]exportColumn, 0, len(views))
		for _, v := range views {
			cols = append(cols, exportColumn{standard: v.StandardPath, private: v.PrivatePath})
		}
		return cols
	}
	// 回退：从首条命令的 param_paths 读（standard=private）。
	for _, entry := range commands {
		if pp, ok := entry["param_paths"].([]interface{}); ok {
			cols := make([]exportColumn, 0, len(pp))
			for _, p := range pp {
				if s, ok := p.(string); ok && s != "" {
					cols = append(cols, exportColumn{standard: s, private: s})
				}
			}
			if len(cols) > 0 {
				return cols
			}
		}
	}
	return nil
}

func taskIsRead(commands []map[string]interface{}) bool {
	for _, entry := range commands {
		if op, ok := entry["operation_type"].(string); ok {
			return op == "LST" || op == "DSP"
		}
	}
	return false
}

// exportRow 一台设备一行的导出数据。
type exportRow struct {
	sn         string
	statusText string
	values     map[string]string // standardPath -> 读回值（读类）
	faultCode  string
	faultMsg   string
	sentAt     string
	completedAt string
}

func deviceStatusText(status string, errorCode int) string {
	if status == "completed" && errorCode == 0 {
		return "成功"
	}
	switch status {
	case "completed":
		return "成功"
	case "failed":
		return "失败"
	case "expired":
		return "超时"
	case "pending", "running":
		return "执行中"
	default:
		if status == "" {
			return "未知"
		}
		return status
	}
}

func leafOf(path string) string {
	parts := strings.Split(path, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return path
}

// rowToExport 把 device_tasks 结果行转为导出行；读类解析 GPV 原始报文回填列值。
func rowToExport(row DeviceTaskResultRowView, cols []exportColumn, read bool) exportRow {
	er := exportRow{
		sn:         row.DeviceSN,
		statusText: deviceStatusText(row.Status, row.ErrorCode),
		values:     map[string]string{},
		faultMsg:   row.ErrorMessage,
	}
	if row.ErrorCode != 0 {
		er.faultCode = strconv.Itoa(row.ErrorCode)
	}
	if row.SentAt != nil {
		er.sentAt = row.SentAt.Format("2006-01-02 15:04:05")
	}
	if row.CompletedAt != nil {
		er.completedAt = row.CompletedAt.Format("2006-01-02 15:04:05")
	}

	if read && row.Status == "completed" {
		raw := rawResponseOf(row.Result)
		if raw != "" {
			if pvs, _, err := soap.DecodeGetParameterValuesResponse(strings.NewReader(raw)); err == nil {
				byPath := make(map[string]string, len(pvs))
				byLeaf := make(map[string]string, len(pvs))
				for _, pv := range pvs {
					byPath[pv.Name] = pv.Value
					byLeaf[leafOf(pv.Name)] = pv.Value
				}
				// 列对齐：优先 privatePath（GPV 实际 key）→ standardPath → 叶子名兜底。
				for _, c := range cols {
					if v, ok := byPath[c.private]; ok {
						er.values[c.standard] = v
					} else if v, ok := byPath[c.standard]; ok {
						er.values[c.standard] = v
					} else if v, ok := byLeaf[leafOf(c.private)]; ok {
						er.values[c.standard] = v
					}
				}
			}
		}
	}
	return er
}

// rawResponseOf 从 device_tasks.result JSONB 抽出 raw_response（CWMP SOAP 原文）。
func rawResponseOf(result json.RawMessage) string {
	if len(result) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		return ""
	}
	if rr, ok := m["raw_response"].(string); ok {
		return rr
	}
	return ""
}

const utf8BOM = "\xEF\xBB\xBF"

// buildAggregateCSV 全设备汇总：行=设备，列=序号/设备SN/状态/<参数列…>/故障码/故障信息/下发时间/响应时间。
func buildAggregateCSV(cols []exportColumn, rows []exportRow) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(utf8BOM) // Excel 正确识别中文
	w := csv.NewWriter(&buf)

	header := []string{"序号", "设备SN", "状态"}
	for _, c := range cols {
		header = append(header, c.standard)
	}
	header = append(header, "故障码", "故障信息", "下发时间", "响应时间")
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for i, r := range rows {
		rec := []string{strconv.Itoa(i + 1), r.sn, r.statusText}
		for _, c := range cols {
			rec = append(rec, r.values[c.standard])
		}
		rec = append(rec, r.faultCode, r.faultMsg, r.sentAt, r.completedAt)
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildDeviceCSV 单设备：纵向「参数路径,读回值」表 + 顶部设备/状态摘要。
func buildDeviceCSV(cols []exportColumn, r exportRow) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	w := csv.NewWriter(&buf)

	summary := [][]string{
		{"设备SN", r.sn},
		{"状态", r.statusText},
		{"故障码", r.faultCode},
		{"故障信息", r.faultMsg},
		{"下发时间", r.sentAt},
		{"响应时间", r.completedAt},
		{},
		{"参数路径", "读回值"},
	}
	for _, line := range summary {
		if err := w.Write(line); err != nil {
			return nil, err
		}
	}
	for _, c := range cols {
		if err := w.Write([]string{c.standard, r.values[c.standard]}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ── Service 导出方法 ──────────────────────────────────────────────────────────

// SetExporter 注入结果导出器（provider 装配）。
func (s *Service) SetExporter(e *Exporter) { s.exporter = e }

// allDeviceResults 取任务全部设备结果（不分页；单批上限 200 台，远低于硬顶）。
func (s *Service) allDeviceResults(ctx context.Context, taskID uuid.UUID) ([]DeviceTaskResultRowView, error) {
	if s.deviceTaskResultLister == nil {
		return nil, ErrExporterNotConfigured
	}
	rows, _, err := s.deviceTaskResultLister.ListResultsBySourceID(ctx, taskID.String(), 1, 100000)
	if err != nil {
		return nil, fmt.Errorf("list device task results: %w", err)
	}
	return rows, nil
}

// ExportTaskResultsCSV 生成「全设备汇总」CSV，上传 MinIO，并把 object key 记录到 mml_tasks。
// 返回 object key 与浏览器可下载的预签名 URL。
func (s *Service) ExportTaskResultsCSV(ctx context.Context, taskID uuid.UUID) (objectKey, downloadURL string, err error) {
	if !s.exporter.ready() {
		return "", "", ErrExporterNotConfigured
	}
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return "", "", fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, taskID)
	if err != nil {
		return "", "", err
	}
	exportRows := make([]exportRow, 0, len(rows))
	for _, r := range rows {
		exportRows = append(exportRows, rowToExport(r, cols, read))
	}
	data, err := buildAggregateCSV(cols, exportRows)
	if err != nil {
		return "", "", fmt.Errorf("build csv: %w", err)
	}
	now := time.Now().UTC()
	key := aggregateObjectKey(taskID, now)
	if err := s.exporter.upload(ctx, key, data); err != nil {
		return "", "", err
	}
	if err := s.taskRepo.UpdateExportAggregate(ctx, taskID, key, now); err != nil {
		s.logger.Warn("record mml_tasks export_object failed", zap.String("task_id", taskID.String()), zap.Error(err))
	}
	url, err := s.exporter.presign(ctx, key)
	if err != nil {
		return key, "", err
	}
	return key, url, nil
}

// ExportTaskDeviceCSV 生成「单设备」CSV，上传 MinIO，并把 object key 记入 mml_tasks 的
// device_export_objects 映射。返回 object key 与预签名 URL。
func (s *Service) ExportTaskDeviceCSV(ctx context.Context, taskID uuid.UUID, deviceSN string) (objectKey, downloadURL string, err error) {
	if !s.exporter.ready() {
		return "", "", ErrExporterNotConfigured
	}
	if deviceSN == "" {
		return "", "", commonInvalidDeviceSN
	}
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return "", "", fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, taskID)
	if err != nil {
		return "", "", err
	}
	var matched *DeviceTaskResultRowView
	for i := range rows {
		if rows[i].DeviceSN == deviceSN {
			matched = &rows[i]
			break
		}
	}
	if matched == nil {
		return "", "", commonDeviceResultNotFound
	}
	data, err := buildDeviceCSV(cols, rowToExport(*matched, cols, read))
	if err != nil {
		return "", "", fmt.Errorf("build csv: %w", err)
	}
	now := time.Now().UTC()
	key := deviceObjectKey(taskID, deviceSN, now)
	if err := s.exporter.upload(ctx, key, data); err != nil {
		return "", "", err
	}
	if err := s.taskRepo.UpdateExportDevice(ctx, taskID, deviceSN, key, now); err != nil {
		s.logger.Warn("record mml_tasks device export failed",
			zap.String("task_id", taskID.String()), zap.String("device_sn", deviceSN), zap.Error(err))
	}
	url, err := s.exporter.presign(ctx, key)
	if err != nil {
		return key, "", err
	}
	return key, url, nil
}

var (
	commonInvalidDeviceSN      = errors.New("mml: device_sn is required")
	commonDeviceResultNotFound = errors.New("mml: device result not found in task")
)
