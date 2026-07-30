# Issue #227 Alarm Instance Reconciliation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make CurrentAlarm and ExpeditedEvent converge on the same active alarm instance while preserving legitimate same-identifier alarms on different managed objects.

**Architecture:** Keep the existing New/Changed/Cleared and CurrentAlarm reconciliation flow. Replace transport-container-based identity with a stable business-object qualifier, add unique-candidate fallback for device clear events, publish one sync request per ExpeditedEvent batch, and make the sync diff preserve and archive duplicate local rows instead of overwriting them in a map.

**Tech Stack:** Go, pgx/Squirrel-backed alarm store, Redis alarm cache, NATS event bus, TR-069/CWMP, testify.

## Global Constraints

- Do not add database tables, columns, or `000002+` migrations.
- Do not add alarm-identifier-specific branches for `11109`, `11112`, `11189`, or `11190`.
- Preserve raw `managed_object_instance`, `additional_text`, and `additional_information` in `AdditionalInfo`.
- Carrier-specific behavior, if needed, must live under `internal/core/carrier/`; do not scatter carrier conditionals.
- CurrentAlarm remains authoritative for eventual consistency.
- All implementation tasks use TDD: failing test, minimal implementation, passing focused test.
- Do not modify the frontend or manually delete production alarm rows.

---

## File Structure

- Modify `omcgo/internal/alarm/alarm_identity.go`: normalize transient FaultMgmt containers and derive stable business-object qualifiers.
- Create `omcgo/internal/alarm/alarm_identity_test.go`: focused identity, cross-channel matching, and candidate-selection tests.
- Modify `omcgo/internal/alarm/engine.go`: unique-identifier fallback for AutoClear and consistent Redis key deletion.
- Modify `omcgo/internal/alarm/engine_test.go`: direct AutoClear and Redis-key regression coverage.
- Modify `omcgo/internal/alarm/metrics.go`: expose alarm reconciliation outcome counters.
- Create `omcgo/internal/alarm/metrics_test.go`: verify reconciliation counter registration and labels.
- Create `omcgo/internal/alarm/sync_request.go`: shared batch-level `alarm.sync.requested` publisher used by both receivers.
- Modify `omcgo/internal/alarm/receiver.go`: delegate the existing sync publish operation to the shared helper.
- Modify `omcgo/internal/alarm/expedited_receiver.go`: publish one sync request after a valid ExpeditedEvent batch.
- Modify `omcgo/internal/alarm/expedited_receiver_test.go`: batch publish and partial-failure regression tests.
- Modify `omcgo/internal/alarm/sync_model.go`: return explicit local update, clear, and duplicate-clear records.
- Modify `omcgo/internal/alarm/sync_model_test.go`: duplicate grouping, keeper selection, and parallel-object tests.
- Modify `omcgo/internal/alarm/sync_processor.go`: consume explicit diff records and archive duplicates with an audit reason.
- Modify `omcgo/internal/alarm/sync_processor_test.go`: six-local/two-remote issue #227 regression fixture.

---

### Task 1: Stable Cross-Channel Alarm Identity

**Files:**
- Create: `omcgo/internal/alarm/alarm_identity_test.go`
- Modify: `omcgo/internal/alarm/alarm_identity.go`

**Interfaces:**
- Produces: `activeAlarmQualifier(*model.Alarm) string`
- Produces: `activeAlarmMatchKey(*model.Alarm) string`
- Produces: `findActiveAlarmsByIdentifier([]*model.Alarm, *model.Alarm) []*model.Alarm`
- Consumed by: `AlarmEngine`, `ComputeDiff`, and Redis key operations.

- [ ] **Step 1: Write failing identity tests**

Add table-driven tests covering:

```go
func TestActiveAlarmMatchKey_CorrelatesCurrentAndExpeditedChannels(t *testing.T) {
	current := &model.Alarm{
		AlarmIdentifier: "11109",
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.CurrentAlarm.3.",
			"additional_text": "LTE0",
			"additional_information": "LTE0(73828545);S1setup fail. The MME ID: MME1",
		},
	}
	expedited := &model.Alarm{
		AlarmIdentifier: "11109",
		AdditionalInfo: map[string]string{
			"managed_object_instance": "Device.FaultMgmt.ExpeditedEvent.",
			"additional_text": "LTE0",
			"additional_information": "LTE0(73828545);S1setup fail. The MME ID: MME1",
		},
	}
	assert.Equal(t, activeAlarmMatchKey(current), activeAlarmMatchKey(expedited))
	assert.Equal(t, "11109|LTE0(73828545)", activeAlarmMatchKey(current))
}
```

Also assert:

