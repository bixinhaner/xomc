package loginpwd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// MaxPasswordLength 限制解密后明文密码的字节数，防御异常超长输入耗费 bcrypt CPU。
// bcrypt 自身只取前 72 字节，这里取 128 给 UTF-8 编码（汉字 3 字节）留余量。
const MaxPasswordLength = 128

// Payload 是前端 RSA-OAEP 加密前的 JSON 载荷。
//
// 字段约定（前后端必须保持一致）：
//   - password — 明文密码（UTF-8）
//   - ts       — 客户端 Unix 秒时间戳，由后端 ReplayGuard 校验 ±5 min 容差
//   - nonce    — 至少 16 字节随机串（hex / base64 / urlsafe 任意），全局唯一防重放
type Payload struct {
	Password string `json:"password"`
	TS       int64  `json:"ts"`
	Nonce    string `json:"nonce"`
}

// ErrEmptyCiphertext 表示请求未携带密文字段（或为空字符串）。
var ErrEmptyCiphertext = errors.New("encrypted password is empty")

// ErrEmptyKeyID 表示请求未携带 keyID。
var ErrEmptyKeyID = errors.New("key_id is empty")

// ErrPasswordTooLong 表示解密后明文密码超过 MaxPasswordLength。
var ErrPasswordTooLong = errors.New("decrypted password exceeds max length")

// ErrInvalidPayload 表示密文解密成功但 JSON 解析失败 / 字段缺失。
var ErrInvalidPayload = errors.New("invalid encrypted payload")

// Cipher 聚合 Keystore + ReplayGuard，对外暴露"密文 → 明文密码"的单一入口。
type Cipher struct {
	store  *Keystore
	replay ReplayGuard
}

// NewCipher 构造 Cipher。两个依赖均不可为空。
func NewCipher(store *Keystore, replay ReplayGuard) *Cipher {
	if store == nil {
		panic("loginpwd: NewCipher: store is nil")
	}
	if replay == nil {
		panic("loginpwd: NewCipher: replay is nil")
	}
	return &Cipher{store: store, replay: replay}
}

// ActiveKeyID 透传 keystore 的当前 keyID，便于上层对外暴露。
func (c *Cipher) ActiveKeyID() string {
	return c.store.ActiveKeyID()
}

// ActivePublicKeyPEM 透传 keystore 的当前 PEM 公钥。
func (c *Cipher) ActivePublicKeyPEM() string {
	return c.store.ActivePublicKeyPEM()
}

// Decrypt 把请求中携带的 base64 密文 + keyID 还原为明文密码。
//
// 完整流程：
//  1. 校验非空 → base64 decode → RSA-OAEP 解密 → JSON unmarshal
//  2. 长度上限校验
//  3. ReplayGuard 校验 ts 窗口 + nonce 唯一
//
// 任何一步失败都会返回 wrap 后的 error，调用方应将其映射为 401（防探测对外不区分原因）。
func (c *Cipher) Decrypt(ctx context.Context, keyID, ciphertextB64 string) (string, error) {
	keyID = strings.TrimSpace(keyID)
	ciphertextB64 = strings.TrimSpace(ciphertextB64)

	if keyID == "" {
		return "", ErrEmptyKeyID
	}
	if ciphertextB64 == "" {
		return "", ErrEmptyCiphertext
	}

	ciphertext, err := decodeBase64Flexible(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	plaintext, err := c.store.Decrypt(keyID, ciphertext)
	if err != nil {
		return "", err
	}

	var payload Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if payload.Password == "" || payload.Nonce == "" || payload.TS == 0 {
		return "", fmt.Errorf("%w: missing field", ErrInvalidPayload)
	}
	if len(payload.Password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	if err := c.replay.Check(ctx, payload.TS, payload.Nonce); err != nil {
		return "", err
	}
	return payload.Password, nil
}

// decodeBase64Flexible 同时接受标准 base64 与 URL-safe base64，并自动补齐 padding。
// 前端不同库（atob / Buffer / Web Crypto）输出风格不统一，统一兼容降低集成成本。
func decodeBase64Flexible(s string) ([]byte, error) {
	// 自动补齐 '=' padding
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	if strings.ContainsAny(s, "-_") {
		return base64.URLEncoding.DecodeString(s)
	}
	return base64.StdEncoding.DecodeString(s)
}
