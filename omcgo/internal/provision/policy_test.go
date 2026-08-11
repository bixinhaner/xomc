package provision

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnabledPolicyModulesPreservesDispatchOrderAndSkipsDisabledModules(t *testing.T) {
	tests := []struct {
		name   string
		policy *PlugAndPlayPolicy
		want   []policyModule
	}{
		{
			name: "all modules",
			policy: &PlugAndPlayPolicy{
				UpgradeEnabled: true, LicenseEnabled: true, SelfConfigEnabled: true,
			},
			want: []policyModule{policyModuleUpgrade, policyModuleLicense, policyModuleSelfConfig},
		},
		{
			name: "upgrade then self config",
			policy: &PlugAndPlayPolicy{
				UpgradeEnabled: true, SelfConfigEnabled: true,
			},
			want: []policyModule{policyModuleUpgrade, policyModuleSelfConfig},
		},
		{
			name: "license then self config",
			policy: &PlugAndPlayPolicy{
				LicenseEnabled: true, SelfConfigEnabled: true,
			},
			want: []policyModule{policyModuleLicense, policyModuleSelfConfig},
		},
		{
			name:   "license only",
			policy: &PlugAndPlayPolicy{LicenseEnabled: true},
			want:   []policyModule{policyModuleLicense},
		},
		{
			name:   "none",
			policy: &PlugAndPlayPolicy{},
			want:   []policyModule{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, enabledPolicyModules(tt.policy))
		})
	}
}

func TestNormalizePolicyProductClassesSupportsMultipleAndLegacyValues(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		ProductClass:   "legacy-is-replaced",
		ProductClasses: []string{" FAP/A ", "FAP/B", "FAP/A", ""},
	}

	normalizePolicyProductClasses(policy)

	assert.Equal(t, []string{"FAP/A", "FAP/B"}, policy.ProductClasses)
	assert.Equal(t, "FAP/A", policy.ProductClass)
	assert.True(t, policySupportsProductClass(policy, "FAP/B"))
	assert.False(t, policySupportsProductClass(policy, "FAP/C"))

	legacy := &PlugAndPlayPolicy{ProductClass: " FAP/LEGACY "}
	normalizePolicyProductClasses(legacy)
	assert.Equal(t, []string{"FAP/LEGACY"}, legacy.ProductClasses)
	assert.Equal(t, "FAP/LEGACY", legacy.ProductClass)
}

func TestPolicySupportsProductNameUsesNamesAndLegacyFallback(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		ProductNames:   []string{" XG-500 ", "XG-800"},
		ProductClasses: []string{"legacy-class"},
	}

	assert.True(t, policySupportsProductName(policy, "xg-500"))
	assert.False(t, policySupportsProductName(policy, "legacy-class"))
	assert.True(t, policySupportsProductName(&PlugAndPlayPolicy{ProductClass: " FAP/LEGACY "}, "fap/legacy"))
}
