package rawcleanup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPressureProbeTreatsUnavailableMonitoringAsUnavailable(t *testing.T) {
	probe := NewPrometheusPressureProbe("http://127.0.0.1:1", 10*time.Millisecond, nil)
	if got := probe.Sample(context.Background()); got.MonitoringAvailable {
		t.Fatalf("unreachable monitoring reported available: %+v", got)
	}
}

func TestPressureProbePausesOnlyWhenQueueBacklogGrows(t *testing.T) {
	values := []QueueSample{
		{"pm": {Pending: 10}},
		{"pm": {Pending: 10}},
		{"pm": {Pending: 11}},
		{"pm": {Pending: 12}},
	}
	index := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"resultType": "vector", "result": []any{map[string]any{"value": []any{float64(time.Now().Unix()), "1"}}}},
		})
	}))
	defer server.Close()
	probe := NewPrometheusPressureProbe(server.URL, time.Second, func(context.Context) (QueueSample, error) {
		got := values[index]
		index++
		return got, nil
	})
	if probe.Sample(context.Background()).NATSBacklog {
		t.Fatal("first sample must establish baseline")
	}
	if probe.Sample(context.Background()).NATSBacklog {
		t.Fatal("stable backlog must not pause")
	}
	if probe.Sample(context.Background()).NATSBacklog {
		t.Fatal("one increase is not sustained growth")
	}
	if !probe.Sample(context.Background()).NATSBacklog {
		t.Fatal("two consecutive increases must pause")
	}
}

func TestPressureProbeDoesNotLetDrainingQueueMaskAnotherGrowingQueue(t *testing.T) {
	values := []QueueSample{
		{"growing": {Pending: 10}, "draining": {Pending: 100}},
		{"growing": {Pending: 11}, "draining": {Pending: 50}},
		{"growing": {Pending: 12}, "draining": {Pending: 1}},
	}
	index := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"resultType": "vector", "result": []any{map[string]any{"value": []any{float64(time.Now().Unix()), "1"}}}},
		})
	}))
	defer server.Close()
	probe := NewPrometheusPressureProbe(server.URL, time.Second, func(context.Context) (QueueSample, error) {
		got := values[index]
		index++
		return got, nil
	})
	_ = probe.Sample(context.Background())
	_ = probe.Sample(context.Background())
	if !probe.Sample(context.Background()).NATSBacklog {
		t.Fatal("draining queue masked sustained growth in another durable")
	}
}
