package indicator

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// fakeIndicatorRepo is a minimal IndicatorRepository for exercising buildIDMap.
// Only ListAll does real work; every other method is an unused stub so the type
// satisfies the interface. It records the filter ListAll was called with so the
// test can assert buildIDMap bypasses the (empty-on-fresh-stack) group table and
// scopes by device type.
type fakeIndicatorRepo struct {
	items          []IndicatorListItem
	listAllErr     error
	listAllFilter  IndicatorListFilter
	listAllCalled  bool
	groupIDsCalled bool // must stay false: regression guard against the old group-table path
}

func (f *fakeIndicatorRepo) ListAll(ctx context.Context, filter IndicatorListFilter) ([]IndicatorListItem, error) {
	f.listAllCalled = true
	f.listAllFilter = filter
	if f.listAllErr != nil {
		return nil, f.listAllErr
	}
	return f.items, nil
}

func (f *fakeIndicatorRepo) GetIDsByGroupID(ctx context.Context, dt DeviceType, groupID string) ([]string, error) {
	f.groupIDsCalled = true
	return nil, nil
}

// ── unused stubs (interface satisfaction only) ──────────────────────────────
func (f *fakeIndicatorRepo) List(ctx context.Context, filter IndicatorListFilter) (*model.ListResponse[IndicatorListItem], error) {
	return nil, nil
}
func (f *fakeIndicatorRepo) GetByID(ctx context.Context, dt DeviceType, id string) (*PerfIndicator, error) {
	return nil, nil
}
func (f *fakeIndicatorRepo) Create(ctx context.Context, dt DeviceType, indicator *PerfIndicator, tx pgx.Tx) error {
	return nil
}
func (f *fakeIndicatorRepo) Update(ctx context.Context, dt DeviceType, id string, req *UpdateIndicatorRequest, tx pgx.Tx) error {
	return nil
}
func (f *fakeIndicatorRepo) Delete(ctx context.Context, dt DeviceType, id string, tx pgx.Tx) error {
	return nil
}
func (f *fakeIndicatorRepo) DeleteByGroupID(ctx context.Context, dt DeviceType, groupID string, tx pgx.Tx) error {
	return nil
}
func (f *fakeIndicatorRepo) GetNextKPIID(ctx context.Context, dt DeviceType, operatorCode string) (string, error) {
	return "", nil
}
func (f *fakeIndicatorRepo) GetNextCounterID(ctx context.Context) (string, error) {
	return "", nil
}
func (f *fakeIndicatorRepo) ListByIDs(ctx context.Context, dt DeviceType, ids []string) ([]*PerfIndicator, error) {
	return nil, nil
}

func item(id, arithmetic string) IndicatorListItem {
	var a *string
	if arithmetic != "" {
		a = &arithmetic
	}
	return IndicatorListItem{PerfIndicator: PerfIndicator{ID: id, Arithmetic: a}}
}

// TestBuildIDMap_BypassesGroupTable is the #134 regression guard: on a fresh
// stack the indicator group table only holds a `default` placeholder and every
// indicator's group_id is dangling, so the legacy group-table walk yielded an
// empty idMap and the validator rejected every legal formula. buildIDMap must
// instead read all indicators directly via ListAll, scoped only by device type.
func TestBuildIDMap_BypassesGroupTable(t *testing.T) {
	repo := &fakeIndicatorRepo{
		items: []IndicatorListItem{
			item("C000200015", "C000200015"),
			item("C000000175", "C000000175"),
			item("defaultK900000001", "C000200015+C000000175"),
		},
	}
	svc := &IndicatorManagementService{indicatorRepo: repo}

	idMap, err := svc.buildIDMap(context.Background(), DeviceTypeENB)
	if err != nil {
		t.Fatalf("buildIDMap returned error: %v", err)
	}

	if !repo.listAllCalled {
		t.Fatal("buildIDMap must source indicators via ListAll")
	}
	if repo.groupIDsCalled {
		t.Fatal("buildIDMap must NOT walk the group table (GetIDsByGroupID) — that path is empty on a fresh stack")
	}
	if repo.listAllFilter.DeviceType != string(DeviceTypeENB) {
		t.Errorf("ListAll filter device type = %q, want ENB (platform/device dimension must be preserved)", repo.listAllFilter.DeviceType)
	}
	// No operator filter: custom KPIs written under any operator must be visible
	// so K90000* recursive expansion can resolve them.
	if repo.listAllFilter.OperatorCode != nil {
		t.Errorf("ListAll filter operator_code = %v, want nil (all operators)", *repo.listAllFilter.OperatorCode)
	}

	if len(idMap) != 3 {
		t.Fatalf("idMap size = %d, want 3", len(idMap))
	}
	if got := idMap["defaultK900000001"]; got != "C000200015+C000000175" {
		t.Errorf("custom KPI arithmetic = %q, want C000200015+C000000175", got)
	}
	if _, ok := idMap["C000200015"]; !ok {
		t.Error("idMap missing real counter C000200015")
	}
}

