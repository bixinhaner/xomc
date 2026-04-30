package backup

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// mockKMSClient — deterministic in-process KMSClient for tests
// =============================================================================
//
// Models the AWS KMS / Vault Transit shape: a master key wraps a freshly-
// generated 32B data key under AES-256-GCM. Real adapters do this on the
// service side; this mock does it locally so tests are deterministic and
// network-free. Production code MUST NOT depend on mockKMSClient — it
// lives in the test file and is unexported.

type mockKMSClient struct {
	masterKey []byte // 32B AES-256
	keyID     string
	// failNext is the operation name that must fail on its next invocation.
	// Reset to "" after one failure so subsequent calls succeed.
	failNext string
}

func newMockKMSClient(t *testing.T, keyID string) *mockKMSClient {
	t.Helper()
	mk := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, mk)
	require.NoError(t, err)
	return &mockKMSClient{masterKey: mk, keyID: keyID}
}

func (m *mockKMSClient) KeyID() string { return m.keyID }

// GenerateDataKey produces a fresh 32B plaintext + the AES-256-GCM
// ciphertextBlob (12B nonce ‖ ciphertext+16B tag). Mirrors the AWS KMS
// blob layout closely enough for tests but is not interoperable with any
// real KMS — that is by design (mocks must be unmistakably non-real).
func (m *mockKMSClient) GenerateDataKey(_ context.Context) ([]byte, []byte, error) {
	if m.failNext == "generate" {
		m.failNext = ""
		return nil, nil, errors.New("mock kms: GenerateDataKey injected failure")
	}
	plaintext := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, plaintext); err != nil {
		return nil, nil, err
	}
	blob, err := m.wrap(plaintext)
	if err != nil {
		return nil, nil, err
	}
	return plaintext, blob, nil
}

func (m *mockKMSClient) Decrypt(_ context.Context, blob []byte) ([]byte, error) {
	if m.failNext == "decrypt" {
		m.failNext = ""
		return nil, errors.New("mock kms: Decrypt injected failure")
	}
	return m.unwrap(blob)
}

func (m *mockKMSClient) wrap(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func (m *mockKMSClient) unwrap(blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(blob) < gcm.NonceSize()+16 {
		return nil, errors.New("mock kms: blob too short")
	}
	nonce, ct := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("mock kms: GCM verification failed")
	}
	return pt, nil
}

// =============================================================================
// V1 — KMSKeyProvider 构造成功 + KEK round-trip
// =============================================================================

func TestKMSKeyProvider_V1_RoundTrip(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")
	plaintext, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)
	require.Len(t, plaintext, 32)
	require.NotEmpty(t, blob)

	kp, err := NewKMSKeyProvider(ctx, client, blob)
	require.NoError(t, err)
	require.NotNil(t, kp)
	assert.True(t, kp.Available())
	assert.Equal(t, "mock-kms-master-1", kp.KeyID())

	got, err := kp.KEK(ctx)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(plaintext, got),
		"KEK(ctx) must return the same plaintext that GenerateDataKey produced")
}

// =============================================================================
// V2 — KEK 返回 copy 防 caller mutation
// =============================================================================

func TestKMSKeyProvider_V2_KEKReturnsCopy(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")
	_, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)
	kp, err := NewKMSKeyProvider(ctx, client, blob)
	require.NoError(t, err)

	first, err := kp.KEK(ctx)
	require.NoError(t, err)
	// Mutate the returned slice in place.
	for i := range first {
		first[i] ^= 0xFF
	}

	second, err := kp.KEK(ctx)
	require.NoError(t, err)
	assert.False(t, bytes.Equal(first, second),
		"second KEK call must NOT see caller's mutation of the first call (M-2 fix mirror)")
}

// =============================================================================
// V3 — KMS Decrypt 失败路径
// =============================================================================

func TestKMSKeyProvider_V3_DecryptFailure(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")
	_, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)

	client.failNext = "decrypt"
	kp, err := NewKMSKeyProvider(ctx, client, blob)
	require.Error(t, err)
	assert.Nil(t, kp)
	assert.Contains(t, err.Error(), "decrypt blob")
}

// =============================================================================
// V4 — ciphertextBlob 篡改检测
// =============================================================================

func TestKMSKeyProvider_V4_BlobTampered(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")
	_, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)

	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[len(tampered)-1] ^= 0x01

	kp, err := NewKMSKeyProvider(ctx, client, tampered)
	require.Error(t, err)
	assert.Nil(t, kp)
}

// =============================================================================
// V5 — KMSKeyProvider satisfies KeyProvider (compile-time + runtime)
// =============================================================================

