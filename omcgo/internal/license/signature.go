// Package license — signature verification (T-0100-P4-C).
//
// 替代 P3 stub：加载 configs/oem_public_keys/*.pem 中的 OEM RSA 公钥，
// 对 license 文件做 RSA-PSS 签名验证。
//
// 签名格式约定（PRD §16 Q4=B 决议 + 用户拍板"嵌入 JSON signature 字段"）：
//
//	license 文件本身是 JSON，多一个顶层字段：
//	  {
//	    "license_code": "...",
//	    "license_name": "...",
//	    ...其他 license 字段...,
//	    "signature": "<base64-rsa-pss-of-canonical-without-signature>",
//	    "signature_key_id": "<optional sha256 fingerprint of public key>"
//	  }
//
// 验证流程：
//  1. 解析 JSON，提取 signature + signature_key_id 字段
//  2. 构造 canonical payload = JSON marshal of license（按 key 字典序，去掉
//     signature/signature_key_id 字段）
//  3. RSA-PSS 验证：sha256(canonical) + signature + 公钥
//  4. signature_key_id 命中：用对应公钥；未命中：依次尝试所有已加载公钥
//
// 兼容性：
//   - strict=false 时（默认）：未签名 / 公钥未配置 / 验证失败 仍允许导入
//     但 signature_status 反映真实状态供前端展示
//   - strict=true（prod 推荐）：上述 3 种状态返 (Invalid, error)，handler 拒绝
//     导入返 400
package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// SignatureStatus 是 license 数字签名校验结果。
//
// 与 PRD §5.3.1 / Q4=B 决议保持一致：
//   - SignatureUnverified — 未签名 / 无公钥 / 未启用强校验
//   - SignatureVerified   — 签名验证通过
//   - SignatureInvalid    — 有签名但验证失败（公钥不匹配 / 篡改）
type SignatureStatus string

const (
	SignatureUnverified SignatureStatus = "unverified"
	SignatureVerified   SignatureStatus = "verified"
	SignatureInvalid    SignatureStatus = "invalid"
)

// signatureFieldName / keyIDFieldName — 签名嵌入 JSON 时的字段名。
const (
	signatureFieldName = "signature"
	keyIDFieldName     = "signature_key_id"
)

// rsaPSSSaltLength — RSA-PSS 盐长度，与签名端约定一致；本期硬编码为 32 字节
// （SHA-256 输出长度），与 crypto.SHA256 + PSSSaltLengthEqualsHash 等价。
const rsaPSSSaltLength = 32

// SignatureVerifier 是 license 数字签名校验器，线程安全。
//
// Strict=false（默认）：unverified/invalid 不阻断导入，只反映 signature_status。
// Strict=true（prod 推荐）：unverified/invalid 直接 return error，handler 拒绝。
type SignatureVerifier struct {
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey // keyID（公钥 SHA-256 fingerprint，hex）→ *rsa.PublicKey
	strict bool
}

// NewSignatureVerifier 构造空 verifier。strict=false 时所有非 verified 状态放过。
//
// 调用方需通过 LoadKeysFromDir 加载公钥；nil verifier 也可用，等价于"所有
// license 都返 unverified"，与 P3 stub 行为一致。
func NewSignatureVerifier(strict bool) *SignatureVerifier {
	return &SignatureVerifier{
		keys:   map[string]*rsa.PublicKey{},
		strict: strict,
	}
}

// SetStrict 切换 strict 模式（运维热更新场景，例如演示部署改 prod 收紧）。
func (v *SignatureVerifier) SetStrict(strict bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.strict = strict
}

