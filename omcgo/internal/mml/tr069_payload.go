package mml

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/omcgo/omcgo/pkg/tr069"
)

// 把 MML 控制台/脚本里"用户友好的命令参数"翻译为 ACS RPC handler 需要的
// TR-069 wire 格式 JSON。ACS 侧（internal/acs/rpc/dispatcher.go）的 schema 是
// 协议级硬约束，本翻译层是数据库面/业务面与协议面之间的唯一桥梁。
//
// 输入：
//   - rpcMethod    canonical RPC 名（GetParameterValues / SetParameterValues / ...）
//   - paramRefs    命令绑定的参数定义（mml_command_sub_fields JOIN standard_params），
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

// 路径合规校验（TR-069 §3.3 / §A.2.2.x，含厂商私有根扩展）。
// 前置在 BuildTR069Params 之前过滤掉 CPE 必拒的路径，避免下发到 ACS 后被静默丢弃。
// 被过滤的路径会通过 Skip 列表由 fanout 层日志输出，运维侧能立刻看到具体哪条 path 出问题、为什么。
//
// 三类不合规：
//   1. 顶层既不是 TR-069 标准根（Device. / InternetGatewayDevice.）也不是已知
//      厂商私有根。厂商根从 param_mappings.private_path 已有 root 集合扩展（白名单），
//      详见 paramPathLegalRoot 注释。
//   2. 含 {i} / {n} / {idx} 占位符未替换
//      （TR-181 写法约定，发 SOAP 必须替换为实际索引）
//   3. 含非 ASCII 或非 TR-069 合法字符
//      （只允许字母数字、_ . - [] {} ）

// paramPathLegalRoot 接受 TR-069 标准根 + 厂商私有根。
//
//   - Device. / InternetGatewayDevice. — TR-069/TR-181 标准
//   - boardconf. — BLQ/MLN 等百怡私有板卡配置根（present in 7+ param_mappings）
//   - DeviceGSM. — GSM 设备子树（T-0171 扩展 catalog 派生）
//   - aldconfig. — BLQ ALD 配置（present in 1 param_mapping）
//   - FAPService. — TR-196 FAPService 根（无 Device. 前缀的私有变体）
//
// 历史教训：仅放 Device. / InternetGatewayDevice. → boardconf.HALOD.* path 在
// MML MOD 时被 validator 拒绝（bad_prefix），即便 param_mappings 已声明该路径合法、
// CPE 实际能识别。修复方式：扩展白名单覆盖 param_mappings.private_path 的实际根集合。
var paramPathLegalRoot = regexp.MustCompile(`^(Device|InternetGatewayDevice|boardconf|DeviceGSM|aldconfig|FAPService)\.`)

// 占位符匹配所有 {<letter>+} 形式：TR-069 spec 标准是 {i}，
// 但实际项目种子里也出现 {j}（多层实例索引）等同类问题，CPE 同样无法识别。
// 一律视作"未替换的实例索引"。
var paramPathPlaceholder = regexp.MustCompile(`\{[A-Za-z]+\}`)
var paramPathLegalChars = regexp.MustCompile(`^[A-Za-z0-9_.\[\]\-]+$`)

// PathSkipReason 描述路径被过滤的原因（仅用于日志/诊断，不暴露给协议层）。
type PathSkipReason string

const (
	PathSkipBadPrefix   PathSkipReason = "bad_prefix"      // 不是 Device. / InternetGatewayDevice.
	PathSkipPlaceholder PathSkipReason = "has_placeholder" // 含 {i}/{n}/{idx}
	PathSkipBadChars    PathSkipReason = "bad_chars"       // 含非法字符
)

// SkippedPath 是 ValidatePath 返回的不合规条目。
type SkippedPath struct {
	Path   string
	Reason PathSkipReason
}

