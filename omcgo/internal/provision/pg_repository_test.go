package provision

import (
	"strings"
	"testing"
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
