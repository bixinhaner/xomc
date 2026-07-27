package mml

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type captureDeviceTaskResultLister struct {
	sourceIDs []string
	rows      []DeviceTaskResultRowView
}

func (l *captureDeviceTaskResultLister) ListResultsBySourceID(
	_ context.Context,
	sourceID string,
	_, _ int,
) ([]DeviceTaskResultRowView, int64, error) {
	l.sourceIDs = append(l.sourceIDs, sourceID)
	return l.rows, int64(len(l.rows)), nil
}

func TestGetTaskResultsPeriodicParentUsesLatestChildResults(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	parentIDCopy := parentID
	parentTask := &MMLTask{
		ID:              parentID,
		ExecuteType:     ExecutePeriodic,
		ExecuteMode:     TaskExecuteModeDeviceBound,
		ProductResolved: true,
		Commands: []map[string]interface{}{
			{
				"command_code":   "RAW LST",
				"operation_type": "LST",
				"plan_raw_line":  "LST Device.Test.Value;SN-1",
			},
		},
	}
	childTask := &MMLTask{
		ID:               childID,
		ExecuteType:      ExecuteImmediate,
		ExecuteMode:      TaskExecuteModeDeviceBound,
		PeriodicParentID: &parentIDCopy,
		ProductResolved:  true,
		Commands:         parentTask.Commands,
	}
	repo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			require.Equal(t, parentID, id)
			return parentTask, nil
		},
		getLatestPeriodicChildFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			require.Equal(t, parentID, id)
			return childTask, nil
		},
	}
	lister := &captureDeviceTaskResultLister{
		rows: []DeviceTaskResultRowView{
			{
				DeviceSN:     "SN-1",
				Status:       "completed",
				ErrorCode:    0,
				CommandIndex: 0,
				DeviceIndex:  0,
			},
		},
	}
	svc := NewService(&mockCommandRepo{}, nil, repo, nil, nil, zap.NewNop())
	svc.SetDeviceTaskResultLister(lister)

	resp, err := svc.GetTaskResults(context.Background(), parentID, 1, 20)

	require.NoError(t, err)
	require.Equal(t, []string{childID.String()}, lister.sourceIDs)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "SN-1", resp.Items[0]["device_sn"])
	assert.Equal(t, "RAW LST", resp.Items[0]["command_code"])
	assert.Equal(t, "LST Device.Test.Value", resp.Items[0]["mml_script"])
}
