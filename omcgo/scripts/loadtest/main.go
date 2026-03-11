// Package main provides a load testing tool that simulates large-scale device
// TR069 sessions against the ACS engine.
//
// Supports four modes:
//   - acs:  Full TR069 session (Inform → InformResponse → Empty → RPC/Empty → loop)
//   - kpi:  PM file upload session (Inform → ATC FileType=4 → session end)
//   - mr:   MR file upload session (Inform → ATC FileType=5 → session end)
//   - all:  Round-robin across acs/kpi/mr
//
// Also supports gradual ramp (-ramp) for 5-round escalating load tests.
//
// Usage:
//
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -devices 1000
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -mode kpi -devices 500
//	go run scripts/loadtest/main.go -url http://localhost:7547/acs -mode all -ramp -json
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

// ---------------------------------------------------------------------------
// SOAP / XML Templates
// ---------------------------------------------------------------------------

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

var atcTemplate = template.Must(template.New("atc").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"
  xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">{{.ID}}</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:AutonomousTransferComplete>
      <AnnounceURL></AnnounceURL>
      <TransferURL>{{.TransferURL}}</TransferURL>
      <IsDownload>false</IsDownload>
      <FileType>{{.FileType}}</FileType>
      <FileSize>{{.FileSize}}</FileSize>
      <TargetFileName>{{.TargetFileName}}</TargetFileName>
      <StartTime>{{.StartTime}}</StartTime>
      <CompleteTime>{{.CompleteTime}}</CompleteTime>
    </cwmp:AutonomousTransferComplete>
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

// ---------------------------------------------------------------------------
// Sample PM / MR XML for the built-in file server
// ---------------------------------------------------------------------------

// 3GPP 32.435 PM XML: 4 counters x 2 cells
const samplePMXML = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile xmlns="http://www.3gpp.org/ftp/specs/archive/32_series/32.435#measCollec">
  <fileHeader dnPrefix="DC=cmcc" vendorName="LoadTestVendor" fileFormatVersion="32.435 V10.0"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=%s" swVersion="V2.1"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="%s"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measType p="2">rrc_conn_setup_succ</measType>
      <measType p="3">erab_setup_att</measType>
      <measType p="4">erab_setup_succ</measType>
      <measValue measObjLdn="CellId=%s-Cell1">
        <r p="1">%d</r><r p="2">%d</r><r p="3">%d</r><r p="4">%d</r>
      </measValue>
      <measValue measObjLdn="CellId=%s-Cell2">
        <r p="1">%d</r><r p="2">%d</r><r p="3">%d</r><r p="4">%d</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

// MRO format: RSRP/RSRQ/SINR x 2 cells, 3 UE records
const sampleMROXML = `<?xml version="1.0" encoding="UTF-8"?>
<bulkPmMrDataFile>
  <fileHeader startTime="%s" reportingPeriod="PT900S"/>
  <eNB id="%s">
    <measurement mrType="MRO">
      <smr>MR.LteScRSRP MR.LteScRSRQ MR.LteScSinrUL</smr>
      <object id="%s-Cell-1">
        <v>-%.1f -%.1f %.1f</v>
        <v>-%.1f -%.1f %.1f</v>
      </object>
      <object id="%s-Cell-2">
        <v>-%.1f -%.1f %.1f</v>
      </object>
    </measurement>
  </eNB>
</bulkPmMrDataFile>`

// ---------------------------------------------------------------------------
// Data Types
// ---------------------------------------------------------------------------

type informData struct {
	ID           string
	OUI          string
	ProductClass string
	SerialNumber string
	Timestamp    string
	IP           string
}

type atcData struct {
	ID             string
	TransferURL    string
	FileType       string // "4" for PM, "5" for MR
	FileSize       int64
	TargetFileName string
	StartTime      string
	CompleteTime   string
}

type stats struct {
	successCount    atomic.Int64
	failCount       atomic.Int64
	errorCount      atomic.Int64
	rateLimitCount  atomic.Int64 // 503/429 rate-limited responses (subset of failCount)
	sessionCount    atomic.Int64 // complete sessions
	rpcRoundCount   atomic.Int64 // total RPC round-trips across all sessions
	fileUploadCount atomic.Int64 // successful ATC messages (PM/MR uploads)
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

// rampRound defines one round in the 5-round escalating load test.
type rampRound struct {
	name        string
	duration    time.Duration
	concurrency int
	devices     int
}

var rampSchedule = []rampRound{
	{"R1-warmup", 3 * time.Minute, 20, 200},
	{"R2-light", 3 * time.Minute, 50, 500},
	{"R3-medium", 3 * time.Minute, 100, 1000},
	{"R4-heavy", 3 * time.Minute, 200, 2000},
	{"R5-stress", 3 * time.Minute, 500, 5000},
}

// roundConfig holds all parameters for a single load test round.
type roundConfig struct {
	url            string
	mode           string
	devices        int
	concurrency    int
	duration       time.Duration
	interval       time.Duration
	fullSession    bool
	fileServerBase string // e.g. "http://localhost:9876"
}

// ---------------------------------------------------------------------------
// Device Transport Pool
// ---------------------------------------------------------------------------

// deviceTransportPool manages per-device HTTP transports.
// Each device gets a dedicated transport with MaxConnsPerHost=1,
// ensuring all requests for the same device use the same TCP connection
// (same RemoteAddr) while allowing TCP connection reuse across sessions.
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

// ---------------------------------------------------------------------------
// Built-in File Server (serves sample PM/MR XML for TransferBridge)
// ---------------------------------------------------------------------------

func startFileServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/pm/", servePM)
	mux.HandleFunc("/mr/", serveMR)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "file server error: %v\n", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // let listener start
	return srv
}

func servePM(w http.ResponseWriter, r *http.Request) {
	sn := strings.TrimSuffix(path.Base(r.URL.Path), ".xml")
	now := time.Now().Format("2006-01-02T15:04:05+08:00")
	// Generate slightly randomized counter values
	h := fnvHash(sn)
	xml := fmt.Sprintf(samplePMXML,
		sn, now,
		sn, 900+h%200, 880+h%200, 800+h%300, 780+h%300,
		sn, 850+h%250, 830+h%250, 750+h%350, 730+h%350,
	)
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml))
}

