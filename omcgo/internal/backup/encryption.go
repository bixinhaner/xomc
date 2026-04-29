// Package backup — backup encryption (T-0075 / R-102 final core capability).
//
// Envelope encryption: a per-file Data-Encryption-Key (DEK, 32B random) is
// generated, used to AES-256-GCM-encrypt the plaintext, and itself wrapped
// under the deployment-wide KEK (also AES-256-GCM). The wrapped DEK and
// both nonces ride alongside the ciphertext in a custom file format
// (see `encMagic` and the layout comment below).
//
// Design choices (PRD §2):
//   - AES-256-GCM only in MVP; CBC and ChaCha20-Poly1305 stub-error
//   - AAD = filename binds ciphertext to its expected container (anti-swap)
//   - Buffer-then-encrypt with 64MB ceiling — backups are < 10MB realistic;
//     streaming chunked-GCM format is T-0085 material
//   - File format starts with magic "OENC" so download handler can detect
//     encrypted objects even without `.enc` extension
//
// Security:
//   - Nonces from crypto/rand (Linux: /dev/urandom)
//   - GCM tag verification is constant-time (Go stdlib)
//   - Key bytes never appear in error messages or logs
//   - This file imports only stdlib crypto + math + encoding — no third-party
//     crypto deps
package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	encNonceSize      = 12 // GCM standard
	encDEKSize        = 32 // AES-256
	encGCMTagSize     = 16 // GCM standard
	encWrappedDEKSize = encDEKSize + encGCMTagSize // 48
	encMaxPlaintext   = 64 * 1024 * 1024 // 64 MB cap, T-0075 PRD §2.3
)

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
// one of AES-256-GCM|AES-256-CBC|ChaCha20-Poly1305. CBC and ChaCha20-Poly1305
// currently return ErrEncryptionAlgorithmNotImplemented (T-0085 followup).
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
	case "AES-256-CBC", "ChaCha20-Poly1305":
		return nil, fmt.Errorf("algorithm=%q: %w", algorithm, ErrEncryptionAlgorithmNotImplemented)
	default:
		return nil, fmt.Errorf("algorithm=%q: %w", algorithm, ErrEncryptionAlgorithmInvalid)
	}
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
