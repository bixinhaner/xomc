package devsweep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// Prober 是 GPV 探测的最小依赖。
//
// 单条/小批量 path 入参，返回每条 path 的 ProbeOutcome 与可选 fault 元数据。
// 实现侧负责：构造 GPV task → 入队 → 等待 terminal → 解析结果。
// 调用侧（Service.Run）负责：rate-limit、批次划分、聚合统计。
//
// 设计取舍：Prober 不内部限流 — 串行由 Service 通过 rate.Limiter 控制，
// 测试侧用 in-memory mock 实现，零网络依赖。
type Prober interface {
	Probe(ctx context.Context, deviceSN string, paths []string, batchIdx int) []ProbeRecord
}

// TaskSubmitter 是 taskProber 的入队依赖。生产环境由 *task.TaskService 满足。
//
// 故意只暴露 CreateTask + GetTask 两个方法（不是整个 *task.TaskService），
// 让单测只 mock 这两个调用即可。
type TaskSubmitter interface {
	CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	GetTask(ctx context.Context, taskID string) (*task.Task, error)
}

// CWMP 9005 = Invalid parameter name。仅此 code 能确证 path 在 CPE 数据模型
// 中不存在，可安全标 is_supported=false。其它 fault code 含义不同，
// 走 OutcomeUnknown 不污染真值源。
const cwmpFaultInvalidParameterName = 9005

// taskProber 用 task service 提交 GPV 任务并轮询 terminal 状态。
type taskProber struct {
	tasks        TaskSubmitter
	pollInterval time.Duration
	rpcTimeout   time.Duration
	logger       *zap.Logger
}

// NewTaskProber 构造默认的 taskProber。
// pollInterval=0 → 默认 500ms；rpcTimeout=0 → 默认 30s。
// logger nil-safe。
func NewTaskProber(tasks TaskSubmitter, pollInterval, rpcTimeout time.Duration, logger *zap.Logger) Prober {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	if rpcTimeout <= 0 {
		rpcTimeout = 30 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &taskProber{
		tasks:        tasks,
		pollInterval: pollInterval,
		rpcTimeout:   rpcTimeout,
		logger:       logger.Named("devsweep.prober"),
	}
}

// Probe 实现 Prober 接口。
//
// 入参 paths 已由 Service 做 {i} → .0. 替换后传入；这里直接发给 CPE。
// outcome 解析规则：
//   - task.Status=completed → 所有 path 均 supported
//   - task.Status=failed && ErrorCode=9005 → 所有 path 均 unsupported
//     （batch>1 时只能从 ErrorMessage 抽出一条 badPath；其余记 unknown
//     以避免误标 — 因此推荐 BatchSize=1）
//   - 其它 failed / expired / cancelled / timeout / 解析错 → 全 unknown
func (p *taskProber) Probe(ctx context.Context, deviceSN string, paths []string, batchIdx int) []ProbeRecord {
	out := make([]ProbeRecord, 0, len(paths))
	if len(paths) == 0 {
		return out
	}

	start := time.Now()

	params, err := json.Marshal(map[string]any{"names": paths})
	if err != nil {
		p.logger.Warn("marshal gpv params", zap.Error(err))
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				DurationMS: time.Since(start).Milliseconds(),
				FaultMessage: "marshal gpv params: " + err.Error(),
			})
		}
		return out
	}

	created, err := p.tasks.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      "GetParameterValues",
		Params:      params,
		Priority:    20,
		ExpiresIn:   int(p.rpcTimeout.Seconds()),
		Source:      task.TaskSourceOps,
		Description: fmt.Sprintf("devsweep batch=%d size=%d", batchIdx, len(paths)),
	})
	if err != nil {
		p.logger.Warn("enqueue gpv task",
			zap.String("device_sn", deviceSN),
			zap.Int("batch", batchIdx),
			zap.Error(err))
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				DurationMS:   time.Since(start).Milliseconds(),
				FaultMessage: "enqueue task: " + err.Error(),
			})
		}
		return out
	}

	final, pollErr := p.waitForTerminal(ctx, created.ID)
	dur := time.Since(start).Milliseconds()

	if pollErr != nil {
		// timeout / ctx cancel — 不标真值源
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				DurationMS: dur, TaskID: created.ID,
				FaultMessage: pollErr.Error(),
			})
		}
		return out
	}

	switch final.Status {
	case task.TaskStatusCompleted:
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeSupported, Batch: batchIdx,
				DurationMS: dur, TaskID: created.ID,
			})
		}
	case task.TaskStatusFailed:
		outcomes := classifyGPVFailure(final, paths)
		for i, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome:      outcomes[i],
				FaultCode:    final.ErrorCode,
				FaultMessage: final.ErrorMessage,
				Batch:        batchIdx,
				DurationMS:   dur,
				TaskID:       created.ID,
			})
		}
	default:
		// expired / cancelled — 不标真值源
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				FaultCode:    final.ErrorCode,
				FaultMessage: fmt.Sprintf("task status=%s", final.Status),
				DurationMS:   dur,
				TaskID:       created.ID,
			})
		}
	}
	return out
}