func serveMR(w http.ResponseWriter, r *http.Request) {
	sn := strings.TrimSuffix(path.Base(r.URL.Path), ".xml")
	now := time.Now().Format("2006-01-02T15:04:05+08:00")
	h := fnvHash(sn)
	rsrp := func() float64 { return 70.0 + float64(h%30) }
	rsrq := func() float64 { return 7.0 + float64(h%8) }
	sinr := func() float64 { return 5.0 + float64(h%20) }
	xml := fmt.Sprintf(sampleMROXML,
		now, sn,
		sn, rsrp(), rsrq(), sinr(), rsrp()+5, rsrq()+2, sinr()-3,
		sn, rsrp()+3, rsrq()+1, sinr()-1,
	)
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml))
}

// fnvHash returns a simple deterministic hash for a string.
func fnvHash(s string) int {
	h := 2166136261
	for i := 0; i < len(s); i++ {
		h ^= int(s[i])
		h *= 16777619
	}
	if h < 0 {
		h = -h
	}
	return h
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	url := flag.String("url", "http://localhost:7547/acs", "ACS endpoint URL")
	devices := flag.Int("devices", 1000, "Number of simulated devices")
	duration := flag.Duration("duration", 2*time.Minute, "Test duration")
	concurrency := flag.Int("concurrency", 100, "Max concurrent sessions")
	interval := flag.Duration("interval", 1*time.Second, "Inform interval per device")
	fullSession := flag.Bool("full-session", true, "Simulate full TR069 session (Inform+Empty+RPC)")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	mode := flag.String("mode", "acs", "Test mode: acs|kpi|mr|all")
	filePort := flag.Int("file-port", 9876, "Built-in file server port (for kpi/mr/all modes)")
	fileHost := flag.String("file-host", "localhost", "Hostname for TransferURL (must be reachable by TransferBridge)")
	ramp := flag.Bool("ramp", false, "Run 5-round gradual ramp (overrides -devices/-concurrency/-duration)")
	flag.Parse()

	// Validate mode
	switch *mode {
	case "acs", "kpi", "mr", "all":
	default:
		fmt.Fprintf(os.Stderr, "ERROR: invalid mode %q (must be acs|kpi|mr|all)\n", *mode)
		os.Exit(1)
	}

	// kpi/mr/all require full-session (ATC needs Inform first for connSessions binding)
	if *mode != "acs" {
		*fullSession = true
	}

	// File server base URL (only used for kpi/mr/all)
	fileServerBase := fmt.Sprintf("http://%s:%d", *fileHost, *filePort)

	// Start file server if needed
	var fileSrv *http.Server
	if *mode != "acs" {
		fileSrv = startFileServer(*filePort)
		defer fileSrv.Shutdown(context.Background())
	}

	// Ramp mode: run 5-round escalating test
	if *ramp {
		runRamp(*url, *mode, *interval, fileServerBase, *jsonOutput)
		return
	}

	// --- Single round mode ---
	cfg := roundConfig{
		url:            *url,
		mode:           *mode,
		devices:        *devices,
		concurrency:    *concurrency,
		duration:       *duration,
		interval:       *interval,
		fullSession:    *fullSession,
		fileServerBase: fileServerBase,
	}

	fmt.Fprintf(os.Stderr, "Load Test Configuration:\n")
	fmt.Fprintf(os.Stderr, "  URL:          %s\n", cfg.url)
	fmt.Fprintf(os.Stderr, "  Mode:         %s\n", cfg.mode)
	fmt.Fprintf(os.Stderr, "  Devices:      %d\n", cfg.devices)
	fmt.Fprintf(os.Stderr, "  Duration:     %s\n", cfg.duration)
	fmt.Fprintf(os.Stderr, "  Concurrency:  %d\n", cfg.concurrency)
	fmt.Fprintf(os.Stderr, "  Interval:     %s\n", cfg.interval)
	fmt.Fprintf(os.Stderr, "  Full Session: %v\n", cfg.fullSession)
	if cfg.mode != "acs" {
		fmt.Fprintf(os.Stderr, "  File Server:  %s\n", cfg.fileServerBase)
	}
	fmt.Fprintln(os.Stderr)

	// Rate limiter hint
	if cfg.fullSession {
		fmt.Fprintf(os.Stderr, "NOTE: ACS default rate limit is 10/min/device. For maximum throughput,\n")
		fmt.Fprintf(os.Stderr, "      use stress config: ./bin/omcgo-acs --config configs/acs-stress.yaml\n\n")
	}
	if cfg.concurrency >= 1000 {
		fmt.Fprintf(os.Stderr, "HINT: High concurrency (%d). Ensure fd limit >= %d: ulimit -n %d\n\n",
			cfg.concurrency, cfg.concurrency*3, cfg.concurrency*3)
	}

	s := &stats{}
	pool := newDeviceTransportPool()
	defer pool.CloseAll()

	fmt.Fprintf(os.Stderr, "Starting load test at %s...\n\n", time.Now().Format(time.RFC3339))
	totalDuration := runLoadRound(cfg, pool, s)

	if *jsonOutput {
		printJSONReport(s, totalDuration, cfg)
	} else {
		printReport(s, totalDuration, cfg)
	}
}

