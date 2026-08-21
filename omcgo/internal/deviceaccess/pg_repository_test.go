package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type repositoryTestDB struct {
	storage.DB
	row      pgx.Row
	queryRow func(string, ...any) pgx.Row
	tx       pgx.Tx
	lastSQL  string
	lastArgs []any
	allSQL   []string
	allArgs  [][]any
}

func (d *repositoryTestDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	d.lastSQL = query
	d.lastArgs = append([]any(nil), args...)
	d.allSQL = append(d.allSQL, query)
	d.allArgs = append(d.allArgs, append([]any(nil), args...))
	if d.queryRow != nil {
		return d.queryRow(query, args...)
	}
	return d.row
}

func (d *repositoryTestDB) Begin(context.Context) (pgx.Tx, error) {
	return d.tx, nil
}

func (d *repositoryTestDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	d.lastSQL = query
	d.lastArgs = append([]any(nil), args...)
	d.allSQL = append(d.allSQL, query)
	d.allArgs = append(d.allArgs, append([]any(nil), args...))
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type repositoryTestRow struct {
	scan func(...any) error
}

func (r repositoryTestRow) Scan(dest ...any) error {
	return r.scan(dest...)
}

type repositoryTestTx struct {
	pgx.Tx
	queryRow   func(string, ...any) pgx.Row
	execSQL    []string
	failAt     int
	committed  bool
	rolledBack bool
}

func (tx *repositoryTestTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	return tx.queryRow(query, args...)
}

func (tx *repositoryTestTx) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	tx.execSQL = append(tx.execSQL, query)
	if tx.failAt > 0 && len(tx.execSQL) == tx.failAt {
		return pgconn.CommandTag{}, errors.New("forced transaction write failure")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (tx *repositoryTestTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *repositoryTestTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

func TestPgRepositoryUpsertCandidateUsesTenantScopedParameterizedConflictUpdate(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	id := uuid.New()
	ip := netip.MustParseAddr("10.21.7.9")
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(dest ...any) error {
		require.Len(t, dest, 16)
		*dest[0].(*uuid.UUID) = id
		*dest[1].(*string) = "cmcc"
		*dest[2].(*string) = "SN-CANDIDATE"
		*dest[3].(*string) = "48BF74"
		*dest[4].(*string) = "BaiStation"
		*dest[5].(*string) = "V1.0"
		*dest[6].(*model.Technology) = model.TechLTE
		*dest[7].(*[]byte) = []byte(`["Device.Services.FAPService.1.FAPControl.LTE.AdminState"]`)
		*dest[8].(*netip.Addr) = ip
		*dest[9].(*time.Time) = now
		*dest[10].(*time.Time) = now
		*dest[11].(*int64) = 2
		*dest[12].(*string) = "pending"
		*dest[13].(**uuid.UUID) = nil
		*dest[14].(**time.Time) = nil
		*dest[15].(*time.Time) = now.Add(24 * time.Hour)
		return nil
	}}}
	repo := newPgRepositoryWithDB(db)

	got, err := repo.UpsertCandidateObservation(context.Background(), Observation{
		Carrier: "cmcc", SerialNumber: "SN-CANDIDATE", OUI: "48BF74",
		ProductClass: "BaiStation", SoftwareVersion: "V1.0",
		Technology: model.TechLTE, RFControlPaths: []string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState"},
		RemoteIP: ip, ObservedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	})

	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, int64(2), got.InformCount)
	assert.Contains(t, db.lastSQL, "ON CONFLICT (carrier,serial_number) DO UPDATE")
	assert.Contains(t, db.lastSQL, "inform_count = device_access_candidates.inform_count + 1")
	assert.Contains(t, db.lastSQL, "carrier")
	assert.Contains(t, db.lastSQL, "serial_number")
	assert.NotContains(t, db.lastSQL, "SN-CANDIDATE")
	assert.NotContains(t, db.lastSQL, "cmcc")
	assert.Contains(t, db.lastSQL, "$1")
	assert.Contains(t, db.lastArgs, "cmcc")
	assert.Contains(t, db.lastArgs, "SN-CANDIDATE")
}

