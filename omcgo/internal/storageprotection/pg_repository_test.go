package storageprotection

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPolicyContract(t *testing.T) {
	policy := DefaultPolicy()

	require.Equal(t, DefaultPolicyID, policy.ID)
	require.Equal(t, TargetFilesystem, policy.TargetType)
	require.Equal(t, UnifiedStorageTargetID, policy.TargetID)
	require.Equal(t, WriteScopeAll, policy.WriteScope)
	require.True(t, policy.Enabled)
	require.Equal(t, 80, policy.WarnUsedPercent)
	require.Equal(t, 85, policy.RecoverUsedPercent)
	require.Equal(t, 90, policy.BlockUsedPercent)
	require.Equal(t, 30, policy.CheckIntervalSeconds)
	require.Equal(t, UnknownAllowWithAlarm, policy.UnknownBehavior)
	require.Equal(t, StateNormal, policy.CurrentState)
	require.Equal(t, 0, policy.StateObservations)
	require.Equal(t, "system", policy.UpdatedBy)
	require.Equal(t, int64(1), policy.Version)
	require.NoError(t, validatePolicy(&policy))
}

func TestPolicyLookupScopes(t *testing.T) {
	tests := []struct {
		name  string
		scope WriteScope
		want  []WriteScope
	}{
		{name: "all scope queries all policy", scope: WriteScopeAll, want: []WriteScope{WriteScopeAll}},
		{name: "specific scope falls back to all", scope: WriteScopeLog, want: []WriteScope{WriteScopeLog, WriteScopeAll}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policyLookupScopes(tt.scope); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("policyLookupScopes(%q) = %#v, want %#v", tt.scope, got, tt.want)
			}
		})
	}
}
