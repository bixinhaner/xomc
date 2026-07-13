// response/timezone.go —— 响应序列化层的「系统时区统一转换」（issue #457，子单 B / 母单 #455）。
//
// 背景：DB 为 timestamptz，内存中时间一律 UTC；标准 time.Time 默认按 RFC3339 带偏移序列化，
// 但偏移恒为 +00:00（Z）。需求是「后端统一出口按系统时区转换后再返回前端」——即把同一 instant
// 以系统时区（来自 issue #456 的 systimezone.Provider，解析失败回落 UTC）的钟面时间 + 偏移展示，
// 例如系统设 Asia/Tokyo 时返回 +09:00。
//
// 单一改动点 = 本包 renderJSON（所有成功响应 OK / OKWithStatus / OKWithMsg 的统一出口）。
// 在序列化前递归把响应体内的时间值（标准 time.Time + 自定义 model.Time 两类）的 Location 改为
// 系统时区。instant（绝对时刻）不变，只改展示用的钟面 + 偏移。
//
// 北向排除：internal/northbound 路由组在中间件里打上下文标记（c.Set(ContextKeyNorthbound,true)），
// 北向对上游 OSS 必须保持 UTC 标准格式，转换层识别该标记后整体跳过，保持原样输出。
//
// 性能：反射遍历仅发生在响应序列化（非热路径）。按「类型是否可能含时间字段」做短路——
// 不含任何时间字段的类型（绝大多数大数组 DTO，如纯数值/字符串行）直接原样返回，不进递归。
// 含时间字段时才逐元素重建副本。类型探测结果按 reflect.Type 缓存，单类型只算一次。
//
// 配置生效：系统时区改后由 systimezone.Provider 的缓存失效刷新，后续响应即生效，无需重启 app。
package response

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ContextKeyNorthbound 北向响应的上下文标记键。北向路由组中间件 c.Set 此键为 true，
// 转换层据此整体跳过时区转换，保持 UTC 标准格式输出给上游 OSS。
const ContextKeyNorthbound = "northbound"

// TimezoneProvider 抽象「读取当前系统时区」的能力，由 app 启动期用 systimezone.Provider 注入。
// 解耦设计：response 包不直接依赖 systimezone 包，避免基础设施层互相耦合，也便于单测注入桩。
type TimezoneProvider interface {
	Location(ctx context.Context) *time.Location
}

// tzProvider 进程级系统时区入口。nil 时转换层不做任何转换（安全默认：保持 UTC 原样）。
// 由 SetTimezoneProvider 在 app 启动 wiring 时设置；并发安全由 tzMu 保护。
var (
	tzMu       sync.RWMutex
	tzProvider TimezoneProvider
)

// SetTimezoneProvider 注入系统时区入口。传 nil 可关闭转换（恢复 UTC 原样输出）。
// app 启动期调用一次即可；后续系统时区变更由 Provider 自身的缓存刷新承接，无需重设。
func SetTimezoneProvider(p TimezoneProvider) {
	tzMu.Lock()
	tzProvider = p
	tzMu.Unlock()
}

// currentLocation 返回当前应使用的展示时区；未注入 Provider 时返回 nil（表示不转换）。
func currentLocation(ctx context.Context) *time.Location {
	tzMu.RLock()
	p := tzProvider
	tzMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.Location(ctx)
}

// TimeInCurrentLocation returns t converted to the currently configured system
// timezone. It is intended for streaming responses (for example CSV) that cannot
// pass through renderJSON's recursive conversion path.
func TimeInCurrentLocation(ctx context.Context, t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	loc := currentLocation(ctx)
	if loc == nil {
		return t
	}
	return t.In(loc)
}

// FormatTimeInCurrentLocation formats an optional time using the same system
// timezone source as JSON success responses. Nil or zero values return "".
func FormatTimeInCurrentLocation(ctx context.Context, t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return TimeInCurrentLocation(ctx, *t).Format(time.RFC3339)
}

// timeType / modelTimeType 用于反射识别两类需转换的时间值。
var (
	timeType      = reflect.TypeOf(time.Time{})
	modelTimeType = reflect.TypeOf(model.Time{})
)

// convertResponseTimezone 把 body 内所有时间值（time.Time + model.Time）转换为系统时区后返回副本。
// 满足以下任一条件时原样返回入参（不转换、零拷贝）：
//   - 北向响应（上下文标记 ContextKeyNorthbound=true）→ 保持 UTC；
//   - 未注入 TimezoneProvider；
//   - body 类型不可能含时间字段（短路）。
func convertResponseTimezone(c *gin.Context, body any) any {
	if c != nil {
		if v, ok := c.Get(ContextKeyNorthbound); ok {
			if b, _ := v.(bool); b {
				return body // 北向：保持 UTC 原样
			}
		}
	}
	loc := currentLocation(c)
	if loc == nil {
		return body // 未注入 Provider：不转换
	}
	if body == nil {
		return nil
	}
	rv := reflect.ValueOf(body)
	if !typeMayContainTime(rv.Type()) {
		return body // 短路：该类型不可能含时间字段
	}
	return convertValue(rv, loc)
}

