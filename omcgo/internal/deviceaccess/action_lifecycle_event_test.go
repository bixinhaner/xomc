package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type actionLifecycleTestDB struct{ tx *actionLifecycleTestTx }

func (d *actionLifecycleTestDB) Begin(context.Context) (pgx.Tx, error) { return d.tx, nil }
func (d *actionLifecycleTestDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}
func (d *actionLifecycleTestDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}
func (d *actionLifecycleTestDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

type actionLifecycleTestTx struct {
	pgx.Tx
	details   actionLifecycleContext
	execSQL   []string
	execArgs  [][]any
	querySQL  []string
	committed bool
}

func (tx *actionLifecycleTestTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.execSQL = append(tx.execSQL, sql)
	tx.execArgs = append(tx.execArgs, append([]any(nil), args...))
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *actionLifecycleTestTx) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	tx.querySQL = append(tx.querySQL, query)
	return repositoryTestRow{scan: func(dest ...any) error {
		if len(dest) != 12 {
			return fmt.Errorf("unexpected lifecycle scan destination count: %d", len(dest))
		}
		*dest[0].(*uuid.UUID) = tx.details.DecisionID
		*dest[1].(**uuid.UUID) = tx.details.DeviceID
		*dest[2].(**uuid.UUID) = tx.details.CandidateID
		*dest[3].(*ActionType) = tx.details.ActionType
		*dest[4].(*ActionStatus) = tx.details.Status
		*dest[5].(*int) = tx.details.Attempts
		*dest[6].(*string) = tx.details.RequestID
		*dest[7].(*string) = tx.details.LastFailureCode
		*dest[8].(*string) = tx.details.ErrorMessage
		*dest[9].(*string) = tx.details.Carrier
		*dest[10].(*string) = tx.details.SerialNumber
		*dest[11].(**uuid.UUID) = tx.details.PolicyVersionID
		return nil
	}}
}

func (tx *actionLifecycleTestTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *actionLifecycleTestTx) Rollback(context.Context) error { return nil }

func TestScheduleRetryWritesFailureOutboxInSameTransaction(t *testing.T) {
	policyID := uuid.New()
	tx := &actionLifecycleTestTx{details: actionLifecycleContext{
		DecisionID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOff,
		Status: ActionStatusRetryWait, Attempts: 1, RequestID: "request-1",
		LastFailureCode: "cwmp_timeout", ErrorMessage: "timed out",
		Carrier: "cmcc", SerialNumber: "SN-1", PolicyVersionID: &policyID,
	}}
	store := newPgActionStoreWithDB(&actionLifecycleTestDB{tx: tx})

	err := store.ScheduleRetry(context.Background(), uuid.New(), "cwmp_timeout", "timed out", time.Now(), false)

	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 2)
	require.Contains(t, tx.execSQL[0], "UPDATE device_access_actions")
	require.Contains(t, tx.execSQL[1], "INSERT INTO device_access_outbox")
	require.Contains(t, tx.execArgs[1], "device.access.action_failed")
	require.Len(t, tx.querySQL, 1)
	require.Contains(t, tx.querySQL[0], "dec.trigger_event_id")
	require.NotContains(t, tx.querySQL[0], "a.idempotency_key")
	payload := findJSONArgument(t, tx.execArgs[1])
	require.Equal(t, "DEVICE_ACCESS_ACTION_FAILED", payload["event_name"])
	require.Equal(t, "cwmp_timeout", payload["reason_code"])
	require.Equal(t, "SN-1", payload["serial_number"])
	require.Equal(t, "request-1", payload["request_id"])
}

func TestSuccessfulRetryWritesRecoveredOutbox(t *testing.T) {
	tx := &actionLifecycleTestTx{details: actionLifecycleContext{
		DecisionID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOn,
		Status: ActionStatusSucceeded, Attempts: 2, RequestID: "request-2",
		LastFailureCode: "gpv_readback_mismatch", Carrier: "cmcc", SerialNumber: "SN-2",
	}}
	store := newPgActionStoreWithDB(&actionLifecycleTestDB{tx: tx})

	err := store.MarkTerminal(context.Background(), uuid.New(), ActionStatusSucceeded, false, "", "", time.Now())

	require.NoError(t, err)
	require.Len(t, tx.execSQL, 2)
	require.Contains(t, tx.execSQL[0], "device_access_action_attempts")
	require.Contains(t, tx.execSQL[0], "readback_gpv")
	require.Contains(t, tx.execArgs[1], "device.access.action_recovered")
	payload := findJSONArgument(t, tx.execArgs[1])
	require.Equal(t, "DEVICE_ACCESS_ACTION_RECOVERED", payload["event_name"])
}

func TestFirstAttemptSuccessDoesNotWriteRecoveryOutbox(t *testing.T) {
	tx := &actionLifecycleTestTx{details: actionLifecycleContext{
		DecisionID: uuid.New(), DeviceID: uuidPointer(uuid.New()), ActionType: ActionTypeRFOn,
		Status: ActionStatusSucceeded, Attempts: 1, RequestID: "request-3",
		Carrier: "cmcc", SerialNumber: "SN-3",
	}}
	store := newPgActionStoreWithDB(&actionLifecycleTestDB{tx: tx})

	err := store.MarkTerminal(context.Background(), uuid.New(), ActionStatusSucceeded, false, "", "", time.Now())

	require.NoError(t, err)
	require.Len(t, tx.execSQL, 1)
}

func TestCandidateActionFailureOutboxKeepsCandidateIdentity(t *testing.T) {
	candidateID := uuid.New()
	tx := &actionLifecycleTestTx{details: actionLifecycleContext{
		DecisionID: uuid.New(), CandidateID: &candidateID, ActionType: ActionTypeRFOff,
		Status: ActionStatusRetryWait, Attempts: 1, RequestID: "request-candidate",
		LastFailureCode: "cwmp_timeout", Carrier: "cmcc", SerialNumber: "SN-CANDIDATE",
	}}
	store := newPgActionStoreWithDB(&actionLifecycleTestDB{tx: tx})

	err := store.ScheduleRetry(context.Background(), uuid.New(), "cwmp_timeout", "timed out", time.Now(), false)

	require.NoError(t, err)
	payload := findJSONArgument(t, tx.execArgs[1])
	require.Equal(t, candidateID.String(), payload["candidate_id"])
	require.NotContains(t, payload, "device_id")
}

func findJSONArgument(t *testing.T, args []any) map[string]any {
	t.Helper()
	for _, arg := range args {
		raw, ok := arg.([]byte)
		if !ok {
			continue
		}
		var payload map[string]any
		if json.Unmarshal(raw, &payload) == nil && payload["event_name"] != nil {
			return payload
		}
	}
	t.Fatal("event payload argument not found")
	return nil
}