// LoadKeysFromDir 加载目录下所有 *.pem 文件作为 OEM 公钥。
//
// 行为：
//   - 目录不存在 / 没有 .pem 文件 → 返 nil error，verifier 保持空（unverified）
//   - 单文件解析失败 → warn-level 错误（聚合所有错误后返）；其他文件继续加载
//   - 文件可包含一个或多个 PEM block，仅识别"PUBLIC KEY"和"RSA PUBLIC KEY"
//     类型；其他类型（如 PRIVATE KEY、CERTIFICATE）忽略
//
// keyID 格式：公钥 DER 编码的 SHA-256 fingerprint hex，与签名端约定一致。
func (v *SignatureVerifier) LoadKeysFromDir(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // 目录不存在不算错误，保持 unverified 行为
		}
		return fmt.Errorf("read OEM public key dir %s: %w", dir, err)
	}

	loaded := map[string]*rsa.PublicKey{}
	var errs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".pem") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), rerr))
			continue
		}
		keys, perr := parsePublicKeysPEM(raw)
		if perr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), perr))
			continue
		}
		for keyID, k := range keys {
			loaded[keyID] = k
		}
	}

	v.mu.Lock()
	v.keys = loaded
	v.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("load OEM public keys: %s", strings.Join(errs, "; "))
	}
	return nil
}

// AddKey 直接注入公钥（测试用）。返回 keyID。
func (v *SignatureVerifier) AddKey(pub *rsa.PublicKey) string {
	keyID := fingerprintRSAPublicKey(pub)
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys[keyID] = pub
	return keyID
}

// KeyCount 返回当前已加载的公钥数（监控 / 启动诊断用）。
func (v *SignatureVerifier) KeyCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.keys)
}

