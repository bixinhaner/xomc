package northbound

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

// 过滤参数白名单：北向导出/同步端点只接受白名单内的过滤字段，未知字段一律拒绝
// （fail-closed）。防止：① 上游误用未支持的参数被静默忽略导致取数错误；
// ② 攻击者探测 / 注入未预期的过滤维度绕过数据隔离。
//
// 白名单从各 *Request 结构体的 form/json tag 枚举得来（与 model.go 保持同步）。

// allowedSyncParams 是 /northbound/sync/{full,incremental} 接受的 query 参数。
var allowedSyncParams = map[string]struct{}{
	"data_type": {},
	"since":     {}, // incremental 专用
	"format":    {},
}

// allowedPMExportParams 是 POST /northbound/export/pm 接受的 body 字段（对齐 ExportPMRequest）。
var allowedPMExportParams = map[string]struct{}{
	"device_id":     {},
	"cell_id":       {},
	"counter_group": {},
	"start_time":    {},
	"end_time":      {},
	"format":        {},
}

// allowedAlarmExportParams 是 POST /northbound/export/alarms 接受的 body 字段（对齐 ExportAlarmRequest）。
var allowedAlarmExportParams = map[string]struct{}{
	"device_sn":  {},
	"severity":   {},
	"status":     {},
	"start_time": {},
	"end_time":   {},
	"format":     {},
}

// unknownFilterParams 返回 params 中不在白名单 allowed 内的键（保持出现顺序无关）。
// 返回空切片表示全部合法。
func unknownFilterParams(params map[string][]string, allowed map[string]struct{}) []string {
	var unknown []string
	for k := range params {
		if _, ok := allowed[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	return unknown
}

// unknownJSONFields 返回 fields（解析后的 JSON object 顶层键）中不在白名单内的键。
func unknownJSONFields(fields map[string]any, allowed map[string]struct{}) []string {
	var unknown []string
	for k := range fields {
		if _, ok := allowed[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	return unknown
}

// peekJSONBody 读取并复原 gin 请求体，返回解析后的顶层 JSON object（用于白名单校验）。
// 复原 c.Request.Body 后下游 ShouldBindJSON 仍可正常解码。空 body 视为合法的空 object。
// 第二个返回值为 false 表示 body 不是合法 JSON object（调用方应回 400）。
func peekJSONBody(c *gin.Context) (map[string]any, bool) {
	if c.Request == nil || c.Request.Body == nil {
		return map[string]any{}, true
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 1MiB 上限
	_ = c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, true
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil, false
	}
	return m, true
}

// joinFields 把字段名列表拼成逗号分隔字符串（用于错误信息）。
func joinFields(fields []string) string {
	return strings.Join(fields, ", ")
}
