package mml

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// 把 MML 控制台/脚本里"用户友好的命令参数"翻译为 ACS RPC handler 需要的
// TR-069 wire 格式 JSON。ACS 侧（internal/acs/rpc/dispatcher.go）的 schema 是
// 协议级硬约束，本翻译层是数据库面/业务面与协议面之间的唯一桥梁。
//
// 输入：
//   - rpcMethod    canonical RPC 名（GetParameterValues / SetParameterValues / ...）
//   - paramRefs    命令绑定的参数定义（mml_command_param_refs JOIN mml_params），
//                  已按 (command_id, tr069_path) 去重排序
//   - formValues   前端表单 / 脚本入参的 map[param_code]value（可能含空字符串）
//   - operationType MML 操作语义（LST/MOD/ADD/RMV/DSP/...），目前用于日志/诊断
//
// 输出：
//   - device_tasks.params 列要写入的 JSON
//   - 缺失关键输入时返回 error，由调用方决定降级策略

// ErrNoUsableParams indicates the translation produced an empty payload, which is
// a strong signal that the command/parameter configuration upstream is broken.
var ErrNoUsableParams = errors.New("no usable params for tr069 payload")

// BuildTR069Params 把 MML 命令参数翻译为 TR-069 wire 格式 JSON。
//
// 路由表（与 internal/acs/rpc/dispatcher.go 各 Handler.BuildRequest 严格对齐）：
//
//	GetParameterValues / GetParameterAttributes  → {"names": [<tr069_path>...]}
//	SetParameterValues                            → {"values": [{name,value,type}...]}
//	SetParameterAttributes                        → {"attributes": [{name,...}...]}
//	GetParameterNames                             → {"path": "...", "next_level": bool}
//	AddObject / DeleteObject                      → {"object_name": "..."}
//	Reboot / FactoryReset                         → {}
//	Download / Upload                             → 透传 formValues（这两类目前由
//	                                                upgrade/backup 模块自行构造，本
//	                                                函数仅做兜底）
//
// 未识别的 rpcMethod 透传 formValues，让 dispatcher 自己判定（日志可观察）。
func BuildTR069Params(
	rpcMethod string,
	paramRefs []MMLParamRef,
	formValues map[string]interface{},
	operationType string,
) (json.RawMessage, error) {
	switch rpcMethod {
	case "GetParameterValues", "GetParameterAttributes":
		return buildParameterNames(paramRefs)
	case "SetParameterValues":
		return buildParameterValues(paramRefs, formValues)
	case "SetParameterAttributes":
		return buildSetAttributes(paramRefs, formValues)
	case "GetParameterNames":
		return buildGetParameterNames(paramRefs, formValues)
	case "AddObject", "DeleteObject":
		return buildObjectName(paramRefs, formValues)
	case "Reboot", "FactoryReset":
		return json.RawMessage(`{}`), nil
	case "Download", "Upload":
		return marshalRaw(formValues)
	default:
		return marshalRaw(formValues)
	}
}

// xsdType 把 mml_params.value_type（业务定义）映射为 TR-069 SOAP 报文的 xsd 类型字符串。
// 列表型（stringList / unsignedIntList）按 TR-069 规范以 CSV string 传输。
func xsdType(valueType string) string {
	switch valueType {
	case "boolean":
		return "xsd:boolean"
	case "unsignedInt":
		return "xsd:unsignedInt"
	case "int", "uniqueInt":
		return "xsd:int"
	default:
		// string / enum / stringList / unsignedIntList → xsd:string
		return "xsd:string"
	}
}

// buildParameterNames 收集所有 paramRefs 的 tr069_path → {"names":[...]}。
// LST/DSP 不依赖 formValues：用户在控制台不填表单，命令的 param_refs 即为读取范围。
func buildParameterNames(paramRefs []MMLParamRef) (json.RawMessage, error) {
	names := make([]string, 0, len(paramRefs))
	seen := make(map[string]struct{}, len(paramRefs))
	for _, ref := range paramRefs {
		if ref.Tr069Path == "" {
			continue
		}
		if _, dup := seen[ref.Tr069Path]; dup {
			continue
		}
		seen[ref.Tr069Path] = struct{}{}
		names = append(names, ref.Tr069Path)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%w: GetParameterValues found 0 tr069_path in command param_refs", ErrNoUsableParams)
	}
	return json.Marshal(map[string]interface{}{"names": names})
}

// buildParameterValues 把 formValues（map[param_code]value）翻译为
// {"values":[{name,value,type}...]}。
// - 空值（nil / "" / "  "）跳过：MOD 提交时已在前端过滤未填字段，这里再次防御
// - 找不到 param_code 对应的 ref 时跳过并记入 unknown，由上层日志暴露
// - 全部跳过时返回 ErrNoUsableParams
func buildParameterValues(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	if len(formValues) == 0 {
		return nil, fmt.Errorf("%w: SetParameterValues requires form values, got empty map", ErrNoUsableParams)
	}

	refsByCode := make(map[string]MMLParamRef, len(paramRefs))
	for _, ref := range paramRefs {
		refsByCode[ref.ParamCode] = ref
	}

	type valueEntry struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Type  string `json:"type"`
	}
	values := make([]valueEntry, 0, len(formValues))
	var unknown []string

	for code, raw := range formValues {
		strVal, isEmpty := stringifyValue(raw)
		if isEmpty {
			continue
		}
		ref, ok := refsByCode[code]
		if !ok || ref.Tr069Path == "" {
			unknown = append(unknown, code)
			continue
		}
		values = append(values, valueEntry{
			Name:  ref.Tr069Path,
			Value: strVal,
			Type:  xsdType(ref.ValueType),
		})
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("%w: SetParameterValues mapped 0 values; unknown_codes=%v", ErrNoUsableParams, unknown)
	}
	return json.Marshal(map[string]interface{}{"values": values})
}

