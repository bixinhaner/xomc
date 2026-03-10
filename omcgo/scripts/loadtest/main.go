// Package main provides a load testing tool that simulates large-scale device
// TR069 sessions against the ACS engine.
//
// Supports two modes:
//   - Inform-only mode: sends Inform messages (legacy, for Inform throughput measurement)
//   - Full-session mode (default): simulates complete TR069 session lifecycle:
//     Inform → InformResponse → Empty POST → RPC/Empty → loop until session ends
//
// Usage:
//
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -devices 10000 -duration 5m
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -devices 1000 -full-session=false
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
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

const gpvResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">%s</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>v1.0.0-loadtest</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

const spvResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">%s</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:SetParameterValuesResponse>
      <Status>0</Status>
    </cwmp:SetParameterValuesResponse>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

const genericResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">%s</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:%sResponse/>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

type informData struct {
	ID           string
	OUI          string
	ProductClass string
	SerialNumber string
	Timestamp    string
	IP           string
}

type stats struct {
	successCount    atomic.Int64
	failCount       atomic.Int64
	errorCount      atomic.Int64
	rateLimitCount  atomic.Int64 // 503/429 rate-limited responses (subset of failCount)
	sessionCount    atomic.Int64 // complete sessions (Inform + Empty + optional RPC rounds)
	rpcRoundCount   atomic.Int64 // total RPC round-trips across all sessions
	latencies       []time.Duration
	sessionTimes    []time.Duration // end-to-end session durations
	mu              sync.Mutex
}

func (s *stats) recordLatency(d time.Duration) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.mu.Unlock()
}

func (s *stats) recordSessionTime(d time.Duration) {
	s.mu.Lock()
	s.sessionTimes = append(s.sessionTimes, d)
	s.mu.Unlock()
}

// deviceTransportPool manages per-device HTTP transports.
// Each device gets a dedicated transport with MaxConnsPerHost=1,
// ensuring all requests for the same device use the same TCP connection
// (same RemoteAddr) while allowing TCP connection reuse across sessions.
// This eliminates repeated TCP handshake overhead from per-session transport.
type deviceTransportPool struct {
	transports sync.Map
}

func newDeviceTransportPool() *deviceTransportPool {
	return &deviceTransportPool{}
}

