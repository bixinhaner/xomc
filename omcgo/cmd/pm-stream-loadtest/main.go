package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/pm/streamtest"
)

type report struct {
	Devices        int       `json:"devices"`
	Metrics        int       `json:"metrics_per_device"`
	Concurrency    int       `json:"concurrency"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	DurationMS     int64     `json:"duration_ms"`
	Published      int64     `json:"published"`
	Failed         int64     `json:"failed"`
	StreamMessages uint64    `json:"stream_messages"`
}

func main() {
	var natsURL, output string
	var devices, metrics, concurrency int
	flag.StringVar(&natsURL, "nats", nats.DefaultURL, "NATS server URL")
	flag.StringVar(&output, "output", "pm-stream-loadtest-result.json", "result JSON path")
	flag.IntVar(&devices, "devices", 10000, "number of normalized device events")
	flag.IntVar(&metrics, "metrics", 16, "KPI values per event")
	flag.IntVar(&concurrency, "concurrency", 32, "publish concurrency")
	flag.Parse()
	if devices <= 0 || metrics <= 0 || concurrency <= 0 {
		fail(fmt.Errorf("devices, metrics and concurrency must be positive"))
	}
	conn, err := nats.Connect(natsURL)
	if err != nil {
		fail(err)
	}
	defer conn.Close()
	js, err := conn.JetStream(nats.PublishAsyncMaxPending(concurrency * 4))
	if err != nil {
		fail(err)
	}
	if err := ensureAggregationStream(js); err != nil {
		fail(err)
	}
	generator := streamtest.Generator{
		Devices: devices, Metrics: metrics,
		SlotStart: time.Now().UTC().Truncate(15 * time.Minute),
	}
	started := time.Now().UTC()
	jobs := make(chan int)
	var published, failedCount atomic.Int64
	var workers sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				payload, eventErr := generator.Event(index)
				if eventErr != nil {
					failedCount.Add(1)
					continue
				}
				envelope, eventErr := event.NewEvent(event.SubjectPMAggregationNormalized, payload)
				if eventErr != nil {
					failedCount.Add(1)
					continue
				}
				data, eventErr := json.Marshal(envelope)
				if eventErr == nil {
					_, eventErr = js.Publish(event.SubjectPMAggregationNormalized, data)
				}
				if eventErr != nil {
					failedCount.Add(1)
				} else {
					published.Add(1)
				}
			}
		}()
	}
	for index := 0; index < devices; index++ {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	finished := time.Now().UTC()
	result := report{
		Devices: devices, Metrics: metrics, Concurrency: concurrency,
		StartedAt: started, FinishedAt: finished,
		DurationMS: finished.Sub(started).Milliseconds(),
		Published:  published.Load(), Failed: failedCount.Load(),
	}
	if info, infoErr := js.StreamInfo("PM_AGG_15M"); infoErr == nil {
		result.StreamMessages = info.State.Msgs
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(output, append(data, '\n'), 0o644); err != nil {
		fail(err)
	}
	fmt.Println(string(data))
	if result.Failed > 0 {
		os.Exit(1)
	}
}

func ensureAggregationStream(js nats.JetStreamContext) error {
	if _, err := js.StreamInfo("PM_AGG_15M"); err == nil {
		return nil
	} else if err != nats.ErrStreamNotFound {
		return fmt.Errorf("inspect PM_AGG_15M stream: %w", err)
	}
	_, err := js.AddStream(&nats.StreamConfig{
		Name:        "PM_AGG_15M",
		Subjects:    []string{"pmaggregation.15m.>"},
		Retention:   nats.LimitsPolicy,
		MaxAge:      2 * time.Hour,
		MaxBytes:    512 << 20,
		AllowDirect: true,
		Compression: nats.S2Compression,
	})
	if err != nil {
		return fmt.Errorf("create PM_AGG_15M stream: %w", err)
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