- `Device.Radio.1` and `Device.Radio.2` remain different.
- `LTE0(73828545)` and `LTE1(73828546)` remain different.
- CurrentAlarm table indices alone do not create different keys.
- whitespace-only differences in AdditionalInformation do not change the key.
- when no stable object hint exists, normalized AdditionalInformation remains the compatibility fallback.

- [ ] **Step 2: Run the identity tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestActiveAlarm(MatchKey|Qualifier)' -count=1
```

Expected: cross-channel equality tests fail because current code returns raw CurrentAlarm/ExpeditedEvent MOI.

- [ ] **Step 3: Implement the stable qualifier**

In `alarm_identity.go`:

- add the `additional_text` key constant;
- normalize whitespace with `strings.Join(strings.Fields(value), " ")`;
- reject MOI values prefixed by `Device.FaultMgmt.CurrentAlarm.`, `Device.FaultMgmt.ExpeditedEvent.`, and `Device.FaultMgmt.HistoryEvent.`;
- when AdditionalInformation begins with `AdditionalText + "("`, use the text before the first semicolon as the object scope;
- otherwise use AdditionalText, then normalized AdditionalInformation;
- retain the existing identifier-only fallback when the qualifier is empty.

Add `findActiveAlarmsByIdentifier` that returns every alarm whose trimmed identifier matches the candidate, without applying a qualifier.

- [ ] **Step 4: Run identity tests and existing same-ID tests**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestActiveAlarm|TestComputeDiff_SeparatesSameIdentifier' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add omcgo/internal/alarm/alarm_identity.go omcgo/internal/alarm/alarm_identity_test.go
git commit -m "fix(alarm): 统一跨通道告警实例标识"
```

### Task 2: Safe AutoClear Fallback and Redis Key Consistency

**Files:**
- Modify: `omcgo/internal/alarm/engine.go`
- Modify: `omcgo/internal/alarm/engine_test.go`
- Modify: `omcgo/internal/alarm/metrics.go`
- Create: `omcgo/internal/alarm/metrics_test.go`

**Interfaces:**
- Consumes: `activeAlarmMatchKey` and `findActiveAlarmsByIdentifier` from Task 1.
- Preserves: `func (e *AlarmEngine) AutoClear(ctx context.Context, alarm *model.Alarm) error`.
- Produces: exact-match-first, unique-identifier-fallback clear semantics.
- Produces: `AlarmMetrics.ReconciliationTotal` with bounded `result` labels.

- [ ] **Step 1: Write failing AutoClear tests**

Add:

```go
func TestAutoClearFallsBackToUniqueIdentifierWhenQualifierChanged(t *testing.T) {
	// Store one NewAlarm with GPS unavailable information.
	// Clear it with the same identifier but a different clear message and transient MOI.
	// Assert active is empty and history contains exactly one alarm.
}

func TestAutoClearDoesNotGuessWhenIdentifierHasMultipleCandidates(t *testing.T) {
	// Store two 11184 alarms scoped to cell=1 and cell=2.
	// Clear with only AlarmIdentifier.
	// Assert both remain active and history is empty.
}
```

Extend the Redis fake or existing Redis-backed test to prove `clearActiveAlarm` and `ClearBySync` delete `activeAlarmMatchKey(alarm)`, not the bare identifier.

Add a metrics test that registers `NewAlarmMetrics` on a fresh registry and verifies these label values are usable:

```text
clear_exact_miss
clear_unique_fallback
clear_ambiguous
sync_duplicate_cleared
```

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestAutoClear(FallsBack|DoesNotGuess)|TestClear.*Redis' -count=1
```

Expected: unique fallback test fails; Redis deletion test reports the wrong field key.

- [ ] **Step 3: Implement AutoClear fallback**

Update `AutoClear`:

1. load all device alarms once;
2. try `findMatchingActiveAlarm`;
3. if no exact match, call `findActiveAlarmsByIdentifier`;
4. clear the only candidate;
5. return nil without clearing when zero candidates;
6. log a structured warning and return nil when multiple candidates exist.

Use `activeAlarmMatchKey(alarm)` in `clearActiveAlarm` and `ClearBySync` Redis deletion paths. Keep PostgreSQL as the source of truth and preserve current warning-only behavior on Redis deletion failure.

Add `ReconciliationTotal *prometheus.CounterVec` to `AlarmMetrics`, register it as `omc_alarm_reconciliation_total{result=...}`, and increment:

- `clear_exact_miss` when exact instance matching fails;
- `clear_unique_fallback` when the only same-identifier candidate is cleared;
- `clear_ambiguous` when multiple candidates prevent a direct clear.

- [ ] **Step 4: Run focused engine tests**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestAutoClear|TestClear.*Redis' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add omcgo/internal/alarm/engine.go omcgo/internal/alarm/engine_test.go omcgo/internal/alarm/metrics.go omcgo/internal/alarm/metrics_test.go
git commit -m "fix(alarm): 安全回退清除唯一告警实例"
```

