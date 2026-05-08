package loginpwd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKeystore_GeneratesAndPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "key.pem")

	ks1, err := NewKeystore(path)
	require.NoError(t, err)
	require.NotEmpty(t, ks1.ActiveKeyID())
	require.Contains(t, ks1.ActivePublicKeyPEM(), "BEGIN PUBLIC KEY")

	// 第二次构造应读取同一文件，得到完全相同的 keyID。
	ks2, err := NewKeystore(path)
	require.NoError(t, err)
	require.Equal(t, ks1.ActiveKeyID(), ks2.ActiveKeyID())
	require.Equal(t, ks1.ActivePublicKeyPEM(), ks2.ActivePublicKeyPEM())
}

func TestKeystore_Decrypt_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	ks, err := NewKeystore(path)
	require.NoError(t, err)

	pubKey := loadPublicKeyFromPEM(t, ks.ActivePublicKeyPEM())

	plaintext := []byte(`{"password":"hunter2","ts":1714896000,"nonce":"abcd1234"}`)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, plaintext, nil)
	require.NoError(t, err)

	got, err := ks.Decrypt(ks.ActiveKeyID(), ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, got)
}

func TestKeystore_Decrypt_UnknownKeyID(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(filepath.Join(dir, "key.pem"))
	require.NoError(t, err)

	_, err = ks.Decrypt("not-a-real-key-id", []byte("ciphertext-bytes"))
	require.ErrorIs(t, err, ErrUnknownKeyID)
}

// loadPublicKeyFromPEM 把 keystore 返回的 PEM 公钥解析回 *rsa.PublicKey，方便测试加密。
func loadPublicKeyFromPEM(t *testing.T, pemStr string) *rsa.PublicKey {
	t.Helper()
	block, _ := pem.Decode([]byte(pemStr))
	require.NotNil(t, block)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	rsaPub, ok := pub.(*rsa.PublicKey)
	require.True(t, ok, "expected *rsa.PublicKey")
	return rsaPub
}

// payloadJSON 序列化 Payload 用于测试。本地 helper，不暴露给生产代码。
func payloadJSON(t *testing.T, p Payload) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return b
}
