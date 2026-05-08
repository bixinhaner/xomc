package loginpwd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newCipherForTest(t *testing.T, now time.Time) (*Cipher, *rsa.PublicKey, *MemReplayGuard, string) {
	t.Helper()
	ks, err := NewKeystore(filepath.Join(t.TempDir(), "key.pem"))
	require.NoError(t, err)
	guard := NewMemReplayGuard(func() time.Time { return now })
	cipher := NewCipher(ks, guard)
	pub := loadPublicKeyFromPEM(t, ks.ActivePublicKeyPEM())
	return cipher, pub, guard, ks.ActiveKeyID()
}

func encryptForTest(t *testing.T, pub *rsa.PublicKey, payload Payload) string {
	t.Helper()
	body := payloadJSON(t, payload)
	ct, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, body, nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(ct)
}

func TestCipher_Decrypt_HappyPath(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, keyID := newCipherForTest(t, now)

	ciphertext := encryptForTest(t, pub, Payload{
		Password: "hunter2",
		TS:       now.Unix(),
		Nonce:    "abcd-1234-efgh-5678",
	})

	pwd, err := cipher.Decrypt(context.Background(), keyID, ciphertext)
	require.NoError(t, err)
	require.Equal(t, "hunter2", pwd)
}

func TestCipher_Decrypt_RejectsEmptyInputs(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, _, _, keyID := newCipherForTest(t, now)

	_, err := cipher.Decrypt(context.Background(), "", "ciphertext")
	require.ErrorIs(t, err, ErrEmptyKeyID)

	_, err = cipher.Decrypt(context.Background(), keyID, "")
	require.ErrorIs(t, err, ErrEmptyCiphertext)
}

func TestCipher_Decrypt_RejectsReplay(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, keyID := newCipherForTest(t, now)

	payload := Payload{Password: "p", TS: now.Unix(), Nonce: "same-nonce"}
	first := encryptForTest(t, pub, payload)
	second := encryptForTest(t, pub, payload) // 相同 nonce，不同密文（OAEP 含随机源）

	_, err := cipher.Decrypt(context.Background(), keyID, first)
	require.NoError(t, err)
	_, err = cipher.Decrypt(context.Background(), keyID, second)
	require.ErrorIs(t, err, ErrReplayDetected)
}

func TestCipher_Decrypt_RejectsExpiredTimestamp(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, keyID := newCipherForTest(t, now)

	ciphertext := encryptForTest(t, pub, Payload{
		Password: "p",
		TS:       now.Add(-10 * time.Minute).Unix(),
		Nonce:    "n1",
	})
	_, err := cipher.Decrypt(context.Background(), keyID, ciphertext)
	require.ErrorIs(t, err, ErrTimestampOutOfRange)
}

func TestCipher_Decrypt_RejectsTooLongPassword(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, keyID := newCipherForTest(t, now)

	ciphertext := encryptForTest(t, pub, Payload{
		Password: strings.Repeat("x", MaxPasswordLength+1),
		TS:       now.Unix(),
		Nonce:    "n1",
	})
	_, err := cipher.Decrypt(context.Background(), keyID, ciphertext)
	require.ErrorIs(t, err, ErrPasswordTooLong)
}

func TestCipher_Decrypt_RejectsUnknownKeyID(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, _ := newCipherForTest(t, now)

	ciphertext := encryptForTest(t, pub, Payload{Password: "p", TS: now.Unix(), Nonce: "n"})
	_, err := cipher.Decrypt(context.Background(), "wrong-key-id", ciphertext)
	require.ErrorIs(t, err, ErrUnknownKeyID)
}

func TestCipher_Decrypt_AcceptsURLSafeBase64(t *testing.T) {
	now := time.Unix(1_714_896_000, 0)
	cipher, pub, _, keyID := newCipherForTest(t, now)

	body := payloadJSON(t, Payload{Password: "p", TS: now.Unix(), Nonce: "n2"})
	ct, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, body, nil)
	require.NoError(t, err)

	// URL-safe base64（含 - 或 _）
	urlsafe := base64.URLEncoding.EncodeToString(ct)
	_, err = cipher.Decrypt(context.Background(), keyID, urlsafe)
	require.NoError(t, err)
}
