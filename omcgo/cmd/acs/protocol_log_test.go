package main

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

// issue #218：协议日志级别解析。空/非法回退 warn（默认抑制每报文全量 XML 编码），
// 合法值按字面解析（排障时可调 info/debug 恢复全量抓包）。
func TestParseProtocolLogLevel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want zapcore.Level
	}{
		// 成功路径：合法级别按字面解析。
		{"debug", "debug", zapcore.DebugLevel},
		{"info", "info", zapcore.InfoLevel},
		{"warn", "warn", zapcore.WarnLevel},
		{"error", "error", zapcore.ErrorLevel},
		{"mixed-case info", "Info", zapcore.InfoLevel},
		{"padded warn", "  warn  ", zapcore.WarnLevel},
		// 失败/缺省路径：空串与非法值回退 warn（稳态零开销默认）。
		{"empty defaults warn", "", zapcore.WarnLevel},
		{"blank defaults warn", "   ", zapcore.WarnLevel},
		{"garbage defaults warn", "verbose", zapcore.WarnLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseProtocolLogLevel(tc.in); got != tc.want {
				t.Fatalf("parseProtocolLogLevel(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
