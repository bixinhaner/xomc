package parammodel

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MappingValidator 是 P2-07 T-0098 引入的"映射型"参数校验器，等价于旧
// `datamodel.ParameterValidator`，但只消费 ParamMapping（来自 P2-02 Registry
// 返回的 MappingSet），而非完整 datamodel.DataModel。
//
// 设计契约（设计 §1.11 / §1.12）：
//   - 索引按 PrivatePath 建立（设备实际上报路径），LookupParam / LookupObject
//     入参亦为设备视角的 PrivatePath
//   - param_mappings 表元数据集仅 5 项（access / data_type / change_applies /
//     min_value / max_value），故无 max_instances / min_instances / forced_inform
//     / description 等概念；ValidateAddObject / ValidateDeleteObject 退化为
//     "权限 + entry_type 检查"，不强制实例数上下限
//   - 模板路径形如 "Foo.{i}.Bar" 与运行时实例路径 "Foo.1.Bar" 通过
//     normalizeInstancePath 折叠为同一索引键，与旧 datamodel.ExtractInstanceNumbers
//     等价
type MappingValidator struct {
	set            *MappingSet
	byPrivateExact map[string]ParamMapping // 原样 PrivatePath（含 "{i}"）
	byPrivateNorm  map[string]ParamMapping // 实例号 → "{i}" 归一化
}

// NewMappingValidator 构造 MappingValidator。set 为 nil 时返回 nil（调用方需做空检查）。
func NewMappingValidator(set *MappingSet) *MappingValidator {
	if set == nil {
		return nil
	}
	v := &MappingValidator{
		set:            set,
		byPrivateExact: make(map[string]ParamMapping, len(set.Mappings)),
		byPrivateNorm:  make(map[string]ParamMapping, len(set.Mappings)),
	}
	for _, m := range set.Mappings {
		v.byPrivateExact[m.PrivatePath] = m
		v.byPrivateNorm[normalizeInstancePath(m.PrivatePath)] = m
	}
	return v
}

// MappingValidationError 是 MappingValidator 各 Validate* 方法的错误返回。
//
// 与旧 datamodel.ValidationError 字段集兼容（Path/Code/Message），便于消费者按需适配。
type MappingValidationError struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *MappingValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Path, e.Message)
}

// LookupParam 按设备 privatePath 查找 entry_type=parameter 的映射。
//
// 命中规则：先精确匹配 privatePath；未命中则按实例归一化匹配（运行时 "Foo.1.Bar"
// 也能命中模板 "Foo.{i}.Bar"）。返回 nil 表示该路径不在当前 MappingSet 中。
func (v *MappingValidator) LookupParam(privatePath string) *ParamMapping {
	if v == nil {
		return nil
	}
	if m, ok := v.byPrivateExact[privatePath]; ok && m.EntryType == "parameter" {
		cp := m
		return &cp
	}
	if m, ok := v.byPrivateNorm[normalizeInstancePath(privatePath)]; ok && m.EntryType == "parameter" {
		cp := m
		return &cp
	}
	return nil
}

// LookupObject 按 privatePath 查找 entry_type=object 的映射（含末尾 "."）。
func (v *MappingValidator) LookupObject(privatePath string) *ParamMapping {
	if v == nil {
		return nil
	}
	for _, candidate := range objectLookupCandidates(privatePath) {
		if m, ok := v.byPrivateExact[candidate]; ok && m.EntryType == "object" {
			cp := m
			return &cp
		}
		if m, ok := v.byPrivateNorm[normalizeInstancePath(candidate)]; ok && m.EntryType == "object" {
			cp := m
			return &cp
		}
	}
	return nil
}

// lookupObjectCollection 按 AddObject 的集合路径优先查找其多实例模板。
// 例如 DeviceGSM.Bts. 必须由 DeviceGSM.Bts.{i}. 授权，不能误用同名的
// 无索引单实例对象 DeviceGSM.Bts.。旧模型若只声明集合路径，调用方仍可回退到
// LookupObject，保持兼容。
func (v *MappingValidator) lookupObjectCollection(collectionPath string) *ParamMapping {
	if v == nil {
		return nil
	}
	collectionPath = strings.TrimSpace(collectionPath)
	if collectionPath == "" {
		return nil
	}
	if !strings.HasSuffix(collectionPath, ".") {
		collectionPath += "."
	}
	templatePath := collectionPath
	if !strings.HasSuffix(templatePath, ".{i}.") {
		templatePath += "{i}."
	}
	if m, ok := v.byPrivateExact[templatePath]; ok && m.EntryType == "object" {
		cp := m
		return &cp
	}
	if m, ok := v.byPrivateNorm[normalizeInstancePath(templatePath)]; ok && m.EntryType == "object" {
		cp := m
		return &cp
	}
	return nil
}

