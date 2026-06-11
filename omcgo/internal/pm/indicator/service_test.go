package indicator

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

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
