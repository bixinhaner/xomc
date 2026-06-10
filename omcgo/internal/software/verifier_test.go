package software

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// HashOnlyVerifier (default, always-on integrity gate)
// ---------------------------------------------------------------------------

func TestHashOnlyVerifier_Verify(t *testing.T) {
	v := NewHashOnlyVerifier()

	tests := []struct {
		name       string
		fw         *FirmwareVersion
		wantErr    bool
		wantCode   FailureCode
		wantLegacy bool
	}{
		{
			name:       "sha256 present passes (preferred)",
			fw:         &FirmwareVersion{SHA256Val: "deadbeef", MD5Val: "abc"},
			wantErr:    false,
			wantLegacy: false,
		},
		{
			name:       "md5-only passes on legacy fallback",
			fw:         &FirmwareVersion{MD5Val: "abc123"},
			wantErr:    false,
			wantLegacy: true,
		},
		{
			name:     "no digest at all is rejected",
			fw:       &FirmwareVersion{Version: "V1"},
			wantErr:  true,
			wantCode: FailureIntegrityCheck,
		},
		{
			name:     "whitespace-only digests are rejected",
			fw:       &FirmwareVersion{SHA256Val: "  ", MD5Val: "\t"},
			wantErr:  true,
			wantCode: FailureIntegrityCheck,
		},
		{
			name:     "nil firmware is rejected",
			fw:       nil,
			wantErr:  true,
			wantCode: FailureIntegrityCheck,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Verify(tc.fw)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.wantCode, VerificationFailureCode(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantLegacy, IsLegacyMD5Only(tc.fw))
		})
	}
}

// ---------------------------------------------------------------------------
// SignatureVerifier (scaffold: enforce-when-present + config-gated for unsigned)
// ---------------------------------------------------------------------------

func TestSignatureVerifier_UnsignedFirmware(t *testing.T) {
	fw := &FirmwareVersion{SHA256Val: "deadbeef"} // no signature

	// RequireSignature=false → unsigned firmware passes (happy-path preserved).
	lenient := NewSignatureVerifier(nil, false, nil)
	require.NoError(t, lenient.Verify(fw), "unsigned firmware must pass when enforcement is off")

	// RequireSignature=true → unsigned firmware is rejected.
	strict := NewSignatureVerifier(nil, true, nil)
	err := strict.Verify(fw)
	require.Error(t, err, "unsigned firmware must be rejected when enforcement is on")
	assert.Equal(t, FailureSignatureCheck, VerificationFailureCode(err))
}

func TestSignatureVerifier_SignedFirmware_NoVerifyFn(t *testing.T) {
	// Signed firmware but no verification key wired → must reject (never pretend
	// it verified). Holds regardless of RequireSignature.
	fw := &FirmwareVersion{SHA256Val: "deadbeef", Signature: "c2lnbmF0dXJl"}
	v := NewSignatureVerifier(nil, false, nil)
	err := v.Verify(fw)
	require.Error(t, err)
	assert.Equal(t, FailureSignatureCheck, VerificationFailureCode(err))
}

func TestSignatureVerifier_SignedFirmware_VerifyFnPass(t *testing.T) {
	var gotDigest, gotSig, gotAlg, gotKeyID string
	fn := func(digestHex, signatureB64, alg, publicKeyID string) error {
		gotDigest, gotSig, gotAlg, gotKeyID = digestHex, signatureB64, alg, publicKeyID
		return nil
	}
	fw := &FirmwareVersion{
		SHA256Val:    "deadbeef",
		Signature:    "c2lnbmF0dXJl",
		SignatureAlg: "rsa-pss-sha256",
		PublicKeyID:  "vendor-key-1",
	}
	v := NewSignatureVerifier(nil, true, fn)
	require.NoError(t, v.Verify(fw))
	assert.Equal(t, "deadbeef", gotDigest)
	assert.Equal(t, "c2lnbmF0dXJl", gotSig)
	assert.Equal(t, "rsa-pss-sha256", gotAlg)
	assert.Equal(t, "vendor-key-1", gotKeyID)
}

func TestSignatureVerifier_SignedFirmware_VerifyFnFail(t *testing.T) {
	fn := func(_, _, _, _ string) error { return stderrors.New("bad signature") }
	fw := &FirmwareVersion{SHA256Val: "deadbeef", Signature: "c2lnbmF0dXJl"}
	v := NewSignatureVerifier(nil, false, fn)
	err := v.Verify(fw)
	require.Error(t, err)
	assert.Equal(t, FailureSignatureCheck, VerificationFailureCode(err))
}

func TestSignatureVerifier_SignedFirmware_NoDigest(t *testing.T) {
	// Signature anchored on sha256; a signed firmware lacking sha256 cannot be
	// verified. Note md5-only passes the integrity gate, so this reaches the
	// signature stage and must be rejected there.
	fn := func(_, _, _, _ string) error { return nil }
	fw := &FirmwareVersion{MD5Val: "abc", Signature: "c2lnbmF0dXJl"}
	v := NewSignatureVerifier(nil, false, fn)
	err := v.Verify(fw)
	require.Error(t, err)
	assert.Equal(t, FailureSignatureCheck, VerificationFailureCode(err))
}

func TestSignatureVerifier_IntegrityGateRunsFirst(t *testing.T) {
	// No digest at all → fails on the integrity gate before signature logic,
	// even with a passing verifyFn.
	fn := func(_, _, _, _ string) error { return nil }
	v := NewSignatureVerifier(nil, false, fn)
	err := v.Verify(&FirmwareVersion{Signature: "c2lnbmF0dXJl"})
	require.Error(t, err)
	assert.Equal(t, FailureIntegrityCheck, VerificationFailureCode(err))
}

func TestVerificationFailureCode_NonVerificationError(t *testing.T) {
	// A plain error degrades to the generic integrity code so the executor always
	// has a concrete failure reason to persist.
	assert.Equal(t, FailureIntegrityCheck, VerificationFailureCode(stderrors.New("boom")))
}

func TestConstantTimeDigestEqual(t *testing.T) {
	assert.True(t, ConstantTimeDigestEqual("DEADBEEF", "deadbeef"))
	assert.True(t, ConstantTimeDigestEqual(" abc ", "ABC"))
	assert.False(t, ConstantTimeDigestEqual("dead", "beef"))
	assert.False(t, ConstantTimeDigestEqual("dead", "deadbeef"))
	assert.False(t, ConstantTimeDigestEqual("", ""))
}

func TestFirmwareMetrics_NilSafe(t *testing.T) {
	var m *FirmwareMetrics
	// No panic on nil receiver.
	m.RecordPass(true)
	m.RecordPass(false)
	m.RecordFail(FailureSignatureCheck)

	// Constructed without a registry is also usable.
	m2 := NewFirmwareMetrics(nil)
	m2.RecordPass(false)
	m2.RecordFail(FailureIntegrityCheck)
}
