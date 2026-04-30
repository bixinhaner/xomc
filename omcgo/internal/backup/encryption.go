// Package backup — backup encryption (T-0075 / R-102 final core capability;
// T-0085 closes the algorithm matrix).
//
// Envelope encryption: a per-file Data-Encryption-Key (DEK, 32B random) is
// generated, used to encrypt the plaintext under one of three supported
// AEAD-equivalent constructions, and itself wrapped under the
// deployment-wide KEK using AES-256-GCM. The wrapped DEK and the
// algorithm-specific framing ride alongside the ciphertext in a single
// custom file format (see `encMagic` and the layout comments below).
//
// Supported algorithms (algo byte in header):
//   - 'G' AES-256-GCM            — T-0075 (AEAD; reference implementation)
//   - 'C' AES-256-CBC + HMAC-SHA256 (encrypt-then-MAC) — T-0085
//   - 'P' ChaCha20-Poly1305       — T-0085 (AEAD; ARM-friendly)
//
// Design choices (PRD §2):
//   - AAD = filename binds ciphertext to its expected container (anti-swap).
//     For CBC the AAD is folded into the HMAC scope; for AEADs it's the
//     standard authenticated-data slot.
//   - Buffer-then-encrypt with 64MB ceiling — backups are <10MB realistic;
//     streaming chunked formats are deferred.
//   - File format starts with magic "OENC" so download handler can detect
//     encrypted objects even without `.enc` extension.
//   - KEK→DEK wrapping is fixed at AES-256-GCM regardless of the body
//     algorithm (PRD §2.5): KEK is a 32B opaque key, wrapped_DEK has
//     constant 48B length, and the wrapping primitive choice does not
//     affect interop with the body cipher.
//
// Security (T-0085 specifics):
//   - CBC mode requires encrypt-then-MAC (PRD §2.1) to defeat
//     padding-oracle attacks. Decrypt verifies the HMAC FIRST in
//     constant time before unpadding.
//   - The HMAC key is derived from the DEK via HKDF-SHA256 with
//     domain-separation info "backup-cbc-mac" (PRD §2.2), so the
//     wrapped-DEK length stays at 48B.
//   - HMAC scope = algo_byte ‖ IV ‖ AAD ‖ ciphertext, which binds the
//     algorithm tag, the IV, and the AAD to the ciphertext —
//     defeating cross-algorithm or cross-file substitution.
//   - ChaCha20-Poly1305 is a standards-track AEAD with the same shape
//     as GCM; the only envelope difference vs GCM is the algo byte.
//
// Security (carried over from T-0075):
//   - Nonces / IVs / DEKs from crypto/rand (Linux: /dev/urandom).
//   - AEAD tag verification is constant-time (Go stdlib for GCM and
//     chacha20poly1305; we use crypto/hmac.Equal for the CBC HMAC).
//   - Key bytes never appear in error messages or logs.
//   - PKCS7 unpad is bounds-checked and rejects malformed padding before
//     the unpadded slice is exposed to callers (post-HMAC-OK path only).
package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// Sentinel errors callers compare via errors.Is.
var (
	ErrEncryptionKeyUnavailable          = errors.New("backup encryption key not configured")
	ErrEncryptionAlgorithmInvalid        = errors.New("backup encryption algorithm must be one of AES-256-GCM|AES-256-CBC|ChaCha20-Poly1305")
	ErrEncryptionAlgorithmNotImplemented = errors.New("backup encryption algorithm not implemented in current build")
	ErrEncryptionInputTooLarge           = errors.New("backup encryption input exceeds buffer ceiling")
	ErrEncryptionFormatInvalid           = errors.New("backup encrypted file format invalid")
	ErrEncryptionAuthFailed              = errors.New("backup encryption auth tag verification failed (tampered, wrong key, or wrong AAD)")
)

