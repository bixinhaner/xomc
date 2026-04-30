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

// =============================================================================
// T-0087 — multi-version env support
// =============================================================================

// TestEnvKeyProvider_T0087_SingleKeyDefault — only OMC_BACKUP_ENCRYPTION_KEY
// set; ActiveKEKID="" and KEKByID("") returns the key. Backwards compat
// with T-0075 deployments — no env var changes required.
func TestEnvKeyProvider_T0087_SingleKeyDefault(t *testing.T) {
	rawKey := make([]byte, kekSize)
	for i := range rawKey {
		rawKey[i] = byte(i)
	}
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(rawKey))
	t.Setenv(EnvBackupEncryptionKeyID, "")
	t.Setenv(EnvBackupEncryptionKeyHistory, "")

	kp, err := NewEnvKeyProvider()
	require.NoError(t, err)
	assert.True(t, kp.Available())
	assert.Equal(t, "", kp.ActiveKEKID())

	got, err := kp.KEKByID(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, rawKey, got)
}

// TestEnvKeyProvider_T0087_ActiveIDOnly — KEY + KEY_ID="v2"; the active
// key is registered under both "v2" and "" (autopopulated for v1
// envelope backwards compat).
func TestEnvKeyProvider_T0087_ActiveIDOnly(t *testing.T) {
	rawKey := make([]byte, kekSize)
	for i := range rawKey {
		rawKey[i] = byte(0xA0 + i)
	}
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(rawKey))
	t.Setenv(EnvBackupEncryptionKeyID, "v2")
	t.Setenv(EnvBackupEncryptionKeyHistory, "")

	kp, err := NewEnvKeyProvider()
	require.NoError(t, err)
	assert.Equal(t, "v2", kp.ActiveKEKID())

	got, err := kp.KEKByID(context.Background(), "v2")
	require.NoError(t, err)
	assert.Equal(t, rawKey, got)

	// Backwards-compat slot also populated
	got, err = kp.KEKByID(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, rawKey, got, "active key autopopulated under \"\" for v1 envelope read")
}

// TestEnvKeyProvider_T0087_MultiVersionWithHistory — KEY + KEY_ID="v2" +
// KEY_HISTORY="v1=hex". Both KEKs accessible by ID; active is v2.
func TestEnvKeyProvider_T0087_MultiVersionWithHistory(t *testing.T) {
	keyV2 := make([]byte, kekSize)
	keyV1 := make([]byte, kekSize)
	for i := range keyV1 {
		keyV1[i] = byte(0x10 + i)
		keyV2[i] = byte(0x20 + i)
	}
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(keyV2))
	t.Setenv(EnvBackupEncryptionKeyID, "v2")
	t.Setenv(EnvBackupEncryptionKeyHistory, "v1="+hex.EncodeToString(keyV1))

	kp, err := NewEnvKeyProvider()
	require.NoError(t, err)
	assert.Equal(t, "v2", kp.ActiveKEKID())

	gotV1, err := kp.KEKByID(context.Background(), "v1")
	require.NoError(t, err)
	assert.Equal(t, keyV1, gotV1)

	gotV2, err := kp.KEKByID(context.Background(), "v2")
	require.NoError(t, err)
	assert.Equal(t, keyV2, gotV2)
}

// TestEnvKeyProvider_T0087_HistoryActiveCollision — active id and history
// id collide → reject construction.
func TestEnvKeyProvider_T0087_HistoryActiveCollision(t *testing.T) {
	keyHex := hex.EncodeToString(make([]byte, kekSize))
	t.Setenv(EnvBackupEncryptionKey, keyHex)
	t.Setenv(EnvBackupEncryptionKeyID, "v2")
	t.Setenv(EnvBackupEncryptionKeyHistory, "v2="+keyHex)

	_, err := NewEnvKeyProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collides with active")
}

// TestEnvKeyProvider_T0087_HistoryDuplicateID — history with two entries
// for the same id → reject.
func TestEnvKeyProvider_T0087_HistoryDuplicateID(t *testing.T) {
	keyHex := hex.EncodeToString(make([]byte, kekSize))
	t.Setenv(EnvBackupEncryptionKey, keyHex)
	t.Setenv(EnvBackupEncryptionKeyID, "v3")
	t.Setenv(EnvBackupEncryptionKeyHistory, "v1="+keyHex+";v1="+keyHex)

	_, err := NewEnvKeyProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")
}

// TestEnvKeyProvider_T0087_HistoryBadHex — history entry with non-hex
// value → construction fails.
func TestEnvKeyProvider_T0087_HistoryBadHex(t *testing.T) {
	keyHex := hex.EncodeToString(make([]byte, kekSize))
	t.Setenv(EnvBackupEncryptionKey, keyHex)
	t.Setenv(EnvBackupEncryptionKeyID, "v2")
	t.Setenv(EnvBackupEncryptionKeyHistory, "v1=not-hex-z")

	_, err := NewEnvKeyProvider()
	require.Error(t, err)
}

// TestEnvKeyProvider_T0087_KEKDelegatesToActive — KEK() returns the
// active KEK (= KEKByID(ActiveKEKID())).
func TestEnvKeyProvider_T0087_KEKDelegatesToActive(t *testing.T) {
	rawKey := make([]byte, kekSize)
	for i := range rawKey {
		rawKey[i] = byte(0xC0 + i)
	}
	t.Setenv(EnvBackupEncryptionKey, hex.EncodeToString(rawKey))
	t.Setenv(EnvBackupEncryptionKeyID, "v2")
	t.Setenv(EnvBackupEncryptionKeyHistory, "")

	kp, err := NewEnvKeyProvider()
	require.NoError(t, err)

	got, err := kp.KEK(context.Background())
	require.NoError(t, err)
	assert.Equal(t, rawKey, got)
}
