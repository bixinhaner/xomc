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

// probeMaxIterations 是 Probe 内部 retry 的硬上限,防止 prober 卡死。
//
// T-0180: batch>1 失败时 CPE 只暴露 1 个 badPath, 剩余 path 标 Unknown 然后重发;
// 收敛速度 O(N_bad_paths)。BLQ 765 path 含 11 unsupported 最坏 11 iters,留充足
// safety margin。超过此值仍未收敛 → 视为环境异常,剩余 path 标 Unknown 不污染。
const probeMaxIterations = 64

// Probe 实现 Prober 接口。
//
// 入参 paths 已由 Service 做 {i} → .0. 替换后传入;这里直接发给 CPE。
//
// T-0180 后的批次重试语义:
//   - 单次 GPV 失败时,ACS handler 把 badPath 写入 device_tasks.result 的
//     param_faults[] (与 SPV schema 对齐)。classifyGPVFailure 据此可信归因。
//   - batch>1 时:命中的 badPath 标 Unsupported,剩余 path 标 Unknown,Probe 自动
//     用剩余 path 作为新 batch 再发一次(O(N_bad) 收敛)。无进展或达 max iter 时
//     剩余 path 终态 Unknown,不污染真值源。
//   - batch=1 时:9005 直接 Unsupported,无 retry。
func (p *taskProber) Probe(ctx context.Context, deviceSN string, paths []string, batchIdx int) []ProbeRecord {
	if len(paths) == 0 {
		return nil
	}

	classified := make(map[string]ProbeRecord, len(paths))
	remaining := append([]string(nil), paths...)

	for iter := 0; iter < probeMaxIterations && len(remaining) > 0; iter++ {
		results := p.probeOnce(ctx, deviceSN, remaining, batchIdx, iter)
		progressed := false
		var nextRemaining []string
		for _, r := range results {
			if r.Outcome == OutcomeUnknown {
				nextRemaining = append(nextRemaining, r.StandardPath)
				continue
			}
			// Supported / Unsupported = 确定归因 → 落 classified
			classified[r.StandardPath] = r
			progressed = true
		}
		if !progressed {
			// 整批没有任何 path 归因(网络错 / 非 9005 fault / param_faults 解析失败) —
			// 不再重试,剩余 path 用本轮的 Unknown record 锁定。
			for _, r := range results {
				if _, ok := classified[r.StandardPath]; !ok {
					classified[r.StandardPath] = r
				}
			}
			remaining = nil
			break
		}
		remaining = nextRemaining
	}

	// max iter 兜底:仍有 path 未归因 → 标 Unknown
	for _, path := range remaining {
		if _, ok := classified[path]; !ok {
			classified[path] = ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				FaultMessage: "probe max iterations exceeded",
			}
		}
	}

	// 按入参顺序返回
	out := make([]ProbeRecord, 0, len(paths))
	for _, path := range paths {
		if r, ok := classified[path]; ok {
			out = append(out, r)
		}
	}
	return out
}

// probeOnce 单批 GPV 探测(不重试),返回 paths 中每条的 ProbeRecord。
//
// T-0180 前的旧 Probe 主体,被 retry 循环包成 helper。
func (p *taskProber) probeOnce(ctx context.Context, deviceSN string, paths []string, batchIdx, iter int) []ProbeRecord {
	out := make([]ProbeRecord, 0, len(paths))
	start := time.Now()

	params, err := json.Marshal(map[string]any{"names": paths})
	if err != nil {
		p.logger.Warn("marshal gpv params", zap.Error(err))
		for _, path := range paths {
			out = append(out, ProbeRecord{
				StandardPath: path, ProbePath: path,
				Outcome: OutcomeUnknown, Batch: batchIdx,
				DurationMS:   time.Since(start).Milliseconds(),
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
		Description: fmt.Sprintf("devsweep batch=%d iter=%d size=%d", batchIdx, iter, len(paths)),
	})
	if err != nil {
		p.logger.Warn("enqueue gpv task",
			zap.String("device_sn", deviceSN),
			zap.Int("batch", batchIdx),
			zap.Int("iter", iter),
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
// 真值源优先级 (T-0180 起):
//  1. t.Result 的 {"param_faults":[{parameter_name,fault_code}]} —— ACS handler
//     对 GPV 9005 同 SPV schema 写入结构化 fault (与 result_aggregator 同款)。
//  2. 兜底:t.ErrorMessage 文本 extractBadPathFromMessage (向后兼容老 ACS)。
//
// 9005 + batch_size=1 → 该 path 必定 unsupported。
// 9005 + batch_size>1 → 仅命中 fault hint 的 path 标 unsupported,其余标 unknown
// (上层 Probe retry 循环把这批 unknown 当下一轮 batch 重发,O(N_bad) 收敛)。
func classifyGPVFailure(t *task.Task, paths []string) []ProbeOutcome {
	out := make([]ProbeOutcome, len(paths))
	if t.ErrorCode != cwmpFaultInvalidParameterName {
		for i := range out {
			out[i] = OutcomeUnknown
		}
		return out
	}

	// 收集所有 badPath: 先 result.param_faults, 再兜底 ErrorMessage
	badSet := extractBadPathsFromResult(t.Result)
	if len(badSet) == 0 {
		if bad := extractBadPathFromMessage(t.ErrorMessage); bad != "" {
			badSet = map[string]struct{}{bad: {}}
		}
	}

	// batch=1: 9005 直接归因(badSet 即使空也归因,因 CPE 只对这 1 个 path 失败)
	if len(paths) == 1 {
		out[0] = OutcomeUnsupported
		return out
	}

	for i, p := range paths {
		if matchesAnyBadPath(p, badSet) {
			out[i] = OutcomeUnsupported
		} else {
			out[i] = OutcomeUnknown
		}
	}
	return out
}

// extractBadPathsFromResult 解析 ACS 写入的 {"param_faults":[...]} 结构 (T-0180
// 起 SPV/GPV 同 schema)。仅取 fault_code==9005 的条目;其它 code 含义不同
// (9007 类型错 / 9008 只读) 不能映射到 unsupported,跳过。
func extractBadPathsFromResult(raw []byte) map[string]struct{} {
	if len(raw) == 0 {
		return nil
	}
	var payload struct {
		ParamFaults []struct {
			ParameterName string `json:"parameter_name"`
			FaultCode     int    `json:"fault_code"`
		} `json:"param_faults"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if len(payload.ParamFaults) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(payload.ParamFaults))
	for _, f := range payload.ParamFaults {
		if f.FaultCode == cwmpFaultInvalidParameterName && f.ParameterName != "" {
			out[f.ParameterName] = struct{}{}
		}
	}
	return out
}

// matchesAnyBadPath 判 path 是否被任一 badPath 命中。
// 与既有 extract 路径一致:精确相等 / path 是 bad 的前缀 / bad 是 path 的前缀
// (CPE 暴露的 badPath 可能是父对象前缀如 "Device.X.")。
func matchesAnyBadPath(p string, badSet map[string]struct{}) bool {
	if len(badSet) == 0 {
		return false
	}
	if _, ok := badSet[p]; ok {
		return true
	}
	for bad := range badSet {
		if strings.HasPrefix(p, bad) || strings.HasPrefix(bad, p) {
			return true
		}
	}
	return false
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