func TestPgRepositoryFormalDeviceObservationClosesPendingCandidate(t *testing.T) {
	now := time.Now().UTC()
	deviceID := uuid.New()
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(dest ...any) error {
		*dest[0].(*uuid.UUID) = uuid.New()
		*dest[1].(*string) = "cmcc"
		*dest[2].(*string) = "SN-FORMAL"
		*dest[3].(*string) = "48BF74"
		*dest[4].(*string) = "BaiStation"
		*dest[5].(*string) = "V1.0"
		*dest[6].(*model.Technology) = model.TechLTE
		*dest[7].(*[]byte) = []byte(`[]`)
		*dest[8].(*netip.Addr) = netip.MustParseAddr("192.0.2.10")
		*dest[9].(*time.Time) = now
		*dest[10].(*time.Time) = now
		*dest[11].(*int64) = 2
		*dest[12].(*string) = "approved"
		*dest[13].(**uuid.UUID) = nil
		*dest[14].(**time.Time) = nil
		*dest[15].(*time.Time) = now.Add(time.Hour)
		return nil
	}}}

	_, err := newPgRepositoryWithDB(db).UpsertCandidateObservation(context.Background(), Observation{
		Carrier: "cmcc", SerialNumber: "SN-FORMAL", DeviceID: &deviceID,
		Technology: model.TechLTE, RemoteIP: netip.MustParseAddr("192.0.2.10"),
		ObservedAt: now, ExpiresAt: now.Add(time.Hour),
	})

	require.NoError(t, err)
	require.Contains(t, db.lastSQL, "device_id = COALESCE(EXCLUDED.device_id")
	require.Contains(t, db.lastSQL, "EXCLUDED.device_id IS NOT NULL AND device_access_candidates.review_status = 'pending'")
	require.Contains(t, db.lastArgs, &deviceID)
}

func TestPgRepositoryPersistsImmutableIdentitySnapshot(t *testing.T) {
	db := &repositoryTestDB{}
	repo := newPgRepositoryWithDB(db)
	now := time.Date(2026, 8, 19, 11, 30, 0, 0, time.UTC)

	err := repo.SaveIdentitySnapshot(context.Background(), IdentitySnapshot{
		ID: uuid.NewString(), RequestID: "request-1", DecisionID: uuid.NewString(), CandidateID: uuid.NewString(),
		Carrier: "cmcc", SerialNumber: "SN-1", CloudKey: "CLOUD1", OUI: "48BF74", ProductClass: "BaiStation",
		RawRemoteIP: net.ParseIP("192.0.2.10"), ObservedRemoteIP: net.ParseIP("192.0.2.10"),
		InformEvent: "1 BOOT", InformTime: now, IdentitySource: "inform_device_id",
		IdentityStatus: IdentityStatusResolved, CreatedAt: now,
	})

	require.NoError(t, err)
	require.Contains(t, db.lastSQL, "INSERT INTO device_access_identity_snapshots")
	require.Contains(t, db.lastSQL, "ON CONFLICT (carrier, serial_number, request_id) DO NOTHING")
	require.NotContains(t, db.lastSQL, "SN-1")
	require.Contains(t, db.lastArgs, "SN-1")
	require.Contains(t, db.lastArgs, "request-1")
}

func TestPgRepositoryRejectsMissingTenantIdentityBeforeQuery(t *testing.T) {
	db := &repositoryTestDB{}
	repo := newPgRepositoryWithDB(db)
	now := time.Now().UTC()

	_, err := repo.UpsertCandidateObservation(context.Background(), Observation{
		SerialNumber: "SN-NO-CARRIER", OUI: "48BF74", ObservedAt: now, ExpiresAt: now.Add(time.Hour),
	})

	require.ErrorIs(t, err, ErrCarrierRequired)
	assert.Empty(t, db.lastSQL)

	_, err = repo.LoadEvaluationContext(context.Background(), "", "SN-NO-CARRIER")
	require.ErrorIs(t, err, ErrCarrierRequired)
}

