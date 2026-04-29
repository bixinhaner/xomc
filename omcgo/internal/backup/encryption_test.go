package backup

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, kekSize)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)
	return key
}

func TestNewEncryptor_invalidAlgorithm(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	_, err := NewEncryptor("AES-666-XYZ", kp)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAlgorithmInvalid))
}

func TestNewEncryptor_stubAlgorithms(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	for _, algo := range []string{"AES-256-CBC", "ChaCha20-Poly1305"} {
		_, err := NewEncryptor(algo, kp)
		require.Error(t, err, "algorithm=%s must return stub error", algo)
		assert.True(t, errors.Is(err, ErrEncryptionAlgorithmNotImplemented))
	}
}

func TestNewEncryptor_keyUnavailable(t *testing.T) {
	// nil provider
	_, err := NewEncryptor("AES-256-GCM", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionKeyUnavailable))
	// provider with no key
	emptyKp := newStaticKeyProvider(nil)
	_, err = NewEncryptor("AES-256-GCM", emptyKp)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionKeyUnavailable))
}

func TestAESGCM_RoundTrip_sizes(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, err := NewEncryptor("AES-256-GCM", kp)
	require.NoError(t, err)

	for _, size := range []int{0, 1, 16, 1023, 1024, 65536, 1024 * 1024} {
		size := size
		t.Run("", func(t *testing.T) {
			plaintext := make([]byte, size)
			_, _ = io.ReadFull(rand.Reader, plaintext)

			blob, err := enc.Encrypt(plaintext, []byte("cfg.xml"))
			require.NoError(t, err)

			// Magic header check
			require.GreaterOrEqual(t, len(blob), 8)
			assert.Equal(t, []byte(encMagic), blob[:4])
			assert.Equal(t, encVersion, blob[4])
			assert.Equal(t, encAlgoGCM, blob[5])
			// Overhead = 100 bytes per file
			assert.Equal(t, len(plaintext)+100, len(blob))

			recovered, err := enc.Decrypt(blob, []byte("cfg.xml"))
			require.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, recovered),
				"size=%d round-trip mismatch", size)
		})
	}
}

func TestAESGCM_TamperDetected(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	plaintext := []byte("sensitive backup config")
	blob, err := enc.Encrypt(plaintext, []byte("cfg.xml"))
	require.NoError(t, err)

	// Flip a byte deep inside the ciphertext (past header + nonces + wrapped_DEK).
	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[len(tampered)-5] ^= 0x01

	_, err = enc.Decrypt(tampered, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

func TestAESGCM_WrongAAD(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	blob, err := enc.Encrypt([]byte("hello"), []byte("filename-A.xml"))
	require.NoError(t, err)

	_, err = enc.Decrypt(blob, []byte("filename-B.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"AAD mismatch must surface as auth failure (anti-substitution)")
}

func TestAESGCM_WrongKey(t *testing.T) {
	kpA := newStaticKeyProvider(makeTestKey(t))
	kpB := newStaticKeyProvider(makeTestKey(t))
	encA, _ := NewEncryptor("AES-256-GCM", kpA)
	encB, _ := NewEncryptor("AES-256-GCM", kpB)

	blob, err := encA.Encrypt([]byte("msg"), nil)
	require.NoError(t, err)

	_, err = encB.Decrypt(blob, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

// TestAESGCM_NonceUniqueness encrypts the same plaintext under the same key
// many times; every ciphertext must be unique because (outer || inner) nonces
// are randomly generated. Catches catastrophic deterministic-encryption bugs.
func TestAESGCM_NonceUniqueness(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	const N = 500
	seen := make(map[string]struct{}, N)
	for i := 0; i < N; i++ {
		blob, err := enc.Encrypt([]byte("constant"), []byte("AAD"))
		require.NoError(t, err)
		key := string(blob)
		_, dup := seen[key]
		require.False(t, dup, "ciphertext collision on iteration %d", i)
		seen[key] = struct{}{}
	}
}

func TestAESGCM_InputTooLarge(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	// Build a slice that EXCEEDS encMaxPlaintext by 1 byte without filling
	// it with random data (we just need len to trip the check).
	oversized := make([]byte, encMaxPlaintext+1)
	_, err := enc.Encrypt(oversized, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionInputTooLarge))
}

// TestAESGCM_AAD_CompressedExtension regression-guards the review HIGH-1
// fix: when compression and encryption are both active, the upload-side
// AAD must equal the on-disk basename minus `.enc` (i.e. include the
// compression extension), so the download-side AAD reconstruction matches.
func TestAESGCM_AAD_CompressedExtension(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	plaintext := []byte("compressed-then-encrypted backup payload")
	// Upload side: AAD = filename + cmp.ext (e.g. "cfg.xml.gz")
	uploadAAD := []byte("cfg.xml.gz")
	blob, err := enc.Encrypt(plaintext, uploadAAD)
	require.NoError(t, err)

	// Download side: object on disk is "cfg.xml.gz.enc"; AAD reconstructed
	// by stripping ".enc" then taking basename = "cfg.xml.gz". Must match.
	downloadAAD := []byte("cfg.xml.gz")
	recovered, err := enc.Decrypt(blob, downloadAAD)
	require.NoError(t, err, "compressed+encrypted round-trip must succeed")
	assert.Equal(t, plaintext, recovered)

	// And: a downloader that builds AAD without the compression extension
	// (the pre-fix bug) MUST get auth failure, never silent success.
	_, err = enc.Decrypt(blob, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"pre-fix AAD (no .gz) must NOT decrypt — guards regression")
}

func TestAESGCM_FormatInvalid(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	cases := [][]byte{
		nil,                                                            // nil
		make([]byte, 50),                                               // too short
		append([]byte("WRONG"), make([]byte, 200)...),                  // bad magic
		append([]byte(encMagic), append([]byte{0x99}, make([]byte, 200)...)...), // bad version
	}
	for i, blob := range cases {
		_, err := enc.Decrypt(blob, nil)
		require.Error(t, err, "case %d should reject", i)
		// Expect either FormatInvalid OR AlgorithmNotImplemented (algo byte branch)
		assert.True(t,
			errors.Is(err, ErrEncryptionFormatInvalid) || errors.Is(err, ErrEncryptionAlgorithmNotImplemented),
			"case %d unexpected error class: %v", i, err)
	}
}