// validatePath 单条路径合规校验。返回（合规则空字符串, 不合规则原因）。
func validatePath(p string) PathSkipReason {
	p = strings.TrimSpace(p)
	if p == "" {
		return PathSkipBadPrefix
	}
	if !paramPathLegalRoot.MatchString(p) {
		return PathSkipBadPrefix
	}
	if paramPathPlaceholder.MatchString(p) {
		return PathSkipPlaceholder
	}
	if !paramPathLegalChars.MatchString(p) {
		return PathSkipBadChars
	}
	return ""
}

// expandInstancePaths 把含 {i} 占位符的路径展开为 TR-069 partial path
// （末段 `.` 结尾）。CPE 按 spec §A.3.2.7 返回该前缀下全部参数。
//
// 规则：
//   - 不含 {i} 的路径：原样保留
//   - 含 {i} 的路径：取首个 {i} 段之前的前缀 + 末尾点
//     例 "Device.X.{i}.Y.{i}.Z" → "Device.X."
//     例 "Device.X.{i}.Y"       → "Device.X."
//   - 罕见以 {i} 开头的路径：跳过（不合规，无法生成 partial path）
//
// 输入顺序保留；自动去重（多个 {i} 路径可能合到同一 partial path）。
// 此函数为 Sprint B Q-V3-2 决议落地：fanout 层透明处理动态实例，
// 上层无需感知 GPN→GPV 链路（实际通过 TR-069 partial path 单 RPC 解决）。
func expandInstancePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !strings.Contains(p, "{i}") {
			if _, dup := seen[p]; !dup {
				seen[p] = struct{}{}
				out = append(out, p)
			}
			continue
		}
		// 找首个 {i} 段位置（`.{i}` 或 `{i}` 开头）
		idx := strings.Index(p, ".{i}")
		if idx < 0 {
			// path 以 {i} 开头（如 "{i}.Foo"），无法构造合规 partial → 跳过
			continue
		}
		// 截到首个 {i} 段之前 + 末尾 `.`
		partial := p[:idx+1] // 含 idx 位置的 `.`
		if _, dup := seen[partial]; !dup {
			seen[partial] = struct{}{}
			out = append(out, partial)
		}
	}
	return out
}

// filterLegalPaths 把字符串列表过滤为仅包含合规路径，同时返回被跳过的明细。
// 顺序保留输入顺序；重复路径在调用方按需去重（builder 已有 seen 逻辑）。
func filterLegalPaths(paths []string) (legal []string, skipped []SkippedPath) {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if reason := validatePath(p); reason != "" {
			skipped = append(skipped, SkippedPath{Path: p, Reason: reason})
			continue
		}
		legal = append(legal, p)
	}
	return
}

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

// xsdType 把参数模型的数据类型映射为 TR-069 SOAP 报文的 xsd 类型字符串。
// 列表型（stringList / unsignedIntList）按 TR-069 规范以 CSV string 传输。
func xsdType(valueType string) string {
	return tr069.XSDType(valueType)
}

// xsdTypeForPath applies device compatibility overrides to the SOAP wire type.
// The Baicells SignallingTrace implementation accepts Enable only when the
// numeric boolean value is sent as xsd:string (the standard xsd:boolean form
// is acknowledged but not applied by the device).
func xsdTypeForPath(path, valueType string) string {
	return tr069.XSDTypeForPath(path, valueType)
}

func normalizeTR069ValueForPath(path, value, valueType string) string {
	return tr069.NormalizeValueForPath(path, value, valueType)
}