func TestLegacyOwnedIsolationWithoutPathsDoesNotBlockFreshContainment(t *testing.T) {
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}}
	deviceID := uuid.New()
	_, err := findConflictingActionPlan(context.Background(), db, ActionPlan{
		DeviceID: &deviceID, DecisionID: uuid.New(), ActionType: ActionTypeRFOff,
		Direction: ActionDirectionContain, IdempotencyKey: "fresh-containment",
	})
	require.NoError(t, err)
	require.Contains(t, db.lastSQL, "jsonb_array_length(COALESCE(rf_change_paths, '[]'::jsonb)) > 0")

	repo := newPgActionStoreWithDB(db)
	_, err = repo.FindOpenContainment(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Contains(t, db.lastSQL, "jsonb_array_length(COALESCE(a.rf_change_paths, '[]'::jsonb)) > 0")
}

func TestPromoteCandidateTargetMovesProjectionAndActionsAtomically(t *testing.T) {
	candidateID, deviceID := uuid.New(), uuid.New()
	queryNo := 0
	tx := &repositoryTestTx{queryRow: func(_ string, _ ...any) pgx.Row {
		queryNo++
		return repositoryTestRow{scan: func(dest ...any) error {
			require.Len(t, dest, 1)
			if queryNo == 1 {
				*dest[0].(*uuid.UUID) = candidateID
			} else {
				*dest[0].(*uuid.UUID) = deviceID
			}
			return nil
		}}
	}}
	store := newPgActionStoreWithDB(&repositoryTestDB{tx: tx})

	got, err := store.PromoteCandidateTarget(context.Background(), candidateID, "cmcc", "SN-CANDIDATE")

	require.NoError(t, err)
	require.Equal(t, deviceID, got)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 3)
	require.Contains(t, tx.execSQL[0], "UPDATE device_access_candidates")
	require.Contains(t, tx.execSQL[1], "UPDATE device_access_states")
	require.Contains(t, tx.execSQL[1], "candidate_id = $2")
	require.Contains(t, tx.execSQL[2], "UPDATE device_access_actions")
	require.Contains(t, tx.execSQL[2], "candidate_id = $2")
}

func TestChangedAgainstBasePreservesConcurrentEvidenceTypes(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	base := []EvidenceRecord{
		{Type: ConditionTypeTAC, ValueHash: "tac-old", ObservedAt: now},
		{Type: ConditionTypeGPS, ValueHash: "gps-old", ObservedAt: now},
	}
	proposedSnapshot := []EvidenceRecord{
		{Type: ConditionTypeTAC, ValueHash: "tac-old", ObservedAt: now},
		{Type: ConditionTypeGPS, ValueHash: "gps-new", ObservedAt: now.Add(time.Minute)},
	}
	latestConcurrent := []EvidenceRecord{
		{Type: ConditionTypeTAC, ValueHash: "tac-concurrent", ObservedAt: now.Add(30 * time.Second)},
		{Type: ConditionTypeGPS, ValueHash: "gps-old", ObservedAt: now},
	}

	replacements := changedAgainstBase(base, proposedSnapshot)
	merged := mergeEvidenceRecords(latestConcurrent, replacements)

	require.Len(t, replacements, 1)
	require.Equal(t, ConditionTypeGPS, replacements[0].Type)
	byType := make(map[ConditionType]EvidenceRecord, len(merged))
	for _, record := range merged {
		byType[record.Type] = record
	}
	require.Equal(t, "tac-concurrent", byType[ConditionTypeTAC].ValueHash)
	require.Equal(t, "gps-new", byType[ConditionTypeGPS].ValueHash)
}

