package software

// Package software — 固件下发前完整性 / 签名校验（issue #8）。
//
// 设计动机：固件下发前必须确认"要下发的固件是上传时那一份、未被篡改"。历史上仅
// 靠 md5_val 做完整性记录，MD5 已被证明可碰撞，不能作为商用网管系统的完整性根。
//
// 校验在 executor.ExecuteOne 派发 Download 命令之前执行：
//   1. 完整性校验（HashOnlyVerifier，默认强制开启）——固件必须带可用的完整性摘要：
//      优先 SHA-256（新上传），存量旧固件回退 MD5。两者皆空 → 拒绝下发。
//   2. 签名校验（SignatureVerifier，脚手架）——当固件带厂商签名时强制验签；不带签名
//      时跳过（向后兼容现网未签名固件，不阻断 happy-path）。是否对"无签名固件"硬性
//      拒绝由 RequireSignature 配置开关控制，默认 false。
//
// 为什么在元数据层而非字节层校验：executor 在 100K~1M 设备规模下对每个子任务都跑一次
// ExecuteOne，若每次都从 MinIO 全量拉固件重算 hash 会打爆对象存储与带宽。SHA-256 在
// 上传时（可信入口、单次串流）算好落库，下发侧据此判定固件记录是否"具备可信完整性根"。
// 字节级的端到端完整性由 CPE 侧在 Download 后用下发的摘要自校验承接（TR-069 既有机制）。

import (
	"crypto/subtle"
	stderrors "errors"
	"fmt"
	"strings"
)

// FirmwareVerifier 在固件下发前对固件记录做完整性 / 签名校验。
// 校验失败必须返回非 nil error；调用方（executor）据此中止下发、打点、记日志，
// 绝不静默放行。接口优先：默认 HashOnlyVerifier，签名链路用 SignatureVerifier 装饰。
type FirmwareVerifier interface {
	// Verify 校验给定固件是否可安全下发。返回 nil 表示通过；返回 error 表示必须中止，
	// 错误经 VerificationError 携带 FailureCode 供 executor 映射为 sub_task 失败码。
	Verify(fw *FirmwareVersion) error
}

// VerificationError 包装校验失败，携带 executor 落库用的 FailureCode。
type VerificationError struct {
	Code   FailureCode
	Reason string
}

func (e *VerificationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Reason)
}

// VerificationFailureCode 从 error 中抽取 FailureCode；非 VerificationError 退化为
// 通用完整性失败码，保证 executor 总能落一个明确的失败原因。
func VerificationFailureCode(err error) FailureCode {
	var ve *VerificationError
	if stderrors.As(err, &ve) {
		return ve.Code
	}
	return FailureIntegrityCheck
}

// HashOnlyVerifier 是默认且始终强制的完整性校验器：固件必须带可用的完整性摘要。
// 优先采信 SHA-256；存量旧固件 SHA-256 为空时回退 MD5（向后兼容，下发不被阻断）。
// 两者皆空 → 视为"无可信完整性根"，拒绝下发。
//
// 注：本校验器不重算字节 hash（见包注释），只确认固件记录具备可信完整性摘要。
type HashOnlyVerifier struct{}

// NewHashOnlyVerifier 构造默认完整性校验器。
func NewHashOnlyVerifier() *HashOnlyVerifier { return &HashOnlyVerifier{} }

// Verify 实现 FirmwareVerifier：固件必须至少带 SHA-256 或 MD5 之一。
func (v *HashOnlyVerifier) Verify(fw *FirmwareVersion) error {
	if fw == nil {
		return &VerificationError{Code: FailureIntegrityCheck, Reason: "firmware record is nil"}
	}
	if strings.TrimSpace(fw.SHA256Val) != "" {
		return nil // 有 SHA-256：最强完整性根，直接通过。
	}
	if strings.TrimSpace(fw.MD5Val) != "" {
		// 仅 MD5 的存量旧固件：放行但语义上是降级路径。调用方会记 warn 日志 + 打点。
		return nil
	}
	return &VerificationError{
		Code:   FailureIntegrityCheck,
		Reason: "firmware has neither sha256 nor md5 integrity digest; refusing to dispatch",
	}
}

// IsLegacyMD5Only 报告固件是否走"仅 MD5"降级路径（无 SHA-256）。executor 据此决定
// 是否打降级 warn / metric，便于运营观测存量固件迁移进度。
func IsLegacyMD5Only(fw *FirmwareVersion) bool {
	return fw != nil &&
		strings.TrimSpace(fw.SHA256Val) == "" &&
		strings.TrimSpace(fw.MD5Val) != ""
}