// TestBuildIDMap_DrivesValidator end-to-ends the fix: the idMap built from a
// real-counter list must let the FormulaValidator accept formulas referencing
// those counters while still rejecting unknown IDs.
func TestBuildIDMap_DrivesValidator(t *testing.T) {
	repo := &fakeIndicatorRepo{
		items: []IndicatorListItem{
			item("C000200015", "C000200015"),
			item("C000000175", "C000000175"),
			item("defaultK900000001", "C000200015+C000000175"),
		},
	}
	svc := &IndicatorManagementService{indicatorRepo: repo}

	idMap, err := svc.buildIDMap(context.Background(), DeviceTypeENB)
	if err != nil {
		t.Fatalf("buildIDMap returned error: %v", err)
	}
	v := NewFormulaValidator(idMap)

	cases := []struct {
		name      string
		formula   string
		wantValid bool
	}{
		{"real counter scaled", "C000200015*100", true},
		{"real counter add const", "C000000175+1", true},
		{"two real counters", "C000200015/C000000175", true},
		{"custom KPI recursive expand", "defaultK900000001*2", true},
		{"unknown ID rejected", "C999999999*100", false},
		{"unknown mixed with known rejected", "C000200015+UNKNOWN_COUNTER", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.Validate(tc.formula)
			if result.IsValid != tc.wantValid {
				t.Fatalf("Validate(%q).IsValid = %v, want %v (err=%q)", tc.formula, result.IsValid, tc.wantValid, result.ErrorMsg)
			}
		})
	}
}

// TestBuildIDMap_PropagatesRepoError ensures repo failures bubble up rather than
// silently producing an empty map (which would mask the bug class again).
func TestBuildIDMap_PropagatesRepoError(t *testing.T) {
	sentinel := errors.New("boom")
	repo := &fakeIndicatorRepo{listAllErr: sentinel}
	svc := &IndicatorManagementService{indicatorRepo: repo}

	_, err := svc.buildIDMap(context.Background(), DeviceTypeENB)
	if !errors.Is(err, sentinel) {
		t.Fatalf("buildIDMap error = %v, want wrap of sentinel", err)
	}
}

func TestUpdateAffectsRoute_OnlyRouteFields(t *testing.T) {
	cnName := "展示名"
	desc := "只影响展示"
	updator := "alice"
	groupID := "G1"
	dataType := "float"
	productTypes := "pico"
	indicatorLevel := "cell"
	status := "0"
	if updateAffectsRoute(&UpdateIndicatorRequest{
		CnName:            &cnName,
		EnDescription:     &desc,
		CnDescription:     &desc,
		GroupID:           &groupID,
		DataType:          &dataType,
		Updator:           &updator,
		ProductTypes:      &productTypes,
		IndicatorLevel:    &indicatorLevel,
		CalculatingStatus: &status,
	}) {
		t.Fatal("pure display/filter fields must not trigger KPI route invalidation")
	}

	enName := "CounterName"
	unit := "percent"
	arithmetic := "C000200015/C000000175"
	statisType := "pct"
	cases := []struct {
		name string
		req  *UpdateIndicatorRequest
	}{
		{"en name feeds route names", &UpdateIndicatorRequest{EnName: &enName}},
		{"unit feeds route defs", &UpdateIndicatorRequest{UnitID: &unit}},
		{"arithmetic feeds kpi formula", &UpdateIndicatorRequest{Arithmetic: &arithmetic}},
		{"statis type feeds aggregation", &UpdateIndicatorRequest{StatisType: &statisType}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !updateAffectsRoute(tc.req) {
				t.Fatal("route-affecting field did not trigger KPI route invalidation")
			}
		})
	}
}