func TestPgRepositoryResolveCarrierUsesDeviceAndAccessAuthority(t *testing.T) {
	carrier := "cmcc"
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(dest ...any) error {
		require.Len(t, dest, 2)
		*dest[0].(*int64) = 1
		*dest[1].(**string) = &carrier
		return nil
	}}}
	repo := newPgRepositoryWithDB(db)

	got, err := repo.ResolveCarrier(context.Background(), "SN-CARRIER")

	require.NoError(t, err)
	require.Equal(t, "cmcc", got)
	require.Len(t, db.allSQL, 3)
	require.Contains(t, db.allSQL[0], "FROM devices")
	require.Contains(t, db.allSQL[0], "deleted_at IS NULL")
	require.Contains(t, db.allSQL[1], "FROM device_access_states")
	require.Contains(t, db.allSQL[2], "FROM device_access_candidates")
	for i := range db.allSQL {
		require.Contains(t, db.allSQL[i], "$1")
		require.Equal(t, []any{"SN-CARRIER"}, db.allArgs[i])
	}
}

func TestPgRepositoryResolveCarrierSupportsFreshRegisteredDevice(t *testing.T) {
	db := &repositoryTestDB{queryRow: func(query string, _ ...any) pgx.Row {
		if strings.Contains(query, "FROM devices") {
			return carrierResolutionRow(1, "cmcc")
		}
		return carrierResolutionRow(0, "")
	}}
	repo := newPgRepositoryWithDB(db)

	got, err := repo.ResolveCarrier(context.Background(), "SN-FRESH")

	require.NoError(t, err)
	require.Equal(t, "cmcc", got)
	require.Len(t, db.allSQL, 3)
}

func TestPgRepositoryResolveCarrierSupportsCandidateBeforeRegistration(t *testing.T) {
	db := &repositoryTestDB{queryRow: func(query string, _ ...any) pgx.Row {
		if strings.Contains(query, "FROM device_access_candidates") {
			return carrierResolutionRow(1, "ctcc")
		}
		return carrierResolutionRow(0, "")
	}}
	repo := newPgRepositoryWithDB(db)

	got, err := repo.ResolveCarrier(context.Background(), "SN-CANDIDATE")

	require.NoError(t, err)
	require.Equal(t, "ctcc", got)
}

func TestPgRepositoryResolveCarrierFailsClosedOnConflictingSources(t *testing.T) {
	db := &repositoryTestDB{queryRow: func(query string, _ ...any) pgx.Row {
		switch {
		case strings.Contains(query, "FROM devices"):
			return carrierResolutionRow(1, "cmcc")
		case strings.Contains(query, "FROM device_access_states"):
			return carrierResolutionRow(1, "ctcc")
		default:
			return carrierResolutionRow(0, "")
		}
	}}
	repo := newPgRepositoryWithDB(db)

	_, err := repo.ResolveCarrier(context.Background(), "SN-CONFLICT")

	require.ErrorIs(t, err, ErrCarrierAmbiguous)
}

func TestPgRepositoryResolveCarrierExcludesSoftDeletedDevice(t *testing.T) {
	db := &repositoryTestDB{queryRow: func(_ string, _ ...any) pgx.Row {
		return carrierResolutionRow(0, "")
	}}
	repo := newPgRepositoryWithDB(db)

	_, err := repo.ResolveCarrier(context.Background(), "SN-DELETED")

	require.ErrorIs(t, err, ErrCarrierNotFound)
	require.NotEmpty(t, db.allSQL)
	require.Contains(t, db.allSQL[0], "FROM devices")
	require.Contains(t, db.allSQL[0], "deleted_at IS NULL")
}

func carrierResolutionRow(count int64, carrier string) pgx.Row {
	return repositoryTestRow{scan: func(dest ...any) error {
		*dest[0].(*int64) = count
		if count == 0 {
			*dest[1].(**string) = nil
			return nil
		}
		value := carrier
		*dest[1].(**string) = &value
		return nil
	}}
}

func TestPgRepositoryAssetQueriesAlwaysCarryCarrierScope(t *testing.T) {
	for name, build := range map[string]func() (string, []any, error){
		"device": func() (string, []any, error) {
			return buildAssetDeviceQuery("cmcc", "SN-1")
		},
		"registration": func() (string, []any, error) {
			return buildAssetRegistrationQuery("cmcc", "SN-1")
		},
		"other carrier": func() (string, []any, error) {
			return buildOtherCarrierQuery("cmcc", "SN-1")
		},
	} {
		t.Run(name, func(t *testing.T) {
			query, args, err := build()
			require.NoError(t, err)
			assert.Contains(t, query, "carrier")
			assert.Contains(t, query, "serial_number")
			assert.NotContains(t, query, "cmcc")
			assert.NotContains(t, query, "SN-1")
			assert.Equal(t, []any{"cmcc", "SN-1"}, args)
		})
	}
}

