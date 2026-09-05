package agentruntime

import "strings"

// The planner sees a read-only operational catalog generated from live routes,
// not a list of business scenarios. Sensitive administration, file downloads,
// and GET endpoints with potentially surprising side effects are excluded.
func (e *ToolExecutor) AssistantCapabilityCatalog() []map[string]any {
	out := []map[string]any{}
	for _, r := range e.HandbookRouteExport().Routes {
		if r.Method != "GET" || r.Risk != "read" || strings.Contains(r.Path, "*") {
			continue
		}
		if !strings.HasPrefix(r.Path, "/api/v1/devices") && !strings.HasPrefix(r.Path, "/api/v1/alarms/") {
			continue
		}
		denied := false
		for _, word := range []string{"export", "download", "file", "password", "credential", "secret", "backup", "reboot", "reset", "sync", "dispatch", "execute"} {
			if strings.Contains(strings.ToLower(r.Path), word) {
				denied = true
				break
			}
		}
		if denied {
			continue
		}
		description := r.Summary + "\n" + r.Description
		if len([]rune(description)) > 3500 {
			description = string([]rune(description)[:3500])
		}
		scoped := r.OperationID == "get.devices.by_id" || r.OperationID == "get.alarms.active" || r.OperationID == "get.alarms.history"
		out = append(out, map[string]any{"operationId": r.OperationID, "title": r.Title, "description": description, "path": r.Path, "deviceScoped": scoped})
	}
	return out
}
