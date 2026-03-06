// Package main provides a load testing tool that simulates large-scale device
// Inform sessions against the ACS engine.
//
// Usage:
//
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -devices 10000 -duration 5m
package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

var informTemplate = template.Must(template.New("inform").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">{{.ID}}</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>LoadTestVendor</Manufacturer>
        <OUI>{{.OUI}}</OUI>
        <ProductClass>LT-{{.ProductClass}}</ProductClass>
        <SerialNumber>{{.SerialNumber}}</SerialNumber>
      </DeviceId>
      <Event>
        <EventStruct>
          <EventCode>2 PERIODIC</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>{{.Timestamp}}</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList SOAP-ENC:arrayType="cwmp:ParameterValueStruct[3]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>v1.0.0-loadtest</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.ConnectionRequestURL</Name>
          <Value>http://{{.IP}}:7547/connreq</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.PeriodicInformInterval</Name>
          <Value>300</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`))

type informData struct {
	ID           string
	OUI          string
	ProductClass string
	SerialNumber string
	Timestamp    string
	IP           string
}

type stats struct {
	successCount atomic.Int64
	failCount    atomic.Int64
	errorCount   atomic.Int64
	latencies    []time.Duration
	mu           sync.Mutex
}

func (s *stats) recordLatency(d time.Duration) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.mu.Unlock()
}

func main() {
	url := flag.String("url", "http://localhost:7547/acs", "ACS endpoint URL")
	devices := flag.Int("devices", 1000, "Number of simulated devices")
	duration := flag.Duration("duration", 2*time.Minute, "Test duration")
	concurrency := flag.Int("concurrency", 100, "Max concurrent requests")
	interval := flag.Duration("interval", 1*time.Second, "Inform interval per device")
	flag.Parse()

	fmt.Printf("Load Test Configuration:\n")
	fmt.Printf("  URL:         %s\n", *url)
	fmt.Printf("  Devices:     %d\n", *devices)
	fmt.Printf("  Duration:    %s\n", *duration)
	fmt.Printf("  Concurrency: %d\n", *concurrency)
	fmt.Printf("  Interval:    %s\n", *interval)
	fmt.Println()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	s := &stats{}
	sem := make(chan struct{}, *concurrency)
	done := make(chan struct{})
	deadline := time.After(*duration)

	fmt.Printf("Starting load test at %s...\n\n", time.Now().Format(time.RFC3339))
	startTime := time.Now()

	go func() {
		<-deadline
		close(done)
	}()

	var wg sync.WaitGroup
	deviceSNs := generateDeviceSNs(*devices)

	for {
		select {
		case <-done:
			goto finish
		default:
		}

		for _, sn := range deviceSNs {
			select {
			case <-done:
				goto finish
			case sem <- struct{}{}:
			}

			wg.Add(1)
			go func(serialNumber string) {
				defer wg.Done()
				defer func() { <-sem }()

				sendInform(client, *url, serialNumber, s)
			}(sn)
		}

		time.Sleep(*interval)
	}

finish:
	fmt.Println("Waiting for in-flight requests to complete...")
	wg.Wait()

	totalDuration := time.Since(startTime)
	printReport(s, totalDuration)
}

func generateDeviceSNs(count int) []string {
	sns := make([]string, count)
	for i := 0; i < count; i++ {
		sns[i] = fmt.Sprintf("LT-%06d", i)
	}
	return sns
}

func sendInform(client *http.Client, url, sn string, s *stats) {
	data := informData{
		ID:           fmt.Sprintf("lt-%d", rand.Int63()),
		OUI:          "AABBCC",
		ProductClass: "SmallCell",
		SerialNumber: sn,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		IP:           fmt.Sprintf("10.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255)),
	}

	var buf bytes.Buffer
	if err := informTemplate.Execute(&buf, data); err != nil {
		s.errorCount.Add(1)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		s.errorCount.Add(1)
		return
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		s.errorCount.Add(1)
		return
	}
	defer resp.Body.Close()

	s.recordLatency(latency)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.successCount.Add(1)
	} else {
		s.failCount.Add(1)
	}
}

func printReport(s *stats, totalDuration time.Duration) {
	success := s.successCount.Load()
	fail := s.failCount.Load()
	errors := s.errorCount.Load()
	total := success + fail + errors

	fmt.Println("\n========================================")
	fmt.Println("           LOAD TEST REPORT")
	fmt.Println("========================================")
	fmt.Printf("Duration:     %s\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Total:        %d\n", total)
	fmt.Printf("Success:      %d (%.1f%%)\n", success, pct(success, total))
	fmt.Printf("Failed:       %d (%.1f%%)\n", fail, pct(fail, total))
	fmt.Printf("Errors:       %d (%.1f%%)\n", errors, pct(errors, total))
	fmt.Printf("Throughput:   %.1f req/s\n", float64(total)/totalDuration.Seconds())
	fmt.Println()

	s.mu.Lock()
	latencies := make([]time.Duration, len(s.latencies))
	copy(latencies, s.latencies)
	s.mu.Unlock()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		fmt.Println("Latency Percentiles:")
		fmt.Printf("  p50:  %s\n", latencies[percentileIdx(latencies, 50)])
		fmt.Printf("  p90:  %s\n", latencies[percentileIdx(latencies, 90)])
		fmt.Printf("  p95:  %s\n", latencies[percentileIdx(latencies, 95)])
		fmt.Printf("  p99:  %s\n", latencies[percentileIdx(latencies, 99)])
		fmt.Printf("  max:  %s\n", latencies[len(latencies)-1])
	}

	fmt.Println("========================================")

	if pct(success, total) < 99.0 {
		fmt.Println("\nWARNING: Success rate below 99%!")
		os.Exit(1)
	}
}

func pct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func percentileIdx(sorted []time.Duration, p int) int {
	idx := len(sorted) * p / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return idx
}
