// Package backup — restore active integrity verification (issue #70).
//
// 背景：配置恢复"成功"此前只看 TransferComplete 的 FaultCode==0（CPE 仅回报
// "文件下载完成"）。这无法回答 DR 演练真正关心的问题——"设备实际生效的配置，
// 是不是我们下发的那一份？"。本文件实现 OMC 侧的主动校验编排：
//
//	下载完成(TransferComplete FaultCode==0)
//	  → MarkDownloaded（进入 downloaded 中间态，记录 downloaded_at）
//	  → RestoreVerifier.Verify（回读设备实际配置指纹 / 关键参数）
//	  → 与 restore_tasks.expected_hash 比对
//	      match    → MarkVerified(completed)
//	      mismatch → MarkVerified(failed) + error_message
//	      无 verifier / 设备不支持回读 → 按 OnNoVerifier 策略处理
//
// ── 为什么 Verify 是一个 hook（interface）而非内联实现 ──
// "回读设备实际生效配置"强依赖设备能力：有的 CPE 支持私有 GetParameterValues
// 回报配置指纹，有的只能逐参数 GPV 比对，有的两者都不支持。这部分是 device-
// dependent 的，故抽象成 RestoreVerifier 接口；OMC 侧的编排（记期望指纹 → 触发
// → 比对 → 落状态）在本仓内完整闭环，具体 Verify 实现由后续接 ACS GPV 的任务
// 或厂商适配器填充（modules.go 里默认不注入 → 走 OnNoVerifier 默认策略）。
package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RestoreVerifyResult 是一次主动校验的结果。
type RestoreVerifyResult struct {
	// Match 为 true 表示设备回读的配置指纹/关键参数与期望一致。
	Match bool
	// ObservedHash 是从设备回读并归一化后的配置指纹（写入 verified_hash）。
	// GPV 关键参数比对方式下可填一个稳定派生指纹；为空表示无法得到指纹（仅做了
	// 参数级比对），此时 Match 仍有效，只是 verified_hash 留空。
	ObservedHash string
	// Method 记录实际走的校验方式（写入 verification_method）。
	Method RestoreVerificationMethod
	// Detail 是失败时附加到 error_message 的可读说明（成功时忽略）。
	Detail string
}

// RestoreVerifier 是"回读设备实际生效配置并与期望比对"的 device-dependent hook。
//
// 入参：
//   - deviceSN：目标设备序列号
//   - expectedHash：下发时记录的期望明文指纹（restore_tasks.expected_hash）
//   - hashAlgo：expectedHash 的算法（md5 / sha256）
//
// 返回 (result, err)：err != nil 表示校验过程本身出错（设备不可达、超时等），
// 由调用方按 OnVerifierError 策略处理；err == nil 时以 result.Match 为准。
//
// 实现要求：必须是设备隔离的、可超时的；不得 panic。
type RestoreVerifier interface {
	Verify(ctx context.Context, deviceSN, expectedHash string, hashAlgo RestoreHashAlgo) (RestoreVerifyResult, error)
}

// NoVerifierPolicy 决定"没有可用 verifier（未装配，或设备不支持回读）"时的归宿。
type NoVerifierPolicy string

const (
	// NoVerifierStayDownloaded（默认，安全）：停在 downloaded 态，标记需人工/后续
	// 校验。绝不把"仅下载完成"谎报为 completed —— 这是 #70 的核心诉求。
	NoVerifierStayDownloaded NoVerifierPolicy = "stay_downloaded"
	// NoVerifierFallbackComplete（兼容旧行为，需显式开启）：退化为旧语义，
	// FaultCode==0 即记 completed。仅在运维明确接受"无主动校验"风险时启用。
	NoVerifierFallbackComplete NoVerifierPolicy = "fallback_complete"
)

// VerifierErrorPolicy 决定 verifier 执行过程出错（设备不可达/超时）时的归宿。
type VerifierErrorPolicy string

const (
	// VerifierErrorStayDownloaded（默认）：校验过程出错时停在 downloaded，等下次
	// 重试，不武断判失败（设备临时不可达 ≠ 配置错）。
	VerifierErrorStayDownloaded VerifierErrorPolicy = "stay_downloaded"
	// VerifierErrorFail：校验过程出错即判 failed（严格 DR 场景，宁可误报失败）。
	VerifierErrorFail VerifierErrorPolicy = "fail"
)

