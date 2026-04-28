package syslog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ---------------------------------------------------------------------------
// Service-layer tests reuse the package-private mockSyslogRepo defined in
// handler_test.go (same package). Keeping the mock single-source-of-truth
// avoids drift between handler and service test assertions.
// ---------------------------------------------------------------------------

func newSvc(repo SyslogRepository) *Service {
	return NewService(repo, zap.NewNop())
}

// ---------------------------------------------------------------------------
// ListSystemLogs
// ---------------------------------------------------------------------------

func TestService_ListSystemLogs_Success(t *testing.T) {
	expected := sampleSystemLogs()
	repo := &mockSyslogRepo{systemLogs: expected}

	got, err := newSvc(repo).ListSystemLogs(context.Background(), SystemLogFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_ListSystemLogs_InvalidRange(t *testing.T) {
	repo := &mockSyslogRepo{systemLogs: sampleSystemLogs()}

	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	start := end.Add(time.Hour) // start > end → invalid
	filter := SystemLogFilter{
		StartTime: &start,
		EndTime:   &end,
	}

	got, err := newSvc(repo).ListSystemLogs(context.Background(), filter)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestService_ListSystemLogs_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockSyslogRepo{systemLogsErr: repoErr}

	got, err := newSvc(repo).ListSystemLogs(context.Background(), SystemLogFilter{})

	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, repoErr), "service should wrap repo error with %%w")
}

// ---------------------------------------------------------------------------
// ListNEMessageLogs
// ---------------------------------------------------------------------------

func TestService_ListNEMessageLogs_Success(t *testing.T) {
	expected := sampleNEMessageLogs()
	repo := &mockSyslogRepo{neMsgLogs: expected}

	sn := "DEV001"
	filter := NEMessageLogFilter{DeviceSN: &sn}

	got, err := newSvc(repo).ListNEMessageLogs(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
	require.NotNil(t, repo.lastNEMessageLogFilter.DeviceSN)
	assert.Equal(t, sn, *repo.lastNEMessageLogFilter.DeviceSN)
}

func TestService_ListNEMessageLogs_InvalidRange(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}

	end := time.Now()
	start := end.Add(2 * time.Hour) // start > end
	filter := NEMessageLogFilter{
		StartTime: &start,
		EndTime:   &end,
	}

	got, err := newSvc(repo).ListNEMessageLogs(context.Background(), filter)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// ---------------------------------------------------------------------------
// ListNEMessagesByDevice
// ---------------------------------------------------------------------------

func TestService_ListNEMessagesByDevice_Success(t *testing.T) {
	expected := sampleNEMessageLogs()
	repo := &mockSyslogRepo{neMsgLogs: expected}

	deviceID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	window := TimeWindow{Start: &start, End: &end}
	list := model.ListRequest{Page: 2, PageSize: 50}

	got, err := newSvc(repo).ListNEMessagesByDevice(context.Background(), deviceID, window, list)

	require.NoError(t, err)
	assert.Equal(t, expected, got)

	// Verify the filter was assembled correctly.
	require.NotNil(t, repo.lastNEMessageLogFilter.DeviceID)
	assert.Equal(t, deviceID, *repo.lastNEMessageLogFilter.DeviceID)
	require.NotNil(t, repo.lastNEMessageLogFilter.StartTime)
	assert.Equal(t, start, *repo.lastNEMessageLogFilter.StartTime)
	require.NotNil(t, repo.lastNEMessageLogFilter.EndTime)
	assert.Equal(t, end, *repo.lastNEMessageLogFilter.EndTime)
	assert.Equal(t, 2, repo.lastNEMessageLogFilter.Page)
	assert.Equal(t, 50, repo.lastNEMessageLogFilter.PageSize)
}

func TestService_ListNEMessagesByDevice_NilDeviceID(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}

	got, err := newSvc(repo).ListNEMessagesByDevice(context.Background(), uuid.Nil, TimeWindow{}, model.ListRequest{})

	require.Error(t, err)
	assert.Nil(t, got)
	var bErr *commonerrors.BusinessError
	require.True(t, errors.As(err, &bErr))
	assert.Equal(t, 4001, bErr.Code)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestService_ListNEMessagesByDevice_InvalidWindow(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}

	end := time.Now()
	start := end.Add(time.Hour) // start after end
	window := TimeWindow{Start: &start, End: &end}

	got, err := newSvc(repo).ListNEMessagesByDevice(context.Background(), uuid.New(), window, model.ListRequest{})

	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// ---------------------------------------------------------------------------
// Misc
// ---------------------------------------------------------------------------

func TestNewService_NilLogger(t *testing.T) {
	// Should not panic when logger is nil.
	svc := NewService(&mockSyslogRepo{}, nil)
	require.NotNil(t, svc)
	require.NotNil(t, svc.logger)
}

func TestTimeWindow_Validate(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	cases := []struct {
		name    string
		w       TimeWindow
		wantErr bool
	}{
		{"both nil", TimeWindow{}, false},
		{"only start", TimeWindow{Start: &earlier}, false},
		{"only end", TimeWindow{End: &now}, false},
		{"start before end", TimeWindow{Start: &earlier, End: &now}, false},
		{"start after end", TimeWindow{Start: &now, End: &earlier}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.w.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
