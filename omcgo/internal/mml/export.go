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

// PresignClientProvider 抽象"按需取当前 MinIO 预签名 client"的能力（issue #548 切片 4）。
// 主线生产实现是 internal/core/components/minio.PresignBridge；sys_configs 写入
// storage.minio_public_endpoint 后下一次 Get() 拿到新 endpoint 对应的 client。
type PresignClientProvider interface {
	Get() *minio.Client
}

// Exporter 把 MML 任务结果 CSV 上传到 MinIO 并签发下载 URL。
//   - put：内部客户端（走 docker 内网 endpoint），用于 PutObject
//   - sign：默认预签名客户端（走 public_endpoint），作 PresignedGetObject
//     启动期兑底；运行期优先使用 signProvider.Get()（sys_configs 热改生效）
//   - bucket：reports bucket；对象统一落在 mml-results/ 独立目录下
type Exporter struct {
	put          *minio.Client
	sign         *minio.Client         // 启动期默认。signProvider != nil 且 Get()!=nil 时优先走 provider。
	signProvider PresignClientProvider // issue #548 切片 4：sys_configs 热改 endpoint 后下次 presign 即生效
	bucket       string
	logger       *zap.Logger
}

// NewExporter 构造导出器。put/sign 任一为 nil 视为未配置（ExportXxx 返回 503）。
func NewExporter(put, sign *minio.Client, bucket string, logger *zap.Logger) *Exporter {
	return &Exporter{put: put, sign: sign, bucket: bucket, logger: logger}
}

// SetSignProvider 注入 运行期感知 sys_configs 变更的 presign client provider（issue #548 切片 4）。
// nil 时回退使用 e.sign（启动期静态 client）。
func (e *Exporter) SetSignProvider(p PresignClientProvider) {
	if e == nil {
		return
	}
	e.signProvider = p
}

// signClient 返回当前该用于签发预签名 URL 的 client：优先 provider.Get()，其次 e.sign。
func (e *Exporter) signClient() *minio.Client {
	if e.signProvider != nil {
		if c := e.signProvider.Get(); c != nil {
			return c
		}
	}
	return e.sign
}

