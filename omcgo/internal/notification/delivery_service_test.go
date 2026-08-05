package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

type deliveryAuthorizationStub struct {
	delivery AuthorizedDelivery
	err      error
	id       uuid.UUID
	now      time.Time
}

type deliveryQueryStub struct {
	records     []DeliveryRecord
	attempts    []DomainDeliveryAttempt
	filter      DeliveryFilter
	getGrants   []model.DeviceVisibilityGrant
	retryIDs    []uuid.UUID
	retryGrants []model.DeviceVisibilityGrant
	retryReason string
	retryActor  string
}

func (s *deliveryQueryStub) ListDeliveries(_ context.Context, filter DeliveryFilter) ([]DeliveryRecord, error) {
	s.filter = filter
	return s.records, nil
}

func (s *deliveryQueryStub) GetDelivery(_ context.Context, _ uuid.UUID, grants []model.DeviceVisibilityGrant) (*DeliveryRecord, error) {
	s.getGrants = grants
	if len(s.records) == 0 {
		return nil, ErrDeliveryNotFound
	}
	return &s.records[0], nil
}

func (s *deliveryQueryStub) ListDeliveryAttempts(context.Context, uuid.UUID) ([]DomainDeliveryAttempt, error) {
	return s.attempts, nil
}

func (s *deliveryQueryStub) RetryDeadLetters(_ context.Context, ids []uuid.UUID, grants []model.DeviceVisibilityGrant, reason, actor string, _ time.Time) (int, error) {
	s.retryIDs, s.retryGrants, s.retryReason, s.retryActor = ids, grants, reason, actor
	return len(ids), nil
}

type deliveryPermissionStub struct{ grants []model.DeviceVisibilityGrant }

func (s deliveryPermissionStub) GetUserVisibleDeviceGrants(context.Context, uuid.UUID, bool) ([]model.DeviceVisibilityGrant, error) {
	return s.grants, nil
}

func (s *deliveryAuthorizationStub) AuthorizeSend(_ context.Context, id uuid.UUID, now time.Time) (AuthorizedDelivery, error) {
	s.id, s.now = id, now
	return s.delivery, s.err
}

func TestDeliveryService_AuthorizeSendDelegatesWithStableTime(t *testing.T) {
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	id := uuid.New()
	repository := &deliveryAuthorizationStub{delivery: AuthorizedDelivery{DomainDelivery: DomainDelivery{ID: id}}}
	service := NewDeliveryService(repository)
	service.now = func() time.Time { return now }

	delivery, err := service.AuthorizeSend(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, id, delivery.ID)
	require.Equal(t, id, repository.id)
	require.Equal(t, now, repository.now)
}

func TestEvaluateDeliveryAuthorization_RequiresLeaseOccurrenceAndClosedChannel(t *testing.T) {
	now := time.Now().UTC()
	snapshot := authorizedSnapshot(now, DispatchKindInitial, "active")
	require.Nil(t, evaluateDeliveryAuthorization(snapshot, now))

	tests := []struct {
		name   string
		mutate func(*deliveryAuthorizationSnapshot)
		want   error
		flow   string
		reason string
	}{
		{name: "expired lease", mutate: func(s *deliveryAuthorizationSnapshot) { expired := now; s.Delivery.LeaseExpiresAt = &expired }, want: ErrDeliveryLeaseInvalid},
		{name: "acknowledged", mutate: func(s *deliveryAuthorizationSnapshot) { s.OccurrenceStatus = "acknowledged" }, want: ErrDeliveryOccurrenceFenced, flow: "cancelled", reason: "occurrence_fenced"},
		{name: "stale generation", mutate: func(s *deliveryAuthorizationSnapshot) { s.OccurrenceGeneration++ }, want: ErrDeliveryOccurrenceFenced, flow: "cancelled", reason: "occurrence_fenced"},
		{name: "disabled channel", mutate: func(s *deliveryAuthorizationSnapshot) { s.ChannelEnabled = false }, want: ErrDeliveryChannelUnavailable, flow: "suppressed", reason: "channel_disabled"},
		{name: "open circuit", mutate: func(s *deliveryAuthorizationSnapshot) { s.CircuitState = "open" }, want: ErrDeliveryChannelUnavailable, flow: "suppressed", reason: "channel_circuit_open"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := snapshot
			test.mutate(&candidate)
			rejection := evaluateDeliveryAuthorization(candidate, now)
			require.NotNil(t, rejection)
			require.True(t, errors.Is(rejection.err, test.want))
			require.Equal(t, test.flow, rejection.flowState)
			require.Equal(t, test.reason, rejection.reason)
		})
	}
}

