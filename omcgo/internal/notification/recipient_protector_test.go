package notification

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAESGCMRecipientProtector_RoundTripAndStableOpaqueFingerprint(t *testing.T) {
	protector, err := NewAESGCMRecipientProtector(bytes.Repeat([]byte{0x2a}, 32))
	require.NoError(t, err)
	one, version, fingerprintOne, err := protector.Protect("email", " NOC@Example.com ")
	require.NoError(t, err)
	two, _, fingerprintTwo, err := protector.Protect("email", "noc@example.com")
	require.NoError(t, err)
	require.NotEqual(t, one, two, "random nonces must prevent deterministic ciphertext")
	require.Equal(t, fingerprintOne, fingerprintTwo)
	require.NotContains(t, string(one), "noc@example.com")

	address, err := protector.Unprotect("email", one, version)
	require.NoError(t, err)
	require.Equal(t, "noc@example.com", address)
}

func TestAESGCMRecipientProtector_BindsCiphertextToChannel(t *testing.T) {
	protector, err := NewAESGCMRecipientProtector(bytes.Repeat([]byte{0x3b}, 32))
	require.NoError(t, err)
	ciphertext, version, _, err := protector.Protect("email", "noc@example.com")
	require.NoError(t, err)
	_, err = protector.Unprotect("sms_kafka", ciphertext, version)
	require.Error(t, err)
}
