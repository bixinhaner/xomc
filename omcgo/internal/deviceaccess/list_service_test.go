package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type listStoreStub struct {
	carrier string
	entry   CompiledListEntry
}

func (s *listStoreStub) UpsertListEntryAndQueue(_ context.Context, carrier string, entry CompiledListEntry) error {
	s.carrier = carrier
	s.entry = entry
	return nil
}

func TestListServicePersistsChangeAndQueuesOnlyChangedSerialNumber(t *testing.T) {
	entries := &listStoreStub{}
	service := NewListService(entries)

	err := service.UpsertAndReevaluate(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1",
	}, CompiledListEntry{
		Type:          ListEntryTypeDeny,
		IdentityType:  IdentityTypeSerialNumber,
		IdentityValue: "SN-1",
		Status:        ListEntryStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, "cmcc", entries.carrier)
	require.Equal(t, "SN-1", entries.entry.IdentityValue)
}

func TestListServiceRejectsNonSerialIdentity(t *testing.T) {
	service := NewListService(&listStoreStub{})

	err := service.UpsertAndReevaluate(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1",
	}, CompiledListEntry{IdentityType: IdentityType("certificate_fingerprint")})

	require.Error(t, err)
}

func TestListServiceRejectsUnsupportedTypeAndStatus(t *testing.T) {
	service := NewListService(&listStoreStub{})
	actor := PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}

	err := service.UpsertAndReevaluate(context.Background(), actor, CompiledListEntry{
		Type: ListEntryType("unexpected"), IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-1",
	})
	require.Error(t, err)

	err = service.UpsertAndReevaluate(context.Background(), actor, CompiledListEntry{
		Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-1",
		Status: ListEntryStatus("unexpected"),
	})
	require.Error(t, err)
}

func TestListServiceNormalizesIdentityAndDefaultsStatus(t *testing.T) {
	entries := &listStoreStub{}
	service := NewListService(entries)

	err := service.UpsertAndReevaluate(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1",
	}, CompiledListEntry{
		Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber,
		IdentityValue: "  SN-TRIM  ",
	})

	require.NoError(t, err)
	require.Equal(t, "SN-TRIM", entries.entry.IdentityValue)
	require.Equal(t, ListEntryStatusActive, entries.entry.Status)
}

func TestListServiceEnforcesTargetIdentityScope(t *testing.T) {
	entries := &listStoreStub{}
	checker := &identityVisibilityStub{}
	service := NewListService(entries)
	service.SetIdentityVisibilityChecker(checker)
	groupID := uuid.New()

	err := service.UpsertAndReevaluate(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1", VisibleGroups: []uuid.UUID{groupID},
	}, CompiledListEntry{
		Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-OTHER-GROUP",
	})

	require.Error(t, err)
	require.Empty(t, entries.entry.IdentityValue)
	require.Equal(t, []uuid.UUID{groupID}, checker.groups)
}