// VerificationConfig 控制主动校验的默认行为，便于运维按场景调。零值即安全默认：
// 无 verifier → 停 downloaded；verifier 出错 → 停 downloaded。
type VerificationConfig struct {
	OnNoVerifier    NoVerifierPolicy
	OnVerifierError VerifierErrorPolicy
}

// DefaultVerificationConfig 返回安全默认：绝不把未校验的恢复记为 completed。
func DefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		OnNoVerifier:    NoVerifierStayDownloaded,
		OnVerifierError: VerifierErrorStayDownloaded,
	}
}

// restoreVerifyOutcome 是编排的内部结论，描述应如何落 restore_tasks 状态。
type restoreVerifyOutcome struct {
	// terminal 为 true 表示要落终态（completed/failed）走 MarkVerified；
	// 为 false 表示停在 downloaded（已由 MarkDownloaded 落）。
	terminal     bool
	status       RestoreStatus
	result       int16
	verifiedHash string
	method       RestoreVerificationMethod
	errMsg       string
}

// RestoreVerificationOrchestrator 把"下载完成 → 校验 → 落状态"串起来。被
// TransferCompleteRouter 在 restore 成功分支调用，替代旧的"直接 MarkComplete"。
//
// 设计：repo + verifier(可空) + config + logger。verifier 为空走 OnNoVerifier。
type RestoreVerificationOrchestrator struct {
	repo     RestoreTaskRepository
	verifier RestoreVerifier // 可空（device-dependent，未接 GPV 时为 nil）
	cfg      VerificationConfig
	metrics  *RestoreMetrics // 复用，nil-safe
	logger   *zap.Logger
}

// NewRestoreVerificationOrchestrator 构造编排器。verifier 可为 nil。
func NewRestoreVerificationOrchestrator(
	repo RestoreTaskRepository,
	verifier RestoreVerifier,
	cfg VerificationConfig,
	metrics *RestoreMetrics,
	logger *zap.Logger,
) *RestoreVerificationOrchestrator {
	return &RestoreVerificationOrchestrator{
		repo:     repo,
		verifier: verifier,
		cfg:      cfg,
		metrics:  metrics,
		logger:   logger.Named("restore-verify"),
	}
}

// HandleDownloaded 在 CPE 回报 Download 成功（FaultCode==0）后被调用。task 是已
// 取到的 restore_tasks 行（含 expected_hash）。它先把行落到 downloaded，再对每个
// 目标设备执行主动校验，按 match/mismatch/无 verifier/出错 落终态或停 downloaded。
//
// 返回 error 仅在 DB 写失败时（让 NATS 重投兜底）；校验业务结论（含失败）都通过
// 状态列体现，不返回 error。
func (o *RestoreVerificationOrchestrator) HandleDownloaded(
	ctx context.Context, task *RestoreTask, completedAt time.Time,
) error {
	// 1) 先落 downloaded 中间态（幂等；仅从 pending/running 推进）。
	if err := o.repo.MarkDownloaded(ctx, task.ID, completedAt); err != nil {
		o.recordOutcome("downloaded_mark_error")
		return fmt.Errorf("mark restore_task %s downloaded: %w", task.ID, err)
	}
	o.recordOutcome("downloaded")

	// 2) 推导本次校验结论。
	outcome := o.evaluate(ctx, task)

	// 3) 停在 downloaded（无 verifier / 出错且策略为 stay）→ 不落终态，直接返回。
	if !outcome.terminal {
		o.recordOutcome("stay_downloaded")
		o.logger.Info("restore stays in downloaded (active verification pending)",
			zap.String("restore_id", task.ID.String()),
			zap.String("method", string(outcome.method)),
			zap.String("reason", outcome.errMsg))
		return nil
	}

	// 4) 落终态。
	if err := o.repo.MarkVerified(ctx, task.ID, outcome.status, outcome.result,
		outcome.verifiedHash, outcome.method, completedAt, outcome.errMsg); err != nil {
		o.recordOutcome("verified_mark_error")
		return fmt.Errorf("mark restore_task %s verified: %w", task.ID, err)
	}
	o.recordOutcome("verified_" + string(outcome.status))
	o.logger.Info("restore active verification done",
		zap.String("restore_id", task.ID.String()),
		zap.String("status", string(outcome.status)),
		zap.String("method", string(outcome.method)))
	return nil
}