func TestEvaluateDeliveryAuthorization_RecoveryRequiresClearedOccurrence(t *testing.T) {
	now := time.Now().UTC()
	require.Nil(t, evaluateDeliveryAuthorization(authorizedSnapshot(now, DispatchKindRecovery, "cleared"), now))
	rejection := evaluateDeliveryAuthorization(authorizedSnapshot(now, DispatchKindRecovery, "active"), now)
	require.ErrorIs(t, rejection.err, ErrDeliveryOccurrenceFenced)
}

func TestEvaluateDeliveryAuthorization_DigestSurvivesClearGenerationChange(t *testing.T) {
	now := time.Now().UTC()
	snapshot := authorizedSnapshot(now, DispatchKindDigest, "cleared")
	snapshot.OccurrenceGeneration++
	require.Nil(t, evaluateDeliveryAuthorization(snapshot, now))
}

func TestDeliveryService_ListAppliesGrantsAndNeverReturnsCiphertext(t *testing.T) {
	groupID := uuid.New()
	query := &deliveryQueryStub{records: []DeliveryRecord{{
		DomainDelivery: DomainDelivery{
			ID: uuid.New(), AddressCiphertext: []byte("secret@example.com"),
			RecipientFingerprint: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		DeviceID: uuid.New(), DeviceSN: "SN001", Technology: "nr",
	}}}
	service := NewDeliveryService(&deliveryAuthorizationStub{})
	service.SetQueryRepository(query)
	service.SetPermissionResolver(deliveryPermissionStub{grants: []model.DeviceVisibilityGrant{{
		GroupIDs: []uuid.UUID{groupID}, Technologies: []model.Technology{model.TechNR},
	}}})

	views, err := service.List(context.Background(), uuid.New(), false, DeliveryFilter{Limit: 500})
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, "recipient-…bbccddee", views[0].MaskedAddress)
	require.NotContains(t, views[0].MaskedAddress, "secret")
	require.Equal(t, 200, query.filter.Limit)
	require.Equal(t, groupID, query.filter.Grants[0].GroupIDs[0])
}

func TestDeliveryService_RetryRequiresBoundedIDsReasonAndVisibility(t *testing.T) {
	query := &deliveryQueryStub{}
	service := NewDeliveryService(&deliveryAuthorizationStub{})
	service.SetQueryRepository(query)
	service.SetPermissionResolver(deliveryPermissionStub{grants: []model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{uuid.New()}}}})

	_, err := service.Retry(context.Background(), uuid.New(), false, nil, "fixed", "operator")
	require.Error(t, err)
	_, err = service.Retry(context.Background(), uuid.New(), false, []uuid.UUID{uuid.New()}, "  ", "operator")
	require.Error(t, err)

	id := uuid.New()
	count, err := service.Retry(context.Background(), uuid.New(), false, []uuid.UUID{id}, " SMTP configuration verified ", "operator")
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, []uuid.UUID{id}, query.retryIDs)
	require.Equal(t, "SMTP configuration verified", query.retryReason)
	require.Equal(t, "operator", query.retryActor)
	require.NotEmpty(t, query.retryGrants)
}

func authorizedSnapshot(now time.Time, kind, status string) deliveryAuthorizationSnapshot {
	lease := now.Add(time.Minute)
	return deliveryAuthorizationSnapshot{
		Delivery: DomainDelivery{
			ID: uuid.New(), FlowState: "sending", DispatchKind: kind, LeaseExpiresAt: &lease,
			OccurrenceVersion: 2, ScheduleGeneration: 3,
		},
		OccurrenceStatus: status, OccurrenceVersion: 2, OccurrenceGeneration: 3,
		ChannelEnabled: true, CircuitState: "closed",
	}
}
