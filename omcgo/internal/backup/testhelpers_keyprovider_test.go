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

func (p *staticKeyProvider) KEK(_ context.Context) ([]byte, error) {
	if len(p.key) != kekSize {
		return nil, ErrEncryptionKeyUnavailable
	}
	out := make([]byte, len(p.key))
	copy(out, p.key)
	return out, nil
}

func (p *staticKeyProvider) Available() bool { return len(p.key) == kekSize }
