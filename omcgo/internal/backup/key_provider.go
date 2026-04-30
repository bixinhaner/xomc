// Package backup — backup encryption key management (T-0075).
//
// KeyProvider returns the Key-Encryption-Key (KEK) used to wrap per-file
// Data-Encryption-Keys (DEK) under envelope encryption. Two implementations
// satisfy the interface today:
//
//   - EnvKeyProvider (this file)        — reads OMC_BACKUP_ENCRYPTION_KEY
//     (64-hex-char string ⇒ 32 bytes / 256 bits). The default for local /
//     systemd / docker-compose deployments without external KMS.
//   - KMSKeyProvider (key_provider_kms.go) — fetches the KEK from a
//     KMSClient at construction time. Skeleton + mock; the production
//     adapters (AWS KMS / Vault / HSM) plug into KMSClient via T-0091.
//
// Both store the plaintext KEK in process memory; the proper "KEK never
// in process" flow is the KEKWrapper interface (T-0087+ alongside
// envelope kek_id field for rotation).
//
// Security notes:
//   - The KEK lives in process memory and (transitively) in the env var
//     visible via /proc/<pid>/environ for EnvKeyProvider. Operators are
//     expected to restrict access via systemd EnvironmentFile= + chmod 600
//     or equivalent.
//   - KEK rotation is out of scope (T-0087); changing the key invalidates
//     all previously-encrypted backups.
//   - This package never logs the key bytes or returns them in error
//     messages. Tests must not log key fixtures either.
package backup

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
)

// EnvBackupEncryptionKey is the env var name read by EnvKeyProvider.
const EnvBackupEncryptionKey = "OMC_BACKUP_ENCRYPTION_KEY"

// kekSize is the AES-256 key length: 32 bytes.
const kekSize = 32

// KeyProvider is the narrow contract Encryptor consumes for KEK lookup.
// Available() lets callers (e.g. PolicyService.validatePolicy) check whether
// encryption can be enabled at all without actually invoking KEK retrieval.
type KeyProvider interface {
	// KEK returns the 32-byte key. Returns ErrEncryptionKeyUnavailable when
	// the underlying provider is unconfigured.
	KEK(ctx context.Context) ([]byte, error)
	// Available reports whether KEK retrieval is expected to succeed without
	// actually performing it. Used by PolicyService to fail fast at PUT time.
	Available() bool
}

// EnvKeyProvider reads OMC_BACKUP_ENCRYPTION_KEY at construction time. The
// value is a 64-character hex string encoding a 32-byte AES-256 key.
//
// Construction does not error on missing/invalid env var — Available() simply
// reports false. This lets the binary start in deployments that do not (yet)
// enable encryption; enabling at runtime requires a restart with the env var
// set.
type EnvKeyProvider struct {
	key []byte // nil when unavailable; never logged
}

// NewEnvKeyProvider reads the env var and returns a provider. An invalid
// hex string or wrong length is a configuration error and returns nil
// provider + error so cmd/acs/main.go can surface it at startup; an unset
// env var returns a non-nil provider with Available()=false.
func NewEnvKeyProvider() (*EnvKeyProvider, error) {
	hexKey := os.Getenv(EnvBackupEncryptionKey)
	if hexKey == "" {
		return &EnvKeyProvider{}, nil
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid hex: %w", EnvBackupEncryptionKey, err)
	}
	if len(key) != kekSize {
		return nil, fmt.Errorf("%s must decode to %d bytes (got %d)",
			EnvBackupEncryptionKey, kekSize, len(key))
	}
	return &EnvKeyProvider{key: key}, nil
}

// KEK returns a fresh copy of the 32-byte key. The copy isolates the caller
// from accidentally corrupting the provider's internal storage (review M-2
// fix — defense-in-depth even though current callers don't mutate). The
// 32-byte allocation is negligible per encrypt.
func (p *EnvKeyProvider) KEK(_ context.Context) ([]byte, error) {
	if !p.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	out := make([]byte, len(p.key))
	copy(out, p.key)
	return out, nil
}

// Available reports whether the provider was constructed with a usable key.
func (p *EnvKeyProvider) Available() bool {
	return p != nil && len(p.key) == kekSize
}
