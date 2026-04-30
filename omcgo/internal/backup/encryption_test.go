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

// T-0085 closed the encryption algorithm matrix — CBC + ChaCha20-Poly1305
// now construct successfully. The previous T-0075 stub-rejection assertion
// is reversed here (rather than deleted) to leave a hard regression guard:
// if anyone re-introduces a stub, this test fails fast.
func TestNewEncryptor_allAlgorithmsConstruct(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	for _, algo := range []string{"AES-256-GCM", "AES-256-CBC", "ChaCha20-Poly1305"} {
		enc, err := NewEncryptor(algo, kp)
		require.NoError(t, err, "algorithm=%s must construct successfully (T-0085)", algo)
		require.NotNil(t, enc)
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

// =============================================================================
// T-0085 — AES-256-CBC + HMAC-SHA256 (encrypt-then-MAC)
// =============================================================================

// TestAESCBC_RoundTrip exercises the full size grid PRD §4 V1: empty,
// single-byte, exactly one block, sub-block, exactly one block + 1, and
// large inputs (64 KiB, 1 MiB) through encrypt → decrypt with matching
// AAD, including the algoByte assertion.
func TestAESCBC_RoundTrip(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, err := NewEncryptor("AES-256-CBC", kp)
	require.NoError(t, err)

	for _, size := range []int{0, 1, 16, 1023, 1024, 65536, 1024 * 1024} {
		size := size
		t.Run("", func(t *testing.T) {
			plaintext := make([]byte, size)
			_, _ = io.ReadFull(rand.Reader, plaintext)

			blob, err := enc.Encrypt(plaintext, []byte("cfg.xml"))
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(blob), 8)
			assert.Equal(t, []byte(encMagic), blob[:4])
			assert.Equal(t, encVersion, blob[4])
			assert.Equal(t, encAlgoCBC, blob[5])

			recovered, err := enc.Decrypt(blob, []byte("cfg.xml"))
			require.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, recovered),
				"size=%d round-trip mismatch", size)
		})
	}
}

// TestAESCBC_TamperDetected — flip a byte in the CBC ciphertext and assert
// HMAC catches it. CRITICAL for padding-oracle defence.
func TestAESCBC_TamperDetected(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-CBC", kp)

	plaintext := []byte("sensitive backup config")
	blob, err := enc.Encrypt(plaintext, []byte("cfg.xml"))
	require.NoError(t, err)

	// Flip one byte deep inside ciphertext (before the trailing 32B HMAC).
	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	// Pick a byte midway between header end and HMAC start.
	bitFlipPos := len(blob) - encHMACSize - 5
	tampered[bitFlipPos] ^= 0x01

	_, err = enc.Decrypt(tampered, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"CBC tamper must surface as auth failure (HMAC verify)")
}

// TestAESCBC_WrongAAD — Decrypt with different AAD must fail HMAC verify.
func TestAESCBC_WrongAAD(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-CBC", kp)

	blob, err := enc.Encrypt([]byte("hello"), []byte("filename-A.xml"))
	require.NoError(t, err)

	_, err = enc.Decrypt(blob, []byte("filename-B.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"AAD mismatch must surface as auth failure (anti-substitution)")
}

// TestAESCBC_WrongKey — Decrypt with a different KEK must fail at the
// unwrap-DEK GCM step (NOT propagate the underlying error type, must
// surface as ErrEncryptionAuthFailed for caller-uniform handling).
func TestAESCBC_WrongKey(t *testing.T) {
	kpA := newStaticKeyProvider(makeTestKey(t))
	kpB := newStaticKeyProvider(makeTestKey(t))
	encA, _ := NewEncryptor("AES-256-CBC", kpA)
	encB, _ := NewEncryptor("AES-256-CBC", kpB)

	blob, err := encA.Encrypt([]byte("msg"), nil)
	require.NoError(t, err)

	_, err = encB.Decrypt(blob, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"wrong KEK must surface as auth failure (unwrap-DEK GCM)")
}

// TestAESCBC_TamperedHMAC — flip a byte in the trailing HMAC block.
func TestAESCBC_TamperedHMAC(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-CBC", kp)

	blob, err := enc.Encrypt([]byte("payload"), []byte("cfg.xml"))
	require.NoError(t, err)

	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[len(tampered)-1] ^= 0x01

	_, err = enc.Decrypt(tampered, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

// TestAESCBC_TamperedIV — flip a byte in the IV block. The HMAC scope
// includes the IV (PRD §2.1) precisely to catch this.
func TestAESCBC_TamperedIV(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-CBC", kp)

	blob, err := enc.Encrypt([]byte("payload"), []byte("cfg.xml"))
	require.NoError(t, err)

	// IV starts after header(8) + outer_nonce(12) + len(4) + wrapped(48).
	ivOff := 8 + encNonceSize + 4 + encWrappedDEKSize
	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[ivOff] ^= 0x01

	_, err = enc.Decrypt(tampered, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed),
		"IV tamper must be caught by HMAC scope (anti IV-substitution)")
}

// TestAESCBC_AlgoByteSubstitution — change the algo byte from 'C' to 'G'
// and confirm the CBC decryptor rejects it (algo binding to HMAC scope
// would also catch it during HMAC verify, but the explicit algo-byte
// gate at parse time is the first line of defence).
func TestAESCBC_AlgoByteSubstitution(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-CBC", kp)

	blob, err := enc.Encrypt([]byte("payload"), nil)
	require.NoError(t, err)

	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[5] = encAlgoGCM

	_, err = enc.Decrypt(tampered, nil)
	require.Error(t, err)
}

// =============================================================================
// T-0085 — ChaCha20-Poly1305
// =============================================================================

func TestChaCha20Poly1305_RoundTrip(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, err := NewEncryptor("ChaCha20-Poly1305", kp)
	require.NoError(t, err)

	for _, size := range []int{0, 1, 16, 1023, 1024, 65536, 1024 * 1024} {
		size := size
		t.Run("", func(t *testing.T) {
			plaintext := make([]byte, size)
			_, _ = io.ReadFull(rand.Reader, plaintext)

			blob, err := enc.Encrypt(plaintext, []byte("cfg.xml"))
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(blob), 8)
			assert.Equal(t, []byte(encMagic), blob[:4])
			assert.Equal(t, encVersion, blob[4])
			assert.Equal(t, encAlgoChaCha20, blob[5])
			// ChaCha20-Poly1305 envelope shape mirrors GCM exactly:
			// 100-byte overhead per file (header 8 + outer 12 + len 4 +
			// wrapped 48 + inner 12 + tag 16).
			assert.Equal(t, len(plaintext)+100, len(blob))

			recovered, err := enc.Decrypt(blob, []byte("cfg.xml"))
			require.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, recovered),
				"size=%d round-trip mismatch", size)
		})
	}
}