// SignatureVerificationFunc 是"用厂商公钥验签固件摘要"的可插拔实现。
// 入参：固件的完整性摘要（hex SHA-256）、签名（base64）、签名算法、公钥标识。
// 返回 nil 表示验签通过。完整的厂商 PKI / 公钥装载 / 密钥轮换 / 吊销由独立子系统提供；
// 在该子系统落地前，注入方可传 nil（SignatureVerifier 退化为"无签名固件直接放行、
// 带签名固件按 RequireSignature 决定拒绝与否"）。
type SignatureVerificationFunc func(digestHex, signatureB64, alg, publicKeyID string) error

// SignatureVerifier 在 HashOnlyVerifier 之上叠加厂商签名校验（脚手架）。
//
// 行为矩阵（gated by RequireSignature，默认 false → 不破坏 happy-path）：
//
//	┌────────────────┬──────────────────────┬──────────────────────────────┐
//	│ 固件是否带签名 │ RequireSignature=false│ RequireSignature=true        │
//	├────────────────┼──────────────────────┼──────────────────────────────┤
//	│ 带签名         │ 强制验签（失败拒绝） │ 强制验签（失败拒绝）         │
//	│ 不带签名       │ 跳过验签（放行）     │ 拒绝下发（SIGNATURE_CHECK）  │
//	└────────────────┴──────────────────────┴──────────────────────────────┘
//
// 这样可以"现在就对已签名固件强制验签"，同时不阻断现网大量存量未签名固件——
// 验签的全面强制（RequireSignature=true）留待厂商签名供应链全量铺开后切换。
type SignatureVerifier struct {
	inner FirmwareVerifier
	// RequireSignature 为 true 时，无签名固件直接拒绝下发；默认 false（仅对带签名固件验签）。
	RequireSignature bool
	// verifyFn 是注入的实际验签实现；nil 表示厂商 PKI 子系统尚未装配，
	// 带签名固件在此情形下也会被拒绝（不能在缺验签能力时假装验过 —— 商用级不撒谎）。
	verifyFn SignatureVerificationFunc
}

// NewSignatureVerifier 用一个底层完整性校验器 + 可选验签实现构造签名校验器。
// inner 为 nil 时默认包一层 HashOnlyVerifier，保证完整性校验始终在签名校验之前执行。
func NewSignatureVerifier(inner FirmwareVerifier, requireSignature bool, verifyFn SignatureVerificationFunc) *SignatureVerifier {
	if inner == nil {
		inner = NewHashOnlyVerifier()
	}
	return &SignatureVerifier{inner: inner, RequireSignature: requireSignature, verifyFn: verifyFn}
}

// Verify 先跑完整性校验，再按行为矩阵处理签名。
func (v *SignatureVerifier) Verify(fw *FirmwareVersion) error {
	if err := v.inner.Verify(fw); err != nil {
		return err
	}
	if fw == nil {
		return &VerificationError{Code: FailureIntegrityCheck, Reason: "firmware record is nil"}
	}

	hasSignature := strings.TrimSpace(fw.Signature) != ""
	if !hasSignature {
		if v.RequireSignature {
			return &VerificationError{
				Code:   FailureSignatureCheck,
				Reason: "signature enforcement enabled but firmware is unsigned; refusing to dispatch",
			}
		}
		return nil // 无签名固件 + 未强制：跳过验签放行（向后兼容）。
	}

	// 带签名固件：必须验签。缺验签能力（verifyFn 为 nil）→ 拒绝，不假装验过。
	if v.verifyFn == nil {
		return &VerificationError{
			Code:   FailureSignatureCheck,
			Reason: "firmware carries a signature but no signature-verification key is configured; refusing to dispatch",
		}
	}
	digest := strings.TrimSpace(fw.SHA256Val)
	if digest == "" {
		// 签名锚定在 SHA-256 摘要上；没有 SHA-256 无法验签。
		return &VerificationError{
			Code:   FailureSignatureCheck,
			Reason: "signed firmware lacks a sha256 digest to verify the signature against",
		}
	}
	if err := v.verifyFn(digest, strings.TrimSpace(fw.Signature), strings.TrimSpace(fw.SignatureAlg), strings.TrimSpace(fw.PublicKeyID)); err != nil {
		return &VerificationError{
			Code:   FailureSignatureCheck,
			Reason: fmt.Sprintf("vendor signature verification failed: %v", err),
		}
	}
	return nil
}

// ConstantTimeDigestEqual 是给未来字节级重算 hash 路径预留的等值比较：用常量时间
// 比较两个 hex 摘要，避免时序侧信道。当前完整性校验走元数据路径（见包注释）暂未用到，
// 但验签子系统落地时多半需要它做摘要比对，故先放在这里作为统一入口。
func ConstantTimeDigestEqual(a, b string) bool {
	ab := []byte(strings.ToLower(strings.TrimSpace(a)))
	bb := []byte(strings.ToLower(strings.TrimSpace(b)))
	if len(ab) != len(bb) || len(ab) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(ab, bb) == 1
}
