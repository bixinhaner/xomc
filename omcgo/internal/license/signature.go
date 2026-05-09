package license

// SignatureStatus 是 license 数字签名校验结果。
//
// T-0100-P3 / PRD §5.3.1（Q4=B 决议）：MVP 暂未接入 OEM 公钥强校验，所有
// import 默认 'unverified' + 后端 zap.Warn；GA 前 P4-C 阶段加载
// configs/oem_public_keys/*.pem 后会返回 'verified' 或 'invalid'。
//
// 前端用此字段在 import 成功 Modal 上显示警告 Tag，提示用户当前阶段未做
// 强校验。
type SignatureStatus string

const (
	// SignatureUnverified — 当前 MVP 默认值（未配置 OEM 公钥）。
	SignatureUnverified SignatureStatus = "unverified"

	// SignatureVerified — P4-C 阶段：签名验证通过。
	SignatureVerified SignatureStatus = "verified"

	// SignatureInvalid — P4-C 阶段：签名验证失败（MVP 暂不返）。
	SignatureInvalid SignatureStatus = "invalid"
)

// VerifySignature 是 license 数字签名校验入口（P3 stub）。
//
// MVP（P3）：不做实际校验，固定返回 SignatureUnverified + 一段说明。
// P4-C：替换实现为加载 OEM 公钥 + RSA/ECDSA 校验；strict=true 时签名无效
// 直接返 (Invalid, error)，调用方 Import 拒绝。
//
// 入参 raw 是 license 文件原始字节（本 stub 暂不读，预留给 P4-C 解析签名块）。
func VerifySignature(_ []byte) (SignatureStatus, string) {
	// P3 MVP 行为：不做校验，全部放过，附 warning note 供前端展示。
	return SignatureUnverified, "OEM public key not configured; signature verification deferred to P4-C"
}
