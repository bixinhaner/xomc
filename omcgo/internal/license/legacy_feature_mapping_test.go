package license

import (
	"encoding/json"
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

// issue #311: 现网无独立 Monitor 页/设备激活/射频开关 → 特性列表隐藏展示，
// 但授权树必须仍按 code 授权（设备列表等菜单由 *_MONITOR 门禁，不能因隐藏而失效）。
func TestLoadLegacyFeatureMappingHiddenStillAuthorized(t *testing.T) {
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)

	features := mapping.Normalize(nil, []string{"CODE_ENB_MONITOR", "CODE_GNB_ACTIVE", "CODE_ENB_RF_ENABLE", "CODE_DASHBOARD"})
	require.Len(t, features, 4)
	hidden := 0
	for _, f := range features {
		if f.FeatureCode == "CODE_DASHBOARD" {
			require.False(t, f.Hidden, "dashboard feature stays visible")
			continue
		}
		require.True(t, f.Hidden, "feature %s should be hidden from display", f.FeatureCode)
		require.True(t, f.Licensed, "hidden feature %s must stay licensed", f.FeatureCode)
		hidden++
	}
	require.Equal(t, 3, hidden)

	tree := mapping.AuthorizationTree(nil, []string{"CODE_ENB_MONITOR"})
	raw, err := json.Marshal(tree)
	require.NoError(t, err)
	require.True(t, HasFeature(FeatureList(raw), "eNB", "Monitor"),
		"hidden monitor code must still authorize eNB.Monitor")
}

func TestLoadLegacyFeatureMappingUPSAuthorizesMonitor(t *testing.T) {
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)

	features := mapping.Normalize([]string{"81"}, nil)
	require.Len(t, features, 1)
	require.Equal(t, "CODE_UPS", features[0].FeatureCode)

	tree := mapping.AuthorizationTree([]string{"81"}, nil)
	raw, err := json.Marshal(tree)
	require.NoError(t, err)
	require.True(t, HasFeature(FeatureList(raw), "UPS", "Monitor"),
		"CODE_UPS should authorize UPS.Monitor through UPS=All")
}