// evaluate 计算校验结论（不写 DB），便于单测直接断言策略矩阵。
func (o *RestoreVerificationOrchestrator) evaluate(ctx context.Context, task *RestoreTask) restoreVerifyOutcome {
	// 无 verifier 装配 → 按 OnNoVerifier 策略。
	if o.verifier == nil {
		return o.noVerifierOutcome("no active verifier configured (device read-back not wired)")
	}
	// 无期望指纹（下发时未记 expected_hash，例如按路径恢复且 stater 未注入）→
	// 无从比对，等同无 verifier 能力。
	expected := ""
	if task.ExpectedHash != nil {
		expected = strings.TrimSpace(*task.ExpectedHash)
	}
	if expected == "" {
		return o.noVerifierOutcome("expected_hash not recorded; nothing to compare")
	}
	algo := RestoreHashMD5
	if task.HashAlgo != nil && *task.HashAlgo != "" {
		algo = *task.HashAlgo
	}

	// 逐目标设备校验：任一不匹配 → 整体 failed（恢复是"该批配置全部生效"语义）。
	var firstHash string
	var usedMethod RestoreVerificationMethod = RestoreVerifyNone
	for _, sn := range task.TargetDeviceSNs {
		res, err := o.verifier.Verify(ctx, sn, expected, algo)
		if err != nil {
			return o.verifierErrorOutcome(sn, err)
		}
		if res.Method != "" {
			usedMethod = res.Method
		}
		if firstHash == "" && res.ObservedHash != "" {
			firstHash = res.ObservedHash
		}
		if !res.Match {
			detail := res.Detail
			if detail == "" {
				detail = "device-readback config hash mismatch"
			}
			return restoreVerifyOutcome{
				terminal:     true,
				status:       RestoreFailed,
				result:       int16(TaskResultFailed),
				verifiedHash: res.ObservedHash,
				method:       usedMethod,
				errMsg: fmt.Sprintf("restore verification failed for %s: %s (expected %s:%s)",
					sn, detail, algo, expected),
			}
		}
	}
	// 全部匹配 → completed。
	return restoreVerifyOutcome{
		terminal:     true,
		status:       RestoreCompleted,
		result:       int16(TaskResultSuccess),
		verifiedHash: firstHash,
		method:       usedMethod,
	}
}

// noVerifierOutcome 按 OnNoVerifier 策略给出停 downloaded 或退化 completed。
func (o *RestoreVerificationOrchestrator) noVerifierOutcome(reason string) restoreVerifyOutcome {
	if o.cfg.OnNoVerifier == NoVerifierFallbackComplete {
		return restoreVerifyOutcome{
			terminal: true,
			status:   RestoreCompleted,
			result:   int16(TaskResultSuccess),
			method:   RestoreVerifyNone,
		}
	}
	// 默认：停 downloaded。
	return restoreVerifyOutcome{terminal: false, method: RestoreVerifyNone, errMsg: reason}
}

// verifierErrorOutcome 按 OnVerifierError 策略给出停 downloaded 或判 failed。
func (o *RestoreVerificationOrchestrator) verifierErrorOutcome(sn string, err error) restoreVerifyOutcome {
	if o.cfg.OnVerifierError == VerifierErrorFail {
		return restoreVerifyOutcome{
			terminal: true,
			status:   RestoreFailed,
			result:   int16(TaskResultFailed),
			method:   RestoreVerifyNone,
			errMsg:   fmt.Sprintf("restore verification error for %s: %v", sn, err),
		}
	}
	// 默认：停 downloaded，等重试。
	return restoreVerifyOutcome{
		terminal: false,
		method:   RestoreVerifyNone,
		errMsg:   fmt.Sprintf("verifier error for %s (staying downloaded): %v", sn, err),
	}
}

func (o *RestoreVerificationOrchestrator) recordOutcome(outcome string) {
	if o.metrics == nil {
		return
	}
	o.metrics.RecordRequest("verify_" + outcome)
}
