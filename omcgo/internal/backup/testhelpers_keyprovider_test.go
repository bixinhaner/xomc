package backup

import "context"

// staticKeyProvider is a test-only KeyProvider built from raw bytes.
// Lives in a *_test.go file so it never compiles into production binaries
// (review L-1 fix: removes a small attack surface from the production
// package). Used by encryption_test.go and policy_service_test.go.
type staticKeyProvider struct{ key []byte }

func newStaticKeyProvider(key []byte) *staticKeyProvider {
	return &staticKeyProvider{key: key}
}

func (p *staticKeyProvider) KEK(ctx context.Context) ([]byte, error) {
	return p.KEKByID(ctx, p.ActiveKEKID())
}

func (p *staticKeyProvider) Available() bool { return len(p.key) == kekSize }

// T-0087 additions — single-key static provider returns the same key for
// the active ID ("") and rejects any non-empty kekID, mirroring the
// "single-key envelope-v1 backwards-compat" behaviour of EnvKeyProvider's
// legacy mode.
func (p *staticKeyProvider) ActiveKEKID() string { return "" }

func (p *staticKeyProvider) KEKByID(_ context.Context, kekID string) ([]byte, error) {
	if !p.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	if kekID != "" {
		return nil, ErrEncryptionKeyUnavailable
	}
	out := make([]byte, len(p.key))
	copy(out, p.key)
	return out, nil
}

// multiKeyProvider is a test-only KeyProvider holding a map of named KEKs
// for rotation scenarios. The active id is settable so tests can simulate
// "encrypt under v1, switch active to v2, decrypt v1 file" flows without
// depending on env vars.
type multiKeyProvider struct {
	activeID string
	keys     map[string][]byte
}

func newMultiKeyProvider(activeID string, keys map[string][]byte) *multiKeyProvider {
	cp := make(map[string][]byte, len(keys))
	for k, v := range keys {
		cp[k] = append([]byte(nil), v...)
	}
	return &multiKeyProvider{activeID: activeID, keys: cp}
}

func (p *multiKeyProvider) KEK(ctx context.Context) ([]byte, error) {
	return p.KEKByID(ctx, p.ActiveKEKID())
}
func (p *multiKeyProvider) Available() bool { return p != nil && len(p.keys) > 0 }
func (p *multiKeyProvider) ActiveKEKID() string {
	if p == nil {
		return ""
	}
	return p.activeID
}
func (p *multiKeyProvider) KEKByID(_ context.Context, kekID string) ([]byte, error) {
	if !p.Available() {
		return nil, ErrEncryptionKeyUnavailable
	}
	k, ok := p.keys[kekID]
	if !ok {
		return nil, ErrEncryptionKeyUnavailable
	}
	out := make([]byte, len(k))
	copy(out, k)
	return out, nil
}
