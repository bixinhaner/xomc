package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestBuildListEntryUpsertUsesParameterizedIdentityConflict(t *testing.T) {
	entry := CompiledListEntry{
		Type:          ListEntryTypeDeny,
		IdentityType:  IdentityTypeSerialNumber,
		IdentityValue: "SN-LIST",
		Status:        ListEntryStatusActive,
	}

	query, args, err := buildListEntryUpsert("cmcc", entry)

	require.NoError(t, err)
	require.Contains(t, query, "INSERT INTO device_access_list_entries")
	require.Contains(t, query, "ON CONFLICT (carrier,entry_type,identity_type,identity_value) DO UPDATE")
	require.Contains(t, query, "$1")
	require.NotContains(t, query, "SN-LIST")
	require.Contains(t, args, "cmcc")
	require.Contains(t, args, "SN-LIST")
}

func TestPgPolicyStoreWritesEntryAndOutboxInOneTransaction(t *testing.T) {
	tx := &repositoryTestTx{}
	store := NewPgPolicyStore(&repositoryTestDB{tx: tx})

	err := store.UpsertListEntryAndQueue(context.Background(), "cmcc", CompiledListEntry{
		Type: ListEntryTypeAllow, IdentityType: IdentityTypeSerialNumber,
		IdentityValue: "SN-1", Status: ListEntryStatusActive,
	})

	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 2)
	require.Contains(t, tx.execSQL[0], "device_access_list_entries")
	require.Contains(t, tx.execSQL[1], "device_access_outbox")
}

func TestFindOrCreatePolicySetReturnsPersistedFamilyName(t *testing.T) {
	wantID := uuid.New()
	tx := &repositoryTestTx{queryRow: func(query string, _ ...any) pgx.Row {
		require.Contains(t, query, "SELECT id, name")
		return repositoryTestRow{scan: func(dest ...any) error {
			*dest[0].(*uuid.UUID) = wantID
			*dest[1].(*string) = "CMCC 接入策略"
			return nil
		}}
	}}

	gotID, gotName, err := findOrCreatePolicySet(context.Background(), tx, PolicyVersion{
		Carrier: "cmcc", Name: "不会作为新版本名称保存",
	})

	require.NoError(t, err)
	require.Equal(t, wantID, gotID)
	require.Equal(t, "CMCC 接入策略", gotName)
	require.Empty(t, tx.execSQL)
}

func TestFindOrCreatePolicySetCreatesTrimmedFamilyName(t *testing.T) {
	tx := &repositoryTestTx{queryRow: func(string, ...any) pgx.Row {
		return repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}
	}}

	gotID, gotName, err := findOrCreatePolicySet(context.Background(), tx, PolicyVersion{
		Carrier: "cmcc", Name: "  CMCC 接入策略  ",
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, gotID)
	require.Equal(t, "CMCC 接入策略", gotName)
	require.Len(t, tx.execSQL, 1)
}
