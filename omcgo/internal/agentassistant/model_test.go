package agentassistant

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testDefinition() Definition {
	return Definition{Name: "My independent assistant", Goal: "Check the selected device using real observations.", Scope: Scope{Kind: "visible"}, Trigger: Trigger{Kind: "manual", Conditions: []Condition{}}, Operations: []string{"get.devices.by_id"}, Notify: "findings", CooldownMinutes: 30}
}
func testCapabilities() []Capability {
	return []Capability{{OperationID: "get.devices.by_id", Path: "/api/v1/devices/:id", DeviceScoped: true}, {OperationID: "get.devices", Path: "/api/v1/devices"}}
}
func TestDefinitionValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Definition)
		valid  bool
	}{
		{"arbitrary goal, no scenario enum", func(d *Definition) { d.Goal = "Investigate missing labels on this fleet" }, true},
		{"unknown capability", func(d *Definition) { d.Operations = []string{"post.devices.reboot"} }, false},
		{"duplicate capability", func(d *Definition) { d.Operations = append(d.Operations, d.Operations[0]) }, false},
		{"unresolved device", func(d *Definition) { d.Scope = Scope{Kind: "device", DeviceID: "East region"} }, false},
		{"unscopable list", func(d *Definition) {
			d.Scope = Scope{Kind: "device", DeviceID: "00000000-0000-4000-8000-000000000001"}
			d.Operations = []string{"get.devices"}
		}, false},
		{"interval budget", func(d *Definition) { d.Trigger = Trigger{Kind: "interval", IntervalMinutes: 1} }, false},
		{"custom local schedule", func(d *Definition) {
			d.Trigger = Trigger{Kind: "schedule", Time: "09:05", Timezone: "Asia/Shanghai", Weekdays: []int{1, 3, 5}}
		}, true},
		{"bad timezone", func(d *Definition) {
			d.Trigger = Trigger{Kind: "schedule", Time: "09:05", Timezone: "Not/AZone", Weekdays: []int{1}}
		}, false},
		{"unknown event", func(d *Definition) { d.Trigger = Trigger{Kind: "event", EventType: "invented"} }, false},
		{"unknown event field", func(d *Definition) {
			d.Trigger = Trigger{Kind: "event", EventType: Events[0].Type, Conditions: []Condition{{Field: "invented", Op: "eq", Value: true}}}
		}, false},
		{"numeric condition", func(d *Definition) {
			d.Trigger = Trigger{Kind: "event", EventType: Events[1].Type, Conditions: []Condition{{Field: "retryCount", Op: "gte", Value: float64(2)}}}
		}, true},
		{"invalid numeric condition", func(d *Definition) {
			d.Trigger = Trigger{Kind: "event", EventType: Events[1].Type, Conditions: []Condition{{Field: "retryCount", Op: "gte", Value: "2"}}}
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := testDefinition()
			tc.change(&d)
			err := ValidateDefinition(&d, testCapabilities())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
func TestTypedConditions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		conditions []Condition
		data       map[string]any
		want       bool
	}{
		{"missing fields are not a match", []Condition{{Field: "a", Op: "ne", Value: 1}}, map[string]any{}, false},
		{"no string/number coercion", []Condition{{Field: "a", Op: "eq", Value: "2"}}, map[string]any{"a": 2}, false},
		{"numeric input formats", []Condition{{Field: "a", Op: "eq", Value: json.Number("2")}}, map[string]any{"a": float64(2)}, true},
		{"in set", []Condition{{Field: "a", Op: "in", Value: []any{"major", "critical"}}}, map[string]any{"a": "critical"}, true},
		{"threshold", []Condition{{Field: "a", Op: "gte", Value: 2}}, map[string]any{"a": float64(1)}, false},
		{"unknown operator fails closed", []Condition{{Field: "a", Op: "eval", Value: true}}, map[string]any{"a": true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Matches(tc.conditions, tc.data); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
func TestNextRunTimezonesAndDST(t *testing.T) {
	for _, tc := range []struct {
		name, now, want, zone, clock string
		days                         []int
	}{
		{"Shanghai tomorrow", "2026-09-06T01:00:00Z", "2026-09-07T01:00:00Z", "Asia/Shanghai", "09:00", []int{0, 1, 2, 3, 4, 5, 6}},
		{"weekly selection", "2026-09-04T20:00:00Z", "2026-09-07T16:00:00Z", "America/Los_Angeles", "09:00", []int{1}},
		{"spring gap skipped", "2026-03-08T09:00:00Z", "2026-03-09T09:30:00Z", "America/Los_Angeles", "02:30", []int{0, 1, 2, 3, 4, 5, 6}},
		{"fall first fold", "2026-11-01T08:00:00Z", "2026-11-01T08:30:00Z", "America/Los_Angeles", "01:30", []int{0, 1, 2, 3, 4, 5, 6}},
		{"fall fold not duplicated", "2026-11-01T08:30:00Z", "2026-11-02T09:30:00Z", "America/Los_Angeles", "01:30", []int{0, 1, 2, 3, 4, 5, 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.now)
			got := NextRun(Trigger{Kind: "schedule", Timezone: tc.zone, Time: tc.clock, Weekdays: tc.days}, now)
			if got == nil || got.UTC().Format(time.RFC3339) != tc.want {
				t.Fatalf("got %v want %s", got, tc.want)
			}
		})
	}
	if NextRun(Trigger{Kind: "event"}, time.Now()) != nil {
		t.Fatal("event must not get a timer")
	}
}
func TestOperationPathBinding(t *testing.T) {
	for _, path := range []string{"/api/v1/devices/../secrets", "/api/v1/devices/%2fadmin", "/api/v1/devices/abc?all=true", "https://evil.invalid/api/v1/devices/x", "/api/v1/alarms/abc", "/api/v1/devices/.."} {
		if _, err := BindOperation("/api/v1/devices/:id", path); err == nil {
			t.Errorf("accepted %q", path)
		}
	}
	if _, err := BindOperation("/api/v1/devices/:id", "/api/v1/devices/abc"); err != nil {
		t.Fatal(err)
	}
}
func TestNotificationSemantics(t *testing.T) {
	d := testDefinition()
	now := time.Now()
	last := now.Add(-time.Minute)
	for _, tc := range []struct {
		kind, status, outcome string
		last                  *time.Time
		want                  bool
	}{
		{"trial", "COMPLETED", "finding", nil, false}, {"schedule", "CANCELLED", "finding", nil, false},
		{"schedule", "COMPLETED", "no_change", nil, false}, {"event", "COMPLETED", "finding", nil, true},
		{"event", "COMPLETED", "finding", &last, false}, {"event", "FAILED", "", nil, true},
	} {
		if got := ShouldNotify(tc.kind, tc.status, tc.outcome, d, tc.last, now); got != tc.want {
			t.Errorf("%+v got %v", tc, got)
		}
	}
	d.Notify = "always"
	if !ShouldNotify("schedule", "COMPLETED", "no_change", d, nil, now) {
		t.Fatal("always reports must deliver no-change")
	}
}
func TestDigestStableAndBoundToDefinition(t *testing.T) {
	a := testDefinition()
	x := DefinitionDigest(a)
	if x != DefinitionDigest(a) || !strings.HasPrefix(x, "sha256:") {
		t.Fatal(x)
	}
	a.Goal = "different"
	if x == DefinitionDigest(a) {
		t.Fatal("goal not bound")
	}
}
