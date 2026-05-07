// Package response: testing helpers for envelope unwrapping.
//
// 历史背景：handler 在 v0.1 之前直接 `c.JSON(http.StatusOK, X)` 返回 X，
// 大量单测使用 `json.NewDecoder(w.Body).Decode(&resp)`（resp 类型为 X）。
// v0.1 起所有 handler 改走统一信封 {ret, msg, data}，这些 helper 用于把
// data 字段提取出来再 Unmarshal 成原始 X 类型，避免逐个测试重写解码逻辑。
package response

import (
	"encoding/json"
	"io"
	"testing"
)

// envelope 是测试侧用于解包的内部结构（与 response.OK / Fail 写出的 body 对齐）。
type envelope struct {
	Ret     int             `json:"ret"`
	Msg     string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
	BizCode int             `json:"biz_code,omitempty"`
}

// DecodeData 解析统一信封并把 data 段反序列化进 v。
//
// v 可以为 nil（仅断言 ret/msg 时使用）。
// 当 data 段为 null 或空时不写入 v；其他情况下不匹配会 t.Fatalf。
//
// 返回 envelope 的 ret 与 msg，便于额外断言。
func DecodeData(t *testing.T, body io.Reader, v any) (ret int, msg string) {
	t.Helper()
	var env envelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if v != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, v); err != nil {
			t.Fatalf("decode data: %v", err)
		}
	}
	return env.Ret, env.Msg
}

// DecodeFail 解析失败信封并返回 (msg, biz_code)。
//
// 用于 4xx/5xx 测试断言场景：body 形如 {ret:0, msg:"...", data:null, biz_code?:N}。
func DecodeFail(t *testing.T, body io.Reader) (msg string, bizCode int) {
	t.Helper()
	var env envelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env.Msg, env.BizCode
}
