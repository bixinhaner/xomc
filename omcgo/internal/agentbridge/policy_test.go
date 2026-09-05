package agentbridge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBindScopedOperationPathIgnoresRemoteLiteralPath(t *testing.T) {
	resources := []ResourceRef{
		{Type: "task", ID: "task-in-scope", Role: "task"},
		{Type: "device", ID: "device-in-scope", Role: "device"},
	}
	devicePath, err := bindScopedOperationPath("get.devices.by_id", "/forged", resources)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/devices/device-in-scope", devicePath)
	taskPath, err := bindScopedOperationPath("get.devices.tasks.by_task_id", "/forged", resources)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/devices/tasks/task-in-scope", taskPath)
}

func TestBindScopedOperationPathFailsClosed(t *testing.T) {
	path, err := bindScopedOperationPath("get.alarms.active", "/forged", []ResourceRef{{Type: "device", ID: "device-1"}})
	require.NoError(t, err)
	require.Equal(t, "/api/v1/alarms/active", path)
	_, err = bindScopedOperationPath("get.devices.tasks.by_task_id", "", []ResourceRef{{Type: "device", ID: "device-1"}})
	require.ErrorContains(t, err, "task resource is missing")
	_, err = bindScopedOperationPath("delete.devices", "", nil)
	require.ErrorContains(t, err, "no local route binding")
}

func TestBindHandbookOperationPath(t *testing.T) {
	path, err := bindScopedOperationPath("get.agent.handbook.manifest", "/forged", nil)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/agent/handbook/manifest", path)

	path, err = bindScopedOperationPath("get.agent.handbook.chunks.by_index", "/api/v1/agent/handbook/chunks/3", nil)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/agent/handbook/chunks/3", path)

	_, err = bindScopedOperationPath("get.agent.handbook.chunks.by_index", "/api/v1/agent/handbook/chunks/../admin", nil)
	require.ErrorContains(t, err, "numeric index")
}

func TestValidateFindingDeliveryRequiresEvidenceAndScope(t *testing.T) {
	delivery := FindingDelivery{
		ContractVersion: ContractVersion, DeliveryID: "delivery-1", FindingID: "finding-1", RunID: "run-1",
		ScenarioKey: "task-failure-analysis", Finding: AgentFinding{
			SchemaVersion: ContractVersion, Title: "title", Summary: "summary", Severity: "high", Confidence: .8,
			ResourceRefs: []ResourceRef{{Type: "task", ID: "task-1", Role: "task"}},
			Facts:        []map[string]any{{"id": "fact-1", "text": "fact", "evidenceRefs": []any{"tool:one"}}},
		},
	}
	require.NoError(t, validateFindingDelivery(delivery))
	delivery.Finding.Facts[0]["evidenceRefs"] = []any{}
	require.ErrorContains(t, validateFindingDelivery(delivery), "evidenceRefs")
	delivery.Finding.Facts = nil
	delivery.Finding.ResourceRefs = nil
	require.ErrorContains(t, validateFindingDelivery(delivery), "authoritative resource")
}