func TestChaCha20Poly1305_TamperDetected(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("ChaCha20-Poly1305", kp)

	blob, err := enc.Encrypt([]byte("sensitive"), []byte("cfg.xml"))
	require.NoError(t, err)

	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[len(tampered)-5] ^= 0x01

	_, err = enc.Decrypt(tampered, []byte("cfg.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

func TestChaCha20Poly1305_WrongAAD(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("ChaCha20-Poly1305", kp)

	blob, err := enc.Encrypt([]byte("hello"), []byte("filename-A.xml"))
	require.NoError(t, err)

	_, err = enc.Decrypt(blob, []byte("filename-B.xml"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

func TestChaCha20Poly1305_WrongKey(t *testing.T) {
	kpA := newStaticKeyProvider(makeTestKey(t))
	kpB := newStaticKeyProvider(makeTestKey(t))
	encA, _ := NewEncryptor("ChaCha20-Poly1305", kpA)
	encB, _ := NewEncryptor("ChaCha20-Poly1305", kpB)

	blob, err := encA.Encrypt([]byte("msg"), nil)
	require.NoError(t, err)

	_, err = encB.Decrypt(blob, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
}

// TestNonceUniqueness_AllAlgos — PRD §4 V7: 500-iteration nonce uniqueness
// for all three algorithms. Catches catastrophic deterministic-encryption
// regressions in any of the three implementations.
func TestNonceUniqueness_AllAlgos(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	for _, algo := range []string{"AES-256-GCM", "AES-256-CBC", "ChaCha20-Poly1305"} {
		algo := algo
		t.Run(algo, func(t *testing.T) {
			enc, err := NewEncryptor(algo, kp)
			require.NoError(t, err)
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
		})
	}
}

// TestPKCS7_PadUnpad_RoundTrip exercises padding correctness for empty,
// 1-byte, exactly-block-sized, and odd-sized inputs.
func TestPKCS7_PadUnpad_RoundTrip(t *testing.T) {
	cases := []int{0, 1, 5, 15, 16, 17, 31, 32, 33}
	for _, n := range cases {
		n := n
		t.Run("", func(t *testing.T) {
			data := make([]byte, n)
			_, _ = io.ReadFull(rand.Reader, data)
			padded := pkcs7Pad(data, encCBCBlockSize)
			require.Equal(t, 0, len(padded)%encCBCBlockSize, "padded must be block-aligned")
			require.GreaterOrEqual(t, len(padded), len(data)+1, "PKCS7 always adds ≥1 byte")

			unpadded, err := pkcs7Unpad(padded, encCBCBlockSize)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(data, unpadded))
		})
	}
}

// TestPKCS7_Unpad_RejectsMalformed — exhaustive negative cases catch
// padding-oracle prerequisites: bad pad value, non-uniform padding,
// non-aligned length, empty input.
func TestPKCS7_Unpad_RejectsMalformed(t *testing.T) {
	cases := [][]byte{
		nil,                            // empty
		make([]byte, 7),                // not aligned (and pad byte = 0)
		append(make([]byte, 15), 0x00), // pad value 0
		append(make([]byte, 15), 0x11), // pad value > blockSize
		append(append(make([]byte, 14), 0x05), 0x05), // 0x05 says "5 bytes pad" but only last 1 matches
	}
	for i, c := range cases {
		_, err := pkcs7Unpad(c, encCBCBlockSize)
		require.Error(t, err, "case %d should reject", i)
		assert.True(t, errors.Is(err, ErrEncryptionFormatInvalid))
	}
}

func TestAESGCM_FormatInvalid(t *testing.T) {
	kp := newStaticKeyProvider(makeTestKey(t))
	enc, _ := NewEncryptor("AES-256-GCM", kp)

	cases := [][]byte{
		nil,              // nil
		make([]byte, 50), // too short
		append([]byte("WRONG"), make([]byte, 200)...),                           // bad magic
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
