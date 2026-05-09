package license

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signLicenseWithKey 模拟 OEM 签名工具：把 license payload (无 signature/key_id 字段)
// 用 RSA-PSS 签名后嵌入 signature + signature_key_id 字段；返回完整 license JSON。
func signLicenseWithKey(t *testing.T, payload map[string]any, priv *rsa.PrivateKey) []byte {
	t.Helper()

	// 注入 keyID 但还不签名
	keyID := fingerprintRSAPublicKey(&priv.PublicKey)

	// canonicalize payload（与 verifier 同算法）
	docJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(docJSON, &doc))
	canonical, err := canonicalizeLicensePayload(doc)
	require.NoError(t, err)

	hashed := sha256.Sum256(canonical)
	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], &rsa.PSSOptions{
		SaltLength: rsaPSSSaltLength,
		Hash:       crypto.SHA256,
	})
	require.NoError(t, err)

	// 加 signature + key_id 字段
	signed := make(map[string]any, len(payload)+2)
	for k, v := range payload {
		signed[k] = v
	}
	signed[signatureFieldName] = base64.StdEncoding.EncodeToString(sig)
	signed[keyIDFieldName] = keyID

	out, err := json.Marshal(signed)
	require.NoError(t, err)
	return out
}

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv
}

func writePEMPublicKey(t *testing.T, path string, pub *rsa.PublicKey) {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))
}

// --- Tests ---

func TestSignatureVerifier_Verified(t *testing.T) {
	priv := newTestRSAKey(t)
	v := NewSignatureVerifier(false)
	v.AddKey(&priv.PublicKey)

	signed := signLicenseWithKey(t, map[string]any{
		"license_code": "LIC-VER-001",
		"license_name": "Verified",
		"max_devices":  100,
	}, priv)

	status, note, err := v.VerifyLicenseJSON(signed)
	require.NoError(t, err)
	assert.Equal(t, SignatureVerified, status)
	assert.Contains(t, note, "verified with key_id=")
}

func TestSignatureVerifier_InvalidSignature_Tampered(t *testing.T) {
	priv := newTestRSAKey(t)
	v := NewSignatureVerifier(false)
	v.AddKey(&priv.PublicKey)

	signed := signLicenseWithKey(t, map[string]any{
		"license_code": "LIC-TAMP-001",
		"max_devices":  100,
	}, priv)

	// 篡改 max_devices
	var doc map[string]any
	require.NoError(t, json.Unmarshal(signed, &doc))
	doc["max_devices"] = 9999
	tampered, err := json.Marshal(doc)
	require.NoError(t, err)

	status, note, err := v.VerifyLicenseJSON(tampered)
	require.NoError(t, err) // strict=false → no error
	assert.Equal(t, SignatureInvalid, status)
	assert.Contains(t, note, "did not verify")
}

func TestSignatureVerifier_InvalidSignature_StrictRejects(t *testing.T) {
	priv := newTestRSAKey(t)
	v := NewSignatureVerifier(true) // strict
	v.AddKey(&priv.PublicKey)

	signed := signLicenseWithKey(t, map[string]any{
		"license_code": "LIC-TAMP-002",
		"max_devices":  100,
	}, priv)

	// 篡改
	var doc map[string]any
	require.NoError(t, json.Unmarshal(signed, &doc))
	doc["max_devices"] = 9999
	tampered, err := json.Marshal(doc)
	require.NoError(t, err)

	status, _, err := v.VerifyLicenseJSON(tampered)
	require.Error(t, err)
	assert.Equal(t, SignatureInvalid, status)
}

func TestSignatureVerifier_MissingSignatureField(t *testing.T) {
	v := NewSignatureVerifier(false)
	priv := newTestRSAKey(t)
	v.AddKey(&priv.PublicKey)

	noSig, err := json.Marshal(map[string]any{
		"license_code": "LIC-NOSIG-001",
		"max_devices":  100,
	})
	require.NoError(t, err)

	status, note, err := v.VerifyLicenseJSON(noSig)
	require.NoError(t, err)
	assert.Equal(t, SignatureUnverified, status)
	assert.Contains(t, note, "missing 'signature' field")
}

func TestSignatureVerifier_MissingSignature_StrictRejects(t *testing.T) {
	v := NewSignatureVerifier(true)
	priv := newTestRSAKey(t)
	v.AddKey(&priv.PublicKey)

	noSig, err := json.Marshal(map[string]any{"license_code": "LIC-NOSIG-002"})
	require.NoError(t, err)

	status, _, err := v.VerifyLicenseJSON(noSig)
	require.Error(t, err)
	assert.Equal(t, SignatureUnverified, status)
}