// File-format constants. Layout (all multi-byte fields little-endian):
//
//	[magic "OENC" 4B][version 1B][algo 1B][reserved 2B 0x0000]   ← header 8B
//	[outer_nonce 12B]                                            ← KEK→DEK GCM nonce
//	[wrapped_DEK_len 4B u32]
//	[wrapped_DEK ?B (= 32B DEK + 16B GCM tag = 48B)]
//	[inner_nonce 12B]                                            ← DEK→data GCM nonce
//	[ciphertext + 16B GCM tag]                                   ← AAD bound
//
// Header overhead = 100 bytes per encrypted file (8 header + 12 outer + 4
// length + 48 wrapped + 12 inner + 16 tag).
const (
	encMagic          = "OENC"
	encVersion        = byte(0x01)
	encAlgoGCM        = byte('G')
	encAlgoCBC        = byte('C')                  // T-0085: AES-256-CBC + HMAC-SHA256
	encAlgoChaCha20   = byte('P')                  // T-0085: ChaCha20-Poly1305 (Poly1305 → 'P')
	encNonceSize      = 12                         // GCM / ChaCha20-Poly1305 standard
	encDEKSize        = 32                         // AES-256
	encGCMTagSize     = 16                         // GCM standard
	encWrappedDEKSize = encDEKSize + encGCMTagSize // 48

	// T-0085 CBC framing
	encCBCBlockSize = 16 // AES block size (also IV size)
	encHMACSize     = 32 // HMAC-SHA256 output

	// T-0085 ChaCha20-Poly1305 framing
	encChaChaTagSize = chacha20poly1305.Overhead // 16
	// chacha20poly1305.NonceSize is 12 — same as encNonceSize, so no separate const.

	encMaxPlaintext = 64 * 1024 * 1024 // 64 MB cap, T-0075 PRD §2.3
)

// hkdfMACInfoCBC binds the HKDF-derived HMAC key to the AES-CBC backup-mac
// purpose so accidental reuse with another HKDF caller cannot equate keys.
// Changing this string invalidates all previously-encrypted CBC backups,
// so it MUST stay constant for the lifetime of the OENC v1 envelope.
const hkdfMACInfoCBC = "backup-cbc-mac"

// Encryptor wraps plaintext bytes and produces the on-disk encrypted blob.
// Decrypt is the symmetric reverse.
type Encryptor interface {
	// Encrypt reads up to encMaxPlaintext bytes from plaintext and returns
	// the encrypted blob. aad participates in GCM auth — the same aad must
	// be passed to Decrypt or auth verification fails.
	Encrypt(plaintext, aad []byte) ([]byte, error)
	// Decrypt reverses Encrypt. Returns ErrEncryptionAuthFailed when the
	// stored ciphertext / tag / AAD combination does not verify (tamper,
	// wrong key, wrong AAD, or substitution).
	Decrypt(blob, aad []byte) ([]byte, error)
	// Format returns the canonical algorithm name ("aes-256-gcm").
	Format() string
	// Extension returns the file-name suffix appended after encryption
	// ("enc" without dot — combined as ".enc" by callers).
	Extension() string
}

// NewEncryptor returns the Encryptor matching algorithm. algorithm must be
// one of AES-256-GCM|AES-256-CBC|ChaCha20-Poly1305. T-0085 closes the matrix —
// no algorithm in this set returns ErrEncryptionAlgorithmNotImplemented.
//
// kp must be non-nil and Available() must return true at construction time;
// otherwise ErrEncryptionKeyUnavailable.
func NewEncryptor(algorithm string, kp KeyProvider) (Encryptor, error) {
	if kp == nil || !kp.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	switch algorithm {
	case "AES-256-GCM":
		return &aesGCMEncryptor{kp: kp}, nil
	case "AES-256-CBC":
		return &aesCBCEncryptor{kp: kp}, nil
	case "ChaCha20-Poly1305":
		return &chaCha20Encryptor{kp: kp}, nil
	default:
		return nil, fmt.Errorf("algorithm=%q: %w", algorithm, ErrEncryptionAlgorithmInvalid)
	}
}

// wrapDEKWithKEK seals dek under kek using AES-256-GCM, returning the
// wrapped-DEK byte slice that goes on disk. outerNonce must be 12B random.
// Used by all three body algorithms — KEK→DEK wrap is fixed (PRD §2.5).
func wrapDEKWithKEK(kek, outerNonce, dek []byte) ([]byte, error) {
	kekBlock, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (KEK): %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (KEK): %w", err)
	}
	return kekGCM.Seal(nil, outerNonce, dek, nil), nil
}

