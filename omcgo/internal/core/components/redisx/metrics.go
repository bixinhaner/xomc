package redisx

import (
	"context"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// PrometheusHook 实现 redis.Hook，为每次 Redis 命令/管线打点到 Prometheus。
//
// 使用方式：
//   hook := redisx.NewPrometheusHook(monitor.RedisOpDuration)
//   client, _ := redisx.NewClient(cfg, hook)
//
// 指标标签：operation = 命令名称（GET/SET/...）或 "PIPELINE(N)"。
type PrometheusHook struct {
	histogram *prometheus.HistogramVec
}

// NewPrometheusHook 创建一个 hook，把命令耗时写入 histogram。
// histogram 应为带一个 "operation" 标签的 HistogramVec（例如
// monitor.InfraMetrics.RedisOpDuration），调用方负责注册到 registry。
func NewPrometheusHook(histogram *prometheus.HistogramVec) *PrometheusHook {
	return &PrometheusHook{histogram: histogram}
}

// DialHook 透传——连接建立不在本指标覆盖范围内。
func (h *PrometheusHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook 为单条命令打点。
func (h *PrometheusHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h == nil || h.histogram == nil {
			return next(ctx, cmd)
		}
		start := time.Now()
		err := next(ctx, cmd)
		h.histogram.WithLabelValues(cmd.Name()).Observe(time.Since(start).Seconds())
		return err
	}
}

// ProcessPipelineHook 为整条 pipeline 打点。operation 标签形如 "PIPELINE(3)"。
func (h *PrometheusHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h == nil || h.histogram == nil {
			return next(ctx, cmds)
		}
		start := time.Now()
		err := next(ctx, cmds)
		h.histogram.WithLabelValues(pipelineLabel(len(cmds))).Observe(time.Since(start).Seconds())
		return err
	}
}

func pipelineLabel(n int) string {
	switch {
	case n <= 0:
		return "PIPELINE(0)"
	case n == 1:
		return "PIPELINE(1)"
	case n <= 10:
		return "PIPELINE(<=10)"
	case n <= 100:
		return "PIPELINE(<=100)"
	default:
		return "PIPELINE(>100)"
	}
}
