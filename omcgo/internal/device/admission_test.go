package device

import "testing"

func TestAccessDecisionAllowsRegistration(t *testing.T) {
	tests := map[string]struct {
		state string
		want  bool
	}{
		"accepted":        {state: AccessDecisionAccepted, want: true},
		"bypassed":        {state: AccessDecisionBypassed, want: true},
		"review required": {state: AccessDecisionReviewRequired, want: false},
		"rejected":        {state: AccessDecisionRejected, want: false},
		"revoked":         {state: AccessDecisionRevoked, want: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := accessDecisionAllowsRegistration(AccessDecision{State: test.state}); got != test.want {
				t.Fatalf("accessDecisionAllowsRegistration(%q) = %t, want %t", test.state, got, test.want)
			}
		})
	}
}
