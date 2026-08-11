package license

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapLegacyTrueLicenseRepositorySample(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "omc.lic"))
	if os.IsNotExist(err) {
		t.Skip("repository legacy license sample is not available")
	}
	require.NoError(t, err)
	artifact, err := DecodeLegacyTrueLicense(raw, "bcb9omc6")
	require.NoError(t, err)

	claims, err := MapLegacyTrueLicense(artifact)
	require.NoError(t, err)
	require.Equal(t, "NO2022-03-14002", claims.LicenseID)
	require.Equal(t, SystemLicenseTypeCommercial, claims.LicenseType)
	require.Equal(t, 10000, claims.DevicesSupport["eNB"])
	require.Equal(t, 10000, claims.DevicesSupport["gNB"])
	require.Equal(t, 10000, claims.DevicesSupport["CPE"])
	require.Equal(t, 1000, claims.DevicesSupport["EGW"])
	require.Equal(t, 1000, claims.DevicesSupport["UPS"])
	require.NotEmpty(t, claims.FeatureIDs)
	require.NotEmpty(t, claims.FeatureCodes)
	require.Equal(t, "2049-01-01 00:00:00 +0000 UTC", claims.OMCNotAfter.String())
}
