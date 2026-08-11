package mml

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/storageprotection"
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
	admission    storageprotection.WriteAdmission
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

func (e *Exporter) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	if e == nil {
		return
	}
	e.admission = admission
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
	if e.admission != nil {
		decision, err := e.admission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, storageprotection.WriteScopeReport)
		if err != nil {
			return fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}
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
	values      map[string]string // standardPath -> 读回值（读类）
	faultCode   string
	faultMsg    string
	sentAt      string
	completedAt string
}

type exportParameterValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func standardParameterValuesOf(result json.RawMessage) []exportParameterValue {
	if len(result) == 0 {
		return nil
	}
	var payload struct {
		Values []exportParameterValue `json:"standard_parameter_values"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil
	}
	return payload.Values
}

func localizedDeviceStatusText(status string, errorCode int, locale appcontext.Locale) string {
	if locale == appcontext.LocaleEN {
		if status == "completed" && errorCode == 0 {
			return "Success"
		}
		switch status {
		case "completed":
			return "Success"
		case "failed":
			return "Failed"
		case "expired":
			return "Timeout"
		case "pending", "running":
			return "Running"
		default:
			if status == "" {
				return "Unknown"
			}
			return status
		}
	}
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

type exportLabels struct {
	header        []string
	deviceSN      string
	command       string
	operation     string
	executionMode string
	deviceStatus  string
	parameterPath string
	parameterName string
	status        string
	readbackValue string
	faultCode     string
	faultMessage  string
	subtaskID     string
	sentAt        string
	responseAt    string
	resultXML     string
	pending       string
}

func labelsForLocale(locale appcontext.Locale) exportLabels {
	if locale == appcontext.LocaleEN {
		return exportLabels{
			header:        []string{"Index", "Device SN", "Command", "Operation", "Parameter Path", "Parameter Name", "Status", "Readback Value", "Fault Code", "Fault Message", "Sent Time", "Response Time", "Result XML"},
			deviceSN:      "Device SN",
			command:       "Command",
			operation:     "Operation",
			executionMode: "Execution Mode",
			deviceStatus:  "Device Status",
			parameterPath: "Parameter Path",
			parameterName: "Parameter Name",
			status:        "Status",
			readbackValue: "Readback Value",
			faultCode:     "Fault Code",
			faultMessage:  "Fault Message",
			subtaskID:     "Subtask ID",
			sentAt:        "Sent Time",
			responseAt:    "Response Time",
			resultXML:     "Result XML",
			pending:       "Pending",
		}
	}
	return exportLabels{
		header:        []string{"序号", "设备SN", "命令", "操作类型", "参数路径", "参数名称", "状态", "读回值", "故障码", "故障信息", "下发时间", "响应时间", "结果报文(XML)"},
		deviceSN:      "设备SN",
		command:       "命令",
		operation:     "操作类型",
		executionMode: "执行模式",
		deviceStatus:  "设备状态",
		parameterPath: "参数路径",
		parameterName: "参数名称",
		status:        "状态",
		readbackValue: "读回值",
		faultCode:     "故障码",
		faultMessage:  "故障信息",
		subtaskID:     "子任务ID",
		sentAt:        "下发时间",
		responseAt:    "响应时间",
		resultXML:     "结果报文(XML)",
		pending:       "待执行",
	}
}

func execModeLabelForLocale(commands []map[string]interface{}, locale appcontext.Locale) string {
	if len(commands) > 1 {
		if locale == appcontext.LocaleEN {
			return "Per PATH"
		}
		return "逐 PATH"
	}
	if locale == appcontext.LocaleEN {
		return "Whole"
	}
	return "整体执行"
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

func pathMatchesExportColumn(path string, col exportColumn) bool {
	if path == col.standard {
		return true
	}
	return strings.HasSuffix(col.standard, ".") && strings.HasPrefix(path, col.standard)
}

func matchingStandardParameterValues(result json.RawMessage, cols []exportColumn) []exportParameterValue {
	values := standardParameterValuesOf(result)
	matched := make([]exportParameterValue, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.Name == "" {
			continue
		}
		for _, col := range cols {
			if !pathMatchesExportColumn(value.Name, col) {
				continue
			}
			if _, ok := seen[value.Name]; !ok {
				seen[value.Name] = struct{}{}
				matched = append(matched, value)
			}
			break
		}
	}
	return matched
}

func longestPrefixColumn(path string, cols []exportColumn, private bool) (exportColumn, bool) {
	var matched exportColumn
	matchedLength := -1
	for _, col := range cols {
		prefix := col.standard
		if private {
			prefix = col.private
		}
		if !strings.HasSuffix(prefix, ".") || !strings.HasPrefix(path, prefix) {
			continue
		}
		if len(prefix) > matchedLength {
			matched = col
			matchedLength = len(prefix)
		}
	}
	return matched, matchedLength >= 0
}

func standardPathForRawName(path string, cols []exportColumn) (string, bool) {
	for _, col := range cols {
		if path == col.private {
			return col.standard, true
		}
	}
	for _, col := range cols {
		if path == col.standard {
			return col.standard, true
		}
	}
	if col, ok := longestPrefixColumn(path, cols, true); ok {
		return col.standard + strings.TrimPrefix(path, col.private), true
	}
	if _, ok := longestPrefixColumn(path, cols, false); ok {
		return path, true
	}
	for _, col := range cols {
		if !strings.HasSuffix(col.standard, ".") && leafOf(path) == leafOf(col.private) {
			return col.standard, true
		}
	}
	return "", false
}

func matchingRawParameterValues(result json.RawMessage, cols []exportColumn) []exportParameterValue {
	raw := rawResponseOf(result)
	if raw == "" {
		return nil
	}
	values, _, err := soap.DecodeGetParameterValuesResponse(strings.NewReader(raw))
	if err != nil {
		return nil
	}
	matched := make([]exportParameterValue, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		standardPath, ok := standardPathForRawName(value.Name, cols)
		if !ok {
			continue
		}
		if _, ok := seen[standardPath]; ok {
			continue
		}
		seen[standardPath] = struct{}{}
		matched = append(matched, exportParameterValue{Name: standardPath, Value: value.Value})
	}
	return matched
}

func matchingResultParameterValues(result json.RawMessage, cols []exportColumn) []exportParameterValue {
	if values := matchingStandardParameterValues(result, cols); len(values) > 0 {
		return values
	}
	return matchingRawParameterValues(result, cols)
}

// rowToExport 把 device_tasks 结果行转为导出行；读类解析 GPV 原始报文回填列值。
func rowToExport(row DeviceTaskResultRowView, cols []exportColumn, read bool) exportRow {
	er := exportRow{
		sn:       row.DeviceSN,
		values:   map[string]string{},
		faultMsg: row.ErrorMessage,
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
		for _, value := range matchingResultParameterValues(row.Result, cols) {
			er.values[value.Name] = value.Value
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

func commandExportColumns(commands []map[string]interface{}, cols []exportColumn, ci int) []exportColumn {
	paths := commandStandardPaths(commands, ci)
	out := make([]exportColumn, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		col := exportColumn{standard: path, private: path}
		for _, candidate := range cols {
			if candidate.standard == path {
				col = candidate
				break
			}
		}
		out = append(out, col)
	}
	return out
}

func exportPathsByCommand(
	cols []exportColumn,
	rows []DeviceTaskResultRowView,
	commands []map[string]interface{},
	read bool,
) map[int][]string {
	matchedPathsByCommand := make(map[int][]string, len(commands))
	seenByCommand := make(map[int]map[string]struct{}, len(commands))
	if read {
		for _, row := range rows {
			if row.Status != "completed" {
				continue
			}
			commandCols := commandExportColumns(commands, cols, row.CommandIndex)
			for _, value := range matchingResultParameterValues(row.Result, commandCols) {
				if seenByCommand[row.CommandIndex] == nil {
					seenByCommand[row.CommandIndex] = make(map[string]struct{})
				}
				if _, ok := seenByCommand[row.CommandIndex][value.Name]; ok {
					continue
				}
				seenByCommand[row.CommandIndex][value.Name] = struct{}{}
				matchedPathsByCommand[row.CommandIndex] = append(
					matchedPathsByCommand[row.CommandIndex],
					value.Name,
				)
			}
		}
	}

	pathsByCommand := make(map[int][]string, len(commands))
	for ci := range commands {
		seen := make(map[string]struct{})
		appendPath := func(path string) {
			if path == "" {
				return
			}
			if _, ok := seen[path]; ok {
				return
			}
			seen[path] = struct{}{}
			pathsByCommand[ci] = append(pathsByCommand[ci], path)
		}

		for _, originalPath := range commandStandardPaths(commands, ci) {
			matched := false
			originalColumn := exportColumn{standard: originalPath}
			for _, resultPath := range matchedPathsByCommand[ci] {
				if !pathMatchesExportColumn(resultPath, originalColumn) {
					continue
				}
				matched = true
				appendPath(resultPath)
			}
			if !matched {
				appendPath(originalPath)
			}
		}
	}
	return pathsByCommand
}

// commandPathNames 从 commands 的 param_refs 抽 standardPath → 参数名称（旧字段名 param_name_zh）映射。
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

func commandDisplayField(commands []map[string]interface{}, ci int) string {
	if name := commandStringField(commands, ci, "command_name"); name != "" {
		return name
	}
	return commandStringField(commands, ci, "command_code")
}

// flattenXML 把多行 XML 报文压成一行（空白折叠为单空格），便于放进 CSV 单元格（方式 B）。
func flattenXML(s string) string {
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

// buildLongFormatCSV 多设备汇总（长表，一行 = 设备 × PATH）：
// 保留该函数供旧的包内调用使用；HTTP 导出入口使用带 locale 的实现。
func buildLongFormatCSV(
	cols []exportColumn, rows []DeviceTaskResultRowView, commands []map[string]interface{},
	nameMap map[string]string, read bool,
) ([]byte, error) {
	return buildLongFormatCSVForLocale(cols, rows, commands, nameMap, read, appcontext.LocaleZH)
}

// buildLongFormatCSVForLocale 多设备汇总（长表，一行 = 设备 × PATH）。
// 汇总表不再输出执行模式和子任务 ID 两列，也不插入设备间空行，保证 CSV 可以按 PATH
// 连续筛选和处理；设备/任务信息仍由任务详情接口提供。
func buildLongFormatCSVForLocale(
	cols []exportColumn, rows []DeviceTaskResultRowView, commands []map[string]interface{},
	nameMap map[string]string, read bool, locale appcontext.Locale,
) ([]byte, error) {

	pathsByCommand := exportPathsByCommand(cols, rows, commands, read)
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
	labels := labelsForLocale(locale)
	if err := w.Write(labels.header); err != nil {
		return nil, err
	}

	deviceSeq := 0
	for _, sn := range order {
		deviceSeq++
		firstRowOfDevice := true
		deviceRows := groups[sn]
		sort.SliceStable(deviceRows, func(i, j int) bool {
			if deviceRows[i].CommandIndex != deviceRows[j].CommandIndex {
				return deviceRows[i].CommandIndex < deviceRows[j].CommandIndex
			}
			return deviceRows[i].CreatedAt.Before(deviceRows[j].CreatedAt)
		})
		for _, row := range deviceRows {
			commandCols := commandExportColumns(commands, cols, row.CommandIndex)
			er := rowToExport(row, commandCols, read)
			status := localizedDeviceStatusText(row.Status, row.ErrorCode, locale)
			faultCode, faultMsg := "", ""
			if row.ErrorCode != 0 {
				faultCode = strconv.Itoa(row.ErrorCode)
				faultMsg = row.ErrorMessage
			}
			raw := flattenXML(rawResponseOf(row.Result))
			paths := pathsByCommand[row.CommandIndex]
			firstRowOfTask := true
			for _, p := range paths {
				rec := make([]string, len(labels.header))
				if firstRowOfDevice { // 设备级公共字段：仅设备首行
					rec[0] = strconv.Itoa(deviceSeq)
					rec[1] = sn
					firstRowOfDevice = false
				}
				if firstRowOfTask { // 下发/响应/报文仅在该 device_task 首行填，避免重复
					rec[2] = commandDisplayField(commands, row.CommandIndex)
					rec[3] = commandStringField(commands, row.CommandIndex, "operation_type")
					rec[10] = er.sentAt
					rec[11] = er.completedAt
					rec[12] = raw
					firstRowOfTask = false
				}
				rec[4] = p // PATH 级：每行
				rec[5] = nameMap[p]
				if rec[5] == "" {
					rec[5] = p
				}
				rec[6] = status
				rec[7] = er.values[p]
				rec[8] = faultCode
				rec[9] = faultMsg
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
	return buildDeviceCSVMultiForLocale(cols, devRows, commands, nameMap, read, appcontext.LocaleZH)
}

func buildDeviceCSVMultiForLocale(
	cols []exportColumn, devRows []DeviceTaskResultRowView, commands []map[string]interface{},
	nameMap map[string]string, read bool, locale appcontext.Locale,
) ([]byte, error) {
	labels := labelsForLocale(locale)
	pathsByCommand := exportPathsByCommand(cols, devRows, commands, read)
	// 设备级状态：全成功=成功 / 全失败=失败 / 混合=部分失败。
	anySucc, anyFail := false, false
	for _, row := range devRows {
		if row.Status == "completed" && row.ErrorCode == 0 {
			anySucc = true
		} else {
			anyFail = true
		}
	}
	devStatus := localizedDeviceStatusText("completed", 0, locale)
	switch {
	case anyFail && anySucc:
		if locale == appcontext.LocaleEN {
			devStatus = "Partially Failed"
		} else {
			devStatus = "部分失败"
		}
	case anyFail:
		devStatus = localizedDeviceStatusText("failed", 1, locale)
	}

	cmdCode := ""
	opType := ""
	if len(devRows) > 0 {
		cmdCode = commandDisplayField(commands, devRows[0].CommandIndex)
		opType = commandStringField(commands, devRows[0].CommandIndex, "operation_type")
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	w := csv.NewWriter(&buf)
	// 顶部设备摘要（公共字段一次）。
	for _, line := range [][]string{
		{labels.deviceSN, devRows[0].DeviceSN},
		{labels.command, cmdCode},
		{labels.operation, opType},
		{labels.executionMode, execModeLabelForLocale(commands, locale)},
		{labels.deviceStatus, devStatus},
		{},
		{labels.parameterPath, labels.parameterName, labels.status, labels.readbackValue, labels.faultCode, labels.faultMessage, labels.subtaskID, labels.sentAt, labels.responseAt, labels.resultXML},
	} {
		if err := w.Write(line); err != nil {
			return nil, err
		}
	}

	// 逐 PATH 明细：子任务级字段（子任务ID/下发/响应/报文）只在该 device_task 首行填。
	for _, row := range devRows {
		commandCols := commandExportColumns(commands, cols, row.CommandIndex)
		er := rowToExport(row, commandCols, read)
		status := localizedDeviceStatusText(row.Status, row.ErrorCode, locale)
		faultCode, faultMsg := "", ""
		if row.ErrorCode != 0 {
			faultCode = strconv.Itoa(row.ErrorCode)
			faultMsg = row.ErrorMessage
		}
		raw := flattenXML(rawResponseOf(row.Result))
		firstRowOfTask := true
		for _, p := range pathsByCommand[row.CommandIndex] {
			name := nameMap[p]
			if name == "" {
				name = p
			}
			rec := []string{p, name, status, er.values[p], faultCode, faultMsg, "", "", "", ""}
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
	return buildDeviceBoundPlanCSVForLocale(task, rows, deviceSN, appcontext.LocaleZH)
}

func buildDeviceBoundPlanCSVForLocale(task *MMLTask, rows []DeviceTaskResultRowView, deviceSN string, locale appcontext.Locale) ([]byte, error) {
	labels := labelsForLocale(locale)
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
	header := deviceBoundPlanHeader
	if locale == appcontext.LocaleEN {
		header = []string{"Task ID", "Task Name", "Line No.", "Device SN", "Order", "Raw Script Line", "Command Code", "RPC Method", "Operation", "Parameters", "Subtask ID", "Status", "Fault Code", "Failure Reason", "Sent Time", "Response Time", "Result XML"}
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	written := 0
	for idx, item := range task.PlanItems {
		if deviceSN != "" && item.DeviceSN != deviceSN {
			continue
		}
		row, hasRow := rowByPlanIndex[idx]
		status := labels.pending
		faultCode, faultMsg, sentAt, completedAt, raw, deviceTaskID := "", "", "", "", "", ""
		if hasRow {
			status = localizedDeviceStatusText(row.Status, row.ErrorCode, locale)
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

// resolvePathNames 解析 standardPath → 友好名（CSV「参数名称」列）。优先使用带 locale 的
// pathNameResolver；中文缺失时回退任务快照中的旧 param_name_zh，英文缺失时由调用方回退 PATH。
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
	if appcontext.GetLocale(ctx) != appcontext.LocaleEN {
		for path, name := range commandPathNames(commands) {
			if _, ok := names[path]; !ok && name != "" && name != path {
				names[path] = name
			}
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
	task, resultSourceID, err := s.resolveTaskResultSource(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, resultSourceID)
	if err != nil {
		return nil, err
	}
	if task.ExecuteMode == TaskExecuteModeDeviceBound {
		return buildDeviceBoundPlanCSVForLocale(task, rows, "", appcontext.GetLocale(ctx))
	}
	s.enrichCommandNames(ctx, task.Commands)
	nameMap := s.resolvePathNames(ctx, cols, task.Commands)
	return buildLongFormatCSVForLocale(cols, rows, task.Commands, nameMap, read, appcontext.GetLocale(ctx))
}

// DeviceCSVBytes 直接生成「单设备」CSV 字节（不落 MinIO），供同源流式下载。
func (s *Service) DeviceCSVBytes(ctx context.Context, taskID uuid.UUID, deviceSN string) ([]byte, error) {
	if deviceSN == "" {
		return nil, commonInvalidDeviceSN
	}
	task, resultSourceID, err := s.resolveTaskResultSource(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}
	cols := exportColumnsFromTask(task.Commands)
	read := taskIsRead(task.Commands)
	rows, err := s.allDeviceResults(ctx, resultSourceID)
	if err != nil {
		return nil, err
	}
	if task.ExecuteMode == TaskExecuteModeDeviceBound {
		return buildDeviceBoundPlanCSVForLocale(task, rows, deviceSN, appcontext.GetLocale(ctx))
	}
	s.enrichCommandNames(ctx, task.Commands)
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
	return buildDeviceCSVMultiForLocale(cols, devRows, task.Commands, nameMap, read, appcontext.GetLocale(ctx))
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
