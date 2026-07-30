package storageprotection

import (
	"reflect"
	"testing"
)

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
