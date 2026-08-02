package paramsync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecideRunConvergence(t *testing.T) {
	tests := []struct {
		name string
		run  SyncRun
		want convergenceDecision
	}{
		{
			name: "all terminal and processed succeeds",
			run: SyncRun{
				Status:             RunStatusExecuting,
				ExpectedTaskCount:  20,
				TerminalTaskCount:  20,
				ProcessedTaskCount: 20,
			},
			want: convergenceDecision{Ready: true, Status: RunStatusSucceeded},
		},
		{
			name: "terminal failures converge failed",
			run: SyncRun{
				Status:             RunStatusExecuting,
				ExpectedTaskCount:  20,
				TerminalTaskCount:  20,
				ProcessedTaskCount: 20,
				FailedTaskCount:    2,
			},
			want: convergenceDecision{Ready: true, Status: RunStatusFailed},
		},
		{
			name: "cancelling run converges failed even without failed task",
			run: SyncRun{
				Status:             RunStatusCancelling,
				ExpectedTaskCount:  20,
				TerminalTaskCount:  20,
				ProcessedTaskCount: 20,
			},
			want: convergenceDecision{Ready: true, Status: RunStatusFailed},
		},
		{
			name: "processed results missing remains active",
			run: SyncRun{
				Status:             RunStatusExecuting,
				ExpectedTaskCount:  20,
				TerminalTaskCount:  20,
				ProcessedTaskCount: 19,
			},
			want: convergenceDecision{},
		},
		{
			name: "terminal run is idempotent no-op",
			run: SyncRun{
				Status:             RunStatusSucceeded,
				ExpectedTaskCount:  20,
				TerminalTaskCount:  20,
				ProcessedTaskCount: 20,
			},
			want: convergenceDecision{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, decideRunConvergence(tt.run))
		})
	}
}