// unwrapDEKWithKEK reverses wrapDEKWithKEK. A failure here is reported as
// ErrEncryptionAuthFailed (wrong KEK or tampered wrapped_DEK) — never the
// underlying GCM open error, to avoid leaking timing differentiation
// between "key wrong" and "key right but body tampered".
func unwrapDEKWithKEK(kek, outerNonce, wrappedDEK []byte) ([]byte, error) {
	kekBlock, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (KEK): %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (KEK): %w", err)
	}
	dek, err := kekGCM.Open(nil, outerNonce, wrappedDEK, nil)
	if err != nil {
		return nil, fmt.Errorf("unwrap DEK: %w", ErrEncryptionAuthFailed)
	}
	if len(dek) != encDEKSize {
		return nil, fmt.Errorf("unexpected DEK size %d: %w", len(dek), ErrEncryptionFormatInvalid)
	}
	return dek, nil
}

// pkcs7Pad appends PKCS7 padding so output is a multiple of blockSize.
// PKCS7 always adds at least one padding byte; an input whose length is
// already a multiple of blockSize gets a full extra block. This is
// required to keep unpadding unambiguous.
func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

// pkcs7Unpad strips PKCS7 padding, validating every padding byte.
// Returns ErrEncryptionFormatInvalid on any structural issue. This runs
// AFTER HMAC verification has succeeded, so it is not a side-channel
// concern; nevertheless we check the entire padding block (constant work
// vs the maximum padding length) rather than short-circuit, matching the
// CBC-mode best practice.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	n := len(data)
	if n == 0 || n%blockSize != 0 {
		return nil, fmt.Errorf("pkcs7 unpad: bad length %d: %w", n, ErrEncryptionFormatInvalid)
	}
	pad := int(data[n-1])
	if pad == 0 || pad > blockSize {
		return nil, fmt.Errorf("pkcs7 unpad: bad pad value %d: %w", pad, ErrEncryptionFormatInvalid)
	}
	for i := n - pad; i < n; i++ {
		if int(data[i]) != pad {
			return nil, fmt.Errorf("pkcs7 unpad: non-uniform padding: %w", ErrEncryptionFormatInvalid)
		}
	}
	return data[:n-pad], nil
}

// deriveCBCMACKey extracts a 32B HMAC-SHA256 key from the DEK using HKDF
// with domain-separation info hkdfMACInfoCBC. Salt is empty per RFC 5869
// §3.1: when the IKM (DEK) is already a uniformly-random 32B key, salt is
// optional. The info string carries the cross-purpose binding instead.
func deriveCBCMACKey(dek []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, dek, nil, []byte(hkdfMACInfoCBC))
	out := make([]byte, encHMACSize)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("hkdf derive cbc-mac key: %w", err)
	}
	return out, nil
}

// computeCBCMAC computes HMAC-SHA256 over (algo_byte ‖ IV ‖ AAD ‖
// ciphertext). Domain components in this exact order:
//
//   - algo_byte (1B): pins the MAC to the CBC algorithm so cross-algorithm
//     substitution (e.g. swapping the algo byte to 'G') is detected.
//   - IV (16B): prevents IV-substitution attacks where an attacker swaps
//     IV blocks to force flips in the first plaintext block.
//   - AAD: PRD §2.1 explicit; mirrors GCM/Poly1305 AAD semantics so the
//     CBC path enforces the same anti-swap binding (filename ↔ ciphertext).
//   - ciphertext: tampering or substitution detected.
//
// macKey is the HKDF-derived 32B MAC key (NOT the DEK directly).
func computeCBCMAC(macKey, iv, aad, ciphertext []byte) []byte {
	mac := hmac.New(sha256.New, macKey)
	mac.Write([]byte{encAlgoCBC})
	mac.Write(iv)
	mac.Write(aad)
	mac.Write(ciphertext)
	return mac.Sum(nil)
}