// ---------------------------------------------------------------------------
// Core Load Loop
// ---------------------------------------------------------------------------

// runLoadRound runs one load test round with the given configuration and returns the actual duration.
func runLoadRound(cfg roundConfig, pool *deviceTransportPool, s *stats) time.Duration {
	// Shared transport for inform-only mode
	sharedTransport := &http.Transport{
		MaxIdleConns:        cfg.concurrency * 2,
		MaxIdleConnsPerHost: cfg.concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	var deviceInFlight sync.Map
	sem := make(chan struct{}, cfg.concurrency)
	done := make(chan struct{})
	deadline := time.After(cfg.duration)

	startTime := time.Now()

	go func() {
		<-deadline
		close(done)
	}()

	var wg sync.WaitGroup
	deviceSNs := generateDeviceSNs(cfg.devices)

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

				// In-flight guard for session modes
				if cfg.fullSession || cfg.mode != "acs" {
					if _, loaded := deviceInFlight.LoadOrStore(serialNumber, true); loaded {
						return
					}
					defer deviceInFlight.Delete(serialNumber)
				}

				switch cfg.mode {
				case "acs":
					if cfg.fullSession {
						runFullSession(cfg.url, serialNumber, pool.Get(serialNumber), s)
					} else {
						client := &http.Client{
							Timeout:   30 * time.Second,
							Transport: sharedTransport,
						}
						sendInform(client, cfg.url, serialNumber, s)
					}
				case "kpi":
					runFileUploadSession(cfg.url, serialNumber, "4", cfg.fileServerBase, pool.Get(serialNumber), s)
				case "mr":
					runFileUploadSession(cfg.url, serialNumber, "5", cfg.fileServerBase, pool.Get(serialNumber), s)
				case "all":
					idx := deviceIndex(serialNumber)
					switch idx % 3 {
					case 0:
						runFullSession(cfg.url, serialNumber, pool.Get(serialNumber), s)
					case 1:
						runFileUploadSession(cfg.url, serialNumber, "4", cfg.fileServerBase, pool.Get(serialNumber), s)
					case 2:
						runFileUploadSession(cfg.url, serialNumber, "5", cfg.fileServerBase, pool.Get(serialNumber), s)
					}
				}
			}(sn)
		}

		time.Sleep(cfg.interval)
	}