// waitForTerminal 轮询 GetTask 直到 terminal 状态或超时/取消。
func (p *taskProber) waitForTerminal(ctx context.Context, taskID string) (*task.Task, error) {
	deadline := time.Now().Add(p.rpcTimeout + 5*time.Second) // 多给 5s buffer
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		t, err := p.tasks.GetTask(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("get task %s: %w", taskID, err)
		}
		if t == nil {
			return nil, fmt.Errorf("task %s vanished", taskID)
		}
		switch t.Status {
		case task.TaskStatusCompleted, task.TaskStatusFailed,
			task.TaskStatusExpired, task.TaskStatusCancelled:
			return t, nil
		}
		if time.Now().After(deadline) {
			return nil, errors.New("wait for task terminal: deadline exceeded")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// classifyGPVFailure 把 batch 的 paths 分配 Outcome。
//
// 9005 + batch_size=1 → 该 path 必定 unsupported。
// 9005 + batch_size>1 → 仅 ErrorMessage 中含的 badPath 标 unsupported，
// 其余标 unknown（无法可靠归因；BatchSize 推荐 =1 的根因）。
func classifyGPVFailure(t *task.Task, paths []string) []ProbeOutcome {
	out := make([]ProbeOutcome, len(paths))
	if t.ErrorCode == cwmpFaultInvalidParameterName {
		if len(paths) == 1 {
			out[0] = OutcomeUnsupported
			return out
		}
		// batch>1 时谨慎：仅 ErrorMessage 命中的 path 标 unsupported
		bad := extractBadPathFromMessage(t.ErrorMessage)
		for i, p := range paths {
			switch {
			case bad != "" && (p == bad || strings.HasPrefix(p, bad) || strings.HasPrefix(bad, p)):
				out[i] = OutcomeUnsupported
			default:
				out[i] = OutcomeUnknown
			}
		}
		return out
	}
	for i := range out {
		out[i] = OutcomeUnknown
	}
	return out
}

// extractBadPathFromMessage 从 ACS 写入的 ErrorMessage 中抽出 badPath。
//
// ACS handler.detectSOAPFault + extractBadPathFromFaultString 已做主要工作；
// 此处只是把同样的逻辑用纯字符串再做一遍，避免 devsweep 依赖 acs 包。
// 支持的样式（与 ACS handler 对齐）：
//
//	"[Client] Invalid parameter name: Device.X.Y"
//	"Invalid Parameter Names [1], including: Device.X.Y"
//	"Parameter 'Device.X.Y' is not supported"
func extractBadPathFromMessage(msg string) string {
	if msg == "" {
		return ""
	}
	keywords := []string{"including:", "name:", "parameter:", "parameter '"}
	for _, kw := range keywords {
		idx := strings.Index(strings.ToLower(msg), kw)
		if idx < 0 {
			continue
		}
		tail := msg[idx+len(kw):]
		tail = strings.TrimLeft(tail, " '\"")
		end := strings.IndexAny(tail, "' \",;]")
		if end < 0 {
			return strings.TrimSpace(tail)
		}
		return strings.TrimSpace(tail[:end])
	}
	// 兜底找最长 dot-separated 标识符
	var longest string
	for _, tok := range strings.FieldsFunc(msg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ',' || r == ';' || r == ']' || r == '[' || r == '\'' || r == '"'
	}) {
		if strings.Count(tok, ".") >= 2 && len(tok) > len(longest) {
			longest = tok
		}
	}
	return longest
}

// NormalizeForProbe 把 catalog 中含 {i} 的 standardPath 替换为可发给 CPE 的
// 具体实例路径（默认 ".0."）。多 {i} 全部替换。
//
// 设计取舍：不做"实例发现"（先 GetParameterNames 列出活跃实例再挑一个）—
// 那样工具复杂度倍增、慢一个数量级；用 .0. 当作"默认实例" disclaimer，
// 对没有实例的 path 仍会返回 9005，调用方接受这种"伪 unsupported"。
//
// 实际生产中建议：先 CPE 上确认目标对象有 instance 0；否则跑前用 --prefix
// 限定到无 {i} 的子树。
func NormalizeForProbe(path string) string {
	if !strings.Contains(path, "{i}") {
		return path
	}
	return strings.ReplaceAll(path, "{i}", "0")
}

// normalizeBatch 批量 NormalizeForProbe；保留 stable 顺序。
func normalizeBatch(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = NormalizeForProbe(p)
	}
	return out
}