// aesGCMEncryptor implements envelope encryption with AES-256-GCM.
type aesGCMEncryptor struct {
	kp KeyProvider
}

func (e *aesGCMEncryptor) Format() string    { return "aes-256-gcm" }
func (e *aesGCMEncryptor) Extension() string { return "enc" }

func (e *aesGCMEncryptor) Encrypt(plaintext, aad []byte) ([]byte, error) {
	if len(plaintext) > encMaxPlaintext {
		return nil, fmt.Errorf("plaintext %d > max %d: %w",
			len(plaintext), encMaxPlaintext, ErrEncryptionInputTooLarge)
	}
	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}

	// Generate per-file DEK (32 bytes) + outer nonce (12 bytes).
	dek := make([]byte, encDEKSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}
	outerNonce := make([]byte, encNonceSize)
	if _, err := io.ReadFull(rand.Reader, outerNonce); err != nil {
		return nil, fmt.Errorf("generate outer nonce: %w", err)
	}

	// Wrap DEK under KEK.
	kekBlock, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (KEK): %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (KEK): %w", err)
	}
	wrappedDEK := kekGCM.Seal(nil, outerNonce, dek, nil)

	// Encrypt plaintext under DEK with AAD.
	innerNonce := make([]byte, encNonceSize)
	if _, err := io.ReadFull(rand.Reader, innerNonce); err != nil {
		return nil, fmt.Errorf("generate inner nonce: %w", err)
	}
	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (DEK): %w", err)
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (DEK): %w", err)
	}
	ciphertext := dekGCM.Seal(nil, innerNonce, plaintext, aad)

	// Assemble: header(8) + outer_nonce(12) + len(4) + wrapped(48) + inner_nonce(12) + ciphertext.
	totalLen := 8 + encNonceSize + 4 + len(wrappedDEK) + encNonceSize + len(ciphertext)
	out := make([]byte, 0, totalLen)
	out = append(out, encMagic...)
	out = append(out, encVersion, encAlgoGCM, 0x00, 0x00)
	out = append(out, outerNonce...)
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(wrappedDEK)))
	out = append(out, lenBuf[:]...)
	out = append(out, wrappedDEK...)
	out = append(out, innerNonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (e *aesGCMEncryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	// Minimum size: header(8) + outer_nonce(12) + len(4) + wrapped(48) + inner_nonce(12) + tag(16) = 100
	const minSize = 8 + encNonceSize + 4 + encWrappedDEKSize + encNonceSize + encGCMTagSize
	if len(blob) < minSize {
		return nil, fmt.Errorf("blob too short (%d < %d): %w", len(blob), minSize, ErrEncryptionFormatInvalid)
	}

	// Parse header.
	if string(blob[0:4]) != encMagic {
		return nil, fmt.Errorf("magic mismatch: %w", ErrEncryptionFormatInvalid)
	}
	if blob[4] != encVersion {
		return nil, fmt.Errorf("version=%d unsupported: %w", blob[4], ErrEncryptionFormatInvalid)
	}
	if blob[5] != encAlgoGCM {
		return nil, fmt.Errorf("algo=%c unsupported: %w", blob[5], ErrEncryptionAlgorithmNotImplemented)
	}
	// reserved bytes [6:8] ignored
	off := 8

	outerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	wrappedLen := int(binary.LittleEndian.Uint32(blob[off : off+4]))
	off += 4
	if wrappedLen != encWrappedDEKSize {
		return nil, fmt.Errorf("wrapped_DEK_len=%d expected %d: %w",
			wrappedLen, encWrappedDEKSize, ErrEncryptionFormatInvalid)
	}
	if off+wrappedLen+encNonceSize+encGCMTagSize > len(blob) {
		return nil, fmt.Errorf("truncated blob: %w", ErrEncryptionFormatInvalid)
	}
	wrappedDEK := blob[off : off+wrappedLen]
	off += wrappedLen

	innerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	ciphertext := blob[off:]

	// Unwrap DEK.
	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}
	kekBlock, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (KEK): %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (KEK): %w", err)
	}
	dek, err := kekGCM.Open(nil, outerNonce, wrappedDEK, nil)
	if err != nil {
		// Wrong KEK or tampered wrapped_DEK.
		return nil, fmt.Errorf("unwrap DEK: %w", ErrEncryptionAuthFailed)
	}
	if len(dek) != encDEKSize {
		return nil, fmt.Errorf("unexpected DEK size %d: %w", len(dek), ErrEncryptionFormatInvalid)
	}

	// Decrypt body.
	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (DEK): %w", err)
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, fmt.Errorf("aead (DEK): %w", err)
	}
	plaintext, err := dekGCM.Open(nil, innerNonce, ciphertext, aad)
	if err != nil {
		// Tampered ciphertext, wrong DEK, or wrong AAD.
		return nil, fmt.Errorf("decrypt body: %w", ErrEncryptionAuthFailed)
	}
	return plaintext, nil
}

