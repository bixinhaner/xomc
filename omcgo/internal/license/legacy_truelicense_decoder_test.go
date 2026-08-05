package license

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeLegacyTrueLicenseRepositorySample(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "omc.lic"))
	if os.IsNotExist(err) {
		t.Skip("repository legacy license sample is not available")
	}
	require.NoError(t, err)

	artifact, err := DecodeLegacyTrueLicense(raw, "bcb9omc6")
	require.NoError(t, err)
	require.Equal(t, "SHA1withDSA", artifact.SignatureAlgorithm)
	require.Equal(t, "US-ASCII/Base64", artifact.SignatureEncoding)
	require.NotEmpty(t, artifact.Signature)
	require.NotEmpty(t, artifact.EncodedContent)
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "NO2022-03-14002"}, artifact.Extra["licenseNum"])
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "10000"}, artifact.Extra["maxDeviceCapability"])
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "10000"}, artifact.Extra["cpeMaxNum"])
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "1000"}, artifact.Extra["egwMaxNum"])
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "2"}, artifact.Extra["productType"])
	require.Equal(t, LegacyXMLValue{Kind: "string", Value: "user"}, artifact.StandardFields["consumerType"])

	signature, err := artifact.DecodeLegacySignature()
	require.NoError(t, err)
	require.Len(t, signature, 46)
}

func TestVerifyLegacyTrueLicenseRepositorySample(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "omc.lic"))
	if os.IsNotExist(err) {
		t.Skip("repository legacy license sample is not available")
	}
	require.NoError(t, err)
	keyStore, err := os.ReadFile(filepath.Join("..", "..", "..", "license-run-time", "keystore", "omcPublicKey.store"))
	require.NoError(t, err)
	artifact, err := DecodeLegacyTrueLicense(raw, "bcb9omc6")
	require.NoError(t, err)
	require.NoError(t, VerifyLegacyTrueLicenseSignature(
		artifact,
		keyStore,
		"bcb9omc6",
		"omcPublicKey",
	))
}

func TestDecodeLegacyTrueLicenseRejectsWrongPasswordAndTampering(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "omc.lic"))
	if os.IsNotExist(err) {
		t.Skip("repository legacy license sample is not available")
	}
	require.NoError(t, err)

	_, err = DecodeLegacyTrueLicense(raw, "wrong-password")
	require.Error(t, err)

	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)/2] ^= 0x01
	_, err = DecodeLegacyTrueLicense(tampered, "bcb9omc6")
	require.Error(t, err)
}

func TestDecodeLegacyTrueLicenseRepositorySamples(t *testing.T) {
	root := filepath.Join("..", "..", "..", "omcmb", "original-omc", "OMCWebServer", "License_100y")
	var samples []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info != nil && !info.IsDir() && strings.EqualFold(info.Name(), "omc.lic") {
			samples = append(samples, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		t.Skip("repository legacy license corpus is not available")
	}
	require.NoError(t, err)
	require.NotEmpty(t, samples)

	for _, path := range samples {
		raw, readErr := os.ReadFile(path)
		require.NoError(t, readErr, path)
		if _, decodeErr := DecodeLegacyTrueLicense(raw, "bcb9omc6"); decodeErr != nil {
			t.Errorf("decode %s: %v", path, decodeErr)
		}
	}
}
