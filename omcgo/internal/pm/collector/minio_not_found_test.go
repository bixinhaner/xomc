package collector

import (
	"errors"
	"fmt"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/core/reliability"
)

// TestIsMinIONotFound 覆盖 isMinIONotFound 的判定分支。
//
// 2026-07-22 压测实测（20000设备规模，磁盘长期逼近满载）：MinIO 对象在 worker 处理前
// 已消失（疑似磁盘压力下的写入/清理异常），导致 GetObject 流首次 Read 时暴露
// NoSuchKey/NoSuchBucket。这类错误重试注定失败（同一个已不存在的 key 重试多少次
// 结果都一样），之前走通用重试+指数退避（约7秒/条），在队列被此类消息大量淹没时
// 会显著拖慢真正可处理消息的吞吐。
func TestIsMinIONotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "NoSuchKey", err: minio.ErrorResponse{Code: "NoSuchKey"}, want: true},
		{name: "NoSuchBucket", err: minio.ErrorResponse{Code: "NoSuchBucket"}, want: true},
		{name: "other minio error", err: minio.ErrorResponse{Code: "AccessDenied"}, want: false},
		{name: "unrelated error", err: errors.New("connection refused"), want: false},
		{
			name: "wrapped NoSuchKey still detected",
			err:  fmt.Errorf("download pm file: %w", minio.ErrorResponse{Code: "NoSuchKey"}),
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isMinIONotFound(tc.err))
		})
	}
}

// TestIsMinIONotFound_ErrorWrappingIsPermanent 验证 handleFileReceived 里对
// NoSuchKey/NoSuchBucket 的包装方式真的能让 errors.Is(err, reliability.ErrPermanent)
// 成立——这是 Runner/NATSEventBus 两层短路重试机制的前提条件（缺一层都不会真正短路，
// 见 internal/core/reliability/retry.go 与 internal/core/event/nats_bus.go 的
// errors.Is(err, ErrPermanent) 判定）。
func TestIsMinIONotFound_ErrorWrappingIsPermanent(t *testing.T) {
	notFoundErr := minio.ErrorResponse{Code: "NoSuchKey"}

	wrapped := fmt.Errorf("parse pm xml: %w: %w", notFoundErr, reliability.ErrPermanent)

	assert.True(t, isMinIONotFound(notFoundErr), "sanity: raw error must be detected as not-found")
	assert.True(t, errors.Is(wrapped, reliability.ErrPermanent),
		"NoSuchKey must be permanent so the runner/event bus stop retrying immediately")
	assert.Contains(t, wrapped.Error(), "parse pm xml",
		"context prefix must stay visible in the final error/DLQ message")
}