// =============================================================================
// AES-256-CBC + HMAC-SHA256 (encrypt-then-MAC) — T-0085
// =============================================================================
//
// CBC alone is unsafe (padding-oracle). The encrypt-then-MAC construction
// here is the standards-compliant safe form: HMAC verification gates
// padding removal, so the decrypt path never reveals padding-validity
// timing differences for attacker-controlled ciphertexts.
//
// On-disk layout (CBC):
//
//	[magic "OENC" 4B][version 1B][algo='C' 1B][reserved 2B 0x0000]
//	[outer_nonce 12B]                                              ← KEK→DEK GCM nonce
//	[wrapped_DEK_len 4B u32][wrapped_DEK 48B]
//	[iv 16B]                                                       ← AES-CBC IV
//	[ciphertext (PKCS7 padded, multiple of 16B)]
//	[hmac 32B]                                                     ← HMAC-SHA256(algo ‖ IV ‖ AAD ‖ ct)
//
// Header overhead before ciphertext = 8 + 12 + 4 + 48 + 16 = 88 B.
// Trailing HMAC-SHA256 = 32 B. PKCS7 padding adds 1..16 B.
// Total overhead lower bound = 88 + 32 + 1 = 121 B (vs GCM's 100 B).

// aesCBCEncryptor implements envelope encryption with AES-256-CBC body and
// HMAC-SHA256 over the framing + ciphertext + AAD.
type aesCBCEncryptor struct {
	kp KeyProvider
}

func (e *aesCBCEncryptor) Format() string    { return "aes-256-cbc" }
func (e *aesCBCEncryptor) Extension() string { return "enc" }

func (e *aesCBCEncryptor) Encrypt(plaintext, aad []byte) ([]byte, error) {
	if len(plaintext) > encMaxPlaintext {
		return nil, fmt.Errorf("plaintext %d > max %d: %w",
			len(plaintext), encMaxPlaintext, ErrEncryptionInputTooLarge)
	}
	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}

	dek := make([]byte, encDEKSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}
	outerNonce := make([]byte, encNonceSize)
	if _, err := io.ReadFull(rand.Reader, outerNonce); err != nil {
		return nil, fmt.Errorf("generate outer nonce: %w", err)
	}
	wrappedDEK, err := wrapDEKWithKEK(kek, outerNonce, dek)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, encCBCBlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("generate cbc iv: %w", err)
	}

	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (DEK): %w", err)
	}
	padded := pkcs7Pad(plaintext, encCBCBlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(dekBlock, iv).CryptBlocks(ciphertext, padded)

	macKey, err := deriveCBCMACKey(dek)
	if err != nil {
		return nil, err
	}
	mac := computeCBCMAC(macKey, iv, aad, ciphertext)

	totalLen := 8 + encNonceSize + 4 + len(wrappedDEK) + encCBCBlockSize + len(ciphertext) + encHMACSize
	out := make([]byte, 0, totalLen)
	out = append(out, encMagic...)
	out = append(out, encVersion, encAlgoCBC, 0x00, 0x00)
	out = append(out, outerNonce...)
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(wrappedDEK)))
	out = append(out, lenBuf[:]...)
	out = append(out, wrappedDEK...)
	out = append(out, iv...)
	out = append(out, ciphertext...)
	out = append(out, mac...)
	return out, nil
}

