// response/sanitize.go —— 响应序列化层的「非有限浮点」集中兜底（issue #387）。
//
// 背景：Go 的 encoding/json 遇到非有限浮点（NaN / +Inf / -Inf）直接返回 error；
// 而 Gin 的 c.JSON 是「先写 200 状态头、再边编码边写 body」，一旦编码到非有限浮点
// 那一行就中断——客户端收到「HTTP 200 + 空 body」，前端误判为「暂无数据」。
//
// PM 聚合结果里的「平均型 / 比率型」指标，分母（样本数）为 0 时会合法地算出 NaN，
// 是真实数据普遍会踩的坑。受影响的不止某一个端点：adhoc results、/pm/metrics/aggregated、
// /pm/counters、/pm/counters/aggregated、/pm/kpis 等所有回传浮点指标值的端点都同病。
//
// 集中兜底落点 = 本包的 OK/OKWithStatus/OKWithMsg 统一调用 renderJSON，在真正写出 body 前
// 对 payload 做一次「非有限浮点 → null」的 sanitize。这样所有走统一响应信封的端点（含未来
// 新增的端点）一次性免疫，无需逐个 DTO 改类型。null 表达「缺测」语义，绝不伪造成 0 以免污染统计。
//
// 性能与保真：只有当 payload 内确实存在非有限浮点时，才把对应结构净化为 map 副本；
// 不含非有限浮点的绝大多数响应原样交给 encoding/json，输出与修复前逐字节一致
// （omitempty / ,string / 字段顺序等语义完全不受影响）。
package response

import (
	"math"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
)

// renderJSON 是所有成功响应（OK / OKWithStatus / OKWithMsg）的统一出口。
// 序列化前依次做两层处理（顺序无关，互不影响）：
//  1. 时区转换（issue #457）：把响应体内时间值（time.Time / model.Time）按系统时区展示；
//     北向响应 / 未注入 Provider / 类型不含时间时短路，原样输出 UTC。
//  2. 非有限浮点兜底（issue #387）：把 NaN/±Inf 规整为 null，避免 Gin 边写边编码中断。
func renderJSON(c *gin.Context, statusCode int, body gin.H) {
	converted := convertResponseTimezone(c, body)
	c.JSON(statusCode, sanitizeNonFiniteFloats(converted))
}

// jsonMarshalerType 用于识别自定义 JSON 序列化类型（如 time.Time、model.Time、jsonx.Float），
// 遇到则原样保留不拆解——它们的输出由自身 MarshalJSON 决定，不会产生裸 NaN。
var jsonMarshalerType = reflect.TypeOf((*interface{ MarshalJSON() ([]byte, error) })(nil)).Elem()

// sanitizeNonFiniteFloats 返回一个「净化后的副本」：把其中的非有限浮点替换为 nil（JSON null）。
// 若整个 payload 不含非有限浮点，则原样返回入参，让 encoding/json 以完全保真的方式编码。
func sanitizeNonFiniteFloats(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if !containsNonFinite(rv) {
		return v
	}
	return sanitizeValue(rv)
}

// containsNonFinite 深度探测值内是否存在非有限浮点。命中自定义 Marshaler 即停止下钻
// （其输出自洽），故 time.Time / jsonx.Float 等不会被误判。
func containsNonFinite(rv reflect.Value) bool {
	if !rv.IsValid() {
		return false
	}
	if implementsJSONMarshaler(rv) {
		return false
	}
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		return math.IsNaN(f) || math.IsInf(f, 0)
	case reflect.Interface, reflect.Ptr:
		return !rv.IsNil() && containsNonFinite(rv.Elem())
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if containsNonFinite(rv.Index(i)) {
				return true
			}
		}
		return false
	case reflect.Map:
		iter := rv.MapRange()
		for iter.Next() {
			if containsNonFinite(iter.Value()) {
				return true
			}
		}
		return false
	case reflect.Struct:
		t := rv.Type()
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).PkgPath != "" {
				continue // 未导出字段
			}
			if containsNonFinite(rv.Field(i)) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func implementsJSONMarshaler(rv reflect.Value) bool {
	if rv.Type().Implements(jsonMarshalerType) {
		return true
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(jsonMarshalerType) {
		return true
	}
	return false
}

// sanitizeValue 把值净化为可安全 JSON 序列化的副本（非有限浮点 → nil）。
// 仅在 containsNonFinite 为真的子树上调用，故对未受影响子树仍会复制（可接受，调用极罕见）。
func sanitizeValue(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}
	if implementsJSONMarshaler(rv) {
		if rv.Type().Implements(jsonMarshalerType) {
			return rv.Interface()
		}
		return rv.Addr().Interface()
	}

	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		return rv.Interface()

	case reflect.Interface, reflect.Ptr:
		if rv.IsNil() {
			return nil
		}
		return sanitizeValue(rv.Elem())

	case reflect.Slice:
		if rv.IsNil() {
			return rv.Interface()
		}
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = sanitizeValue(rv.Index(i))
		}
		return out

	case reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = sanitizeValue(rv.Index(i))
		}
		return out

	case reflect.Map:
		if rv.IsNil() {
			return rv.Interface()
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[mapKeyString(iter.Key())] = sanitizeValue(iter.Value())
		}
		return out

	case reflect.Struct:
		return sanitizeStruct(rv)

	default:
		return rv.Interface()
	}
}