### Task 3: ExpeditedEvent Batch-Level Alarm Sync

**Files:**
- Create: `omcgo/internal/alarm/sync_request.go`
- Modify: `omcgo/internal/alarm/receiver.go`
- Modify: `omcgo/internal/alarm/expedited_receiver.go`
- Modify: `omcgo/internal/alarm/expedited_receiver_test.go`
- Modify: `omcgo/internal/alarm/receiver_test.go`

**Interfaces:**
- Produces: `publishAlarmSyncRequest(context.Context, event.EventBus, *zap.Logger, string)`.
- Preserves: existing NATS subject `event.SubjectAlarmSyncRequested`.
- Consumed by: `AlarmReceiver` and `ExpeditedEventReceiver`.

- [ ] **Step 1: Write failing batch publication tests**

Add tests asserting:

```go
func TestExpeditedEventReceiverPublishesOneSyncRequestPerValidBatch(t *testing.T) {
	// Build one payload containing two valid NewAlarm instances.
	// Assert exactly one SubjectAlarmSyncRequested event was published.
}

func TestExpeditedEventReceiverPublishesSyncAfterPartialProcessingFailure(t *testing.T) {
	// Include one valid event and one event that fails engine processing.
	// Assert one sync request is still published.
}

func TestExpeditedEventReceiverDoesNotPublishSyncForInvalidBatch(t *testing.T) {
	// Invalid notification type only.
	// Assert no sync request.
}
```

Update the existing AlarmReceiver test to continue asserting one sync request through the shared helper.

- [ ] **Step 2: Run receiver tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestExpeditedEventReceiver.*Sync|TestAlarmReceiver.*Sync' -count=1
```

Expected: expedited sync assertions fail because the receiver currently returns without publishing.

- [ ] **Step 3: Implement shared publisher and batch trigger**

- Move the existing event construction/publish code from the `AlarmReceiver` method into `sync_request.go`.
- In `ExpeditedEventReceiver.handleExpeditedAlarmEvent`, track whether at least one event passed validation.
- After the loop, publish once when the batch contained a valid New/Changed/Cleared event, including when an individual engine operation failed.
- Do not publish for an empty or wholly invalid batch.

- [ ] **Step 4: Run all receiver tests**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'Test.*Receiver' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 3**

```bash
git add omcgo/internal/alarm/sync_request.go omcgo/internal/alarm/receiver.go omcgo/internal/alarm/receiver_test.go omcgo/internal/alarm/expedited_receiver.go omcgo/internal/alarm/expedited_receiver_test.go
git commit -m "fix(alarm): 加急告警后触发全量同步"
```

### Task 4: Duplicate-Aware CurrentAlarm Reconciliation

**Files:**
- Modify: `omcgo/internal/alarm/sync_model.go`
- Modify: `omcgo/internal/alarm/sync_model_test.go`
- Modify: `omcgo/internal/alarm/sync_processor.go`
- Modify: `omcgo/internal/alarm/sync_processor_test.go`

**Interfaces:**
- Produces:

```go
type AlarmUpdate struct {
	Local  *model.Alarm
	Remote *model.Alarm
}

type AlarmDiff struct {
	ToAdd             []*model.Alarm
	ToUpdate          []AlarmUpdate
	ToClear           []*model.Alarm
	ToClearDuplicates []*model.Alarm
}
```

- Produces: deterministic local keeper selection based on remote RaisedAt, then newest local RaisedAt, CreatedAt, and ID.
- Consumed by: `AlarmSyncProcessor.processSync`.

- [ ] **Step 1: Write failing diff tests**

Add tests for:

```go
func TestComputeDiffKeepsClosestRaisedAlarmAndClearsDuplicate(t *testing.T) {
	// Remote 11109 raised at 16:25.
	// Local equivalent CurrentAlarm row raised at 15:55 and ExpeditedEvent row at 16:25.
	// Assert the 16:25 local row is the update target and the 15:55 row is a duplicate clear.
}

func TestComputeDiffClearsAllLocalInstancesMissingRemotely(t *testing.T) {
	// Two local rows for a key absent from remote.
	// Assert both appear in ToClear.
}