// VerifyLicenseJSON 验证 license JSON 文件的签名。
//
// 入参 raw：license 文件原始字节（JSON）。
// 返回：
//   - status：unverified / verified / invalid
//   - note：人类可读说明，写到 license_logs.details + import 响应供前端 Modal
//   - err：strict=true 时非 verified 返 commonerrors.ErrInvalidInput；否则 nil
//
// 调用方（handler.Import）：
//   - err != nil：直接返 400 拒绝导入
//   - err == nil：继续导入，把 status / note 落 import response + license_logs
func (v *SignatureVerifier) VerifyLicenseJSON(raw []byte) (SignatureStatus, string, error) {
	if v == nil {
		return SignatureUnverified,
			"signature verifier not configured (T-0100-P3 stub mode)",
			nil
	}

	v.mu.RLock()
	keyCount := len(v.keys)
	keys := make(map[string]*rsa.PublicKey, keyCount)
	for k, val := range v.keys {
		keys[k] = val
	}
	strict := v.strict
	v.mu.RUnlock()

	if keyCount == 0 {
		note := "no OEM public keys configured; signature not verified"
		if strict {
			return SignatureUnverified, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureUnverified, note, nil
	}

	if len(raw) == 0 {
		note := "empty license payload; cannot verify signature"
		if strict {
			return SignatureUnverified, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureUnverified, note, nil
	}

	// 解析 license JSON 为 map，提取 signature/key_id 字段。
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		note := fmt.Sprintf("license JSON parse failed: %v", err)
		if strict {
			return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureInvalid, note, nil
	}

	sigRaw, ok := doc[signatureFieldName]
	if !ok {
		note := "license JSON missing 'signature' field"
		if strict {
			return SignatureUnverified, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureUnverified, note, nil
	}
	var sigB64 string
	if err := json.Unmarshal(sigRaw, &sigB64); err != nil {
		note := fmt.Sprintf("'signature' field is not a JSON string: %v", err)
		if strict {
			return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureInvalid, note, nil
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		note := fmt.Sprintf("base64 decode signature failed: %v", err)
		if strict {
			return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureInvalid, note, nil
	}

	// 可选 key_id：精确匹配优先；否则降级为 try-all 模式。
	var keyID string
	if kidRaw, ok := doc[keyIDFieldName]; ok {
		_ = json.Unmarshal(kidRaw, &keyID) // best-effort
	}

	canonical, err := canonicalizeLicensePayload(doc)
	if err != nil {
		note := fmt.Sprintf("canonicalize license payload failed: %v", err)
		if strict {
			return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
		}
		return SignatureInvalid, note, nil
	}

	hashed := sha256.Sum256(canonical)
	pssOpts := &rsa.PSSOptions{
		SaltLength: rsaPSSSaltLength,
		Hash:       crypto.SHA256,
	}

	// 1) 精确匹配 keyID
	if keyID != "" {
		if pub, ok := keys[keyID]; ok {
			if vErr := rsa.VerifyPSS(pub, crypto.SHA256, hashed[:], sig, pssOpts); vErr == nil {
				return SignatureVerified,
					fmt.Sprintf("verified with key_id=%s", keyID), nil
			}
			note := fmt.Sprintf("signature did not verify against declared key_id=%s", keyID)
			if strict {
				return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
			}
			return SignatureInvalid, note, nil
		}
		// 声明的 key_id 不在我们的 trust set 里；继续 try-all 当兜底
	}

	// 2) 遍历所有公钥
	for kid, pub := range keys {
		if rsa.VerifyPSS(pub, crypto.SHA256, hashed[:], sig, pssOpts) == nil {
			return SignatureVerified,
				fmt.Sprintf("verified with key_id=%s (key_id field absent or mismatched)", kid),
				nil
		}
	}

	note := "signature did not verify against any configured OEM public key"
	if strict {
		return SignatureInvalid, note, fmt.Errorf("strict signature: %s", note)
	}
	return SignatureInvalid, note, nil
}

// VerifySignature — 兼容 P3 stub 签名（旧 handler 暂未重写时的入口）。
//
// MVP 行为：返回 (Unverified, note)；P4-C 真正逻辑在 SignatureVerifier.VerifyLicenseJSON。
// 保留此函数避免 break 已存在的 handler 路径；handler 在 P4-C 阶段切到 verifier。
func VerifySignature(_ []byte) (SignatureStatus, string) {
	return SignatureUnverified, "OEM public key not configured; signature verification deferred to P4-C"
}

// ---- internal helpers ----

// parsePublicKeysPEM 把 PEM bytes 解析成 keyID → *rsa.PublicKey 映射。
//
// 支持的 PEM type：
//   - "PUBLIC KEY"     — PKIX 包装（推荐，OpenSSL `-pubout` 默认输出）
//   - "RSA PUBLIC KEY" — PKCS#1 裸格式
func parsePublicKeysPEM(raw []byte) (map[string]*rsa.PublicKey, error) {
	out := map[string]*rsa.PublicKey{}
	rest := raw
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		var pub *rsa.PublicKey
		switch block.Type {
		case "PUBLIC KEY":
			anyPub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return out, fmt.Errorf("parse PKIX public key: %w", err)
			}
			rsaPub, ok := anyPub.(*rsa.PublicKey)
			if !ok {
				return out, fmt.Errorf("PKIX key is not RSA (got %T)", anyPub)
			}
			pub = rsaPub
		case "RSA PUBLIC KEY":
			rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return out, fmt.Errorf("parse PKCS1 public key: %w", err)
			}
			pub = rsaPub
		default:
			// 忽略 PRIVATE KEY / CERTIFICATE 等其他 block
			continue
		}
		keyID := fingerprintRSAPublicKey(pub)
		out[keyID] = pub
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no RSA public key found in PEM data")
	}
	return out, nil
}

// fingerprintRSAPublicKey 计算公钥的 SHA-256 fingerprint，hex 格式。
//
// 用 PKIX DER 编码（与 OpenSSL `openssl rsa -pubin -outform der | sha256sum`
// 一致），便于跨工具复核。
func fingerprintRSAPublicKey(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		// MarshalPKIXPublicKey 仅在 key 类型不支持时报错；RSA 类型不会失败。
		// fallback：用 N+E 拼接 hash，保证返回非空 fingerprint。
		return fmt.Sprintf("rsa-fallback-%x", sha256.Sum256([]byte(pub.N.String())))
	}
	sum := sha256.Sum256(der)
	return fmt.Sprintf("%x", sum)
}

// canonicalizeLicensePayload 构造签名时使用的 canonical JSON：
//
//   - 删除 signature / signature_key_id 字段
//   - 顶层字段按 key 字典序排序
//   - JSON marshal 不带空格
//
// 嵌套对象不递归排序（约定：license JSON 顶层已扁平，仅 features 是数组）。
// 这与签名端的"sign before adding signature"约定一致。
func canonicalizeLicensePayload(doc map[string]json.RawMessage) ([]byte, error) {
	keys := make([]string, 0, len(doc))
	for k := range doc {
		if k == signatureFieldName || k == keyIDFieldName {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// 用 json.Marshal 转义 key（虽然 license 字段名都是简单 ASCII，但保险）
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(doc[k])
	}
	buf.WriteByte('}')
	return []byte(buf.String()), nil
}
