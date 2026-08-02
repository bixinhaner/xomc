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
			name: "undispatched zero task plan remains active",
			run: SyncRun{
				Status:             RunStatusPlanning,
				ExpectedTaskCount:  0,
				TerminalTaskCount:  0,
				ProcessedTaskCount: 0,
			},
			want: convergenceDecision{},
		},
		{
			name: "executing zero task plan remains active",
			run: SyncRun{
				Status:             RunStatusExecuting,
				ExpectedTaskCount:  0,
				TerminalTaskCount:  0,
				ProcessedTaskCount: 0,
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

func TestClassifyRunBlockReason(t *testing.T) {
	tests := []struct {
		name   string
		run    SyncRun
		counts authoritativeRunCounts
		want   convergenceBlockReason
	}{
		{
			name: "plan not dispatched",
			run: SyncRun{
				Status:            RunStatusEnqueuing,
				ExpectedTaskCount: 20,
			},
			counts: authoritativeRunCounts{},
			want:   convergenceBlockPlanNotDispatched,
		},
		{
			name: "executing plan only partially dispatched",
			run: SyncRun{
				Status:            RunStatusExecuting,
				ExpectedTaskCount: 20,
			},
			counts: authoritativeRunCounts{expected: 10, terminal: 10, processed: 10},
			want:   convergenceBlockPlanNotDispatched,
		},
		{
			name: "executing zero task plan not dispatched",
			run: SyncRun{
				Status: RunStatusExecuting,
			},
			counts: authoritativeRunCounts{},
			want:   convergenceBlockPlanNotDispatched,
		},
		{
			name: "terminal task missing durable result",
			run: SyncRun{
				Status:            RunStatusExecuting,
				ExpectedTaskCount: 20,
			},
			counts: authoritativeRunCounts{expected: 20, terminal: 20, processed: 19},
			want:   convergenceBlockTerminalResultMissing,
		},
		{
			name: "device task remains active",
			run: SyncRun{
				Status:            RunStatusExecuting,
				ExpectedTaskCount: 20,
			},
			counts: authoritativeRunCounts{expected: 20, terminal: 19, processed: 19},
			want:   convergenceBlockDeviceTaskActive,
		},
		{
			name: "ready run is not blocked",
			run: SyncRun{
				Status:            RunStatusExecuting,
				ExpectedTaskCount: 20,
			},
			counts: authoritativeRunCounts{expected: 20, terminal: 20, processed: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, classifyRunBlockReason(tt.run, tt.counts))
		})
	}
}
