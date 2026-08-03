package indicator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	listCalled     bool
	listFilter     IndicatorListFilter
	listResp       *model.ListResponse[IndicatorListItem]
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
	f.listCalled = true
	f.listFilter = filter
	if f.listResp != nil {
		return f.listResp, nil
	}
	return model.NewListResponse([]IndicatorListItem{}, 0, filter.Page, filter.PageSize), nil
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

type fakeEnabledRepo struct {
	enabledIDs  []string
	operators   map[DeviceType][]string
	createCalls int
	createdIDs  []string
	createTx    pgx.Tx
	deleteCalls int
	deletedIDs  []string
	deleteTx    pgx.Tx
}

func (f *fakeEnabledRepo) List(context.Context, DeviceType, string) ([]string, error) {
	return append([]string(nil), f.enabledIDs...), nil
}

func (f *fakeEnabledRepo) ListTx(context.Context, DeviceType, string, pgx.Tx) ([]string, error) {
	return append([]string(nil), f.enabledIDs...), nil
}

func (f *fakeEnabledRepo) ListOperatorCodes(_ context.Context, dt DeviceType) ([]string, error) {
	return append([]string(nil), f.operators[dt]...), nil
}

func (f *fakeEnabledRepo) BatchCreate(_ context.Context, _ DeviceType, _ string, indicatorIDs []string, tx pgx.Tx) error {
	f.createCalls++
	f.createdIDs = append([]string(nil), indicatorIDs...)
	f.createTx = tx
	return nil
}

func (f *fakeEnabledRepo) BatchDelete(_ context.Context, _ DeviceType, _ string, indicatorIDs []string, tx pgx.Tx) error {
	f.deleteCalls++
	f.deletedIDs = append([]string(nil), indicatorIDs...)
	f.deleteTx = tx
	return nil
}

func (f *fakeEnabledRepo) Exists(context.Context, DeviceType, string, string) (bool, error) {
	return false, nil
}

type fakeDashboardLayoutRef struct {
	referenced []string
	err        error
}

func (f fakeDashboardLayoutRef) ReferencedIndicators(context.Context, DeviceType, []string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]string(nil), f.referenced...), nil
}

func item(id, arithmetic string) IndicatorListItem {
	var a *string
	if arithmetic != "" {
		a = &arithmetic
	}
	return IndicatorListItem{PerfIndicator: PerfIndicator{ID: id, Arithmetic: a}}
}

func dependencyItem(id, arithmetic, isCounter string) IndicatorListItem {
	value := arithmetic
	return IndicatorListItem{PerfIndicator: PerfIndicator{
		ID:         id,
		Arithmetic: &value,
		IsCounter:  isCounter,
	}}
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

func TestDisableIndicators_RejectsDashboardLayoutReferences(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{}
	svc := &IndicatorManagementService{
		enabledRepo:     enabledRepo,
		dashboardLayout: fakeDashboardLayoutRef{referenced: []string{"K900010076"}},
	}

	err := svc.DisableIndicators(context.Background(), &EnableIndicatorsRequest{
		DeviceType:   string(DeviceTypeENB),
		OperatorCode: "default",
		IndicatorIDs: []string{"K900010076"},
	})

	if !errors.Is(err, ErrIndicatorUsedByDashboardLayout) {
		t.Fatalf("DisableIndicators error = %v, want ErrIndicatorUsedByDashboardLayout", err)
	}
	if !errors.Is(err, commonerrors.ErrInvalidInput) {
		t.Fatalf("DisableIndicators error = %v, want ErrInvalidInput wrapper", err)
	}
	if enabledRepo.deleteCalls != 0 {
		t.Fatalf("BatchDelete calls = %d, want 0 when dashboard references metric", enabledRepo.deleteCalls)
	}
}

func TestDisableIndicators_AllowsWhenDashboardDoesNotReference(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010040", "C000000001", "0"),
			dependencyItem("C000000001", "C000000001", "1"),
		}},
		enabledRepo:     enabledRepo,
		dashboardLayout: fakeDashboardLayoutRef{},
		pool:            beginner,
	}

	err := svc.DisableIndicators(context.Background(), &EnableIndicatorsRequest{
		DeviceType:   string(DeviceTypeENB),
		OperatorCode: "default",
		IndicatorIDs: []string{"K900010040"},
	})

	if err != nil {
		t.Fatalf("DisableIndicators returned error: %v", err)
	}
	if enabledRepo.deleteCalls != 1 {
		t.Fatalf("BatchDelete calls = %d, want 1", enabledRepo.deleteCalls)
	}
	if got := enabledRepo.deletedIDs; len(got) != 1 || got[0] != "K900010040" {
		t.Fatalf("deleted IDs = %v, want [K900010040]", got)
	}
}

