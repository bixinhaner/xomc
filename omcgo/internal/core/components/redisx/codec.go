package redisx

import (
	"encoding/json"
	"fmt"
)

// Codec 抽象值的序列化。默认 JSON 实现；业务侧可自行替换为 msgpack 等。
//
// 采用接口而非具体函数，使得 task.RedisTaskQueue 等调用方可把 Codec 注入进来，
// 单测里替换为 identity codec 以便断言写入的具体字节。
type Codec interface {
	Encode(v any) ([]byte, error)
	Decode(b []byte, v any) error
}

// JSONCodec 是默认的 JSON 编解码实现。
type JSONCodec struct{}

// Encode 把 v 序列化为 JSON 字节流，失败时 wrap 为带调用上下文的 error。
func (JSONCodec) Encode(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("redisx json encode: %w", err)
	}
	return b, nil
}

// Decode 反序列化 JSON 字节流到 v。
func (JSONCodec) Decode(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("redisx json decode: %w", err)
	}
	return nil
}

// DefaultCodec 项目统一使用的 Codec 实例。
var DefaultCodec Codec = JSONCodec{}