func objectLookupCandidates(privatePath string) []string {
	trimmed := strings.TrimSpace(privatePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasSuffix(trimmed, ".") {
		return []string{
			trimmed,
			strings.TrimSuffix(trimmed, "."),
			strings.TrimSuffix(trimmed, ".") + ".{i}.",
		}
	}
	return []string{trimmed, trimmed + ".", trimmed + ".{i}."}
}

// ValidateValue 按 ParamMapping 的元属性校验单条 set 请求。
//
// 错误码与 datamodel.ValidationError 对齐：
//   - "not_found"     → mapping 中无此 privatePath
//   - "not_writable"  → access != readWrite/writeOnly
//   - "type_mismatch" → 数值类型解析失败
//   - "out_of_range"  → 数值超出 min/max
//
// 注意：没有 enum / pattern / forced inform 概念（param_mappings 不存这些信息）；
// 命中后仅做 access 与数值范围两类检查。
func (v *MappingValidator) ValidateValue(privatePath, value string) *MappingValidationError {
	if v == nil {
		return nil
	}
	m := v.LookupParam(privatePath)
	if m == nil {
		return &MappingValidationError{Path: privatePath, Code: "not_found", Message: "parameter not found in mapping"}
	}
	if !isWritable(m.Access) {
		return &MappingValidationError{Path: privatePath, Code: "not_writable", Message: fmt.Sprintf("access=%s", m.Access)}
	}
	if m.MinValue == nil && m.MaxValue == nil {
		return nil
	}
	if strings.EqualFold(m.DataType, "string") {
		length := int64(utf8.RuneCountInString(value))
		if m.MinValue != nil && length < *m.MinValue {
			return &MappingValidationError{Path: privatePath, Code: "out_of_range", Message: fmt.Sprintf("length %d < min %d", length, *m.MinValue)}
		}
		if m.MaxValue != nil && length > *m.MaxValue {
			return &MappingValidationError{Path: privatePath, Code: "out_of_range", Message: fmt.Sprintf("length %d > max %d", length, *m.MaxValue)}
		}
		return nil
	}
	num, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		// 仅对显式有数值范围的字段报 type_mismatch；
		// 字符串类参数（无 min/max）已在上面 return nil 提前退出
		return &MappingValidationError{Path: privatePath, Code: "type_mismatch", Message: fmt.Sprintf("expected integer: %v", err)}
	}
	if m.MinValue != nil && num < *m.MinValue {
		return &MappingValidationError{Path: privatePath, Code: "out_of_range", Message: fmt.Sprintf("%d < min %d", num, *m.MinValue)}
	}
	if m.MaxValue != nil && num > *m.MaxValue {
		return &MappingValidationError{Path: privatePath, Code: "out_of_range", Message: fmt.Sprintf("%d > max %d", num, *m.MaxValue)}
	}
	return nil
}

// ValidateAddObject 校验 AddObject 请求是否合法。
//
// 退化语义（相对 datamodel.ParameterValidator）：
//   - 命中且 entry_type=object 且 access 含 write → 通过
//   - 实例数上限：param_mappings 不存 max_instances，本方法不做上限检查；
//     上限改由 P3-01 端点结合 product/swVersion 元数据另行处理
func (v *MappingValidator) ValidateAddObject(objectPrefix string, _ int) *MappingValidationError {
	if v == nil {
		return nil
	}
	m := v.lookupObjectCollection(objectPrefix)
	if m == nil {
		m = v.LookupObject(objectPrefix)
	}
	if m == nil {
		return &MappingValidationError{Path: objectPrefix, Code: "not_found", Message: "object not found in mapping"}
	}
	if !isWritable(m.Access) {
		return &MappingValidationError{Path: objectPrefix, Code: "not_writable", Message: fmt.Sprintf("access=%s", m.Access)}
	}
	return nil
}

// ValidateDeleteObject 校验 DeleteObject 请求是否合法。
//
// 退化语义：与 ValidateAddObject 对称；min_instances 概念不在 mapping 范畴内，
// 不做下限检查。
func (v *MappingValidator) ValidateDeleteObject(objectPrefix string, _ int) *MappingValidationError {
	if v == nil {
		return nil
	}
	m := v.lookupObjectCollection(objectPrefix)
	if m == nil {
		m = v.LookupObject(objectPrefix)
	}
	if m == nil {
		return &MappingValidationError{Path: objectPrefix, Code: "not_found", Message: "object not found in mapping"}
	}
	if !isWritable(m.Access) {
		return &MappingValidationError{Path: objectPrefix, Code: "not_writable", Message: fmt.Sprintf("access=%s", m.Access)}
	}
	return nil
}

// Source 透传 MappingSet.Source（discovered / default），便于上层日志/指标。
func (v *MappingValidator) Source() MappingSource {
	if v == nil || v.set == nil {
		return ""
	}
	return v.set.Source
}

// Mappings 返回原始 ParamMapping 列表（保留原序）。供消费者批量遍历。
func (v *MappingValidator) Mappings() []ParamMapping {
	if v == nil || v.set == nil {
		return nil
	}
	return v.set.Mappings
}

// ── helpers ──────────────────────────────────────────────────────────

// normalizeInstancePath 把运行时实例段（连续数字片段）替换为模板占位符 "{i}"，
// 使得 "Foo.1.Bar" 与模板 "Foo.{i}.Bar" 命中同一索引项。
//
// 末尾分号点（"."）保留。仅替换 ".<num>." 形态的片段；其它字符不动。
func normalizeInstancePath(p string) string {
	if p == "" {
		return p
	}
	parts := strings.Split(p, ".")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		if isAllDigits(seg) {
			parts[i] = "{i}"
		}
	}
	return strings.Join(parts, ".")
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsAccessWritable 判定 access 字符串是否可写。容忍多种 XML 命名（readWrite / read_write / RW / writeOnly）。
//
// 导出供 P2-08 interop / P2-04 sync.go 等消费者复用，避免命名漂移。
func IsAccessWritable(access string) bool {
	switch strings.ToLower(strings.ReplaceAll(access, "_", "")) {
	case "readwrite", "rw", "writeonly", "wo", "writeable":
		return true
	}
	return false
}

// isWritable 是 IsAccessWritable 的内部别名（兼容现有调用点）。
func isWritable(access string) bool { return IsAccessWritable(access) }