func TestEnableIndicators_AutomaticallyEnablesDependencyClosure(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010076", "C000000216/C000000273*100", "0"),
			dependencyItem("C000000216", "C000000216", "1"),
			dependencyItem("C000000273", "C000000273", "1"),
		}},
		enabledRepo: enabledRepo,
		pool:        beginner,
	}

	err := svc.EnableIndicators(context.Background(), &EnableIndicatorsRequest{
		DeviceType:   string(DeviceTypeENB),
		OperatorCode: "default",
		IndicatorIDs: []string{"K900010076"},
	})
	if err != nil {
		t.Fatalf("EnableIndicators returned error: %v", err)
	}
	want := []string{"C000000216", "C000000273", "K900010076"}
	if got := enabledRepo.createdIDs; !equalStrings(got, want) {
		t.Fatalf("created IDs = %v, want %v", got, want)
	}
	if enabledRepo.createTx == nil {
		t.Fatal("dependency closure must be enabled in one transaction")
	}
	if beginner.tx.commits != 1 {
		t.Fatalf("transaction commits = %d, want 1", beginner.tx.commits)
	}
}

func TestReconcileEnabledDependencies_RepairsPersistedKPIClosure(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{
		enabledIDs: []string{"K900010040"},
		operators:  map[DeviceType][]string{DeviceTypeENB: {"default"}},
	}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010040", "(C000190005-C000190009)*8/C000190007", "0"),
			dependencyItem("C000190005", "C000190005", "1"),
			dependencyItem("C000190009", "C000190009", "1"),
			dependencyItem("C000190007", "C000190007", "1"),
		}},
		enabledRepo: enabledRepo,
		pool:        beginner,
	}

	err := svc.ReconcileEnabledDependencies(context.Background())
	if err != nil {
		t.Fatalf("ReconcileEnabledDependencies returned error: %v", err)
	}
	want := []string{"C000190005", "C000190007", "C000190009", "K900010040"}
	if got := enabledRepo.createdIDs; !equalStrings(got, want) {
		t.Fatalf("created IDs = %v, want %v", got, want)
	}
	if enabledRepo.createTx == nil {
		t.Fatal("persisted dependency closure must be repaired in one transaction")
	}
	if beginner.tx.commits != 1 {
		t.Fatalf("transaction commits = %d, want 1", beginner.tx.commits)
	}
}

func TestReconcileEnabledDependencies_FailsWhenCacheVersionBumpFails(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{
		enabledIDs: []string{"K900010040"},
		operators:  map[DeviceType][]string{DeviceTypeENB: {"default"}},
	}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	bumpErr := errors.New("redis unavailable")
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010040", "C000190005", "0"),
			dependencyItem("C000190005", "C000190005", "1"),
		}},
		enabledRepo: enabledRepo,
		pool:        beginner,
		cacheVersionBumper: func(context.Context) error {
			return bumpErr
		},
	}

	err := svc.ReconcileEnabledDependencies(context.Background())
	if !errors.Is(err, bumpErr) {
		t.Fatalf("ReconcileEnabledDependencies error = %v, want %v", err, bumpErr)
	}
	if beginner.tx.commits != 1 {
		t.Fatalf("transaction commits = %d, want 1", beginner.tx.commits)
	}
}

func TestDisableIndicators_RejectsCounterRequiredByEnabledKPI(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{
		enabledIDs: []string{"C000000216", "C000000273", "K900010076"},
	}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010076", "C000000216/C000000273*100", "0"),
			dependencyItem("C000000216", "C000000216", "1"),
			dependencyItem("C000000273", "C000000273", "1"),
		}},
		enabledRepo: enabledRepo,
		pool:        beginner,
	}

	err := svc.DisableIndicators(context.Background(), &EnableIndicatorsRequest{
		DeviceType:   string(DeviceTypeENB),
		OperatorCode: "default",
		IndicatorIDs: []string{"C000000216"},
	})
	if !errors.Is(err, ErrEnabledKPIDependency) {
		t.Fatalf("DisableIndicators error = %v, want ErrEnabledKPIDependency", err)
	}
	if enabledRepo.deleteCalls != 0 {
		t.Fatalf("BatchDelete calls = %d, want 0", enabledRepo.deleteCalls)
	}
}