func TestKMSKeyProvider_V5_SatisfiesKeyProvider(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")
	_, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)
	kp, err := NewKMSKeyProvider(ctx, client, blob)
	require.NoError(t, err)

	// Runtime conformance: assignment to KeyProvider interface variable.
	var iface KeyProvider = kp
	assert.True(t, iface.Available())
	got, err := iface.KEK(ctx)
	require.NoError(t, err)
	assert.Len(t, got, kekSize)
}

// =============================================================================
// V6 — KeyID exposure
// =============================================================================

func TestKMSKeyProvider_V6_KeyIDExposed(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "arn:aws:kms:us-east-1:111122223333:key/mrk-abc")
	_, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)
	kp, err := NewKMSKeyProvider(ctx, client, blob)
	require.NoError(t, err)

	assert.Equal(t, "arn:aws:kms:us-east-1:111122223333:key/mrk-abc", kp.KeyID())
	// Nil receiver returns empty string (not panic) — defence-in-depth.
	var nilKp *KMSKeyProvider
	assert.Equal(t, "", nilKp.KeyID())
}

// =============================================================================
// V7 — Available false paths: nil client, empty blob, nil receiver
// =============================================================================

func TestKMSKeyProvider_V7_AvailableFalseForBadConstruction(t *testing.T) {
	ctx := context.Background()

	// nil client
	kp, err := NewKMSKeyProvider(ctx, nil, []byte("any"))
	require.Error(t, err)
	assert.Nil(t, kp)
	assert.True(t, errors.Is(err, ErrEncryptionKeyUnavailable))

	// empty blob
	client := newMockKMSClient(t, "mock-kms-master-1")
	kp, err = NewKMSKeyProvider(ctx, client, nil)
	require.Error(t, err)
	assert.Nil(t, kp)
	assert.True(t, errors.Is(err, ErrEncryptionKeyUnavailable))

	// nil receiver Available
	var nilKp *KMSKeyProvider
	assert.False(t, nilKp.Available())
}

// =============================================================================
// V8 — mockKMSClient self-round-trip + GenerateDataKey failure
// =============================================================================

func TestMockKMSClient_V8_RoundTrip(t *testing.T) {
	ctx := context.Background()
	client := newMockKMSClient(t, "mock-kms-master-1")

	plaintext, blob, err := client.GenerateDataKey(ctx)
	require.NoError(t, err)
	require.Len(t, plaintext, 32)
	// Blob is nonce(12) + ciphertext(32) + tag(16) = 60 bytes
	assert.Equal(t, 60, len(blob))

	got, err := client.Decrypt(ctx, blob)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(plaintext, got))

	// Inject GenerateDataKey failure
	client.failNext = "generate"
	_, _, err = client.GenerateDataKey(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateDataKey injected failure")

	// failNext resets, next call succeeds
	_, _, err = client.GenerateDataKey(ctx)
	require.NoError(t, err)
}

// =============================================================================
// V9 — different masterKey → blob not interoperable
// =============================================================================

func TestMockKMSClient_V9_DifferentMasterKeyNotInteroperable(t *testing.T) {
	ctx := context.Background()
	clientA := newMockKMSClient(t, "master-A")
	clientB := newMockKMSClient(t, "master-B")

	_, blobA, err := clientA.GenerateDataKey(ctx)
	require.NoError(t, err)

	_, err = clientB.Decrypt(ctx, blobA)
	require.Error(t, err, "different master key must reject blob from another KMS instance")
	assert.Contains(t, err.Error(), "GCM verification failed")
}

// =============================================================================
// V10 — Decrypt of a wrong-shape KEK plaintext (e.g., KMS misbehaviour
// returning 16B instead of 32B) must reject construction with
// ErrEncryptionFormatInvalid. Defends against silently accepting a
// truncated KEK.
// =============================================================================

type wrongSizeKMSClient struct{}

func (wrongSizeKMSClient) KeyID() string { return "wrong-size" }
func (wrongSizeKMSClient) GenerateDataKey(_ context.Context) ([]byte, []byte, error) {
	return nil, nil, errors.New("not used")
}
func (wrongSizeKMSClient) Decrypt(_ context.Context, _ []byte) ([]byte, error) {
	return make([]byte, 16), nil // 16 bytes — wrong size for AES-256
}

func TestKMSKeyProvider_V10_RejectsWrongSizeKEK(t *testing.T) {
	kp, err := NewKMSKeyProvider(context.Background(), wrongSizeKMSClient{}, []byte("any"))
	require.Error(t, err)
	assert.Nil(t, kp)
	assert.True(t, errors.Is(err, ErrEncryptionFormatInvalid),
		"wrong KEK size must surface as format invalid (not silently accepted)")
}
