// Package backup — KMS-style KeyProvider skeleton (T-0086).
//
// KMSKeyProvider is a pluggable KeyProvider variant that fetches the KEK
// from an external KMSClient at construction time and caches it for the
// lifetime of the process. It is a *skeleton* — the production AWS KMS /
// HashiCorp Vault Transit / vendor-HSM adapters that satisfy the
// KMSClient interface live in T-0091 (deferred until operators select the
// KMS provider for their deployment) and the multi-version key rotation
// support lives in T-0087 (KEK rotation + envelope kek_id field).
//
// Why a skeleton, not a full integration?
//
//   - AI side has no AWS / Vault / HSM credentials, so a real adapter
//     would be untested.
//   - Adding the AWS or Vault SDK as a direct dependency before the
//     vendor choice is made bloats go.mod by tens of transitive deps
//     for no testable code path.
//   - T-0087 (rotation) needs a KMS-style backend to develop against;
//     a deterministic mockKMSClient (in the test file) is enough.
//
// Architecture caveat (PRD T-0086 §2.1):
//
//	KMSKeyProvider satisfies the existing KeyProvider interface, which
//	means the plaintext KEK lives in process memory after construction.
//	This matches EnvKeyProvider's threat model — *NOT* the proper
//	KMS-native flow where every Encrypt/Decrypt round-trips to the KMS
//	service and the KEK is never extracted. The proper flow is the
//	KEKWrapper interface, which T-0087 will introduce alongside the
//	envelope kek_id field needed for rotation. This file's KMSKeyProvider
//	is the pluggability surface that lets a vendor-specific KMSClient
//	implementation drop in without further refactor — it does NOT close
//	the in-process-KEK gap.
//
// Security:
//
//   - Plaintext KEK is allocated, populated from KMSClient.Decrypt, and
//     held in struct field. KEK(ctx) returns a copy so callers cannot
//     mutate the source.
//   - The ciphertextBlob (KMS-wrapped DEK) is *not* held by this struct
//     post-construction — only the plaintext is needed at runtime;
//     re-fetching across restarts is the operator's responsibility
//     (env var / secret store).
//   - This file imports only stdlib + internal package types — no third-
//     party crypto deps. Real KMS adapters (T-0091) bring their own.
package backup

import (
	"context"
	"fmt"
)

// KMSClient is the narrow contract a KMS adapter must satisfy. Mirrors
// AWS KMS GenerateDataKey / Decrypt API shape so production adapters
// translate 1:1; satisfied by mockKMSClient in tests and by future real
// adapters in T-0091.
//
// Three methods are sufficient for the skeleton:
//
//   - GenerateDataKey: bootstrap a fresh KEK at first deployment.
//     Returns (plaintext, ciphertextBlob). plaintext is used immediately
//     (and discarded after KMSKeyProvider construction); ciphertextBlob
//     is the form the operator persists (env var / secret store) and
//     hands back to NewKMSKeyProvider on subsequent process starts.
//   - Decrypt: recover plaintext from a previously-stored ciphertextBlob.
//     Used at every NewKMSKeyProvider call.
//   - KeyID: opaque master-key identifier for logs and (future)
//     observability — real adapters return AWS ARN / Vault key path /
//     HSM slot; mock returns a constant.
//
// All methods are context-aware so cancellation / timeouts propagate to
// the underlying KMS network call when adapters are real.
type KMSClient interface {
	GenerateDataKey(ctx context.Context) (plaintext, ciphertextBlob []byte, err error)
	Decrypt(ctx context.Context, ciphertextBlob []byte) ([]byte, error)
	KeyID() string
}

// KMSKeyProvider satisfies KeyProvider using a KMSClient as the KEK source.
// Construction calls KMSClient.Decrypt once to recover the plaintext KEK,
// caches it, and serves KEK(ctx) from the cache. See package doc for
// architecture caveats.
type KMSKeyProvider struct {
	kek   []byte // 32B; populated at construction; never logged
	keyID string // for observability; not secret
}

// Compile-time interface conformance (also tested at runtime).
var _ KeyProvider = (*KMSKeyProvider)(nil)

// NewKMSKeyProvider constructs a KMSKeyProvider by Decrypting the supplied
// ciphertextBlob via the KMSClient. The plaintext is validated to be exactly
// kekSize (32 bytes); any other length surfaces ErrEncryptionFormatInvalid
// so wrong-shape KMS responses are rejected early.
//
// Returns nil + error on any of:
//   - client == nil
//   - len(ciphertextBlob) == 0
//   - client.Decrypt returns an error
//   - decrypted plaintext is not exactly kekSize (32 bytes)
//
// On success the returned provider satisfies KeyProvider with Available()=true
// for the lifetime of the process.
func NewKMSKeyProvider(ctx context.Context, client KMSClient, ciphertextBlob []byte) (*KMSKeyProvider, error) {
	if client == nil {
		return nil, fmt.Errorf("kms key provider: client is nil: %w", ErrEncryptionKeyUnavailable)
	}
	if len(ciphertextBlob) == 0 {
		return nil, fmt.Errorf("kms key provider: empty ciphertextBlob: %w", ErrEncryptionKeyUnavailable)
	}
	plaintext, err := client.Decrypt(ctx, ciphertextBlob)
	if err != nil {
		return nil, fmt.Errorf("kms key provider: decrypt blob: %w", err)
	}
	if len(plaintext) != kekSize {
		return nil, fmt.Errorf("kms key provider: plaintext KEK size=%d expected %d: %w",
			len(plaintext), kekSize, ErrEncryptionFormatInvalid)
	}
	return &KMSKeyProvider{kek: plaintext, keyID: client.KeyID()}, nil
}

// KEK returns a fresh copy of the cached plaintext KEK. Mirrors
// EnvKeyProvider.KEK behaviour (review M-2 fix): copy isolates callers from
// the provider's internal storage so mutation in one place can't ripple.
func (p *KMSKeyProvider) KEK(_ context.Context) ([]byte, error) {
	if !p.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	out := make([]byte, len(p.kek))
	copy(out, p.kek)
	return out, nil
}

// Available reports whether the provider holds a usable cached KEK.
func (p *KMSKeyProvider) Available() bool {
	return p != nil && len(p.kek) == kekSize
}

// KeyID returns the KMS master-key identifier for logs / observability.
// Empty string when the provider was constructed under a KMSClient that
// does not expose a stable identifier. Never returns secret material.
func (p *KMSKeyProvider) KeyID() string {
	if p == nil {
		return ""
	}
	return p.keyID
}