func TestDisableIndicators_AllowsKPIAndDependencyTogether(t *testing.T) {
	enabledRepo := &fakeEnabledRepo{
		enabledIDs: []string{"C000000216", "C000000273", "K900010076"},
	}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := &IndicatorManagementService{
		indicatorRepo: &fakeIndicatorRepo{items: []IndicatorListItem{
			dependencyItem("K900010076", "C000000216/C000000273*100", "0"),
			dependencyItem("C000000216", "C000000216", "1"),
			dependencyItem("C000000273", "C000000273", "1"),
		}},
		enabledRepo: enabledRepo,
		pool:        beginner,
	}

	err := svc.DisableIndicators(context.Background(), &EnableIndicatorsRequest{
		DeviceType:   string(DeviceTypeENB),
		OperatorCode: "default",
		IndicatorIDs: []string{"K900010076", "C000000216"},
	})
	if err != nil {
		t.Fatalf("DisableIndicators returned error: %v", err)
	}
	if enabledRepo.deleteTx == nil || beginner.tx.commits != 1 {
		t.Fatalf("disable was not committed transactionally: tx=%v commits=%d", enabledRepo.deleteTx != nil, beginner.tx.commits)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestRESTHandler_ListIndicators_DefaultsOperatorCodeForEnabledFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeIndicatorRepo{}
	svc := &IndicatorManagementService{indicatorRepo: repo}
	handler := NewRESTHandler(svc, nil, nil)
	router := gin.New()
	router.GET("/indicators", handler.ListIndicators)

	req := httptest.NewRequest(http.MethodGet, "/indicators?deviceType=ENB&isEnabled=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !repo.listCalled {
		t.Fatal("ListIndicators must call repository List")
	}
	if repo.listFilter.OperatorCode == nil {
		t.Fatal("operatorCode filter = nil, want default")
	}
	if got := *repo.listFilter.OperatorCode; got != "default" {
		t.Fatalf("operatorCode filter = %q, want default", got)
	}
	if repo.listFilter.IsEnabled == nil || *repo.listFilter.IsEnabled != "1" {
		t.Fatalf("isEnabled filter = %v, want 1", repo.listFilter.IsEnabled)
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

func TestNormalizeDurationArithmetic_UsesRegisteredCounterPerDeviceType(t *testing.T) {
	cases := []struct {
		dt   DeviceType
		want string
	}{
		{DeviceTypeENB, "OTHER_CellServiceTime/C000060273*100"},
		{DeviceTypeGNB, "OTHER_CellServiceTime/C010120025*100"},
		{DeviceTypeGSM, "OTHER_CellServiceTime/CGSM0080001*100"},
	}
	for _, tc := range cases {
		t.Run(string(tc.dt), func(t *testing.T) {
			got := normalizeDurationArithmetic(tc.dt, "OTHER_CellServiceTime/Duration*100")
			if got != tc.want {
				t.Fatalf("normalizeDurationArithmetic() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeDurationArithmetic_ReplacesCompleteTokenOnly(t *testing.T) {
	got := normalizeDurationArithmetic(DeviceTypeENB, "DurationValue+MyDuration+Duration+Duration_Seconds")
	want := "DurationValue+MyDuration+C000060273+Duration_Seconds"
	if got != want {
		t.Fatalf("normalizeDurationArithmetic() = %q, want %q", got, want)
	}
}

func TestCompileRuntimeArithmetic_UnknownDeviceTypeLeavesFormulaUnchanged(t *testing.T) {
	formula := "C001/Duration+MyDuration"
	got := CompileRuntimeArithmetic(DeviceType("UNKNOWN"), formula)
	if got != formula {
		t.Fatalf("CompileRuntimeArithmetic() = %q, want unchanged %q", got, formula)
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

func TestUpsertPlatformFormula_AcceptsDurationReservedWord(t *testing.T) {
	platformRepo := &fakePlatformRepo{deleteRows: 1}
	beginner := &fakeBeginner{tx: &fakeTx{}}
	svc := newFormulaWriteTestService(platformRepo, beginner, nil)

	out, err := svc.UpsertPlatformFormula(context.Background(), DeviceTypeENB, "K900000001", "BLQ", "C000200015/Duration*100")
	if err != nil {
		t.Fatalf("UpsertPlatformFormula returned error: %v", err)
	}
	if out.Formula != "C000200015/Duration*100" {
		t.Fatalf("platform formula should keep editable Duration keyword, got %q", out.Formula)
	}
	if platformRepo.batchCalls != 1 {
		t.Fatalf("batch create calls=%d, want 1", platformRepo.batchCalls)
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
