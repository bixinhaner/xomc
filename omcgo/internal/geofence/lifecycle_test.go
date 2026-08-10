package geofence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDefinitionTransitionAllowsOnlyDocumentedLifecycle(t *testing.T) {
	tests := []struct {
		current DefinitionStatus
		target  DefinitionStatus
	}{
		{current: DefinitionStatusEnabled, target: DefinitionStatusDisabled},
		{current: DefinitionStatusDisabled, target: DefinitionStatusEnabled},
		{current: DefinitionStatusDisabled, target: DefinitionStatusArchived},
	}
	for _, tt := range tests {
		require.NoError(t, validateDefinitionTransition(tt.current, tt.target))
	}
}

func TestValidateDefinitionTransitionRejectsSkippedOrRepeatedStates(t *testing.T) {
	tests := []struct {
		current DefinitionStatus
		target  DefinitionStatus
	}{
		{current: DefinitionStatusDraft, target: DefinitionStatusDisabled},
		{current: DefinitionStatusDraft, target: DefinitionStatusArchived},
		{current: DefinitionStatusEnabled, target: DefinitionStatusArchived},
		{current: DefinitionStatusEnabled, target: DefinitionStatusEnabled},
		{current: DefinitionStatusArchived, target: DefinitionStatusEnabled},
	}
	for _, tt := range tests {
		err := validateDefinitionTransition(tt.current, tt.target)
		require.Error(t, err)
	}
}

func TestLifecyclePreviewFingerprintIsDeterministicAndStateSensitive(t *testing.T) {
	geofenceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	versionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	impact := LifecycleImpact{
		GeofenceID:       geofenceID,
		CurrentVersionID: &versionID,
		CurrentStatus:    DefinitionStatusEnabled,
		TargetStatus:     DefinitionStatusDisabled,
		BindingCount:     5,
		DeviceCount:      5,
	}

	first := lifecyclePreviewFingerprint(impact)
	second := lifecyclePreviewFingerprint(impact)
	require.NotEmpty(t, first)
	assert.Equal(t, first, second)

	impact.DeviceCount = 6
	assert.NotEqual(t, first, lifecyclePreviewFingerprint(impact))
}

func TestLifecyclePreviewFingerprintChangesWithActiveBatchJobs(t *testing.T) {
	base := LifecycleImpact{
		GeofenceID:    uuid.New(),
		CurrentStatus: DefinitionStatusDisabled,
	}
	withJob := base
	withJob.ActiveBatchJobCount = 1

	require.NotEqual(
		t,
		lifecyclePreviewFingerprint(base),
		lifecyclePreviewFingerprint(withJob),
	)
}

func TestLifecycleImpactQueryBindsGeofenceAfterActiveJobFilters(t *testing.T) {
	geofenceID := uuid.New()
	query, args, err := buildLifecycleImpactQuery(geofenceID)
	require.NoError(t, err)

	assert.Contains(t, query, "j.job_type = $1")
	assert.Contains(t, query, "j.status IN ($2,$3)")
	assert.Contains(t, query, "WHERE d.id = $4")
	require.Equal(t, []any{
		ManualBindJobType,
		"pending",
		"running",
		geofenceID.String(),
	}, args)
}

func TestGetLifecycleImpactDoesNotExposePostgresError(t *testing.T) {
	_, err := getLifecycleImpact(
		context.Background(),
		lifecycleImpactErrorQuerier{row: lifecycleImpactErrorRow{err: &pgconn.PgError{
			Code:           "42883",
			Message:        "operator does not exist: uuid = text",
			ConstraintName: "sensitive_constraint",
		}}},
		uuid.New(),
	)

	require.ErrorIs(t, err, commonerrors.ErrInternal)
	assert.NotContains(t, err.Error(), "uuid = text")
	assert.NotContains(t, err.Error(), "sensitive_constraint")
}

type lifecycleImpactErrorQuerier struct {
	row pgx.Row
}

func (q lifecycleImpactErrorQuerier) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	return q.row
}

type lifecycleImpactErrorRow struct {
	err error
}

func (r lifecycleImpactErrorRow) Scan(...any) error {
	return r.err
}

func TestValidateBindingTransitionAllowsOnlyDocumentedLifecycle(t *testing.T) {
	tests := []struct {
		name    string
		current BindingStatus
		target  BindingStatus
		wantErr bool
	}{
		{
			name:    "active binding may be suspended",
			current: BindingStatusActive,
			target:  BindingStatusSuspended,
		},
		{
			name:    "suspended binding may be resumed",
			current: BindingStatusSuspended,
			target:  BindingStatusActive,
		},
		{
			name:    "active binding may be removed",
			current: BindingStatusActive,
			target:  BindingStatusRemoved,
		},
		{
			name:    "suspended binding may be removed",
			current: BindingStatusSuspended,
			target:  BindingStatusRemoved,
		},
		{
			name:    "pending binding cannot bypass activation workflow",
			current: BindingStatusPending,
			target:  BindingStatusActive,
			wantErr: true,
		},
		{
			name:    "removed binding is terminal",
			current: BindingStatusRemoved,
			target:  BindingStatusActive,
			wantErr: true,
		},
		{
			name:    "same-state transition is rejected",
			current: BindingStatusActive,
			target:  BindingStatusActive,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBindingTransition(tt.current, tt.target)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateBindingResumeParentRequiresEnabledPublishedDefinition(
	t *testing.T,
) {
	versionID := uuid.New()

	require.NoError(t, validateBindingResumeParent(
		DefinitionStatusEnabled,
		&versionID,
	))
	require.Error(t, validateBindingResumeParent(
		DefinitionStatusDisabled,
		&versionID,
	))
	require.Error(t, validateBindingResumeParent(
		DefinitionStatusEnabled,
		nil,
	))
}
