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
	"strings"
)

// Env vars read by EnvKeyProvider:
//
//   - OMC_BACKUP_ENCRYPTION_KEY          — 64-hex active KEK (single-key
//     mode, backwards compatible with T-0075 deployments)
//   - OMC_BACKUP_ENCRYPTION_KEY_ID       — opaque ID for the active key
//     (default ""; recommended to set in multi-version mode for clarity)
//   - OMC_BACKUP_ENCRYPTION_KEY_HISTORY  — retired-key list, format
//     "id1=hex;id2=hex" (T-0087 multi-version support; reads only)
const (
	EnvBackupEncryptionKey        = "OMC_BACKUP_ENCRYPTION_KEY"
	EnvBackupEncryptionKeyID      = "OMC_BACKUP_ENCRYPTION_KEY_ID"
	EnvBackupEncryptionKeyHistory = "OMC_BACKUP_ENCRYPTION_KEY_HISTORY"
)

// kekSize is the AES-256 key length: 32 bytes.
const kekSize = 32

// maxKEKIDLen caps how long a key ID may be when serialised into the
// OENC v2 envelope (kek_id_len is u8). 255 is enough for AWS KMS ARNs and
// Vault key paths in practice.
const maxKEKIDLen = 255

// KeyProvider is the narrow contract Encryptor consumes for KEK lookup.
// Available() lets callers (e.g. PolicyService.validatePolicy) check whether
// encryption can be enabled at all without actually invoking KEK retrieval.
//
// T-0087 added ActiveKEKID + KEKByID for multi-version key rotation:
// envelope v2 records the kek_id used to encrypt; decrypt looks it up via
// KEKByID. The single-method KEK(ctx) is preserved as a convenience that
// returns the active key (= KEKByID(ctx, ActiveKEKID())).
type KeyProvider interface {
	// KEK returns the 32-byte active key. Equivalent to KEKByID(ctx,
	// ActiveKEKID()). Returns ErrEncryptionKeyUnavailable when the
	// underlying provider is unconfigured.
	KEK(ctx context.Context) ([]byte, error)
	// Available reports whether KEK retrieval is expected to succeed without
	// actually performing it. Used by PolicyService to fail fast at PUT time.
	Available() bool
	// ActiveKEKID returns the identifier of the currently-active KEK
	// (written into the OENC v2 envelope at encrypt time). Empty string
	// for legacy single-key deployments — that ID is also accepted by
	// KEKByID for backwards compat with T-0075 / T-0085 envelopes.
	ActiveKEKID() string
	// KEKByID returns the KEK matching the given id. Returns
	// ErrEncryptionKeyUnavailable when no KEK with that id is configured.
	// kekID == "" maps to the legacy single-key (or, in multi-version
	// EnvKeyProvider, the active key autopopulated under "" so older
	// envelopes remain readable — see EnvKeyProvider §2.6).
	KEKByID(ctx context.Context, kekID string) ([]byte, error)
}

// EnvKeyProvider reads OMC_BACKUP_ENCRYPTION_KEY at construction time. The
// value is a 64-character hex string encoding a 32-byte AES-256 key.
//
// Two modes (T-0087):
//   - Single-key (T-0075 default): only OMC_BACKUP_ENCRYPTION_KEY set.
//     Active ID = OMC_BACKUP_ENCRYPTION_KEY_ID (default ""). The single
//     key is registered under both "" and the explicit ID so legacy v1
//     envelopes (no kek_id field) and v2 envelopes (with kek_id) both
//     decrypt.
//   - Multi-version (T-0087 rotation): set _HISTORY to retain previously-
//     active KEKs for read access while new writes use the active KEK.
//
// Construction does not error on missing/invalid env var — Available() simply
// reports false. This lets the binary start in deployments that do not (yet)
// enable encryption; enabling at runtime requires a restart with the env var
// set.
type EnvKeyProvider struct {
	activeID string
	keys     map[string][]byte // kekID → 32B; never logged
}