func (p *deviceTransportPool) Get(device string) *http.Transport {
	if v, ok := p.transports.Load(device); ok {
		return v.(*http.Transport)
	}
	t := &http.Transport{
		MaxConnsPerHost:       1,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
		ResponseHeaderTimeout: 15 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	actual, _ := p.transports.LoadOrStore(device, t)
	return actual.(*http.Transport)
}

func (p *deviceTransportPool) CloseAll() {
	p.transports.Range(func(key, value interface{}) bool {
		value.(*http.Transport).CloseIdleConnections()
		return true
	})
}

func main() {
	url := flag.String("url", "http://localhost:7547/acs", "ACS endpoint URL")
	devices := flag.Int("devices", 1000, "Number of simulated devices")
	duration := flag.Duration("duration", 2*time.Minute, "Test duration")
	concurrency := flag.Int("concurrency", 100, "Max concurrent sessions")
	interval := flag.Duration("interval", 1*time.Second, "Inform interval per device")
	fullSession := flag.Bool("full-session", true, "Simulate full TR069 session (Inform+Empty+RPC)")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "Load Test Configuration:\n")
	fmt.Fprintf(os.Stderr, "  URL:          %s\n", *url)
	fmt.Fprintf(os.Stderr, "  Devices:      %d\n", *devices)
	fmt.Fprintf(os.Stderr, "  Duration:     %s\n", *duration)
	fmt.Fprintf(os.Stderr, "  Concurrency:  %d\n", *concurrency)
	fmt.Fprintf(os.Stderr, "  Interval:     %s\n", *interval)
	fmt.Fprintf(os.Stderr, "  Full Session: %v\n", *fullSession)
	fmt.Fprintln(os.Stderr)

	// Shared transport for inform-only mode (no session continuity needed).
	sharedTransport := &http.Transport{
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	// Per-device transport pool for full-session mode.
	// Each device gets a dedicated transport ensuring RemoteAddr consistency
	// while reusing TCP connections across sessions (no repeated TCP handshake).
	pool := newDeviceTransportPool()
	defer pool.CloseAll()

	// Per-device in-flight guard: ensures at most one session per device at a time,
	// preventing RemoteAddr collision and matching real CPE behavior.
	var deviceInFlight sync.Map

	s := &stats{}
	sem := make(chan struct{}, *concurrency)
	done := make(chan struct{})
	deadline := time.After(*duration)

	// Rate limiter hint for full-session mode.
	if *fullSession {
		fmt.Fprintf(os.Stderr, "NOTE: ACS default rate limit is 10/min/device. For maximum throughput,\n")
		fmt.Fprintf(os.Stderr, "      use stress config: ./bin/omcgo-acs --config configs/acs-stress.yaml\n\n")
	}

	// Hint for high concurrency
	if *concurrency >= 1000 {
		fmt.Fprintf(os.Stderr, "HINT: High concurrency (%d). Ensure fd limit >= %d: ulimit -n %d\n\n",
			*concurrency, *concurrency*3, *concurrency*3)
	}

	fmt.Fprintf(os.Stderr, "Starting load test at %s...\n\n", time.Now().Format(time.RFC3339))
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

				if *fullSession {
					// Guard: at most one session per device at a time (matches real CPE behavior).
					if _, loaded := deviceInFlight.LoadOrStore(serialNumber, true); loaded {
						return
					}
					defer deviceInFlight.Delete(serialNumber)
					runFullSession(*url, serialNumber, pool.Get(serialNumber), s)
				} else {
					client := &http.Client{
						Timeout:   30 * time.Second,
						Transport: sharedTransport,
					}
					sendInform(client, *url, serialNumber, s)
				}
			}(sn)
		}

		time.Sleep(*interval)
	}

finish:
	fmt.Fprintln(os.Stderr, "Waiting for in-flight requests to complete...")
	wg.Wait()

	totalDuration := time.Since(startTime)
	if *jsonOutput {
		printJSONReport(s, totalDuration, *concurrency, *devices, *fullSession)
	} else {
		printReport(s, totalDuration, *fullSession)
	}
}

func generateDeviceSNs(count int) []string {
	sns := make([]string, count)
	for i := 0; i < count; i++ {
		sns[i] = fmt.Sprintf("LT-%06d", i)
	}
	return sns
}

