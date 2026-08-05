package notification

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	EnvNotificationRecipientKey = "OMC_NOTIFICATION_RECIPIENT_KEY"
	recipientKeyVersion         = 1
)

var ErrRecipientKeyUnavailable = errors.New("notification recipient encryption key is unavailable")

type AESGCMRecipientProtector struct {
	aead           cipher.AEAD
	fingerprintKey []byte
}

func NewEnvRecipientProtector() (*AESGCMRecipientProtector, error) {
	raw := strings.TrimSpace(os.Getenv(EnvNotificationRecipientKey))
	if raw == "" {
		return nil, ErrRecipientKeyUnavailable
	}
	master, err := hex.DecodeString(raw)
	if err != nil || len(master) != 32 {
		return nil, fmt.Errorf("load notification recipient encryption key: %s must be 64 hexadecimal characters", EnvNotificationRecipientKey)
	}
	return NewAESGCMRecipientProtector(master)
}

func NewAESGCMRecipientProtector(masterKey []byte) (*AESGCMRecipientProtector, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("create notification recipient protector: 32-byte master key is required")
	}
	encryptionKey := deriveRecipientKey(masterKey, "omc-notification-recipient-encryption-v1")
	fingerprintKey := deriveRecipientKey(masterKey, "omc-notification-recipient-fingerprint-v1")
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create notification recipient cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create notification recipient AEAD: %w", err)
	}
	return &AESGCMRecipientProtector{aead: aead, fingerprintKey: fingerprintKey}, nil
}

func (p *AESGCMRecipientProtector) Protect(channel, address string) ([]byte, int, []byte, error) {
	channel, address, err := normalizeRecipientAddress(channel, address)
	if err != nil {
		return nil, 0, nil, err
	}
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, 0, nil, fmt.Errorf("generate notification recipient nonce: %w", err)
	}
	ciphertext := p.aead.Seal(nonce, nonce, []byte(address), []byte(channel))
	mac := hmac.New(sha256.New, p.fingerprintKey)
	_, _ = mac.Write([]byte(channel))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(address))
	return ciphertext, recipientKeyVersion, mac.Sum(nil), nil
}

func (p *AESGCMRecipientProtector) Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error) {
	if p == nil || p.aead == nil || keyVersion != recipientKeyVersion || len(ciphertext) <= p.aead.NonceSize() {
		return "", fmt.Errorf("decrypt notification recipient: invalid key version or ciphertext")
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	nonce, sealed := ciphertext[:p.aead.NonceSize()], ciphertext[p.aead.NonceSize():]
	plaintext, err := p.aead.Open(nil, nonce, sealed, []byte(channel))
	if err != nil {
		return "", fmt.Errorf("decrypt notification recipient: authentication failed")
	}
	return string(plaintext), nil
}

func deriveRecipientKey(masterKey []byte, label string) []byte {
	mac := hmac.New(sha256.New, masterKey)
	_, _ = mac.Write([]byte(label))
	return mac.Sum(nil)
}

func normalizeRecipientAddress(channel, address string) (string, string, error) {
	channel = strings.TrimSpace(strings.ToLower(channel))
	address = strings.TrimSpace(address)
	if channel == "email" {
		address = strings.ToLower(address)
	}
	if channel == "" || address == "" {
		return "", "", fmt.Errorf("protect notification recipient: channel and address are required")
	}
	return channel, address, nil
}