finish:
	fmt.Fprintln(os.Stderr, "Waiting for in-flight requests to complete...")
	wg.Wait()

	return time.Since(startTime)
}

// ---------------------------------------------------------------------------
// Ramp Mode (5-round escalating load)
// ---------------------------------------------------------------------------

func runRamp(url, mode string, interval time.Duration, fileServerBase string, jsonOutput bool) {
	fmt.Fprintf(os.Stderr, "=== RAMP MODE: 5 rounds, ~15 minutes total ===\n")
	fmt.Fprintf(os.Stderr, "  URL:    %s\n", url)
	fmt.Fprintf(os.Stderr, "  Mode:   %s\n", mode)
	if mode != "acs" {
		fmt.Fprintf(os.Stderr, "  Files:  %s\n", fileServerBase)
	}
	fmt.Fprintln(os.Stderr)

	fmt.Fprintf(os.Stderr, "NOTE: ACS default rate limit is 10/min/device. For maximum throughput,\n")
	fmt.Fprintf(os.Stderr, "      use stress config: ./bin/omcgo-acs --config configs/acs-stress.yaml\n\n")

	type roundResult struct {
		Name     string        `json:"name"`
		Duration time.Duration `json:"-"`
		Stats    *stats        `json:"-"`
		Cfg      roundConfig   `json:"-"`
	}

	var results []roundResult

	for i, round := range rampSchedule {
		cfg := roundConfig{
			url:            url,
			mode:           mode,
			devices:        round.devices,
			concurrency:    round.concurrency,
			duration:       round.duration,
			interval:       interval,
			fullSession:    true,
			fileServerBase: fileServerBase,
		}

		fmt.Fprintf(os.Stderr, "━━━ %s: %d devices, %d concurrent, %s ━━━\n",
			round.name, round.devices, round.concurrency, round.duration)

		s := &stats{}
		pool := newDeviceTransportPool()

		actualDuration := runLoadRound(cfg, pool, s)
		pool.CloseAll()

		results = append(results, roundResult{
			Name:     round.name,
			Duration: actualDuration,
			Stats:    s,
			Cfg:      cfg,
		})

		if jsonOutput {
			printJSONReport(s, actualDuration, cfg)
		} else {
			printReport(s, actualDuration, cfg)
		}

		// Pause between rounds (except after last)
		if i < len(rampSchedule)-1 {
			fmt.Fprintf(os.Stderr, "\nPausing 5s before next round...\n\n")
			time.Sleep(5 * time.Second)
		}
	}

	// Print summary table
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "  RAMP SUMMARY")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(os.Stderr, "  %-12s %8s %8s %8s %10s %10s %10s\n",
		"Round", "Conc", "Devices", "Reqs", "Success%", "Throughput", "Uploads")
	fmt.Fprintf(os.Stderr, "  %-12s %8s %8s %8s %10s %10s %10s\n",
		"─────", "────", "───────", "────", "────────", "──────────", "───────")

	var totalReqs, totalSuccess, totalUploads int64
	for _, r := range results {
		success := r.Stats.successCount.Load()
		fail := r.Stats.failCount.Load()
		errors := r.Stats.errorCount.Load()
		total := success + fail + errors
		uploads := r.Stats.fileUploadCount.Load()
		totalReqs += total
		totalSuccess += success
		totalUploads += uploads

		fmt.Fprintf(os.Stderr, "  %-12s %8d %8d %8d %9.1f%% %8.1f/s %10d\n",
			r.Name, r.Cfg.concurrency, r.Cfg.devices, total,
			pct(success, total),
			float64(total)/r.Duration.Seconds(),
			uploads)
	}

	fmt.Fprintf(os.Stderr, "  %-12s %8s %8s %8d %9.1f%% %10s %10d\n",
		"TOTAL", "", "", totalReqs, pct(totalSuccess, totalReqs), "", totalUploads)
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ---------------------------------------------------------------------------
// Session Functions
// ---------------------------------------------------------------------------