func (e *aesCBCEncryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	// Minimum size = 8 + 12 + 4 + 48 + 16 + 16 (one block ciphertext) + 32 = 136
	const minSize = 8 + encNonceSize + 4 + encWrappedDEKSize + encCBCBlockSize + encCBCBlockSize + encHMACSize
	if len(blob) < minSize {
		return nil, fmt.Errorf("blob too short (%d < %d): %w", len(blob), minSize, ErrEncryptionFormatInvalid)
	}
	if string(blob[0:4]) != encMagic {
		return nil, fmt.Errorf("magic mismatch: %w", ErrEncryptionFormatInvalid)
	}
	if blob[4] != encVersion {
		return nil, fmt.Errorf("version=%d unsupported: %w", blob[4], ErrEncryptionFormatInvalid)
	}
	if blob[5] != encAlgoCBC {
		return nil, fmt.Errorf("algo=%c unsupported by aes-256-cbc decryptor: %w", blob[5], ErrEncryptionAlgorithmNotImplemented)
	}
	off := 8
	outerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	wrappedLen := int(binary.LittleEndian.Uint32(blob[off : off+4]))
	off += 4
	if wrappedLen != encWrappedDEKSize {
		return nil, fmt.Errorf("wrapped_DEK_len=%d expected %d: %w",
			wrappedLen, encWrappedDEKSize, ErrEncryptionFormatInvalid)
	}
	if off+wrappedLen+encCBCBlockSize+encHMACSize > len(blob) {
		return nil, fmt.Errorf("truncated blob: %w", ErrEncryptionFormatInvalid)
	}
	wrappedDEK := blob[off : off+wrappedLen]
	off += wrappedLen

	iv := blob[off : off+encCBCBlockSize]
	off += encCBCBlockSize

	// Trailing HMAC-SHA256 occupies the last 32B; ciphertext is everything
	// in between. Reject lengths that would produce an empty ciphertext or
	// a ciphertext not aligned to the block size.
	macStart := len(blob) - encHMACSize
	if macStart <= off {
		return nil, fmt.Errorf("missing ciphertext slot: %w", ErrEncryptionFormatInvalid)
	}
	ciphertext := blob[off:macStart]
	storedMAC := blob[macStart:]
	if len(ciphertext) == 0 || len(ciphertext)%encCBCBlockSize != 0 {
		return nil, fmt.Errorf("ciphertext len=%d not block-aligned: %w", len(ciphertext), ErrEncryptionFormatInvalid)
	}

	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}
	dek, err := unwrapDEKWithKEK(kek, outerNonce, wrappedDEK)
	if err != nil {
		return nil, err
	}
	macKey, err := deriveCBCMACKey(dek)
	if err != nil {
		return nil, err
	}

	expectedMAC := computeCBCMAC(macKey, iv, aad, ciphertext)
	// CRITICAL: HMAC verification in constant time, BEFORE any block
	// decryption or padding inspection. This is the entire reason CBC is
	// safe in this construction.
	if !hmac.Equal(expectedMAC, storedMAC) {
		return nil, fmt.Errorf("cbc hmac verify: %w", ErrEncryptionAuthFailed)
	}

	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher (DEK): %w", err)
	}
	padded := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(dekBlock, iv).CryptBlocks(padded, ciphertext)

	plaintext, err := pkcs7Unpad(padded, encCBCBlockSize)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// =============================================================================
// ChaCha20-Poly1305 — T-0085
// =============================================================================
//
// AEAD with the same envelope shape as AES-256-GCM; the only difference is
// the algo byte ('P' for Poly1305) and the underlying primitive. Useful on
// hosts without AES-NI hardware acceleration (typical ARM small-cell CPU).
//
// On-disk layout (ChaCha20-Poly1305):
//
//	[magic "OENC" 4B][version 1B][algo='P' 1B][reserved 2B 0x0000]
//	[outer_nonce 12B]                              ← KEK→DEK GCM nonce
//	[wrapped_DEK_len 4B u32][wrapped_DEK 48B]
//	[inner_nonce 12B]                              ← ChaCha20-Poly1305 nonce
//	[ciphertext + 16B Poly1305 tag]                ← AAD bound
//
// Header + framing overhead = 100B (identical to GCM).