// buildParameterNames 收集所有 paramRefs 的 tr069_path → {"names":[...]}。
// LST/DSP 不依赖 formValues：用户在控制台不填表单，命令的 param_refs 即为读取范围。
//
// Sprint B Q-V3-2 决议：含 {i} 占位符的路径自动展开为 TR-069 **partial path**
// （末段 `.` 结尾），CPE 按 spec §A.3.2.7 返回前缀下全部参数 — 让命令树
// 自动发现的"含动态实例 LST 命令"也能正常工作，不需要用户手动填实例号。
// 转换示例:
//
//	Device.X.{i}.Y.{i}.Z       → Device.X.（保守 partial path，到首个 {i} 之前）
//	Device.X.{i}.Y             → Device.X.
//	Device.X.Y                 → 原样保留
//
// 路径合规校验：每条（展开后的）tr069_path 经 validatePath 过滤；
// 不合规的路径（前缀错 / 非法字符 / 仍含 {i}）被跳过。
// 全部不合规时返回 ErrNoUsableParams 让上层跳过整条 command。
func buildParameterNames(paramRefs []MMLParamRef) (json.RawMessage, error) {
	pathMode := pathModeFromRefs(paramRefs, nil)
	raw := make([]string, 0, len(paramRefs))
	for _, ref := range paramRefs {
		if path := effectiveParamReadPath(ref, pathMode); path != "" {
			raw = append(raw, path)
		}
	}
	// Sprint B Q-V3-2: 展开 {i} 路径为 partial path（在合规校验之前）。
	if pathMode != rawPathModePrivate {
		raw = expandInstancePaths(raw)
	}
	legal, skipped := filterPathsForMode(raw, pathMode)

	// 去重（保序）
	names := make([]string, 0, len(legal))
	seen := make(map[string]struct{}, len(legal))
	for _, p := range legal {
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		names = append(names, p)
	}

	if len(names) == 0 {
		if len(skipped) > 0 {
			return nil, fmt.Errorf("%w: GetParameterValues all %d paths failed validation (first: %s/%s)",
				ErrNoUsableParams, len(skipped), skipped[0].Path, skipped[0].Reason)
		}
		return nil, fmt.Errorf("%w: GetParameterValues found 0 tr069_path in command param_refs", ErrNoUsableParams)
	}
	return marshalPathPayload(map[string]interface{}{"names": names}, pathMode)
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
	pathMode := pathModeFromRefs(paramRefs, formValues)

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

	var skipped []SkippedPath
	for code, raw := range formValues {
		strVal, isEmpty := stringifyValue(raw)
		if isEmpty {
			continue
		}
		ref, ok := refsByCode[code]
		path := effectiveParamPath(ref, pathMode)
		if !ok || path == "" {
			unknown = append(unknown, code)
			continue
		}
		// 写类参数路径同样必须合规（占位符未替换的 path 写下去 CPE 也会拒）
		if reason := validatePathForMode(path, pathMode); reason != "" {
			skipped = append(skipped, SkippedPath{Path: path, Reason: reason})
			continue
		}
		values = append(values, valueEntry{
			Name: path,
			// TR-069 的 xsd:boolean 允许 true/false 和 1/0，部分基站（包括
			// SignallingTrace.Enable）只接受数字字面量。统一在协议边界编码为
			// 0/1，避免把标准参数的 boolean 语义泄漏成厂商不兼容的 true/false。
			Value: normalizeTR069ValueForPath(path, strVal, ref.ValueType),
			Type:  xsdTypeForPath(path, ref.ValueType),
		})
	}

	if len(values) == 0 {
		if len(skipped) > 0 {
			return nil, fmt.Errorf("%w: SetParameterValues all %d paths failed validation (first: %s/%s)",
				ErrNoUsableParams, len(skipped), skipped[0].Path, skipped[0].Reason)
		}
		return nil, fmt.Errorf("%w: SetParameterValues mapped 0 values; unknown_codes=%v", ErrNoUsableParams, unknown)
	}
	return marshalPathPayload(map[string]interface{}{"values": values}, pathMode)
}