func TestComputeDiffPreservesParallelObjects(t *testing.T) {
	// Same identifier, LTE0 and LTE1 scopes.
	// Assert neither is treated as a duplicate of the other.
}
```

- [ ] **Step 2: Run sync model tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestComputeDiff(Keeps|ClearsAll|PreservesParallel)' -count=1
```

Expected: current map-based diff loses one local duplicate and cannot expose a duplicate-clear list.

- [ ] **Step 3: Implement grouped diff and deterministic keeper**

- group remote and local alarms by `activeAlarmMatchKey`;
- choose one remote record per key using latest RaisedAt for malformed duplicate remote rows;
- choose the local keeper whose RaisedAt has the smallest absolute distance to remote RaisedAt;
- break ties by latest local RaisedAt, latest CreatedAt, then lexical ID;
- put non-keeper locals in `ToClearDuplicates`;
- put every local record for remote-absent keys in `ToClear`;
- return direct local/remote update pairs so `sync_processor.go` no longer reconstructs a lossy local map.

- [ ] **Step 4: Adapt sync processor and write the issue #227 fixture**

In `processSync`:

- apply each `AlarmUpdate` directly;
- clear every `ToClear` alarm normally;
- before clearing each duplicate, set:

```go
clearedBy := "system:alarm_sync_duplicate"
clearNote := "duplicate active alarm reconciled against device CurrentAlarm"
alarm.ClearedBy = &clearedBy
alarm.ClearNote = &clearNote
```

- increment `ReconciliationTotal` with `sync_duplicate_cleared` after each duplicate is successfully archived and removed.

Add a regression fixture with six local alarms:

- old CurrentAlarm `11109/11112` at 15:55;
- ExpeditedEvent `11189/11190` at 15:59;
- new ExpeditedEvent `11109/11112` at 16:25;

and two remote CurrentAlarm alarms `11109/11112` raised at 16:25. Assert:

- two active alarms remain;
- four alarms are archived;
- latest `11109/11112` IDs remain active;
- duplicate history records contain the duplicate clear audit fields;
- `SyncResult.Cleared == 4` and all failed counters are zero.

- [ ] **Step 5: Run sync model and processor tests**

Run:

```bash
cd omcgo
go test ./internal/alarm -run 'TestComputeDiff|TestProcessSync' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add omcgo/internal/alarm/sync_model.go omcgo/internal/alarm/sync_model_test.go omcgo/internal/alarm/sync_processor.go omcgo/internal/alarm/sync_processor_test.go
git commit -m "fix(alarm): 全量同步收敛重复活动告警"
```

### Task 5: Full Verification and Browser Regression

**Files:**
- Modify only if verification exposes a defect in Task 1-4 files.

**Interfaces:**
- Verifies the complete issue #227 behavior without introducing additional scope.

- [ ] **Step 1: Format modified Go files**

Run:

```bash
gofmt -w omcgo/internal/alarm/alarm_identity.go omcgo/internal/alarm/alarm_identity_test.go omcgo/internal/alarm/engine.go omcgo/internal/alarm/engine_test.go omcgo/internal/alarm/metrics.go omcgo/internal/alarm/metrics_test.go omcgo/internal/alarm/sync_request.go omcgo/internal/alarm/receiver.go omcgo/internal/alarm/receiver_test.go omcgo/internal/alarm/expedited_receiver.go omcgo/internal/alarm/expedited_receiver_test.go omcgo/internal/alarm/sync_model.go omcgo/internal/alarm/sync_model_test.go omcgo/internal/alarm/sync_processor.go omcgo/internal/alarm/sync_processor_test.go
```

- [ ] **Step 2: Run the alarm package**

```bash
cd omcgo
go test ./internal/alarm/... -count=1
```

Expected: PASS with zero failures.

- [ ] **Step 3: Run carrier tests and build**

```bash
cd omcgo
go test ./internal/core/carrier/... -count=1
go build ./...
```

Expected: both commands exit 0.

- [ ] **Step 4: Review the complete diff**

Run:

```bash
git diff --check
git status --short
git diff --stat
```

Confirm there are no frontend, migration, generated, or unrelated changes.

- [ ] **Step 5: Browser and environment verification**

On the test environment:

1. deploy the built backend through the existing environment workflow;
2. trigger one Alarm Sync for `120200055922C8B0068`;
3. verify the OMC active list contains the same `11109/11112` pair as LMT;
4. verify the four stale rows are historical with system clear audit fields;
5. generate or observe one ExpeditedEvent and verify one subsequent alarm sync device task.

Do not clear alarms manually as part of this verification.

- [ ] **Step 6: Final commit if verification required fixes**

If Task 5 changes code:

```bash
git add omcgo/internal/alarm
git commit -m "test(alarm): 完善告警实例收敛回归验证"
```