// convertValue 返回把时间值改到 loc 时区后的副本（any）。仅在类型可能含时间时被调用。
func convertValue(rv reflect.Value, loc *time.Location) any {
	if !rv.IsValid() {
		return nil
	}

	switch rv.Type() {
	case timeType:
		t := rv.Interface().(time.Time)
		if t.IsZero() {
			return t // 零值时间：保持原样，绝不转出脏偏移
		}
		return t.In(loc)
	case modelTimeType:
		mt := rv.Interface().(model.Time)
		if mt.IsZero() {
			return mt // 零值时间：保持原样
		}
		return model.Time(mt.Std().In(loc))
	}

	switch rv.Kind() {
	case reflect.Interface, reflect.Ptr:
		if rv.IsNil() {
			return rv.Interface()
		}
		elem := rv.Elem()
		if !typeMayContainTime(elem.Type()) {
			return rv.Interface()
		}
		return convertValue(elem, loc)

	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return rv.Interface()
		}
		if !typeMayContainTime(rv.Type().Elem()) {
			return rv.Interface() // 元素类型无时间：整段短路，不复制
		}
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = convertValue(rv.Index(i), loc)
		}
		return out

	case reflect.Map:
		if rv.IsNil() {
			return rv.Interface()
		}
		if !typeMayContainTime(rv.Type().Elem()) {
			return rv.Interface() // 值类型无时间：整段短路
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[mapKeyString(iter.Key())] = convertValue(iter.Value(), loc)
		}
		return out

	case reflect.Struct:
		return convertStruct(rv, loc)

	default:
		return rv.Interface()
	}
}

// convertStruct 遍历导出字段产出 map，转换其中的时间字段。遵循 json tag 的重命名 / "-" 忽略、
// omitempty 省略、匿名嵌入内联（与 encoding/json 同语义，保证开启时区后输出契约不回归）。
// 仅在 typeMayContainTime 为真的结构上触发。
//
// omitempty 保真（issue #457 回合2 修复）：结构体被重建为 map 时，带 omitempty 且值为空的字段
// 必须像 encoding/json 那样省略——否则开启系统时区后，任何与时间字段同居一结构体的 omitempty
// 字段会从「被省略」变成 null/0/""，破坏前端契约（如 Device 的 product_id/status/deleted_at 等）。
func convertStruct(rv reflect.Value, loc *time.Location) any {
	t := rv.Type()
	out := make(map[string]any, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue // 未导出字段跳过
		}
		name, skip, omitEmpty := jsonFieldTag(field)
		if skip {
			continue
		}
		fv := rv.Field(i)
		if name == "" && field.Anonymous {
			// 匿名嵌入且无 json tag：内联其字段（与 encoding/json 一致）。
			// 注：嵌入字段自身的 omitempty（罕见）由 encoding/json 按整体值判定，此处与其对齐
			// 仅对「值为空」的嵌入跳过内联。
			if omitEmpty && isEmptyJSONValue(fv) {
				continue
			}
			inner := convertValue(fv, loc)
			if m, ok := inner.(map[string]any); ok {
				for k, val := range m {
					out[k] = val
				}
				continue
			}
		}
		if omitEmpty && isEmptyJSONValue(fv) {
			continue // omitempty：空值字段省略，保持与 encoding/json 一致的输出契约
		}
		if name == "" {
			name = field.Name
		}
		out[name] = convertValue(fv, loc)
	}
	return out
}

// typeMayContainTime 判断给定类型「是否可能含有需转换的时间值」。
// 结果按 reflect.Type 缓存（单类型只算一次），使不含时间的大数组 DTO 在序列化时直接短路、
// 不进反射递归。对自引用类型用 inProgress 集合防无限递归。
var timeContainCache sync.Map // map[reflect.Type]bool

func typeMayContainTime(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if v, ok := timeContainCache.Load(t); ok {
		return v.(bool)
	}
	res := computeMayContainTime(t, map[reflect.Type]bool{})
	timeContainCache.Store(t, res)
	return res
}

func computeMayContainTime(t reflect.Type, inProgress map[reflect.Type]bool) bool {
	if t == nil {
		return false
	}
	if t == timeType || t == modelTimeType {
		return true
	}
	if inProgress[t] {
		return false // 自引用：当前路径未发现时间，避免死循环
	}
	inProgress[t] = true
	defer delete(inProgress, t)

	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return computeMayContainTime(t.Elem(), inProgress)
	case reflect.Map:
		return computeMayContainTime(t.Elem(), inProgress)
	case reflect.Interface:
		// 接口动态类型未知（如 gin.H 的 any 值 / data: any）：保守视为可能含时间。
		return true
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue // 未导出字段不参与 JSON，亦不转换
			}
			if _, skip := jsonFieldName(field); skip {
				continue
			}
			if computeMayContainTime(field.Type, inProgress) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