// runFullSession simulates a complete TR069 session:
// Step 1: Inform → InformResponse
// Step 2: Empty POST → RPC Request or Empty Response
// Step 3: If RPC Request → send RPC Response, goto Step 2
// Step 4: If Empty Response → session complete
//
// The transport parameter is a per-device transport from deviceTransportPool.
// MaxConnsPerHost=1 ensures all requests use the same TCP connection (same RemoteAddr).
// TCP connections are reused across sessions for the same device, eliminating handshake overhead.
func runFullSession(url, sn string, transport *http.Transport, s *stats) {
	sessionStart := time.Now()
	rpcRounds := 0

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	// Step 1: Send Inform
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

	start := time.Now()
	respBody, statusCode, err := doPost(client, url, buf.Bytes())
	latency := time.Since(start)

	if err != nil {
		s.errorCount.Add(1)
		return
	}
	s.recordLatency(latency)

	if statusCode < 200 || statusCode >= 300 {
		if statusCode == http.StatusServiceUnavailable || statusCode == http.StatusTooManyRequests {
			s.rateLimitCount.Add(1)
		}
		s.failCount.Add(1)
		return
	}
	s.successCount.Add(1)

	// Verify we got InformResponse
	if !strings.Contains(string(respBody), "InformResponse") {
		s.failCount.Add(1)
		return
	}

	// Step 2-4: Loop — send Empty POST, handle RPC requests
	for i := 0; i < 20; i++ { // max 20 RPC rounds to prevent infinite loops
		start = time.Now()
		respBody, statusCode, err = doPost(client, url, nil) // empty POST
		latency = time.Since(start)

		if err != nil {
			s.errorCount.Add(1)
			return
		}
		s.recordLatency(latency)

		if statusCode == http.StatusNoContent || len(strings.TrimSpace(string(respBody))) == 0 {
			// Session complete — ACS sent empty response
			break
		}

		if statusCode < 200 || statusCode >= 300 {
			if statusCode == http.StatusServiceUnavailable || statusCode == http.StatusTooManyRequests {
				s.rateLimitCount.Add(1)
			}
			s.failCount.Add(1)
			return
		}
		s.successCount.Add(1)

		// Check if ACS sent an RPC request
		bodyStr := string(respBody)
		rpcMethod := detectRPCMethod(bodyStr)
		if rpcMethod == "" {
			// Empty or unrecognized body — session complete
			break
		}

		rpcRounds++

		// Build an appropriate RPC response
		cwmpID := extractCWMPID(bodyStr)
		rpcResp := buildRPCResponse(rpcMethod, cwmpID)

		start = time.Now()
		respBody, statusCode, err = doPost(client, url, []byte(rpcResp))
		latency = time.Since(start)

		if err != nil {
			s.errorCount.Add(1)
			return
		}
		s.recordLatency(latency)

		if statusCode >= 200 && statusCode < 300 {
			s.successCount.Add(1)
		} else {
			s.failCount.Add(1)
			return
		}

		// If response is empty or is another RPC, continue the loop
		// The next iteration will send an empty POST
	}

	// Session completed successfully
	s.sessionCount.Add(1)
	s.rpcRoundCount.Add(int64(rpcRounds))
	s.recordSessionTime(time.Since(sessionStart))
}

// doPost sends a POST request. If body is nil, sends an empty body.
func doPost(client *http.Client, url string, body []byte) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBody, resp.StatusCode, nil
}

// detectRPCMethod detects which RPC method the ACS is requesting from the CPE.
func detectRPCMethod(body string) string {
	methods := []string{
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterNames",
		"GetParameterAttributes",
		"SetParameterAttributes",
		"AddObject",
		"DeleteObject",
		"Download",
		"Upload",
		"Reboot",
		"FactoryReset",
	}
	for _, m := range methods {
		// Look for <cwmp:Method> but not <cwmp:MethodResponse>
		if strings.Contains(body, "cwmp:"+m) && !strings.Contains(body, m+"Response") {
			return m
		}
	}
	return ""
}

// extractCWMPID extracts the CWMP ID from a SOAP message.
func extractCWMPID(body string) string {
	// Simple extraction: find content between <cwmp:ID ...> and </cwmp:ID>
	idx := strings.Index(body, "<cwmp:ID")
	if idx < 0 {
		return "loadtest-1"
	}
	start := strings.Index(body[idx:], ">")
	if start < 0 {
		return "loadtest-1"
	}
	start += idx + 1
	end := strings.Index(body[start:], "</cwmp:ID>")
	if end < 0 {
		return "loadtest-1"
	}
	return body[start : start+end]
}

// buildRPCResponse builds a CPE RPC response for the given method.
func buildRPCResponse(method, cwmpID string) string {
	switch method {
	case "GetParameterValues":
		return fmt.Sprintf(gpvResponseXML, cwmpID)
	case "SetParameterValues":
		return fmt.Sprintf(spvResponseXML, cwmpID)
	default:
		return fmt.Sprintf(genericResponseXML, cwmpID, method)
	}
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
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			s.rateLimitCount.Add(1)
		}
		s.failCount.Add(1)
	}
}

