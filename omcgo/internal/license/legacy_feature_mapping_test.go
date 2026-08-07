package license

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLegacyFeatureMappingResolvesDHCP(t *testing.T) {
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)

	features := mapping.Normalize([]string{"80"}, nil)
	require.Len(t, features, 1)
	require.Equal(t, "CODE_TOOL_DHCP", features[0].FeatureCode)
	require.Equal(t, "DHCP", features[0].NameZH)
	require.Equal(t, "高级 / DHCP", features[0].Path)
	require.True(t, features[0].Recognized)
	require.True(t, features[0].Licensed)
}

func TestLoadLegacyFeatureMappingKeepsUnknownCode(t *testing.T) {
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)

	features := mapping.Normalize(nil, []string{"CODE_NOT_IN_OLD_CATALOG"})
	require.Len(t, features, 1)
	require.Equal(t, "CODE_NOT_IN_OLD_CATALOG", features[0].FeatureCode)
	require.Equal(t, "unknown", features[0].Source)
	require.False(t, features[0].Recognized)
	require.False(t, features[0].Licensed)
}

func TestLoadLegacyFeatureMappingSkipsCatalogNodes(t *testing.T) {
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)

	features := mapping.Normalize([]string{"5", "99", "100", "80"}, nil)
	require.Len(t, features, 1)
	require.Equal(t, "DHCP", features[0].NameZH)
}