func TestInvalidateRouteCache_BestEffort(t *testing.T) {
	calls := 0
	svc := (&IndicatorManagementService{}).WithRouteInvalidator(func(_ context.Context, trigger RouteInvalidationTrigger) error {
		calls++
		if trigger != RouteInvalidationTriggerFormulaWrite {
			t.Fatalf("trigger = %q, want %q", trigger, RouteInvalidationTriggerFormulaWrite)
		}
		return errors.New("redis down")
	})

	svc.InvalidateRouteCache(context.Background(), RouteInvalidationTriggerFormulaWrite)

	if calls != 1 {
		t.Fatalf("route invalidator calls = %d, want 1", calls)
	}
}

type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
	begins   int
}

func (f *fakeBeginner) Begin(context.Context) (pgx.Tx, error) {
	f.begins++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	if f.tx == nil {
		f.tx = &fakeTx{}
	}
	return f.tx, nil
}

type fakeTx struct {
	commitErr error
	commits   int
	rollbacks int
}

func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { return f, nil }
func (f *fakeTx) Commit(context.Context) error {
	f.commits++
	return f.commitErr
}
func (f *fakeTx) Rollback(context.Context) error {
	f.rollbacks++
	return nil
}
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (f *fakeTx) Conn() *pgx.Conn                                         { return nil }

type fakePlatformRepo struct {
	formulas    []*PlatformFormula
	batchErr    error
	deleteRows  int64
	deleteErr   error
	batchCalls  int
	deleteCalls int
	deletedID   string
	deletedPlat string
	batched     []*PlatformFormula
}

func (f *fakePlatformRepo) ListByIndicatorID(context.Context, DeviceType, string) ([]*PlatformFormula, error) {
	return f.formulas, nil
}
func (f *fakePlatformRepo) BatchCreate(_ context.Context, _ DeviceType, formulas []*PlatformFormula, _ pgx.Tx) error {
	f.batchCalls++
	f.batched = append([]*PlatformFormula(nil), formulas...)
	return f.batchErr
}
func (f *fakePlatformRepo) DeleteByIndicatorID(context.Context, DeviceType, string, pgx.Tx) error {
	return nil
}
func (f *fakePlatformRepo) DeleteByIndicatorAndPlatform(_ context.Context, _ DeviceType, indicatorID string, platformName string, _ pgx.Tx) (int64, error) {
	f.deleteCalls++
	f.deletedID = indicatorID
	f.deletedPlat = platformName
	return f.deleteRows, f.deleteErr
}
func (f *fakePlatformRepo) DeleteByIndicatorIDs(context.Context, DeviceType, []string, pgx.Tx) error {
	return nil
}
func (f *fakePlatformRepo) ListPlatformNames(context.Context, DeviceType) ([]string, error) {
	return nil, nil
}
func (f *fakePlatformRepo) ListByPlatform(context.Context, DeviceType, string) ([]*PlatformFormula, error) {
	return nil, nil
}

func newFormulaWriteTestService(platformRepo *fakePlatformRepo, beginner *fakeBeginner, invalidator RouteInvalidator) *IndicatorManagementService {
	return (&IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{
			items: []IndicatorListItem{
				item("C000200015", "C000200015"),
				item("C000000175", "C000000175"),
			},
		},
		platformRepo: platformRepo,
		pool:         beginner,
	}).WithRouteInvalidator(invalidator)
}

