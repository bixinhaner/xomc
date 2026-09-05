package provider

import (
	"os"
	"strings"
	"testing"
)

// Background callbacks can memoize the handbook on their first invocation.
// Guard the initialization barrier against moving any producer above it.
func TestAgentWorkersStartAfterHandbookAndPermissions(t *testing.T) {
	source, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(source)
	handbook := strings.Index(s, "agentRuntimeHandler.PrepareHandbook()")
	permissions := strings.Index(s, "if err := syncApiEndpoints(c, ad)")
	if handbook < 0 || permissions <= handbook {
		t.Fatal("missing route/permission initialization barrier")
	}
	for _, activation := range []string{
		"subscriber.Subscribe()", "c.AlarmEngine.SetRaisedHook(",
		"assistantService.Start()", "forwarder.Start()", "toolWorker.Start()",
		"findingWorker.Start()", "heartbeatWorker.Start()", "dailySummaryScheduler.Start()",
	} {
		if position := strings.Index(s, activation); position <= permissions {
			t.Errorf("%s must activate after handbook and permission initialization", activation)
		}
	}
}