func printReport(s *stats, totalDuration time.Duration, fullSession bool) {
	success := s.successCount.Load()
	fail := s.failCount.Load()
	errors := s.errorCount.Load()
	rateLimited := s.rateLimitCount.Load()
	total := success + fail + errors

	fmt.Println("\n========================================")
	if fullSession {
		fmt.Println("    FULL SESSION LOAD TEST REPORT")
	} else {
		fmt.Println("     INFORM-ONLY LOAD TEST REPORT")
	}
	fmt.Println("========================================")
	fmt.Printf("Duration:     %s\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Total Reqs:   %d\n", total)
	fmt.Printf("Success:      %d (%.1f%%)\n", success, pct(success, total))
	fmt.Printf("Failed:       %d (%.1f%%)\n", fail, pct(fail, total))
	if rateLimited > 0 {
		fmt.Printf("  Rate-Ltd:   %d (%.1f%% of total — ACS 503/429)\n", rateLimited, pct(rateLimited, total))
	}
	fmt.Printf("Errors:       %d (%.1f%%)\n", errors, pct(errors, total))
	fmt.Printf("Throughput:   %.1f req/s\n", float64(total)/totalDuration.Seconds())

	if fullSession {
		sessions := s.sessionCount.Load()
		rpcRounds := s.rpcRoundCount.Load()
		fmt.Println()
		fmt.Printf("Sessions:     %d completed\n", sessions)
		fmt.Printf("Session Rate: %.1f sessions/s\n", float64(sessions)/totalDuration.Seconds())
		if sessions > 0 {
			fmt.Printf("Avg RPC/Sess: %.1f rounds\n", float64(rpcRounds)/float64(sessions))
		}

		s.mu.Lock()
		sessionTimes := make([]time.Duration, len(s.sessionTimes))
		copy(sessionTimes, s.sessionTimes)
		s.mu.Unlock()

		if len(sessionTimes) > 0 {
			sort.Slice(sessionTimes, func(i, j int) bool { return sessionTimes[i] < sessionTimes[j] })
			fmt.Println()
			fmt.Println("Session Duration Percentiles:")
			fmt.Printf("  p50:  %s\n", sessionTimes[percentileIdx(sessionTimes, 50)])
			fmt.Printf("  p90:  %s\n", sessionTimes[percentileIdx(sessionTimes, 90)])
			fmt.Printf("  p95:  %s\n", sessionTimes[percentileIdx(sessionTimes, 95)])
			fmt.Printf("  p99:  %s\n", sessionTimes[percentileIdx(sessionTimes, 99)])
			fmt.Printf("  max:  %s\n", sessionTimes[len(sessionTimes)-1])
		}
	}

	fmt.Println()

	s.mu.Lock()
	latencies := make([]time.Duration, len(s.latencies))
	copy(latencies, s.latencies)
	s.mu.Unlock()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		fmt.Println("Request Latency Percentiles:")
		fmt.Printf("  p50:  %s\n", latencies[percentileIdx(latencies, 50)])
		fmt.Printf("  p90:  %s\n", latencies[percentileIdx(latencies, 90)])
		fmt.Printf("  p95:  %s\n", latencies[percentileIdx(latencies, 95)])
		fmt.Printf("  p99:  %s\n", latencies[percentileIdx(latencies, 99)])
		fmt.Printf("  max:  %s\n", latencies[len(latencies)-1])
	}

	fmt.Println("========================================")

	if pct(success, total) < 99.0 {
		fmt.Println("\nWARNING: Success rate below 99%!")
		if rateLimited > 0 && pct(rateLimited, fail) > 50.0 {
			fmt.Println("ROOT CAUSE: ACS rate limiter (10/min/device) rejected most failed requests.")
			fmt.Println("FIX: Use stress config with relaxed rate limit:")
			fmt.Println("     ./bin/omcgo-acs --config configs/acs-stress.yaml")
		} else if errors > 0 && pct(errors, total) > 10.0 {
			fmt.Println("ROOT CAUSE: TCP connection errors (timeout/refused).")
			fmt.Println("FIX: Reduce -concurrency, or increase fd limit: ulimit -n 65535")
		}
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

type jsonReport struct {
	Mode             string  `json:"mode"`
	Concurrency      int     `json:"concurrency"`
	Devices          int     `json:"devices"`
	DurationMs       int64   `json:"duration_ms"`
	Total            int64   `json:"total"`
	Success          int64   `json:"success"`
	Failed           int64   `json:"failed"`
	RateLimited      int64   `json:"rate_limited"`
	Errors           int64   `json:"errors"`
	SuccessRate      float64 `json:"success_rate"`
	Throughput       float64 `json:"throughput"`
	Sessions         int64   `json:"sessions,omitempty"`
	SessionRate      float64 `json:"session_rate,omitempty"`
	AvgRPCPerSession float64 `json:"avg_rpc_per_session,omitempty"`
	P50Ms            float64 `json:"p50_ms"`
	P90Ms            float64 `json:"p90_ms"`
	P95Ms            float64 `json:"p95_ms"`
	P99Ms            float64 `json:"p99_ms"`
	MaxMs            float64 `json:"max_ms"`
	SessionP50Ms     float64 `json:"session_p50_ms,omitempty"`
	SessionP99Ms     float64 `json:"session_p99_ms,omitempty"`
}

func printJSONReport(s *stats, totalDuration time.Duration, concurrency, devices int, fullSession bool) {
	success := s.successCount.Load()
	fail := s.failCount.Load()
	errors := s.errorCount.Load()
	total := success + fail + errors

	mode := "inform-only"
	if fullSession {
		mode = "full-session"
	}

	rateLimited := s.rateLimitCount.Load()

	r := jsonReport{
		Mode:        mode,
		Concurrency: concurrency,
		Devices:     devices,
		DurationMs:  totalDuration.Milliseconds(),
		Total:       total,
		Success:     success,
		Failed:      fail,
		RateLimited: rateLimited,
		Errors:      errors,
		SuccessRate: pct(success, total),
		Throughput:  float64(total) / totalDuration.Seconds(),
	}

	if fullSession {
		sessions := s.sessionCount.Load()
		rpcRounds := s.rpcRoundCount.Load()
		r.Sessions = sessions
		r.SessionRate = float64(sessions) / totalDuration.Seconds()
		if sessions > 0 {
			r.AvgRPCPerSession = float64(rpcRounds) / float64(sessions)
		}

		s.mu.Lock()
		sessionTimes := make([]time.Duration, len(s.sessionTimes))
		copy(sessionTimes, s.sessionTimes)
		s.mu.Unlock()

		if len(sessionTimes) > 0 {
			sort.Slice(sessionTimes, func(i, j int) bool { return sessionTimes[i] < sessionTimes[j] })
			r.SessionP50Ms = float64(sessionTimes[percentileIdx(sessionTimes, 50)].Microseconds()) / 1000.0
			r.SessionP99Ms = float64(sessionTimes[percentileIdx(sessionTimes, 99)].Microseconds()) / 1000.0
		}
	}

	s.mu.Lock()
	latencies := make([]time.Duration, len(s.latencies))
	copy(latencies, s.latencies)
	s.mu.Unlock()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		r.P50Ms = float64(latencies[percentileIdx(latencies, 50)].Microseconds()) / 1000.0
		r.P90Ms = float64(latencies[percentileIdx(latencies, 90)].Microseconds()) / 1000.0
		r.P95Ms = float64(latencies[percentileIdx(latencies, 95)].Microseconds()) / 1000.0
		r.P99Ms = float64(latencies[percentileIdx(latencies, 99)].Microseconds()) / 1000.0
		r.MaxMs = float64(latencies[len(latencies)-1].Microseconds()) / 1000.0
	}

	data, _ := json.Marshal(r)
	fmt.Println(string(data))
}
