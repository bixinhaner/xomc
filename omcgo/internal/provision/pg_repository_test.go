package provision

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildProvisioningTaskListSQLIncludesSerialNumber(t *testing.T) {
	query, _, err := buildProvisioningTaskListSQL(ProvisioningTaskFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("build list SQL: %v", err)
	}

	normalized := strings.Join(strings.Fields(query), " ")
	if !strings.Contains(normalized, "LEFT JOIN devices d ON d.id = pt.device_id") {
		t.Fatalf("list SQL must join devices by task device_id: %s", normalized)
	}
	if !strings.Contains(normalized, "COALESCE(d.serial_number, '')") {
		t.Fatalf("list SQL must select the device serial number: %s", normalized)
	}
	for _, want := range []string{
		"LEFT JOIN plug_and_play_policies pp ON pp.id = pt.policy_id",
		"LEFT JOIN products prod ON prod.id = d.product_id",
		"COALESCE(prod.product_name, pp.product_class, d.product_class, '')",
		"COALESCE(pp.name, '')",
		"COALESCE(pp.execute_type, '')",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("list SQL must include task context %q: %s", want, normalized)
		}
	}
}

func TestBuildProvisioningTaskListSQLPolicyOnly(t *testing.T) {
	query, _, err := buildProvisioningTaskListSQL(ProvisioningTaskFilter{
		Page: 1, PageSize: 20, PolicyOnly: true,
	})
	if err != nil {
		t.Fatalf("build policy task list SQL: %v", err)
	}

	normalized := strings.Join(strings.Fields(query), " ")
	if !strings.Contains(normalized, "pt.policy_id IS NOT NULL") {
		t.Fatalf("policy-only list SQL must exclude discovery tasks: %s", normalized)
	}
}

func TestBuildProvisioningTaskListSQLIncludesOperationalFilters(t *testing.T) {
	startedAfter := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	startedBefore := time.Date(2026, 8, 7, 23, 59, 59, 0, time.UTC)
	policyID := uuid.New()
	query, args, err := buildProvisioningTaskListSQL(ProvisioningTaskFilter{
		Page: 1, PageSize: 20, PolicyOnly: true, RunningOnly: true,
		PolicyID: &policyID,
		Search:   "Policy A", ProductName: "BaiBNQ", Module: "self_config",
		StartedAfter: &startedAfter, StartedBefore: &startedBefore,
	})
	if err != nil {
		t.Fatalf("build filtered task list SQL: %v", err)
	}
	normalized := strings.Join(strings.Fields(query), " ")
	for _, want := range []string{
		"pt.status NOT IN",
		"pt.policy_id =",
		"d.serial_number ILIKE",
		"pp.name ILIKE",
		"LOWER(COALESCE(prod.product_name, pp.product_class, d.product_class, '')) = LOWER",
		"pt.current_step_name NOT LIKE",
		"COALESCE(pt.started_at, pt.created_at) >=",
		"COALESCE(pt.started_at, pt.created_at) <=",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("filtered task SQL must include %q: %s", want, normalized)
		}
	}
	if len(args) == 0 {
		t.Fatal("filtered task SQL must bind filter values")
	}
}