// buildSetAttributes 翻译 SetParameterAttributes：通常携带 notification 设置。
// formValues 期望已经是结构化形式 {"attributes":[{name,notification_change,notification,...}]}；
// 否则按 paramRefs 兜底构造（每个 ref 一条，notification_change=false）。
func buildSetAttributes(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	if attrs, ok := formValues["attributes"]; ok {
		return marshalPathPayload(map[string]interface{}{"attributes": attrs}, pathModeFromRefs(paramRefs, formValues))
	}
	pathMode := pathModeFromRefs(paramRefs, formValues)
	type attrEntry struct {
		Name               string `json:"name"`
		NotificationChange bool   `json:"notification_change"`
		Notification       int    `json:"notification"`
	}
	entries := make([]attrEntry, 0, len(paramRefs))
	for _, ref := range paramRefs {
		path := effectiveParamPath(ref, pathMode)
		if path == "" {
			continue
		}
		if reason := validatePathForMode(path, pathMode); reason != "" {
			continue // 不合规路径直接跳过
		}
		entries = append(entries, attrEntry{Name: path})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: SetParameterAttributes resolved 0 legal paths", ErrNoUsableParams)
	}
	return marshalPathPayload(map[string]interface{}{"attributes": entries}, pathMode)
}

// buildGetParameterNames 处理 GetParameterNames(path, next_level)。
// 优先 formValues["path"]；否则取 paramRefs 第一条作为兜底。
func buildGetParameterNames(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	pathMode := pathModeFromRefs(paramRefs, formValues)
	var path string
	if p, ok := formValues["path"].(string); ok && p != "" {
		path = p
	} else if len(paramRefs) > 0 {
		path = effectiveParamPath(paramRefs[0], pathMode)
	}
	if path == "" {
		return nil, fmt.Errorf("%w: GetParameterNames requires path", ErrNoUsableParams)
	}
	// GetParameterNames 的 path 允许 partial（以 "." 结尾），所以校验只查前缀和字符集，
	// 占位符 {i} 同样不允许（CPE 解析失败）。
	if pathMode == rawPathModePrivate {
		if reason := validateDirectPath(path); reason != "" {
			return nil, fmt.Errorf("%w: GetParameterNames private path %q failed validation (%s)", ErrNoUsableParams, path, reason)
		}
	} else if !paramPathLegalRoot.MatchString(path) {
		return nil, fmt.Errorf("%w: GetParameterNames path %q does not start with an accepted root (Device./InternetGatewayDevice./boardconf./DeviceGSM./aldconfig./FAPService.)", ErrNoUsableParams, path)
	} else if paramPathPlaceholder.MatchString(path) {
		return nil, fmt.Errorf("%w: GetParameterNames path %q contains unresolved placeholder {i}/{n}", ErrNoUsableParams, path)
	}
	var nextLevel bool
	switch v := formValues["next_level"].(type) {
	case bool:
		nextLevel = v
	case string:
		nextLevel = v == "true" || v == "1"
	}
	return marshalPathPayload(map[string]interface{}{
		"path":       path,
		"next_level": nextLevel,
	}, pathMode)
}