func generateDeviceSNs(count int) []string {
	sns := make([]string, count)
	for i := 0; i < count; i++ {
		sns[i] = fmt.Sprintf("LT-%06d", i)
	}
	return sns
}

// deviceIndex extracts the numeric suffix from a device SN like "LT-000042" → 42.
func deviceIndex(sn string) int {
	parts := strings.SplitN(sn, "-", 2)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}

// buildInformXML builds the Inform SOAP XML for a device.
func buildInformXML(sn string) ([]byte, error) {
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
		return nil, err
	}
	return buf.Bytes(), nil
}

// runFullSession simulates a complete TR069 session:
// Step 1: Inform → InformResponse
// Step 2: Empty POST → RPC Request or Empty Response
// Step 3: If RPC Request → send RPC Response, goto Step 2
// Step 4: If Empty Response → session complete
//
// The transport parameter is a per-device transport from deviceTransportPool.
// MaxConnsPerHost=1 ensures all requests use the same TCP connection (same RemoteAddr).
func runFullSession(url, sn string, transport *http.Transport, s *stats) {
	sessionStart := time.Now()
	rpcRounds := 0

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	// Step 1: Send Inform
	informXML, err := buildInformXML(sn)
	if err != nil {
		s.errorCount.Add(1)
		return
	}

	start := time.Now()
	respBody, statusCode, err := doPost(client, url, informXML)
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
			break
		}

		rpcRounds++

		cwmpID := extractCWMPID(bodyStr)
		rpcResp := buildRPCResponse(rpcMethod, cwmpID)

		start = time.Now()
		_, statusCode, err = doPost(client, url, []byte(rpcResp))
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
	}

	s.sessionCount.Add(1)
	s.rpcRoundCount.Add(int64(rpcRounds))
	s.recordSessionTime(time.Since(sessionStart))
}

// runFileUploadSession simulates a TR069 session with AutonomousTransferComplete:
// Step 1: Inform → InformResponse          (establishes connSessions binding)
// Step 2: ATC    → ATCResponse              (triggers file download pipeline)
// Step 3: Empty POST → Empty/RPC            (session continuation)
// Step 4: Handle RPCs → session end
func runFileUploadSession(acsURL, sn, fileType, fileServerBase string, transport *http.Transport, s *stats) {
	sessionStart := time.Now()
	rpcRounds := 0

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	// Step 1: Send Inform
	informXML, err := buildInformXML(sn)
	if err != nil {
		s.errorCount.Add(1)
		return
	}

	start := time.Now()
	respBody, statusCode, err := doPost(client, acsURL, informXML)
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

	if !strings.Contains(string(respBody), "InformResponse") {
		s.failCount.Add(1)
		return
	}

	// Step 2: Send AutonomousTransferComplete
	now := time.Now()
	var filePath, targetName string
	switch fileType {
	case "4": // PM
		filePath = fmt.Sprintf("/pm/%s.xml", sn)
		targetName = fmt.Sprintf("PM_%s_%s.xml", sn, now.Format("20060102T150405"))
	case "5": // MR
		filePath = fmt.Sprintf("/mr/%s.xml", sn)
		targetName = fmt.Sprintf("MRO_%s_%s.xml", sn, now.Format("20060102T150405"))
	}

	ad := atcData{
		ID:             fmt.Sprintf("lt-atc-%d", rand.Int63()),
		TransferURL:    fileServerBase + filePath,
		FileType:       fileType,
		FileSize:       2048,
		TargetFileName: targetName,
		StartTime:      now.Add(-5 * time.Second).UTC().Format(time.RFC3339),
		CompleteTime:   now.UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := atcTemplate.Execute(&buf, ad); err != nil {
		s.errorCount.Add(1)
		return
	}

	start = time.Now()
	respBody, statusCode, err = doPost(client, acsURL, buf.Bytes())
	latency = time.Since(start)

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

	// Verify AutonomousTransferCompleteResponse
	if !strings.Contains(string(respBody), "AutonomousTransferCompleteResponse") {
		s.failCount.Add(1)
		return
	}
	s.fileUploadCount.Add(1)

	// Step 3-4: Empty POST loop (handle any pending RPC commands from ACS)
	for i := 0; i < 20; i++ {
		start = time.Now()
		respBody, statusCode, err = doPost(client, acsURL, nil)
		latency = time.Since(start)

		if err != nil {
			s.errorCount.Add(1)
			return
		}
		s.recordLatency(latency)

		if statusCode == http.StatusNoContent || len(strings.TrimSpace(string(respBody))) == 0 {
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

		bodyStr := string(respBody)
		rpcMethod := detectRPCMethod(bodyStr)
		if rpcMethod == "" {
			break
		}

		rpcRounds++
		cwmpID := extractCWMPID(bodyStr)
		rpcResp := buildRPCResponse(rpcMethod, cwmpID)

		start = time.Now()
		_, statusCode, err = doPost(client, acsURL, []byte(rpcResp))
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
	}

	s.sessionCount.Add(1)
	s.rpcRoundCount.Add(int64(rpcRounds))
	s.recordSessionTime(time.Since(sessionStart))
}

// ---------------------------------------------------------------------------
// HTTP / SOAP Helpers
// ---------------------------------------------------------------------------

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
		if strings.Contains(body, "cwmp:"+m) && !strings.Contains(body, m+"Response") {
			return m
		}
	}
	return ""
}

