package minio

import (
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// qa-614 #377: PublicEndpoint 为空回退内部 endpoint 时应打 Warn 日志，
// 把「该配 public_endpoint，否则浏览器解析不了」显式化（成功路径=有日志）。
func TestNewPresignClient_FallbackWarnsWhenPublicEndpointEmpty(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	log := zap.New(core)

	cfg := appconfig.MinIOConfig{
		Endpoint:       "minio:9000",
		AccessKey:      "ak",
		SecretKey:      "sk",
		PublicEndpoint: "", // 触发回退
	}

	client, err := NewPresignClient(cfg, log)
	if err != nil {
		t.Fatalf("NewPresignClient err = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("client = nil, want non-nil")
	}

	entries := logs.FilterLevelExact(zap.WarnLevel).All()
	if len(entries) != 1 {
		t.Fatalf("warn log count = %d, want 1", len(entries))
	}
	if !strings.Contains(entries[0].Message, "public_endpoint") {
		t.Errorf("warn message %q does not mention public_endpoint", entries[0].Message)
	}
	// 内部 endpoint 应记录在结构化字段中，便于运维定位。
	if got := entries[0].ContextMap()["internal_endpoint"]; got != "minio:9000" {
		t.Errorf("internal_endpoint field = %v, want minio:9000", got)
	}
}

// 失败路径的镜像：PublicEndpoint 非空时不应回退、不应打 Warn（用对外 host 签发）。
func TestNewPresignClient_NoWarnWhenPublicEndpointSet(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	log := zap.New(core)

	cfg := appconfig.MinIOConfig{
		Endpoint:       "minio:9000",
		AccessKey:      "ak",
		SecretKey:      "sk",
		PublicEndpoint: "files.example.com:9000",
	}

	client, err := NewPresignClient(cfg, log)
	if err != nil {
		t.Fatalf("NewPresignClient err = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("client = nil, want non-nil")
	}
	if n := logs.FilterLevelExact(zap.WarnLevel).Len(); n != 0 {
		t.Errorf("warn log count = %d, want 0 (public_endpoint set, no fallback)", n)
	}
}

// 不传 logger 时回退应静默且不 panic（兼容既有调用点）。
func TestNewPresignClient_NoLoggerSilentFallback(t *testing.T) {
	cfg := appconfig.MinIOConfig{
		Endpoint:       "minio:9000",
		AccessKey:      "ak",
		SecretKey:      "sk",
		PublicEndpoint: "",
	}
	client, err := NewPresignClient(cfg)
	if err != nil {
		t.Fatalf("NewPresignClient err = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("client = nil, want non-nil")
	}
}