// chaCha20Encryptor implements envelope encryption with ChaCha20-Poly1305
// for the body, sharing the AES-256-GCM KEK→DEK wrap with the GCM impl.
type chaCha20Encryptor struct {
	kp KeyProvider
}

func (e *chaCha20Encryptor) Format() string    { return "chacha20-poly1305" }
func (e *chaCha20Encryptor) Extension() string { return "enc" }

func (e *chaCha20Encryptor) Encrypt(plaintext, aad []byte) ([]byte, error) {
	if len(plaintext) > encMaxPlaintext {
		return nil, fmt.Errorf("plaintext %d > max %d: %w",
			len(plaintext), encMaxPlaintext, ErrEncryptionInputTooLarge)
	}
	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}

	dek := make([]byte, encDEKSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}
	outerNonce := make([]byte, encNonceSize)
	if _, err := io.ReadFull(rand.Reader, outerNonce); err != nil {
		return nil, fmt.Errorf("generate outer nonce: %w", err)
	}
	wrappedDEK, err := wrapDEKWithKEK(kek, outerNonce, dek)
	if err != nil {
		return nil, err
	}

	innerNonce := make([]byte, encNonceSize)
	if _, err := io.ReadFull(rand.Reader, innerNonce); err != nil {
		return nil, fmt.Errorf("generate inner nonce: %w", err)
	}
	aead, err := chacha20poly1305.New(dek)
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305 init: %w", err)
	}
	ciphertext := aead.Seal(nil, innerNonce, plaintext, aad)

	totalLen := 8 + encNonceSize + 4 + len(wrappedDEK) + encNonceSize + len(ciphertext)
	out := make([]byte, 0, totalLen)
	out = append(out, encMagic...)
	out = append(out, encVersion, encAlgoChaCha20, 0x00, 0x00)
	out = append(out, outerNonce...)
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(wrappedDEK)))
	out = append(out, lenBuf[:]...)
	out = append(out, wrappedDEK...)
	out = append(out, innerNonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (e *chaCha20Encryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	const minSize = 8 + encNonceSize + 4 + encWrappedDEKSize + encNonceSize + encChaChaTagSize
	if len(blob) < minSize {
		return nil, fmt.Errorf("blob too short (%d < %d): %w", len(blob), minSize, ErrEncryptionFormatInvalid)
	}
	if string(blob[0:4]) != encMagic {
		return nil, fmt.Errorf("magic mismatch: %w", ErrEncryptionFormatInvalid)
	}
	if blob[4] != encVersion {
		return nil, fmt.Errorf("version=%d unsupported: %w", blob[4], ErrEncryptionFormatInvalid)
	}
	if blob[5] != encAlgoChaCha20 {
		return nil, fmt.Errorf("algo=%c unsupported by chacha20-poly1305 decryptor: %w", blob[5], ErrEncryptionAlgorithmNotImplemented)
	}
	off := 8
	outerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	wrappedLen := int(binary.LittleEndian.Uint32(blob[off : off+4]))
	off += 4
	if wrappedLen != encWrappedDEKSize {
		return nil, fmt.Errorf("wrapped_DEK_len=%d expected %d: %w",
			wrappedLen, encWrappedDEKSize, ErrEncryptionFormatInvalid)
	}
	if off+wrappedLen+encNonceSize+encChaChaTagSize > len(blob) {
		return nil, fmt.Errorf("truncated blob: %w", ErrEncryptionFormatInvalid)
	}
	wrappedDEK := blob[off : off+wrappedLen]
	off += wrappedLen

	innerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	ciphertext := blob[off:]

	kek, err := e.kp.KEK(nil)
	if err != nil {
		return nil, fmt.Errorf("get KEK: %w", err)
	}
	dek, err := unwrapDEKWithKEK(kek, outerNonce, wrappedDEK)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.New(dek)
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305 init: %w", err)
	}
	plaintext, err := aead.Open(nil, innerNonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt body: %w", ErrEncryptionAuthFailed)
	}
	return plaintext, nil
}