// buildSetAttributes 翻译 SetParameterAttributes：通常携带 notification 设置。
// formValues 期望已经是结构化形式 {"attributes":[{name,notification_change,notification,...}]}；
// 否则按 paramRefs 兜底构造（每个 ref 一条，notification_change=false）。
func buildSetAttributes(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	if attrs, ok := formValues["attributes"]; ok {
		return json.Marshal(map[string]interface{}{"attributes": attrs})
	}
	type attrEntry struct {
		Name               string `json:"name"`
		NotificationChange bool   `json:"notification_change"`
		Notification       int    `json:"notification"`
	}
	entries := make([]attrEntry, 0, len(paramRefs))
	for _, ref := range paramRefs {
		if ref.Tr069Path == "" {
			continue
		}
		entries = append(entries, attrEntry{Name: ref.Tr069Path})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: SetParameterAttributes resolved 0 paths", ErrNoUsableParams)
	}
	return json.Marshal(map[string]interface{}{"attributes": entries})
}

// buildGetParameterNames 处理 GetParameterNames(path, next_level)。
// 优先 formValues["path"]；否则取 paramRefs 第一条作为兜底。
func buildGetParameterNames(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	var path string
	if p, ok := formValues["path"].(string); ok && p != "" {
		path = p
	} else if len(paramRefs) > 0 && paramRefs[0].Tr069Path != "" {
		path = paramRefs[0].Tr069Path
	}
	if path == "" {
		return nil, fmt.Errorf("%w: GetParameterNames requires path", ErrNoUsableParams)
	}
	var nextLevel bool
	switch v := formValues["next_level"].(type) {
	case bool:
		nextLevel = v
	case string:
		nextLevel = v == "true" || v == "1"
	}
	return json.Marshal(map[string]interface{}{
		"path":       path,
		"next_level": nextLevel,
	})
}

// buildObjectName 翻译 AddObject / DeleteObject：取 formValues["object_name"]
// 或 paramRefs[0].Tr069Path。TR-069 协议规定 object_name 必须以 "." 结尾。
func buildObjectName(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	var name string
	if v, ok := formValues["object_name"].(string); ok && v != "" {
		name = v
	} else if len(paramRefs) > 0 && paramRefs[0].Tr069Path != "" {
		name = paramRefs[0].Tr069Path
	}
	if name == "" {
		return nil, fmt.Errorf("%w: AddObject/DeleteObject requires object_name", ErrNoUsableParams)
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	return json.Marshal(map[string]interface{}{"object_name": name})
}

func marshalRaw(formValues map[string]interface{}) (json.RawMessage, error) {
	if formValues == nil {
		return json.RawMessage(`{}`), nil
	}
	return json.Marshal(formValues)
}

// stringifyValue 统一把 interface{} 转成 TR-069 SOAP body 用的字符串值。
// 第二个返回值 true 表示"应当跳过"（nil / 空白字符串）。
func stringifyValue(v interface{}) (string, bool) {
	if v == nil {
		return "", true
	}
	switch x := v.(type) {
	case string:
		return x, strings.TrimSpace(x) == ""
	case bool:
		if x {
			return "true", false
		}
		return "false", false
	case float64:
		// JSON 原生数字总是 float64；尽量按整数输出避免 1.000000 这种尾巴
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x)), false
		}
		return fmt.Sprintf("%g", x), false
	case int:
		return fmt.Sprintf("%d", x), false
	case int64:
		return fmt.Sprintf("%d", x), false
	default:
		return fmt.Sprintf("%v", v), false
	}
}

// SchemaSummary 返回一个 payload 的 schema 形态摘要，用于日志诊断。
// 目的是不打全量 body 的同时让排查者立刻看出"这次下发是否符合 RPC 契约"。
type SchemaSummary struct {
	HasNames      bool `json:"has_names"`
	HasValues     bool `json:"has_values"`
	HasObjectName bool `json:"has_object_name"`
	HasAttributes bool `json:"has_attributes"`
	HasPath       bool `json:"has_path"`
	NamesCount    int  `json:"names_count"`
	ValuesCount   int  `json:"values_count"`
	PayloadSize   int  `json:"payload_size"`
}

// SummarizeSchema 解析 payload，返回字段在场情况。Unmarshal 失败时返回零值。
func SummarizeSchema(payload json.RawMessage) SchemaSummary {
	s := SchemaSummary{PayloadSize: len(payload)}
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return s
	}
	if v, ok := m["names"].([]interface{}); ok {
		s.HasNames = true
		s.NamesCount = len(v)
	}
	if v, ok := m["values"].([]interface{}); ok {
		s.HasValues = true
		s.ValuesCount = len(v)
	}
	if _, ok := m["object_name"]; ok {
		s.HasObjectName = true
	}
	if _, ok := m["attributes"]; ok {
		s.HasAttributes = true
	}
	if _, ok := m["path"]; ok {
		s.HasPath = true
	}
	return s
}
