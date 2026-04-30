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
// OENC v1 (T-0075 / T-0085):
//
//	[magic "OENC" 4B][version=1 1B][algo 1B][reserved 2B 0x0000]   ← header 8B
//	[outer_nonce 12B]                                              ← KEK→DEK GCM nonce
//	[wrapped_DEK_len 4B u32]
//	[wrapped_DEK 48B]
//	[body algo-specific]                                           ← see Encrypt
//
// OENC v2 (T-0087, current writer):
//
//	[magic "OENC" 4B][version=2 1B][algo 1B][kek_id_len 1B][reserved 1B 0x00]   ← header 8B
//	[kek_id NB (0..255)]                                           ← envelope→KEK lookup
//	[outer_nonce 12B]
//	[wrapped_DEK_len 4B u32]
//	[wrapped_DEK 48B]
//	[body algo-specific]
//
// kek_id is NOT in the AAD/HMAC authenticated scope (PRD §2.3): an attacker
// who flips kek_id to a non-existent ID is rejected at KEKByID lookup; an
// attacker who flips to a known wrong ID still fails at unwrap-DEK GCM
// verification because the wrapped_DEK is bound to the original KEK.
//
// Header overhead before body:
//   - v1 = 8 + 12 + 4 + 48 = 72B
//   - v2 = 8 + N + 12 + 4 + 48 = 72 + N (= 72 for empty kek_id; same as v1)
const (
	encMagic          = "OENC"
	encVersion        = byte(0x01) // legacy reader-only support (T-0075/T-0085 written)
	encVersion2       = byte(0x02) // T-0087: writes carry kek_id field
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

// envelopeHeader holds the parsed-and-validated common prefix shared by
// all three OENC body variants. Produced by parseEnvelopeHeader and
// consumed by each algo's Decrypt to skip past the variable-length kek_id
// (v2) and locate body fields (outer_nonce / wrapped_DEK / body).
type envelopeHeader struct {
	algo       byte
	kekID      string
	outerNonce []byte
	wrappedDEK []byte
	bodyOffset int // byte index where algo-specific body begins
}

// minEnvelopePrefix is the smallest common prefix that always exists in
// any OENC envelope (header 8 + outer_nonce 12 + wrapped_DEK_len 4 +
// wrapped_DEK 48 = 72), excluding the algo-specific body. v2 envelopes
// add kek_id between the header and outer_nonce (0..255 bytes).
const minEnvelopePrefix = 8 + encNonceSize + 4 + encWrappedDEKSize

// parseEnvelopeHeader validates magic / version / algo / kek_id / wrapped
// DEK size and returns the parsed envelopeHeader plus the offset where
// the algo-specific body begins. Accepts both v1 (legacy) and v2
// envelopes; the only behavioural difference is whether a kek_id_len
// byte and kek_id field are present.
//
// Errors are wrapped under ErrEncryptionFormatInvalid for caller-uniform
// handling. The caller is expected to perform algo-specific body checks
// (e.g., CBC HMAC trailing 32B, GCM/ChaCha tag overhead) starting at
// envelopeHeader.bodyOffset.
func parseEnvelopeHeader(blob []byte) (*envelopeHeader, error) {
	if len(blob) < minEnvelopePrefix {
		return nil, fmt.Errorf("blob too short (%d < %d): %w",
			len(blob), minEnvelopePrefix, ErrEncryptionFormatInvalid)
	}
	if string(blob[0:4]) != encMagic {
		return nil, fmt.Errorf("magic mismatch: %w", ErrEncryptionFormatInvalid)
	}
	ver := blob[4]
	algo := blob[5]

	var kekID string
	off := 8
	switch ver {
	case encVersion:
		// v1: bytes 6-7 must be reserved 0x0000. Older code wrote 0x00,
		// 0x00 explicitly; reject if not (defends against attacker
		// repurposing the byte as a hidden kek_id_len).
		if blob[6] != 0x00 || blob[7] != 0x00 {
			return nil, fmt.Errorf("v1 reserved bytes nonzero (%#x, %#x): %w",
				blob[6], blob[7], ErrEncryptionFormatInvalid)
		}
	case encVersion2:
		// v2: byte 6 = kek_id_len, byte 7 reserved.
		if blob[7] != 0x00 {
			return nil, fmt.Errorf("v2 reserved byte nonzero (%#x): %w",
				blob[7], ErrEncryptionFormatInvalid)
		}
		kekIDLen := int(blob[6])
		if 8+kekIDLen+encNonceSize+4+encWrappedDEKSize > len(blob) {
			return nil, fmt.Errorf("truncated kek_id (len=%d): %w",
				kekIDLen, ErrEncryptionFormatInvalid)
		}
		kekID = string(blob[8 : 8+kekIDLen])
		off = 8 + kekIDLen
	default:
		return nil, fmt.Errorf("version=%d unsupported: %w", ver, ErrEncryptionFormatInvalid)
	}

	outerNonce := blob[off : off+encNonceSize]
	off += encNonceSize

	wrappedLen := int(binary.LittleEndian.Uint32(blob[off : off+4]))
	off += 4
	if wrappedLen != encWrappedDEKSize {
		return nil, fmt.Errorf("wrapped_DEK_len=%d expected %d: %w",
			wrappedLen, encWrappedDEKSize, ErrEncryptionFormatInvalid)
	}
	if off+wrappedLen > len(blob) {
		return nil, fmt.Errorf("truncated wrapped_DEK: %w", ErrEncryptionFormatInvalid)
	}
	wrappedDEK := blob[off : off+wrappedLen]
	off += wrappedLen

	return &envelopeHeader{
		algo:       algo,
		kekID:      kekID,
		outerNonce: outerNonce,
		wrappedDEK: wrappedDEK,
		bodyOffset: off,
	}, nil
}

// encodeEnvelopeHeader writes the v2 envelope header + kek_id +
// outer_nonce + wrapped_DEK_len + wrapped_DEK into a fresh slice. The
// returned slice has cap pre-sized to bodyHint to avoid grows when
// callers append the algo-specific body.
//
// kekID is taken verbatim from KeyProvider.ActiveKEKID(); validation
// (max length 255) is enforced via panic-free truncation rejection.
func encodeEnvelopeHeader(algo byte, kekID string, outerNonce, wrappedDEK []byte, bodyHint int) ([]byte, error) {
	if len(kekID) > maxKEKIDLen {
		return nil, fmt.Errorf("kek_id len=%d > %d: %w",
			len(kekID), maxKEKIDLen, ErrEncryptionFormatInvalid)
	}
	headerLen := 8 + len(kekID) + encNonceSize + 4 + len(wrappedDEK)
	out := make([]byte, 0, headerLen+bodyHint)
	out = append(out, encMagic...)
	out = append(out, encVersion2, algo, byte(len(kekID)), 0x00)
	out = append(out, kekID...)
	out = append(out, outerNonce...)
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(wrappedDEK)))
	out = append(out, lenBuf[:]...)
	out = append(out, wrappedDEK...)
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
	kekID := e.kp.ActiveKEKID()
	kek, err := e.kp.KEKByID(nil, kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", kekID, err)
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
	wrappedDEK, err := wrapDEKWithKEK(kek, outerNonce, dek)
	if err != nil {
		return nil, err
	}

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

	// Body = inner_nonce + ciphertext+tag.
	out, err := encodeEnvelopeHeader(encAlgoGCM, kekID, outerNonce, wrappedDEK, encNonceSize+len(ciphertext))
	if err != nil {
		return nil, err
	}
	out = append(out, innerNonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (e *aesGCMEncryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	hdr, err := parseEnvelopeHeader(blob)
	if err != nil {
		return nil, err
	}
	if hdr.algo != encAlgoGCM {
		return nil, fmt.Errorf("algo=%c unsupported by aes-256-gcm decryptor: %w",
			hdr.algo, ErrEncryptionAlgorithmNotImplemented)
	}
	// Body = inner_nonce(12) + ciphertext+tag(>=16).
	if hdr.bodyOffset+encNonceSize+encGCMTagSize > len(blob) {
		return nil, fmt.Errorf("truncated gcm body: %w", ErrEncryptionFormatInvalid)
	}
	innerNonce := blob[hdr.bodyOffset : hdr.bodyOffset+encNonceSize]
	ciphertext := blob[hdr.bodyOffset+encNonceSize:]

	kek, err := e.kp.KEKByID(nil, hdr.kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", hdr.kekID, err)
	}
	dek, err := unwrapDEKWithKEK(kek, hdr.outerNonce, hdr.wrappedDEK)
	if err != nil {
		return nil, err
	}

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
	kekID := e.kp.ActiveKEKID()
	kek, err := e.kp.KEKByID(nil, kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", kekID, err)
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

	bodyHint := encCBCBlockSize + len(ciphertext) + encHMACSize
	out, err := encodeEnvelopeHeader(encAlgoCBC, kekID, outerNonce, wrappedDEK, bodyHint)
	if err != nil {
		return nil, err
	}
	out = append(out, iv...)
	out = append(out, ciphertext...)
	out = append(out, mac...)
	return out, nil
}

func (e *aesCBCEncryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	hdr, err := parseEnvelopeHeader(blob)
	if err != nil {
		return nil, err
	}
	if hdr.algo != encAlgoCBC {
		return nil, fmt.Errorf("algo=%c unsupported by aes-256-cbc decryptor: %w",
			hdr.algo, ErrEncryptionAlgorithmNotImplemented)
	}
	// Body = iv(16) + ciphertext (block-aligned, ≥1 block) + hmac(32).
	if hdr.bodyOffset+encCBCBlockSize+encCBCBlockSize+encHMACSize > len(blob) {
		return nil, fmt.Errorf("truncated cbc body: %w", ErrEncryptionFormatInvalid)
	}
	iv := blob[hdr.bodyOffset : hdr.bodyOffset+encCBCBlockSize]
	macStart := len(blob) - encHMACSize
	ciphertextStart := hdr.bodyOffset + encCBCBlockSize
	if macStart <= ciphertextStart {
		return nil, fmt.Errorf("missing ciphertext slot: %w", ErrEncryptionFormatInvalid)
	}
	ciphertext := blob[ciphertextStart:macStart]
	storedMAC := blob[macStart:]
	if len(ciphertext) == 0 || len(ciphertext)%encCBCBlockSize != 0 {
		return nil, fmt.Errorf("ciphertext len=%d not block-aligned: %w",
			len(ciphertext), ErrEncryptionFormatInvalid)
	}

	kek, err := e.kp.KEKByID(nil, hdr.kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", hdr.kekID, err)
	}
	dek, err := unwrapDEKWithKEK(kek, hdr.outerNonce, hdr.wrappedDEK)
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
	kekID := e.kp.ActiveKEKID()
	kek, err := e.kp.KEKByID(nil, kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", kekID, err)
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

	out, err := encodeEnvelopeHeader(encAlgoChaCha20, kekID, outerNonce, wrappedDEK, encNonceSize+len(ciphertext))
	if err != nil {
		return nil, err
	}
	out = append(out, innerNonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (e *chaCha20Encryptor) Decrypt(blob, aad []byte) ([]byte, error) {
	hdr, err := parseEnvelopeHeader(blob)
	if err != nil {
		return nil, err
	}
	if hdr.algo != encAlgoChaCha20 {
		return nil, fmt.Errorf("algo=%c unsupported by chacha20-poly1305 decryptor: %w",
			hdr.algo, ErrEncryptionAlgorithmNotImplemented)
	}
	if hdr.bodyOffset+encNonceSize+encChaChaTagSize > len(blob) {
		return nil, fmt.Errorf("truncated chacha body: %w", ErrEncryptionFormatInvalid)
	}
	innerNonce := blob[hdr.bodyOffset : hdr.bodyOffset+encNonceSize]
	ciphertext := blob[hdr.bodyOffset+encNonceSize:]

	kek, err := e.kp.KEKByID(nil, hdr.kekID)
	if err != nil {
		return nil, fmt.Errorf("get KEK id=%q: %w", hdr.kekID, err)
	}
	dek, err := unwrapDEKWithKEK(kek, hdr.outerNonce, hdr.wrappedDEK)
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
