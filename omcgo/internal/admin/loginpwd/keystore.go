// Package loginpwd 实现登录类接口的密码加密传输：
// 前端用服务器下发的 RSA 公钥（OAEP / SHA-256）加密 password+ts+nonce 形成密文 →
// 后端按 keyID 选私钥解密 → ts ±5 min 容差 + nonce Redis 防重放 → 取出明文密码走 bcrypt。
//
// 适用接口：
//   - POST /auth/login                       — password
//   - POST /auth/change-password             — old_password / new_password
//   - POST /admin/users/:id/reset-password   — new_password
//   - POST /admin/users                      — password（创建用户初始密码）
//
// 设计要点：
//   - keyID = base64url(SHA-256(SubjectPublicKeyInfo))[:22]，由公钥确定性派生，
//     多副本部署若挂载同一私钥即可获得同一 keyID，无需共享 ID 元数据。
//   - 私钥首次启动时自动生成并落盘（0600），路径由配置 auth.password_encryption.private_key_path
//     指定；生产建议改用 K8s Secret 挂载固定密钥，保证多实例一致。
//   - OAEP padding（SHA-256）规避 PKCS1v1.5 的 padding oracle 风险。
package loginpwd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultRSABits 默认 RSA 密钥长度。2048 位是 NIST 当前建议的下限，足够 2030 前使用。
const DefaultRSABits = 2048

// ErrUnknownKeyID 表示请求中携带的 keyID 不存在于当前 keystore 中。
var ErrUnknownKeyID = errors.New("unknown keyID")

// Keystore 持有一组 RSA 私钥（按 keyID 索引），并维护一个"激活" keyID 用于公钥下发。
//
// 当前实现仅持有单一密钥；保留多键查找能力是为了未来支持密钥轮换（在过渡期内
// 同时接受新旧 keyID 的密文）。
type Keystore struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PrivateKey // keyID → privateKey
	publicPEM map[string]string          // keyID → PEM-encoded SubjectPublicKeyInfo
	activeID  string
}

// NewKeystore 加载或初始化 keystore：
//   - 若 privateKeyPath 已存在，读取并解析为 RSA 私钥
//   - 否则在该路径生成一份 DefaultRSABits 位的新密钥（文件权限 0600，目录权限 0700）
func NewKeystore(privateKeyPath string) (*Keystore, error) {
	if privateKeyPath == "" {
		return nil, errors.New("privateKeyPath is empty")
	}

	priv, err := loadOrGenerate(privateKeyPath)
	if err != nil {
		return nil, err
	}

	keyID, pubPEM, err := derivePublicKeyArtifacts(priv)
	if err != nil {
		return nil, err
	}

	return &Keystore{
		keys:      map[string]*rsa.PrivateKey{keyID: priv},
		publicPEM: map[string]string{keyID: pubPEM},
		activeID:  keyID,
	}, nil
}

// ActiveKeyID 返回当前用于公钥下发的 keyID。
func (k *Keystore) ActiveKeyID() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.activeID
}

// ActivePublicKeyPEM 返回 active keyID 对应的 PEM 编码公钥。
func (k *Keystore) ActivePublicKeyPEM() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.publicPEM[k.activeID]
}

// Decrypt 用 keyID 对应的私钥执行 RSA-OAEP-SHA256 解密。
func (k *Keystore) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	k.mu.RLock()
	priv, ok := k.keys[keyID]
	k.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKeyID, keyID)
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa-oaep decrypt: %w", err)
	}
	return plaintext, nil
}

// loadOrGenerate 读取或生成 RSA 私钥，并以 PKCS#8 PEM 形式落盘。
func loadOrGenerate(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return parsePrivateKeyPEM(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, DefaultRSABits)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal pkcs8: %w", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write private key %s: %w", path, err)
	}
	return priv, nil
}

func parsePrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("decode pem: no block found")
	}
	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs8: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("pkcs8 key is not RSA")
		}
		return rsaKey, nil
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported pem block type: %s", block.Type)
	}
}

// derivePublicKeyArtifacts 从私钥派生 keyID 与 PEM 公钥。
// keyID = base64url(SHA-256(SubjectPublicKeyInfo))[:22]
//
// 用 SPKI 摘要而非 UUID 是为了"同一密钥 → 同一 ID"，多副本部署直接共享密钥即获得一致 ID。
func derivePublicKeyArtifacts(priv *rsa.PrivateKey) (keyID, pubPEM string, err error) {
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal public key: %w", err)
	}
	sum := sha256.Sum256(der)
	keyID = base64.RawURLEncoding.EncodeToString(sum[:])[:22]
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return keyID, pubPEM, nil
}
