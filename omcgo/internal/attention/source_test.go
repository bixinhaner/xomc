package attention

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/deviceaccess"
)

type capturingAlarmReader struct {
	filter alarm.AlarmFilter
	items  []model.Alarm
}

func (r *capturingAlarmReader) ListActive(_ context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	r.filter = filter
	return model.NewListResponse(r.items, int64(len(r.items)), filter.Page, filter.PageSize), nil
}

func TestAlarmAbnormalitySourceNormalizesLegacySeverityAndVisibleScope(t *testing.T) {
	groupID := uuid.New()
	alarmID := uuid.New()
	now := time.Now().UTC()
	reader := &capturingAlarmReader{items: []model.Alarm{{
		ID: alarmID, DeviceID: uuid.New(), DeviceSN: "SN-1", Severity: model.AlarmSeverity(31001),
		Status: model.AlarmActive, Description: "Cell unavailable", RaisedAt: now,
	}}}
	source := NewAlarmAbnormalitySource(reader, model.AlarmCritical)

	result, err := source.ListPrefix(context.Background(), Scope{VisibleGroups: []uuid.UUID{groupID}}, 2)
	require.NoError(t, err)
	require.NotNil(t, reader.filter.Severity)
	assert.Equal(t, model.AlarmCritical, *reader.filter.Severity)
	assert.Equal(t, []uuid.UUID{groupID}, reader.filter.VisibleGroups)
	require.Len(t, result.Items, 1)
	assert.Equal(t, KindActiveAlarm, result.Items[0].Kind)
	assert.Equal(t, "critical", result.Items[0].Severity)
	assert.Contains(t, result.Items[0].DetailRoute, alarmID.String())
	assert.Equal(t, []Action{ActionViewAlarm}, result.Items[0].AllowedActions)
}

type capturingCandidateReader struct {
	filter deviceaccess.ManagementFilter
	item   deviceaccess.CandidateItem
}

func (r *capturingCandidateReader) ListCandidates(_ context.Context, filter deviceaccess.ManagementFilter) ([]deviceaccess.CandidateItem, int64, error) {
	r.filter = filter
	return []deviceaccess.CandidateItem{r.item}, 1, nil
}

func TestCandidateSourceUsesPendingNonExpiredVisibleScope(t *testing.T) {
	groupID := uuid.New()
	candidateID := uuid.New()
	now := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	reader := &capturingCandidateReader{item: deviceaccess.CandidateItem{
		ID: candidateID, Carrier: "cmcc", SerialNumber: "SN-CANDIDATE", FirstSeenAt: now,
	}}
	source := NewCandidateSource(reader)
	source.now = func() time.Time { return now }

	result, err := source.ListPrefix(context.Background(), Scope{VisibleGroups: []uuid.UUID{groupID}}, 2)
	require.NoError(t, err)
	assert.Equal(t, "pending", reader.filter.Status)
	require.NotNil(t, reader.filter.ExpiresAfter)
	assert.Equal(t, now, *reader.filter.ExpiresAfter)
	assert.Equal(t, "first_seen_at", reader.filter.SortBy)
	assert.Equal(t, "asc", reader.filter.SortDir)
	assert.Equal(t, []uuid.UUID{groupID}, reader.filter.VisibleGroups)
	require.Len(t, result.Items, 1)
	assert.Contains(t, result.Items[0].DetailRoute, "candidateId="+candidateID.String())
}

type pagedCandidateReader struct {
	items     []deviceaccess.CandidateItem
	pageSizes []int
}

func (r *pagedCandidateReader) ListCandidates(_ context.Context, filter deviceaccess.ManagementFilter) ([]deviceaccess.CandidateItem, int64, error) {
	r.pageSizes = append(r.pageSizes, filter.PageSize)
	start := (filter.Page - 1) * filter.PageSize
	if start >= len(r.items) {
		return []deviceaccess.CandidateItem{}, int64(len(r.items)), nil
	}
	end := min(start+filter.PageSize, len(r.items))
	return r.items[start:end], int64(len(r.items)), nil
}

func TestCandidateSourceKeepsPageSizeStableAcrossPrefixBoundary(t *testing.T) {
	items := make([]deviceaccess.CandidateItem, 250)
	for index := range items {
		items[index] = deviceaccess.CandidateItem{
			ID: uuid.New(), SerialNumber: "candidate", FirstSeenAt: time.Unix(int64(index), 0).UTC(),
		}
	}
	reader := &pagedCandidateReader{items: items}
	source := NewCandidateSource(reader)

	result, err := source.ListPrefix(context.Background(), Scope{}, 250)

	require.NoError(t, err)
	require.Len(t, result.Items, 250)
	require.Equal(t, []int{200, 200}, reader.pageSizes,
		"changing page_size between pages changes SQL offsets and can duplicate or skip candidates")
	seen := make(map[string]struct{}, len(result.Items))
	for _, item := range result.Items {
		seen[item.SourceID] = struct{}{}
	}
	require.Len(t, seen, 250)

	reader.pageSizes = nil
	window, err := source.ListWindow(context.Background(), Scope{}, 190, 20)
	require.NoError(t, err)
	require.Len(t, window.Items, 20)
	require.Equal(t, items[190].ID.String(), window.Items[0].SourceID)
	require.Equal(t, items[209].ID.String(), window.Items[19].SourceID)
	require.Equal(t, []int{20, 20}, reader.pageSizes,
		"an unaligned offset must use adjacent fixed-size pages")
}
