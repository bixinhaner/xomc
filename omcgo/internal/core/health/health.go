// Package health 提供进程级健康检查 HTTP 处理器，统一 acs/app/worker 三进程的
// /healthz（liveness）与 /readyz（readiness）行为。
//
// 设计要点：
//   - Liveness 检查只回答"进程还活着吗"，永远返回 200，不依赖外部服务。
//   - Readiness 检查依赖外部基础设施（DB/Redis/NATS/MinIO 等）。任一依赖不可用，
//     整体降级为 503，但仍返回每个依赖的明细，便于运维定位。
//   - Checker 是接口而非闭包参数，强制调用方对依赖建模，避免散落的匿名函数。
//   - 与 components.HealthChecker 解耦：调用方可以直接传 Checker，也可以通过
//     FromComponentsChecker 适配既有注册结果。
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Checker 表示单个依赖的就绪检查。
// 实现应当是无副作用的快速探测（如 Ping），耗时控制在百毫秒量级。
type Checker interface {
	// Name 返回依赖名（如 "postgres"、"redis"），用于响应明细。
	Name() string
	// Check 执行就绪检查；返回 nil 表示就绪，非 nil 表示当前不可用。
	Check(ctx context.Context) error
}

// Result 表示单个依赖的检查结果，序列化到 JSON 响应。
type Result struct {
	Name    string `json:"name"`
	Status  string `json:"status"`          // "healthy" | "unhealthy"
	Latency string `json:"latency"`         // 形如 "1.2ms"
	Error   string `json:"error,omitempty"` // 仅 unhealthy 时有值
}

// Response 是 /readyz 的整体响应结构。
// /healthz 复用同一结构（components 字段为 nil）以保持响应形态一致。
type Response struct {
	Status     string   `json:"status"`               // "ok" | "unhealthy"
	Components []Result `json:"components,omitempty"` // /healthz 不返回此字段
}

// LivenessHandler 返回 /healthz 处理器：永远返回 200。
// liveness 探针的语义是"进程未死"，不应该因为依赖不可用而返回 503，
// 否则会导致 Kubernetes 错误地重启正在缓慢恢复的实例。
func LivenessHandler() http.HandlerFunc {
	body, _ := json.Marshal(Response{Status: "ok"})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// ReadinessHandler 返回 /readyz 处理器：并发执行所有 Checker，任一失败返 503。
// timeout 限制单次探测的总时长（外层封口），传 0 时使用默认 5s。
// 当 checkers 为空时退化为 liveness 行为（恒 200）。
func ReadinessHandler(timeout time.Duration, checkers ...Checker) http.HandlerFunc {
	return DynamicReadinessHandler(timeout, func() []Checker {
		return checkers
	})
}

// DynamicReadinessHandler 返回 /readyz 处理器，并在每次请求时重新获取 Checker。
// 适用于 metrics/health 端口需要早于业务模块初始化启动的进程：后续模块继续
// Register 健康检查时，/readyz 不会停留在启动时的旧快照。
func DynamicReadinessHandler(timeout time.Duration, checkerProvider func() []Checker) http.HandlerFunc {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var checkers []Checker
		if checkerProvider != nil {
			checkers = checkerProvider()
		}
		results := runCheckers(ctx, checkers)
		status := http.StatusOK
		overall := "ok"
		for _, res := range results {
			if res.Status != "healthy" {
				status = http.StatusServiceUnavailable
				overall = "unhealthy"
				break
			}
		}

		body, _ := json.Marshal(Response{Status: overall, Components: results})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// runCheckers 并发执行所有 Checker 并按入参顺序返回结果。
// 任一 panic 不会传播，单个 Checker 故障不会影响其他 Checker。
func runCheckers(ctx context.Context, checkers []Checker) []Result {
	if len(checkers) == 0 {
		return nil
	}
	results := make([]Result, len(checkers))
	var wg sync.WaitGroup
	for i, c := range checkers {
		wg.Add(1)
		go func(idx int, chk Checker) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					results[idx] = Result{
						Name:   chk.Name(),
						Status: "unhealthy",
						Error:  "panic in health checker",
					}
				}
			}()
			start := time.Now()
			err := chk.Check(ctx)
			latency := time.Since(start)
			res := Result{
				Name:    chk.Name(),
				Status:  "healthy",
				Latency: latency.String(),
			}
			if err != nil {
				res.Status = "unhealthy"
				res.Error = err.Error()
			}
			results[idx] = res
		}(i, c)
	}
	wg.Wait()
	return results
}

// fnChecker 用 closure 实现 Checker，仅供 NewChecker 内部使用。
type fnChecker struct {
	name string
	fn   func(ctx context.Context) error
}

func (f *fnChecker) Name() string                    { return f.name }
func (f *fnChecker) Check(ctx context.Context) error { return f.fn(ctx) }

// NewChecker 用闭包构造 Checker。
// 适用于 Ping 这类一行就能实现的检查，避免每个依赖都声明一个新类型。
// 上层模块如有更复杂的就绪逻辑，仍应直接实现 Checker 接口。
func NewChecker(name string, fn func(ctx context.Context) error) Checker {
	return &fnChecker{name: name, fn: fn}
}
