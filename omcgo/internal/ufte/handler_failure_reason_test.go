package ufte

import "testing"

func TestTranslateFailureReason_OperatorTerminatedCode(t *testing.T) {
	if got := translateFailureReason("OPERATOR_TERMINATED"); got != "被操作者终止" {
		t.Fatalf("translate OPERATOR_TERMINATED = %q, want %q", got, "被操作者终止")
	}
}
