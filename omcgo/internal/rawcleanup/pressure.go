package rawcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type QueueDepth struct {
	Pending, AckPending uint64
}

type QueueSample map[string]QueueDepth

type QueueProbe func(context.Context) (QueueSample, error)

type PrometheusPressureProbe struct {
	baseURL      string
	client       *http.Client
	queue        QueueProbe
	mu           sync.Mutex
	last         QueueSample
	growthStreak map[string]int
}

func NewPrometheusPressureProbe(baseURL string, timeout time.Duration, queue QueueProbe) *PrometheusPressureProbe {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &PrometheusPressureProbe{
		baseURL:      strings.TrimRight(baseURL, "/"),
		client:       &http.Client{Timeout: timeout},
		queue:        queue,
		growthStreak: make(map[string]int),
	}
}

const (
	cpuPressureQuery = `100 - avg(rate(node_cpu_seconds_total{mode="idle"}[2m])) * 100`
	diskAwaitQuery   = `max(1000 * (rate(node_disk_read_time_seconds_total{device!~"loop.*|ram.*"}[5m]) + rate(node_disk_write_time_seconds_total{device!~"loop.*|ram.*"}[5m])) / clamp_min(rate(node_disk_reads_completed_total{device!~"loop.*|ram.*"}[5m]) + rate(node_disk_writes_completed_total{device!~"loop.*|ram.*"}[5m]), 0.001))`
	diskQueueQuery   = `max(rate(node_disk_io_time_weighted_seconds_total{device!~"loop.*|ram.*"}[5m]))`
)

func (p *PrometheusPressureProbe) Sample(ctx context.Context) Pressure {
	result := Pressure{}
	type queryResult struct {
		name  string
		value float64
		err   error
	}
	results := make(chan queryResult, 3)
	for name, query := range map[string]string{
		"cpu": cpuPressureQuery, "await": diskAwaitQuery, "queue": diskQueueQuery,
	} {
		go func(name, query string) {
			value, err := p.query(ctx, query)
			results <- queryResult{name: name, value: value, err: err}
		}(name, query)
	}
	values := map[string]float64{}
	monitoringOK := true
	for range 3 {
		query := <-results
		if query.err != nil {
			monitoringOK = false
		}
		values[query.name] = query.value
	}
	if monitoringOK {
		result.MonitoringAvailable = true
		result.CPUPercent = values["cpu"]
		result.DiskAwaitMillis = values["await"]
		result.DiskQueue = values["queue"]
	}
	if p.queue != nil {
		current, err := p.queue(ctx)
		if err == nil {
			p.mu.Lock()
			for durable, depth := range current {
				previous, exists := p.last[durable]
				growing := exists &&
					(depth.Pending > previous.Pending || depth.AckPending > previous.AckPending)
				if growing {
					p.growthStreak[durable]++
				} else {
					p.growthStreak[durable] = 0
				}
				if p.growthStreak[durable] >= 2 {
					result.NATSBacklog = true
				}
			}
			p.last = current
			p.mu.Unlock()
		} else {
			result.NATSUnavailable = true
		}
	}
	return result
}

func (p *PrometheusPressureProbe) query(ctx context.Context, expression string) (float64, error) {
	if p.baseURL == "" {
		return 0, fmt.Errorf("Prometheus URL is empty")
	}
	endpoint := p.baseURL + "/api/v1/query?query=" + url.QueryEscape(expression)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build Prometheus request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("query Prometheus: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Prometheus HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, fmt.Errorf("read Prometheus response: %w", err)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, fmt.Errorf("decode Prometheus response: %w", err)
	}
	if envelope.Status != "success" || envelope.Data.ResultType != "vector" || len(envelope.Data.Result) != 1 ||
		len(envelope.Data.Result[0].Value) != 2 {
		return 0, fmt.Errorf("Prometheus pressure query returned no scalar sample")
	}
	var sampleTimestamp float64
	if err := json.Unmarshal(envelope.Data.Result[0].Value[0], &sampleTimestamp); err != nil {
		return 0, fmt.Errorf("decode Prometheus pressure timestamp: %w", err)
	}
	sampleTime := time.Unix(0, int64(sampleTimestamp*float64(time.Second)))
	if age := time.Since(sampleTime); age > time.Minute || age < -5*time.Second {
		return 0, fmt.Errorf("stale Prometheus pressure sample: age=%s", age)
	}
	var raw string
	if err := json.Unmarshal(envelope.Data.Result[0].Value[1], &raw); err != nil {
		return 0, err
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid Prometheus pressure value %q", raw)
	}
	return value, nil
}
