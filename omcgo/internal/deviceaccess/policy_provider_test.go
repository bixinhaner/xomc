package deviceaccess

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type policyProviderEmptyRows struct{ pgx.Rows }

func (policyProviderEmptyRows) Close()            {}
func (policyProviderEmptyRows) Err() error        { return nil }
func (policyProviderEmptyRows) Next() bool        { return false }
func (policyProviderEmptyRows) Scan(...any) error { return nil }

type noActivePolicyDB struct {
	repositoryTestDB
	queryCount int
}

func (d *noActivePolicyDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	d.queryCount++
	return policyProviderEmptyRows{}, nil
}

func TestDecodeConditionExpected(t *testing.T) {
	t.Run("in list", func(t *testing.T) {
		condition := CompiledCondition{Operator: ConditionOperatorIn}
		require.NoError(t, decodeConditionExpected(&condition, json.RawMessage(`["100","200"]`)))
		require.Equal(t, []string{"100", "200"}, condition.ExpectedAny)
	})

	t.Run("cidr string", func(t *testing.T) {
		condition := CompiledCondition{Operator: ConditionOperatorCIDR}
		require.NoError(t, decodeConditionExpected(&condition, json.RawMessage(`"10.0.0.0/8"`)))
		require.Equal(t, "10.0.0.0/8", condition.Expected)
	})

	t.Run("gps radius", func(t *testing.T) {
		condition := CompiledCondition{Operator: ConditionOperatorWithinRadius}
		require.NoError(t, decodeConditionExpected(&condition, json.RawMessage(`{"center":{"latitude":31.2,"longitude":121.5},"radius_meters":100}`)))
		require.Equal(t, 100.0, condition.GeoFence.RadiusMeters)
	})
}

func TestPgPolicyProviderActiveVersionScopesCarrierWithParameterizedQuery(t *testing.T) {
	versionID := uuid.New()
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(dest ...any) error {
		require.Len(t, dest, 2)
		*dest[0].(*uuid.UUID) = versionID
		*dest[1].(*PolicyDefaultAction) = PolicyDefaultActionReject
		return nil
	}}}
	provider := newPgPolicyProviderWithDB(db)

	got, defaultAction, err := provider.loadActiveVersion(context.Background(), "cmcc")

	require.NoError(t, err)
	require.Equal(t, versionID, got)
	require.Equal(t, PolicyDefaultActionReject, defaultAction)
	require.Contains(t, db.lastSQL, "ps.carrier")
	require.Contains(t, db.lastSQL, "$1")
	require.Contains(t, db.lastArgs, "cmcc")
	require.NotContains(t, db.lastSQL, "cmcc")
}

func TestPgPolicyProviderWithoutPublishedVersionDefaultsToRejectAndLoadsLists(t *testing.T) {
	db := &noActivePolicyDB{repositoryTestDB: repositoryTestDB{row: repositoryTestRow{
		scan: func(...any) error { return pgx.ErrNoRows },
	}}}
	provider := newPgPolicyProviderWithDB(db)

	policy, err := provider.Load(context.Background(), "cmcc")

	require.NoError(t, err)
	require.Empty(t, policy.VersionID)
	require.Equal(t, PolicyDefaultActionReject, policy.DefaultAction)
	require.Empty(t, policy.Rules)
	require.Empty(t, policy.ListEntries)
	require.Equal(t, 1, db.queryCount, "active access lists must still be loaded")
}
