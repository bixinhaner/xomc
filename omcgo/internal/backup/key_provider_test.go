package backup

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvKeyProvider_unset(t *testing.T) {
	t.Setenv(EnvBackupEncryptionKey, "")
	kp, err := NewEnvKeyProvider()
	require.NoError(t, err, "unset env var must not error — Available()=false instead")
	assert.False(t, kp.Available())

	_, err = kp.KEK(context.Background())
	require.Error(t, err)
}

func TestEnvKeyProvider_validHex(t *testing.T) {
	rawKey := make([]byte, kekSize)
	for i := range rawKey {
		rawKey[i] = byte(i)
	}
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(rawKey))

	kp, err := NewEnvKeyProvider()
	require.NoError(t, err)
	require.True(t, kp.Available())

	got, err := kp.KEK(context.Background())
	require.NoError(t, err)
	assert.Equal(t, rawKey, got)
}

func TestEnvKeyProvider_invalidHex(t *testing.T) {
	t.Setenv(EnvBackupEncryptionKey, "not-hex-zzzz")
	_, err := NewEnvKeyProvider()
	require.Error(t, err, "invalid hex string must error at startup")
}

func TestEnvKeyProvider_wrongLength(t *testing.T) {
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(make([]byte, 16))) // 16 bytes != 32
	_, err := NewEnvKeyProvider()
	require.Error(t, err, "16-byte key must reject (need exactly 32 bytes for AES-256)")
}