func TestSignatureVerifier_NoKeysConfigured(t *testing.T) {
	// 没加 key 的 verifier 默认 strict=false → unverified 放过
	v := NewSignatureVerifier(false)
	signed := []byte(`{"license_code":"LIC-NOKEY-001","signature":"abc"}`)
	status, note, err := v.VerifyLicenseJSON(signed)
	require.NoError(t, err)
	assert.Equal(t, SignatureUnverified, status)
	assert.Contains(t, note, "no OEM public keys")
}

func TestSignatureVerifier_NoKeys_StrictRejects(t *testing.T) {
	v := NewSignatureVerifier(true)
	signed := []byte(`{"license_code":"LIC-NOKEY-002"}`)
	status, _, err := v.VerifyLicenseJSON(signed)
	require.Error(t, err)
	assert.Equal(t, SignatureUnverified, status)
}

func TestSignatureVerifier_WrongKey_TryAllFails(t *testing.T) {
	priv1 := newTestRSAKey(t) // 签名用
	priv2 := newTestRSAKey(t) // 加载到 verifier
	v := NewSignatureVerifier(false)
	v.AddKey(&priv2.PublicKey)

	signed := signLicenseWithKey(t, map[string]any{"license_code": "LIC-WRONG-001"}, priv1)
	// 不带 key_id 时也要 try-all 兜底——先去掉 key_id
	var doc map[string]any
	require.NoError(t, json.Unmarshal(signed, &doc))
	delete(doc, keyIDFieldName)
	signedNoKID, err := json.Marshal(doc)
	require.NoError(t, err)

	status, note, err := v.VerifyLicenseJSON(signedNoKID)
	require.NoError(t, err)
	assert.Equal(t, SignatureInvalid, status)
	assert.Contains(t, note, "did not verify against any")
}

func TestSignatureVerifier_LoadKeysFromDir_HappyPath(t *testing.T) {
	dir := t.TempDir()
	priv := newTestRSAKey(t)
	writePEMPublicKey(t, filepath.Join(dir, "oem-1.pem"), &priv.PublicKey)

	// 加一个 .txt 不会被识别
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("noop"), 0o600))

	v := NewSignatureVerifier(false)
	require.NoError(t, v.LoadKeysFromDir(dir))
	assert.Equal(t, 1, v.KeyCount())

	signed := signLicenseWithKey(t, map[string]any{"license_code": "LIC-DIR-001"}, priv)
	status, _, err := v.VerifyLicenseJSON(signed)
	require.NoError(t, err)
	assert.Equal(t, SignatureVerified, status)
}

func TestSignatureVerifier_LoadKeysFromDir_NonexistentNoError(t *testing.T) {
	v := NewSignatureVerifier(false)
	require.NoError(t, v.LoadKeysFromDir("/this/path/does/not/exist"))
	assert.Equal(t, 0, v.KeyCount())
}

func TestSignatureVerifier_LoadKeysFromDir_EmptyDirNoError(t *testing.T) {
	dir := t.TempDir()
	v := NewSignatureVerifier(false)
	require.NoError(t, v.LoadKeysFromDir(dir))
	assert.Equal(t, 0, v.KeyCount())
}

func TestSignatureVerifier_NilVerifier_StubBehavior(t *testing.T) {
	var v *SignatureVerifier // nil
	status, note, err := v.VerifyLicenseJSON([]byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, SignatureUnverified, status)
	assert.Contains(t, note, "stub mode")
}

func TestSignatureVerifier_Canonicalize_OrderInvariant(t *testing.T) {
	priv := newTestRSAKey(t)
	v := NewSignatureVerifier(false)
	v.AddKey(&priv.PublicKey)

	// 签名时字段顺序与验证时不同的输入，应该都通过（canonicalize 排序）
	signed := signLicenseWithKey(t, map[string]any{
		"license_code": "LIC-ORDER-001",
		"max_devices":  100,
		"name":         "Order Test",
	}, priv)

	// 解码 → 重新 marshal（字段顺序可能变）
	var doc map[string]any
	require.NoError(t, json.Unmarshal(signed, &doc))
	reEncoded, err := json.Marshal(doc)
	require.NoError(t, err)

	status, _, err := v.VerifyLicenseJSON(reEncoded)
	require.NoError(t, err)
	assert.Equal(t, SignatureVerified, status)
}