func TestUpsertPlatformFormula_CommitsThenInvalidatesRoute(t *testing.T) {
	platformRepo := &fakePlatformRepo{deleteRows: 1}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	var triggers []RouteInvalidationTrigger
	svc := newFormulaWriteTestService(platformRepo, beginner, func(_ context.Context, trigger RouteInvalidationTrigger) error {
		triggers = append(triggers, trigger)
		return nil
	})

	out, err := svc.UpsertPlatformFormula(context.Background(), DeviceTypeENB, "K900000001", " BLQ ", " C000200015/C000000175*100 ")
	if err != nil {
		t.Fatalf("UpsertPlatformFormula returned error: %v", err)
	}

	if out.PlatformName != "BLQ" || out.Formula != "C000200015/C000000175*100" {
		t.Fatalf("formula was not normalized: %#v", out)
	}
	if platformRepo.deleteCalls != 1 || platformRepo.deletedID != "K900000001" || platformRepo.deletedPlat != "BLQ" {
		t.Fatalf("delete before insert not scoped to indicator/platform: calls=%d id=%q platform=%q",
			platformRepo.deleteCalls, platformRepo.deletedID, platformRepo.deletedPlat)
	}
	if platformRepo.batchCalls != 1 || len(platformRepo.batched) != 1 {
		t.Fatalf("batch create calls=%d rows=%d, want 1/1", platformRepo.batchCalls, len(platformRepo.batched))
	}
	if beginner.tx.commits != 1 {
		t.Fatalf("commits = %d, want 1", beginner.tx.commits)
	}
	if len(triggers) != 1 || triggers[0] != RouteInvalidationTriggerFormulaWrite {
		t.Fatalf("route invalidation triggers = %v, want [%s]", triggers, RouteInvalidationTriggerFormulaWrite)
	}
}

func TestUpsertPlatformFormula_InvalidInputDoesNotBeginOrInvalidate(t *testing.T) {
	platformRepo := &fakePlatformRepo{}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	invalidations := 0
	svc := newFormulaWriteTestService(platformRepo, beginner, func(context.Context, RouteInvalidationTrigger) error {
		invalidations++
		return nil
	})

	errCases := []struct {
		name     string
		platform string
		formula  string
	}{
		{"empty platform", " ", "C000200015"},
		{"empty formula", "BLQ", " "},
		{"unknown counter", "BLQ", "C999999999"},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpsertPlatformFormula(context.Background(), DeviceTypeENB, "K900000001", tc.platform, tc.formula)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if beginner.begins != 0 {
		t.Fatalf("transaction begins = %d, want 0 for validation failures", beginner.begins)
	}
	if invalidations != 0 {
		t.Fatalf("route invalidations = %d, want 0", invalidations)
	}
}

func TestUpsertPlatformFormula_CommitFailureDoesNotInvalidate(t *testing.T) {
	platformRepo := &fakePlatformRepo{deleteRows: 1}
	beginner := &fakeBeginner{tx: &fakeTx{commitErr: errors.New("commit failed")}}
	invalidations := 0
	svc := newFormulaWriteTestService(platformRepo, beginner, func(context.Context, RouteInvalidationTrigger) error {
		invalidations++
		return nil
	})

	_, err := svc.UpsertPlatformFormula(context.Background(), DeviceTypeENB, "K900000001", "BLQ", "C000200015")
	if err == nil {
		t.Fatal("expected commit error")
	}
	if invalidations != 0 {
		t.Fatalf("route invalidations = %d, want 0 before successful commit", invalidations)
	}
}

func TestUpsertPlatformFormula_RouteInvalidationFailureDoesNotFailBusinessResult(t *testing.T) {
	platformRepo := &fakePlatformRepo{deleteRows: 1}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := newFormulaWriteTestService(platformRepo, beginner, func(context.Context, RouteInvalidationTrigger) error {
		return errors.New("redis down")
	})

	if _, err := svc.UpsertPlatformFormula(context.Background(), DeviceTypeENB, "K900000001", "BLQ", "C000200015"); err != nil {
		t.Fatalf("route invalidation failure must not fail committed business result: %v", err)
	}
}

func TestDeletePlatformFormula_NotFoundDoesNotCommitOrInvalidate(t *testing.T) {
	platformRepo := &fakePlatformRepo{deleteRows: 0}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	invalidations := 0
	svc := newFormulaWriteTestService(platformRepo, beginner, func(context.Context, RouteInvalidationTrigger) error {
		invalidations++
		return nil
	})

	err := svc.DeletePlatformFormula(context.Background(), DeviceTypeENB, "K900000001", "BLQ")
	if !errors.Is(err, commonerrors.ErrNotFound) {
		t.Fatalf("DeletePlatformFormula error = %v, want not found", err)
	}
	if beginner.tx.commits != 0 {
		t.Fatalf("commits = %d, want 0", beginner.tx.commits)
	}
	if invalidations != 0 {
		t.Fatalf("route invalidations = %d, want 0", invalidations)
	}
}