// sanitizeStruct 遍历导出字段产出 map，净化浮点字段。遵循 json tag 的重命名与 "-" 忽略，
// 以及匿名嵌入字段的内联。注意：本路径仅在子树含非有限浮点时触发（极罕见，均为 PM 数值行），
// omitempty 在此从简不实现，对净化后的数值行无影响。
func sanitizeStruct(rv reflect.Value) any {
	t := rv.Type()
	out := make(map[string]any, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue // 未导出字段跳过
		}
		name, skip := jsonFieldName(field)
		if skip {
			continue
		}
		fv := rv.Field(i)
		if name == "" && field.Anonymous {
			// 匿名嵌入且无 json tag：内联其字段（与 encoding/json 行为一致）。
			inner := sanitizeValue(fv)
			if m, ok := inner.(map[string]any); ok {
				for k, val := range m {
					out[k] = val
				}
				continue
			}
		}
		if name == "" {
			name = field.Name
		}
		out[name] = sanitizeValue(fv)
	}
	return out
}

// jsonFieldName 解析 json tag，返回字段名与是否忽略。
// 空名 + 非忽略表示「用字段原名」（或匿名嵌入内联，由调用方判断）。
func jsonFieldName(field reflect.StructField) (name string, skip bool) {
	name, skip, _ = jsonFieldTag(field)
	return name, skip
}

// jsonFieldTag 解析 json tag，返回字段名、是否忽略、是否带 omitempty 选项。
// 与 encoding/json 对 tag 的解析语义一致（逗号前为名字，逗号后为选项列表）。
func jsonFieldTag(field reflect.StructField) (name string, skip bool, omitEmpty bool) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false, false // 无 tag：用字段原名
	}
	if tag == "-" {
		return "", true, false // 显式忽略
	}
	// 逗号前是名字，逗号后是 omitempty/string 等选项。
	nameEnd := len(tag)
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			nameEnd = i
			break
		}
	}
	name = tag[:nameEnd]
	if nameEnd < len(tag) {
		opts := tag[nameEnd+1:]
		// 选项以逗号分隔，逐个比对 omitempty。
		for len(opts) > 0 {
			var opt string
			if j := indexByte(opts, ','); j >= 0 {
				opt, opts = opts[:j], opts[j+1:]
			} else {
				opt, opts = opts, ""
			}
			if opt == "omitempty" {
				omitEmpty = true
			}
		}
	}
	return name, false, omitEmpty
}

// indexByte 返回 b 在 s 中首次出现的下标，未找到返回 -1（避免引入额外 import）。
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// isEmptyJSONValue 复刻 encoding/json 的 isEmptyValue 语义：
// 带 omitempty 的字段在值为「空」时被省略——空数组/切片/map/字符串、false、0 数值、nil 指针/接口。
func isEmptyJSONValue(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return rv.IsNil()
	}
	return false
}

// mapKeyString 把 map 键转成字符串（JSON 对象键必须为字符串）。
func mapKeyString(k reflect.Value) string {
	switch k.Kind() {
	case reflect.String:
		return k.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(k.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(k.Uint(), 10)
	default:
		// 其余类型回退到 %v；JSON 对象键极少用非字符串/整数键。
		return reflect.ValueOf(k.Interface()).String()
	}
}