// NewEnvKeyProvider reads the env vars and returns a provider. An invalid
// hex string, wrong key length, malformed _HISTORY entry, or active/history
// ID collision is a configuration error and returns nil provider + error
// so cmd/acs/main.go can surface it at startup; an unset
// OMC_BACKUP_ENCRYPTION_KEY returns a non-nil provider with Available()=false.
func NewEnvKeyProvider() (*EnvKeyProvider, error) {
	hexKey := os.Getenv(EnvBackupEncryptionKey)
	if hexKey == "" {
		return &EnvKeyProvider{keys: map[string][]byte{}}, nil
	}
	activeKey, err := decodeKEKHex(hexKey, EnvBackupEncryptionKey)
	if err != nil {
		return nil, err
	}
	activeID := os.Getenv(EnvBackupEncryptionKeyID)
	if len(activeID) > maxKEKIDLen {
		return nil, fmt.Errorf("%s exceeds %d chars (got %d)",
			EnvBackupEncryptionKeyID, maxKEKIDLen, len(activeID))
	}

	keys := map[string][]byte{activeID: activeKey}

	historyRaw := os.Getenv(EnvBackupEncryptionKeyHistory)
	if historyRaw != "" {
		if err := parseKEKHistory(historyRaw, activeID, keys); err != nil {
			return nil, err
		}
	}

	// Backwards compatibility: if active ID is non-empty, also register the
	// active key under "" so legacy v1 envelopes (which carry no kek_id)
	// remain readable. Operators who explicitly set _HISTORY can override
	// "" by including a "=hex" entry there.
	if activeID != "" {
		if _, has := keys[""]; !has {
			keys[""] = activeKey
		}
	}

	return &EnvKeyProvider{activeID: activeID, keys: keys}, nil
}

// decodeKEKHex parses a 64-char hex string into 32 bytes, surfacing the
// env-var name in errors so misconfiguration is easy to fix.
func decodeKEKHex(s, envName string) ([]byte, error) {
	key, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid hex: %w", envName, err)
	}
	if len(key) != kekSize {
		return nil, fmt.Errorf("%s must decode to %d bytes (got %d)",
			envName, kekSize, len(key))
	}
	return key, nil
}

// parseKEKHistory parses "id1=hex1;id2=hex2;..." into the keys map.
// Active ID collisions and duplicate history IDs are rejected. Empty
// entries (e.g. trailing ";") are tolerated.
func parseKEKHistory(raw, activeID string, keys map[string][]byte) error {
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		eq := strings.IndexByte(entry, '=')
		if eq <= 0 {
			return fmt.Errorf("%s entry %q missing 'id=hex' format",
				EnvBackupEncryptionKeyHistory, entry)
		}
		id, hexVal := entry[:eq], entry[eq+1:]
		if len(id) > maxKEKIDLen {
			return fmt.Errorf("%s id=%q exceeds %d chars",
				EnvBackupEncryptionKeyHistory, id, maxKEKIDLen)
		}
		if id == activeID {
			return fmt.Errorf("%s id=%q collides with active key id",
				EnvBackupEncryptionKeyHistory, id)
		}
		if _, dup := keys[id]; dup {
			return fmt.Errorf("%s id=%q duplicated", EnvBackupEncryptionKeyHistory, id)
		}
		key, err := decodeKEKHex(hexVal, fmt.Sprintf("%s[%s]", EnvBackupEncryptionKeyHistory, id))
		if err != nil {
			return err
		}
		keys[id] = key
	}
	return nil
}

// KEK returns a fresh copy of the active 32-byte key. Equivalent to
// KEKByID(ctx, ActiveKEKID()). The copy isolates callers from accidentally
// corrupting the provider's internal storage (review M-2 fix).
func (p *EnvKeyProvider) KEK(ctx context.Context) ([]byte, error) {
	return p.KEKByID(ctx, p.ActiveKEKID())
}

// Available reports whether the provider was constructed with at least one
// usable key. In single-key mode this is simply "active key set"; in
// multi-version mode it tracks the same condition because constructions
// without an active key short-circuit before populating the map.
func (p *EnvKeyProvider) Available() bool {
	return p != nil && len(p.keys) > 0
}

// ActiveKEKID returns the ID written into newly-encrypted envelopes.
// Empty string for legacy single-key deployments without explicit
// OMC_BACKUP_ENCRYPTION_KEY_ID configuration.
func (p *EnvKeyProvider) ActiveKEKID() string {
	if p == nil {
		return ""
	}
	return p.activeID
}

// KEKByID looks up a 32-byte key by id. Empty string maps to the legacy
// "active key registered under """ slot for v1-envelope backwards compat.
func (p *EnvKeyProvider) KEKByID(_ context.Context, kekID string) ([]byte, error) {
	if !p.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	key, ok := p.keys[kekID]
	if !ok {
		return nil, fmt.Errorf("kek id=%q not configured: %w", kekID, ErrEncryptionKeyUnavailable)
	}
	out := make([]byte, len(key))
	copy(out, key)
	return out, nil
}
