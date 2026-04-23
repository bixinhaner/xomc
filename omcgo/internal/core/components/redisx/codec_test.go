package redisx_test

import (
	"bytes"
	"testing"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

func TestJSONCodec_RoundTrip(t *testing.T) {
	codec := redisx.JSONCodec{}

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	in := payload{Name: "alice", Count: 3}

	enc, err := codec.Encode(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Contains(enc, []byte(`"name":"alice"`)) {
		t.Fatalf("encode missing fields: %s", enc)
	}

	var out payload
	if err := codec.Decode(enc, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestJSONCodec_DecodeInvalid(t *testing.T) {
	codec := redisx.JSONCodec{}
	var out map[string]string
	if err := codec.Decode([]byte("{invalid}"), &out); err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestDefaultCodec_IsJSON(t *testing.T) {
	// 默认 codec 与 JSONCodec 行为一致。
	if _, ok := redisx.DefaultCodec.(redisx.JSONCodec); !ok {
		t.Fatalf("DefaultCodec should be JSONCodec by default; got %T", redisx.DefaultCodec)
	}
}