func TestPgRepositorySaveDecisionWritesEvidenceProjectionHistoryChecksAndOutboxAtomically(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	stateID := uuid.New()
	candidateID := uuid.New()
	policyVersionID := uuid.New()
	tx := successfulDecisionTx(t, stateID, candidateID, policyVersionID, 4, AccessStateCollecting)
	repo := newPgRepositoryWithDB(&repositoryTestDB{tx: tx})
	change := decisionChangeFixture(now, candidateID, policyVersionID)

	saved, err := repo.SaveDecision(context.Background(), change)

	require.NoError(t, err)
	assert.Equal(t, int64(5), saved.DecisionVersion)
	assert.Equal(t, AccessStateAccepted, saved.State)
	assert.True(t, tx.committed)
	assert.True(t, tx.rolledBack, "deferred rollback must remain safe after commit")
	assertSQLWriteOrder(t, tx.execSQL, []string{
		"device_access_evidence",
		"device_access_states",
		"device_access_decisions",
		"device_access_identity_snapshots",
		"device_access_decision_checks",
		"device_access_decision_checks",
		"device_access_outbox",
	})
}

func TestPgRepositorySaveDecisionRollsBackEveryWriteWhenAnyAtomicStepFails(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	candidateID := uuid.New()
	policyVersionID := uuid.New()

	for failAt := 1; failAt <= 7; failAt++ {
		t.Run("write "+string(rune('0'+failAt)), func(t *testing.T) {
			tx := successfulDecisionTx(t, uuid.New(), candidateID, policyVersionID, 4, AccessStateCollecting)
			tx.failAt = failAt
			repo := newPgRepositoryWithDB(&repositoryTestDB{tx: tx})

			_, err := repo.SaveDecision(context.Background(), decisionChangeFixture(now, candidateID, policyVersionID))

			require.Error(t, err)
			assert.False(t, tx.committed)
			assert.True(t, tx.rolledBack)
		})
	}
}

func TestPgRepositorySaveDecisionRejectsStaleExpectedVersionWithoutWrites(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	candidateID := uuid.New()
	policyVersionID := uuid.New()
	tx := successfulDecisionTx(t, uuid.New(), candidateID, policyVersionID, 8, AccessStateCollecting)
	repo := newPgRepositoryWithDB(&repositoryTestDB{tx: tx})
	change := decisionChangeFixture(now, candidateID, policyVersionID)
	change.ExpectedDecisionVersion = 4

	_, err := repo.SaveDecision(context.Background(), change)

	require.ErrorIs(t, err, ErrDecisionVersionConflict)
	assert.Empty(t, tx.execSQL)
	assert.False(t, tx.committed)
	assert.True(t, tx.rolledBack)
}

func TestPgRepositorySaveDecisionCreatesInitialStateForNewCandidate(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	candidateID := uuid.New()
	policyVersionID := uuid.New()
	tx := &repositoryTestTx{queryRow: func(query string, _ ...any) pgx.Row {
		switch {
		case strings.Contains(query, "FROM device_access_candidates"):
			return repositoryTestRow{scan: func(dest ...any) error {
				require.Len(t, dest, 1)
				*dest[0].(*uuid.UUID) = candidateID
				return nil
			}}
		case strings.Contains(query, "FROM device_access_decisions"):
			return repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}
		case strings.Contains(query, "FROM device_access_states"):
			return repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}
		default:
			return repositoryTestRow{scan: func(...any) error {
				return errors.New("unexpected query: " + query)
			}}
		}
	}}
	repo := newPgRepositoryWithDB(&repositoryTestDB{tx: tx})
	change := decisionChangeFixture(now, candidateID, policyVersionID)
	change.ExpectedDecisionVersion = 0

	saved, err := repo.SaveDecision(context.Background(), change)

	require.NoError(t, err)
	require.Equal(t, int64(1), saved.DecisionVersion)
	require.True(t, tx.committed)
	assertSQLWriteOrder(t, tx.execSQL, []string{
		"device_access_evidence",
		"device_access_states",
		"device_access_decisions",
		"device_access_identity_snapshots",
		"device_access_decision_checks",
		"device_access_decision_checks",
		"device_access_outbox",
	})
}