func (e *Exporter) ready() bool {
	return e != nil && e.put != nil && e.bucket != "" && e.signClient() != nil
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
	u, err := e.signClient().PresignedGetObject(ctx, e.bucket, key, time.Hour, nil)
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
	sn          string
	statusText  string
	values      map[string]string // standardPath -> 读回值（读类）
	faultCode   string
	faultMsg    string
	sentAt      string
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

// commandStandardPaths 取 commands[ci] 命令覆盖的 standardPath 列表（逐 PATH 时每命令 1 条）。
// 优先 param_refs[].tr069_path（standardPath），缺则回退 param_paths。
func commandStandardPaths(commands []map[string]interface{}, ci int) []string {
	if ci < 0 || ci >= len(commands) {
		return nil
	}
	entry := commands[ci]
	var out []string
	if refs, ok := entry["param_refs"].([]interface{}); ok {
		for _, r := range refs {
			if m, ok := r.(map[string]interface{}); ok {
				if p, ok := m["tr069_path"].(string); ok && p != "" {
					out = append(out, p)
				}
			}
		}
	}
	if len(out) == 0 {
		if pp, ok := entry["param_paths"].([]interface{}); ok {
			for _, p := range pp {
				if s, ok := p.(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// commandPathNames 从 commands 的 param_refs 抽 standardPath → 参数名称（param_name_zh）映射。
func commandPathNames(commands []map[string]interface{}) map[string]string {
	names := make(map[string]string)
	for _, entry := range commands {
		refs, ok := entry["param_refs"].([]interface{})
		if !ok {
			continue
		}
		for _, r := range refs {
			m, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			path, _ := m["tr069_path"].(string)
			name, _ := m["param_name_zh"].(string)
			if path != "" && name != "" {
				names[path] = name
			}
		}
	}
	return names
}

// commandStringField 取 commands[ci] 的字符串字段（command_code / operation_type 等）。
func commandStringField(commands []map[string]interface{}, ci int, key string) string {
	if ci < 0 || ci >= len(commands) {
		return ""
	}
	v, _ := commands[ci][key].(string)
	return v
}

// execModeLabel 推导执行模式：逐 PATH 拆成 N 条 command（每条 1 path）；整体执行为单条 command。
func execModeLabel(commands []map[string]interface{}) string {
	if len(commands) > 1 {
		return "逐 PATH"
	}
	return "整体执行"
}

// flattenXML 把多行 XML 报文压成一行（空白折叠为单空格），便于放进 CSV 单元格（方式 B）。
func flattenXML(s string) string {
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

// longFormatHeader 是多设备汇总长表表头。
var longFormatHeader = []string{
	"序号", "设备SN", "命令", "操作类型", "执行模式", "子任务ID",
	"参数路径", "参数名称", "状态", "读回值", "故障码", "故障信息",
	"下发时间", "响应时间", "结果报文(XML)",
}

// buildLongFormatCSV 多设备汇总（长表，一行 = 设备 × PATH）：
//   - 设备级公共字段（序号/设备SN/命令/操作类型/执行模式）只在该设备首行填；
//   - 子任务级字段（子任务ID/下发/响应/结果报文）只在该 device_task 首行填（整体执行=1 个 RPC→只首行；
//     逐 PATH=N 个 RPC→各自首行），报文压一行不重复；
//   - PATH 级字段（参数路径/参数名称/状态/读回值/故障码/故障信息）每行都填；
//   - 设备之间空一行。
func buildLongFormatCSV(
	cols []exportColumn, rows []DeviceTaskResultRowView, commands []map[string]interface{},
	nameMap map[string]string, read bool,
) ([]byte, error) {
	execMode := execModeLabel(commands)

	order := make([]string, 0)
	groups := make(map[string][]DeviceTaskResultRowView)
	for i := range rows {
		sn := rows[i].DeviceSN
		if _, ok := groups[sn]; !ok {
			order = append(order, sn)
		}
		groups[sn] = append(groups[sn], rows[i])
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	w := csv.NewWriter(&buf)
	if err := w.Write(longFormatHeader); err != nil {
		return nil, err
	}

	deviceSeq := 0
	for di, sn := range order {
		if di > 0 {
			if err := w.Write(make([]string, len(longFormatHeader))); err != nil { // 设备间空行
				return nil, err
			}
		}
		deviceSeq++
		firstRowOfDevice := true
		for _, row := range groups[sn] {
			er := rowToExport(row, cols, read)
			status := deviceStatusText(row.Status, row.ErrorCode)
			faultCode, faultMsg := "", ""
			if row.ErrorCode != 0 {
				faultCode = strconv.Itoa(row.ErrorCode)
				faultMsg = row.ErrorMessage
			}
			raw := flattenXML(rawResponseOf(row.Result))
			paths := commandStandardPaths(commands, row.CommandIndex)
			firstRowOfTask := true
			for _, p := range paths {
				rec := make([]string, len(longFormatHeader))
				if firstRowOfDevice { // 设备级公共字段：仅设备首行
					rec[0] = strconv.Itoa(deviceSeq)
					rec[1] = sn
					rec[2] = commandStringField(commands, row.CommandIndex, "command_code")
					rec[3] = commandStringField(commands, row.CommandIndex, "operation_type")
					rec[4] = execMode
					firstRowOfDevice = false
				}
				if firstRowOfTask { // 子任务级字段：仅该 device_task 首行（整体执行不重复报文）
					rec[5] = row.DeviceTaskID
					rec[12] = er.sentAt
					rec[13] = er.completedAt
					rec[14] = raw
					firstRowOfTask = false
				}
				rec[6] = p // PATH 级：每行
				rec[7] = nameMap[p]
				rec[8] = status
				rec[9] = er.values[p]
				rec[10] = faultCode
				rec[11] = faultMsg
				if err := w.Write(rec); err != nil {
					return nil, err
				}
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildDeviceCSVMulti 单设备结果：顶部设备摘要（公共字段一次）+ 逐 PATH 明细表
// （参数路径/参数名称/状态/读回值/故障码/故障信息/子任务ID/下发/响应/结果报文）。
// 子任务级字段只在该 device_task 首行填（整体执行报文只首行、不重复）。
func buildDeviceCSVMulti(
	cols []exportColumn, devRows []DeviceTaskResultRowView, commands []map[string]interface{},
	nameMap map[string]string, read bool,
) ([]byte, error) {
	// 设备级状态：全成功=成功 / 全失败=失败 / 混合=部分失败。
	anySucc, anyFail := false, false
	for _, row := range devRows {
		if row.Status == "completed" && row.ErrorCode == 0 {
			anySucc = true
		} else {
			anyFail = true
		}
	}
	devStatus := "成功"
	switch {
	case anyFail && anySucc:
		devStatus = "部分失败"
	case anyFail:
		devStatus = "失败"
	}

	cmdCode := ""
	opType := ""
	if len(devRows) > 0 {
		cmdCode = commandStringField(commands, devRows[0].CommandIndex, "command_code")
		opType = commandStringField(commands, devRows[0].CommandIndex, "operation_type")
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	w := csv.NewWriter(&buf)
	// 顶部设备摘要（公共字段一次）。
	for _, line := range [][]string{
		{"设备SN", devRows[0].DeviceSN},
		{"命令", cmdCode},
		{"操作类型", opType},
		{"执行模式", execModeLabel(commands)},
		{"设备状态", devStatus},
		{},
		{"参数路径", "参数名称", "状态", "读回值", "故障码", "故障信息", "子任务ID", "下发时间", "响应时间", "结果报文(XML)"},
	} {
		if err := w.Write(line); err != nil {
			return nil, err
		}
	}

	// 逐 PATH 明细：子任务级字段（子任务ID/下发/响应/报文）只在该 device_task 首行填。
	for _, row := range devRows {
		er := rowToExport(row, cols, read)
		status := deviceStatusText(row.Status, row.ErrorCode)
		faultCode, faultMsg := "", ""
		if row.ErrorCode != 0 {
			faultCode = strconv.Itoa(row.ErrorCode)
			faultMsg = row.ErrorMessage
		}
		raw := flattenXML(rawResponseOf(row.Result))
		firstRowOfTask := true
		for _, p := range commandStandardPaths(commands, row.CommandIndex) {
			rec := []string{p, nameMap[p], status, er.values[p], faultCode, faultMsg, "", "", "", ""}
			if firstRowOfTask {
				rec[6] = row.DeviceTaskID
				rec[7] = er.sentAt
				rec[8] = er.completedAt
				rec[9] = raw
				firstRowOfTask = false
			}
			if err := w.Write(rec); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var deviceBoundPlanHeader = []string{
	"任务ID", "任务名称", "行号", "设备SN", "顺序", "原始脚本行",
	"命令码", "RPC方法", "操作类型", "参数摘要", "子任务ID", "状态",
	"故障码", "失败原因", "下发时间", "响应时间", "结果报文(XML)",
}

func buildDeviceBoundPlanCSV(task *MMLTask, rows []DeviceTaskResultRowView, deviceSN string) ([]byte, error) {
	rowByPlanIndex := make(map[int]DeviceTaskResultRowView, len(rows))
	for _, row := range rows {
		if row.CommandIndex < 0 {
			continue
		}
		if deviceSN != "" && row.DeviceSN != deviceSN {
			continue
		}
		if _, exists := rowByPlanIndex[row.CommandIndex]; !exists {
			rowByPlanIndex[row.CommandIndex] = row
		}
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	w := csv.NewWriter(&buf)
	if err := w.Write(deviceBoundPlanHeader); err != nil {
		return nil, err
	}

	written := 0
	for idx, item := range task.PlanItems {
		if deviceSN != "" && item.DeviceSN != deviceSN {
			continue
		}
		row, hasRow := rowByPlanIndex[idx]
		status := "待执行"
		faultCode, faultMsg, sentAt, completedAt, raw, deviceTaskID := "", "", "", "", "", ""
		if hasRow {
			status = deviceStatusText(row.Status, row.ErrorCode)
			if row.ErrorCode != 0 {
				faultCode = strconv.Itoa(row.ErrorCode)
				faultMsg = row.ErrorMessage
			}
			if row.SentAt != nil {
				sentAt = row.SentAt.Format("2006-01-02 15:04:05")
			}
			if row.CompletedAt != nil {
				completedAt = row.CompletedAt.Format("2006-01-02 15:04:05")
			}
			raw = flattenXML(rawResponseOf(row.Result))
			deviceTaskID = row.DeviceTaskID
		}
		rec := []string{
			task.ID.String(),
			task.TaskName,
			strconv.Itoa(item.LineNo),
			item.DeviceSN,
			strconv.Itoa(item.Order),
			item.RawLine,
			commandString(item.Command, "command_code"),
			commandString(item.Command, "rpc_method"),
			commandString(item.Command, "operation_type"),
			commandParametersSummary(item.Command),
			deviceTaskID,
			status,
			faultCode,
			faultMsg,
			sentAt,
			completedAt,
			raw,
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
		written++
	}
	if written == 0 {
		return nil, commonDeviceResultNotFound
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func commandParametersSummary(command map[string]interface{}) string {
	if params, ok := command["parameters"]; ok && params != nil {
		if b, err := json.Marshal(params); err == nil {
			return string(b)
		}
	}
	if paths, ok := command["param_paths"].([]interface{}); ok && len(paths) > 0 {
		parts := make([]string, 0, len(paths))
		for _, p := range paths {
			if s, ok := p.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ";")
	}
	return ""
}

// ── Service 导出方法 ──────────────────────────────────────────────────────────

// resolvePathNames 解析 standardPath → 友好名（CSV「参数名称」列）。优先 pathNameResolver
// （standard_params.description）；缺失的 path 回退 commandPathNames（param_refs.param_name_zh，
// 排除名==path 的冗余值）。
func (s *Service) resolvePathNames(ctx context.Context, cols []exportColumn, commands []map[string]interface{}) map[string]string {
	names := make(map[string]string)
	if s.pathNameResolver != nil && len(cols) > 0 {
		paths := make([]string, 0, len(cols))
		for _, c := range cols {
			paths = append(paths, c.standard)
		}
		if m, err := s.pathNameResolver(ctx, paths); err == nil {
			for k, v := range m {
				if v != "" {
					names[k] = v
				}
			}
		} else {
			s.logger.Warn("resolve path names for csv export", zap.Error(err))
		}
	}
	for path, name := range commandPathNames(commands) {
		if _, ok := names[path]; !ok && name != "" && name != path {
			names[path] = name
		}
	}
	return names
}

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
// AggregateCSVBytes 直接生成「全设备汇总」CSV 字节（不落 MinIO），供同源流式下载——
// 避免 MinIO 预签名 URL（public_endpoint）在跨主机/反代访问时浏览器不可达的问题。
func (s *Service) AggregateCSVBytes(ctx context.Context, taskID uuid.UUID) ([]byte, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.ExecuteMode == TaskExecuteModeDeviceBound {
		return buildDeviceBoundPlanCSV(task, rows, "")
	}
	nameMap := s.resolvePathNames(ctx, cols, task.Commands)
	return buildLongFormatCSV(cols, rows, task.Commands, nameMap, read)
}

// DeviceCSVBytes 直接生成「单设备」CSV 字节（不落 MinIO），供同源流式下载。
func (s *Service) DeviceCSVBytes(ctx context.Context, taskID uuid.UUID, deviceSN string) ([]byte, error) {
	if deviceSN == "" {
		return nil, commonInvalidDeviceSN
	}
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.ExecuteMode == TaskExecuteModeDeviceBound {
		return buildDeviceBoundPlanCSV(task, rows, deviceSN)
	}
	var devRows []DeviceTaskResultRowView
	for i := range rows {
		if rows[i].DeviceSN == deviceSN {
			devRows = append(devRows, rows[i])
		}
	}
	if len(devRows) == 0 {
		return nil, commonDeviceResultNotFound
	}
	nameMap := s.resolvePathNames(ctx, cols, task.Commands)
	return buildDeviceCSVMulti(cols, devRows, task.Commands, nameMap, read)
}

func (s *Service) ExportTaskResultsCSV(ctx context.Context, taskID uuid.UUID) (objectKey, downloadURL string, err error) {
	if !s.exporter.ready() {
		return "", "", ErrExporterNotConfigured
	}
	data, err := s.AggregateCSVBytes(ctx, taskID)
	if err != nil {
		return "", "", err
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
	data, err := s.DeviceCSVBytes(ctx, taskID, deviceSN)
	if err != nil {
		return "", "", err
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