// buildObjectName 翻译 AddObject / DeleteObject：取 formValues["object_name"]
// 或 paramRefs[0].Tr069Path。TR-069 协议规定 object_name 必须以 "." 结尾。
func buildObjectName(paramRefs []MMLParamRef, formValues map[string]interface{}) (json.RawMessage, error) {
	pathMode := pathModeFromRefs(paramRefs, formValues)
	var name string
	if v, ok := formValues["object_name"].(string); ok && v != "" {
		name = v
	} else if len(paramRefs) > 0 {
		name = effectiveParamPath(paramRefs[0], pathMode)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: AddObject/DeleteObject requires object_name", ErrNoUsableParams)
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	// 对象名同样必须合规。AddObject 的 object_name 是 partial path（以 . 结尾）。
	if reason := validatePathForMode(strings.TrimSuffix(name, "."), pathMode); reason != "" {
		return nil, fmt.Errorf("%w: AddObject/DeleteObject object_name %q failed validation (%s)",
			ErrNoUsableParams, name, reason)
	}
	return marshalPathPayload(map[string]interface{}{"object_name": name}, pathMode)
}

func effectiveParamPath(ref MMLParamRef, pathMode string) string {
	if normalizeRawPathMode(pathMode) == rawPathModePrivate {
		if path := strings.TrimSpace(ref.PrivatePath); path != "" {
			return path
		}
	}
	return strings.TrimSpace(ref.Tr069Path)
}

func effectiveParamReadPath(ref MMLParamRef, pathMode string) string {
	path := effectiveParamPath(ref, pathMode)
	if path == "" {
		return ""
	}
	if normalizeRawPathMode(pathMode) != rawPathModePrivate && isInstanceObjectRef(ref) && !strings.HasSuffix(path, ".") {
		return objectCollectionPath(path)
	}
	if normalizeRawPathMode(pathMode) == rawPathModePrivate && isPrivateInstanceObjectRef(ref) && !strings.HasSuffix(path, ".") {
		return objectCollectionPath(path)
	}
	return path
}

func privatePathOrStandard(ref MMLParamRef) string {
	if path := strings.TrimSpace(ref.PrivatePath); path != "" {
		return path
	}
	return strings.TrimSpace(ref.Tr069Path)
}

func isPrivateInstanceObjectRef(ref MMLParamRef) bool {
	return isInstancePath(privatePathOrStandard(ref))
}

func objectCollectionPath(path string) string {
	path = strings.TrimSuffix(strings.TrimSpace(path), ".")
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return path
	}
	last := parts[len(parts)-1]
	if last == "{i}" || isDigits(last) {
		return strings.Join(parts[:len(parts)-1], ".") + "."
	}
	return path + "."
}

func isInstanceObjectRef(ref MMLParamRef) bool {
	return isInstancePath(ref.Tr069Path)
}

func isInstancePath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	path = strings.TrimSuffix(path, ".")
	if strings.HasSuffix(path, ".{i}") {
		return true
	}
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return false
	}
	return isDigits(parts[len(parts)-1])
}

func isDigits(last string) bool {
	if last == "" {
		return false
	}
	for _, r := range last {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func pathModeFromRefs(paramRefs []MMLParamRef, formValues map[string]interface{}) string {
	if formValues != nil {
		if mode, ok := formValues["path_mode"].(string); ok && normalizeRawPathMode(mode) == rawPathModePrivate {
			return rawPathModePrivate
		}
	}
	for _, ref := range paramRefs {
		if normalizeRawPathMode(ref.PathMode) == rawPathModePrivate {
			return rawPathModePrivate
		}
	}
	return rawPathModeStandard
}

func marshalPathPayload(payload map[string]interface{}, pathMode string) (json.RawMessage, error) {
	if normalizeRawPathMode(pathMode) == rawPathModePrivate {
		payload["path_mode"] = rawPathModePrivate
	}
	return json.Marshal(payload)
}

func filterPathsForMode(paths []string, pathMode string) (legal []string, skipped []SkippedPath) {
	if normalizeRawPathMode(pathMode) != rawPathModePrivate {
		return filterLegalPaths(paths)
	}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if reason := validateDirectPath(p); reason != "" {
			skipped = append(skipped, SkippedPath{Path: p, Reason: reason})
			continue
		}
		legal = append(legal, p)
	}
	return legal, skipped
}

func validatePathForMode(path, pathMode string) PathSkipReason {
	if normalizeRawPathMode(pathMode) == rawPathModePrivate {
		return validateDirectPath(path)
	}
	return validatePath(path)
}

func validateDirectPath(path string) PathSkipReason {
	path = strings.TrimSpace(path)
	if path == "" || !strings.Contains(path, ".") {
		return PathSkipBadPrefix
	}
	if paramPathPlaceholder.MatchString(path) {
		return PathSkipPlaceholder
	}
	if !paramPathLegalChars.MatchString(path) {
		return PathSkipBadChars
	}
	return ""
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

// normalizeTR069Value 规范化需要兼容基站数字布尔约定的协议值。
// 输入可能来自前端表单（"true"/"false"）、JSON bool 或历史脚本（"1"/"0"）。
func normalizeTR069Value(value, valueType string) string {
	return tr069.NormalizeValueForPath("", value, valueType)
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