func successfulDecisionTx(
	t *testing.T,
	stateID uuid.UUID,
	candidateID uuid.UUID,
	policyVersionID uuid.UUID,
	version int64,
	state AccessState,
) *repositoryTestTx {
	t.Helper()
	return &repositoryTestTx{queryRow: func(query string, _ ...any) pgx.Row {
		switch {
		case strings.Contains(query, "FROM device_access_candidates"):
			return repositoryTestRow{scan: func(dest ...any) error {
				require.Len(t, dest, 1)
				*dest[0].(*uuid.UUID) = candidateID
				return nil
			}}
		case strings.Contains(query, "FROM device_access_decisions"):
			return repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}
		case strings.Contains(query, "FROM device_access_states"):
			return repositoryTestRow{scan: func(dest ...any) error {
				require.Len(t, dest, 9)
				*dest[0].(*uuid.UUID) = stateID
				*dest[1].(**uuid.UUID) = nil
				*dest[2].(**uuid.UUID) = &candidateID
				*dest[3].(*AccessState) = state
				*dest[4].(*int64) = version
				*dest[5].(*int64) = 6
				*dest[6].(**uuid.UUID) = &policyVersionID
				*dest[7].(*bool) = true
				*dest[8].(**time.Time) = nil
				return nil
			}}
		default:
			return repositoryTestRow{scan: func(...any) error {
				return errors.New("unexpected query: " + query)
			}}
		}
	}}
}

func decisionChangeFixture(now time.Time, candidateID, policyVersionID uuid.UUID) DecisionChange {
	return DecisionChange{
		Carrier:                 "cmcc",
		SerialNumber:            "SN-DECISION",
		CandidateID:             &candidateID,
		TriggerType:             "inform",
		TriggerEventID:          "event-1",
		ExpectedDecisionVersion: 4,
		PolicyVersionID:         &policyVersionID,
		EvidenceVersion:         7,
		OccurredAt:              now,
		Decision: Decision{
			State:           AccessStateAccepted,
			EffectiveAction: EffectiveActionAccept,
			ReasonCode:      ReasonRuleMatched,
			MatchedRuleID:   uuid.NewString(),
			Checks: []DecisionCheck{
				{CheckID: "tac", CheckType: ConditionTypeTAC, Result: CheckPassed},
				{CheckID: "ip", CheckType: ConditionTypeObservedIP, Result: CheckPassed},
			},
		},
		Evidence: &EvidenceBatch{
			Carrier:      "cmcc",
			SerialNumber: "SN-DECISION",
			Version:      7,
			Records: []EvidenceRecord{{
				Type: ConditionTypeTAC, NormalizedValue: json.RawMessage(`"100"`),
				ValueHash: "sha256:tac", Source: "gpv", ObservedAt: now,
			}},
		},
		IdentitySnapshot: &IdentitySnapshot{
			ID: uuid.NewString(), RequestID: "event-1", Carrier: "cmcc", SerialNumber: "SN-DECISION",
			CandidateID: candidateID.String(), InformTime: now, IdentitySource: "inform_device_id",
			IdentityStatus: IdentityStatusResolved, CreatedAt: now,
		},
		Outbox: OutboxEvent{
			EventType: event.SubjectDeviceAccessAccepted,
			EventKey:  "decision:event-1",
			Payload:   json.RawMessage(`{"event_id":"event-1"}`),
		},
	}
}

func assertSQLWriteOrder(t *testing.T, queries, tables []string) {
	t.Helper()
	require.Len(t, queries, len(tables))
	for i, table := range tables {
		assert.Containsf(t, queries[i], table, "write %d must target %s", i+1, table)
	}
}
