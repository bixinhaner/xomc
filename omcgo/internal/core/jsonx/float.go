// Package jsonx 提供 JSON 序列化层的安全兜底类型。
//
// 背景（issue #387）：Go 的 encoding/json 遇到非有限浮点（NaN / +Inf / -Inf）
// 直接返回 error；而 Gin 是「先写 200 状态头、再边编码边写 body」，编码到非有限
// 浮点那一行即中断——客户端收到「HTTP 200 + 空 body」，前端误判为「暂无数据」。
//
// PM 聚合结果里的「平均型 / 比率型」指标，分母（样本数）为 0 时合法地算出 NaN，
// 是真实数据普遍会踩的坑。本包提供 Float —— 一个底层为 float64 的具名类型，
// 自定义 MarshalJSON 把非有限值规整为 JSON null（缺测语义；绝不伪造成 0 以免污染统计）。
package jsonx

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Float 是 JSON 安全的 float64：非有限值（NaN / +Inf / -Inf）序列化为 null，
// 有限值按普通 JSON number 输出。底层是 float64，可直接参与算术与 pgx 扫描。
type Float float64

// nullLiteral 是 JSON null 的字节表示，复用避免反复分配。
var nullLiteral = []byte("null")

// MarshalJSON 实现 json.Marshaler：非有限值 → null，有限值 → JSON number。
// 有限值委托给 encoding/json 编码原生 float64，确保数字格式与全局其它响应完全一致。
func (f Float) MarshalJSON() ([]byte, error) {
	v := float64(f)
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nullLiteral, nil
	}
	return json.Marshal(v)
}

// Scan 实现 sql.Scanner，使 pgx / database/sql 能把 double precision 列扫描进本类型。
// 仅靠具名 float64 不足以让 pgx 默认扫描计划接受指针目标，显式实现 Scanner 是稳妥契约。
// PostgreSQL 的 'NaN'/'Infinity'/'-Infinity' 会以 float64 的 NaN/Inf 进来，原样保留——
// 序列化时由 MarshalJSON 统一兜底为 null。
func (f *Float) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*f = Float(math.NaN())
	case float64:
		*f = Float(v)
	case float32:
		*f = Float(float64(v))
	case int64:
		*f = Float(float64(v))
	case []byte:
		parsed, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return fmt.Errorf("jsonx.Float scan []byte %q: %w", v, err)
		}
		*f = Float(parsed)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("jsonx.Float scan string %q: %w", v, err)
		}
		*f = Float(parsed)
	default:
		return fmt.Errorf("jsonx.Float: unsupported scan source type %T", src)
	}
	return nil
}
