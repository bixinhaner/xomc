package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type listStoreStub struct {
	carrier         string
	entry           CompiledListEntry
	entries         []CompiledListEntry
	disabledType    ListEntryType
	disabledSerials []string
	disableReason   string
}

func (s *listStoreStub) DisableListEntriesAndQueue(_ context.Context, carrier string, entryType ListEntryType, serialNumbers []string, reason string) error {
	s.carrier = carrier
	s.disabledType = entryType
	s.disabledSerials = append([]string(nil), serialNumbers...)
	s.disableReason = reason
	return nil
}

func (s *listStoreStub) UpsertListEntriesAndQueue(_ context.Context, carrier string, entries []CompiledListEntry) error {
	s.carrier = carrier
	s.entries = append([]CompiledListEntry(nil), entries...)
	return nil
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

func TestListServicePersistsMultiSerialBatchAtomically(t *testing.T) {
	store := &listStoreStub{}
	service := NewListService(store)

	err := service.UpsertManyAndReevaluate(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1",
	}, []CompiledListEntry{
		{Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: " SN-1 "},
		{Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-2"},
	})

	require.NoError(t, err)
	require.Equal(t, "cmcc", store.carrier)
	require.Len(t, store.entries, 2)
	require.Equal(t, "SN-1", store.entries[0].IdentityValue)
	require.Equal(t, ListEntryStatusActive, store.entries[0].Status)
}

func TestListServiceRejectsInvalidMultiSerialBatchBeforePersistence(t *testing.T) {
	tests := []struct {
		name    string
		entries []CompiledListEntry
	}{
		{
			name: "duplicate serial",
			entries: []CompiledListEntry{
				{Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-1"},
				{Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: " SN-1 "},
			},
		},
		{
			name: "mixed list types",
			entries: []CompiledListEntry{
				{Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-1"},
				{Type: ListEntryTypeAllow, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &listStoreStub{}
			service := NewListService(store)
			err := service.UpsertManyAndReevaluate(context.Background(), PolicyActor{
				Carrier: "cmcc", SubjectID: "operator-1",
			}, tt.entries)
			require.Error(t, err)
			require.Empty(t, store.entries)
		})
	}
}

func TestListServiceValidatesBatchDisableBeforePersistence(t *testing.T) {
	store := &listStoreStub{}
	service := NewListService(store)
	actor := PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}

	err := service.DisableManyAndReevaluate(context.Background(), actor, ListEntryTypeDeny, []string{" SN-1 ", "SN-2"}, "expired approval")

	require.NoError(t, err)
	require.Equal(t, "cmcc", store.carrier)
	require.Equal(t, ListEntryTypeDeny, store.disabledType)
	require.Equal(t, []string{"SN-1", "SN-2"}, store.disabledSerials)
	require.Equal(t, "expired approval", store.disableReason)

	err = service.DisableManyAndReevaluate(context.Background(), actor, ListEntryTypeDeny, []string{"SN-1", " SN-1 "}, "duplicate")
	require.Error(t, err)
	err = service.DisableManyAndReevaluate(context.Background(), actor, ListEntryTypeDeny, []string{"SN-1"}, "")
	require.Error(t, err)
}