// extractCWMPID extracts the CWMP ID from a SOAP message.
func extractCWMPID(body string) string {
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
	informXML, err := buildInformXML(sn)
	if err != nil {
		s.errorCount.Add(1)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(informXML))
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

// ---------------------------------------------------------------------------
// Reporting
// ---------------------------------------------------------------------------

func modeLabel(mode string, fullSession bool) string {
	switch mode {
	case "kpi":
		return "KPI UPLOAD"
	case "mr":
		return "MR UPLOAD"
	case "all":
		return "ALL MODES"
	default:
		if fullSession {
			return "FULL SESSION"
		}
		return "INFORM-ONLY"
	}
}

func printReport(s *stats, totalDuration time.Duration, cfg roundConfig) {
	success := s.successCount.Load()
	fail := s.failCount.Load()
	errors := s.errorCount.Load()
	rateLimited := s.rateLimitCount.Load()
	total := success + fail + errors

	fmt.Println("\n========================================")
	fmt.Printf("    %s LOAD TEST REPORT\n", modeLabel(cfg.mode, cfg.fullSession))
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

	// File upload stats
	if fileUploads := s.fileUploadCount.Load(); fileUploads > 0 {
		fmt.Printf("File Uploads: %d (ATC messages)\n", fileUploads)
		fmt.Printf("Upload Rate:  %.1f uploads/s\n", float64(fileUploads)/totalDuration.Seconds())
	}

	// Session stats (for full-session or file upload modes)
	if cfg.fullSession || cfg.mode != "acs" {
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
	FileUploads      int64   `json:"file_uploads,omitempty"`
	UploadRate       float64 `json:"upload_rate,omitempty"`
	P50Ms            float64 `json:"p50_ms"`
	P90Ms            float64 `json:"p90_ms"`
	P95Ms            float64 `json:"p95_ms"`
	P99Ms            float64 `json:"p99_ms"`
	MaxMs            float64 `json:"max_ms"`
	SessionP50Ms     float64 `json:"session_p50_ms,omitempty"`
	SessionP99Ms     float64 `json:"session_p99_ms,omitempty"`
}

func printJSONReport(s *stats, totalDuration time.Duration, cfg roundConfig) {
	success := s.successCount.Load()
	fail := s.failCount.Load()
	errors := s.errorCount.Load()
	total := success + fail + errors
	rateLimited := s.rateLimitCount.Load()

	r := jsonReport{
		Mode:        cfg.mode,
		Concurrency: cfg.concurrency,
		Devices:     cfg.devices,
		DurationMs:  totalDuration.Milliseconds(),
		Total:       total,
		Success:     success,
		Failed:      fail,
		RateLimited: rateLimited,
		Errors:      errors,
		SuccessRate: pct(success, total),
		Throughput:  float64(total) / totalDuration.Seconds(),
	}

	// Session stats
	if cfg.fullSession || cfg.mode != "acs" {
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

	// File upload stats
	if uploads := s.fileUploadCount.Load(); uploads > 0 {
		r.FileUploads = uploads
		r.UploadRate = float64(uploads) / totalDuration.Seconds()
	}

	// Latency percentiles
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
