package nats

import (
	"strings"
	"testing"
	"time"

	gonats "github.com/nats-io/nats.go"
)

// subjectMatchesStream 判断一个具体 subject（如 "log.file.received"）是否被某条
// 流的 subject 模式（如 "log.>"）覆盖。仅支持本项目用到的 ">" 末尾通配。
func subjectMatchesStream(subject, pattern string) bool {
	if strings.HasSuffix(pattern, ".>") {
		prefix := strings.TrimSuffix(pattern, ">")
		return strings.HasPrefix(subject, prefix)
	}
	return subject == pattern
}

func TestDefaultStreams_PMAggregationTieredRetention(t *testing.T) {
	expected := map[string]struct {
		age     time.Duration
		subject string
	}{
		"PM_AGG_15M":    {age: 2 * time.Hour, subject: "pmaggregation.15m.normalized"},
		"PM_AGG_HOURLY": {age: 48 * time.Hour, subject: "pmaggregation.hourly.rollup"},
		"PM_AGG_DAILY":  {age: 40 * 24 * time.Hour, subject: "pmaggregation.daily.rollup"},
	}
	found := map[string]bool{}
	for _, stream := range DefaultStreams() {
		want, ok := expected[stream.Name]
		if !ok {
			continue
		}
		found[stream.Name] = true
		if stream.Retention != gonats.LimitsPolicy {
			t.Fatalf("%s retention = %v, want LimitsPolicy", stream.Name, stream.Retention)
		}
		if stream.MaxAge != want.age {
			t.Fatalf("%s max age = %v, want %v", stream.Name, stream.MaxAge, want.age)
		}
		if stream.MaxBytes <= 0 {
			t.Fatalf("%s must have a hard byte limit", stream.Name)
		}
		if stream.Compression != gonats.S2Compression {
			t.Fatalf("%s compression = %v, want S2", stream.Name, stream.Compression)
		}
		if !subjectMatchesStream(want.subject, stream.Subjects[0]) {
			t.Fatalf("%s subject not covered by %v", want.subject, stream.Subjects)
		}
	}
	for name := range expected {
		if !found[name] {
			t.Fatalf("%s stream is not registered", name)
		}
	}
}

func TestDefaultStreams_DomainAlarmSupportsIndependentDurableConsumers(t *testing.T) {
	streams := DefaultStreams()

	var domainAlarm *StreamDef
	for i := range streams {
		if streams[i].Name == "DOMAIN_ALARM" {
			domainAlarm = &streams[i]
			break
		}
	}
	if domainAlarm == nil {
		t.Fatal("DOMAIN_ALARM stream is not registered")
	}

	if domainAlarm.Retention != gonats.LimitsPolicy {
		t.Fatalf("DOMAIN_ALARM retention = %v, want LimitsPolicy", domainAlarm.Retention)
	}
	if !domainAlarm.AllowDirect {
		t.Fatal("DOMAIN_ALARM must allow direct lookup for operational diagnosis")
	}
	if domainAlarm.MaxAge != 7*24*time.Hour {
		t.Fatalf("DOMAIN_ALARM max age = %v, want 7 days", domainAlarm.MaxAge)
	}
	if domainAlarm.MaxBytes != alarmLifecycleStreamMaxBytes {
		t.Fatalf("DOMAIN_ALARM max bytes = %d, want %d", domainAlarm.MaxBytes, alarmLifecycleStreamMaxBytes)
	}
	if domainAlarm.MaxBytes <= 0 {
		t.Fatal("DOMAIN_ALARM must have a hard byte limit")
	}
	if domainAlarm.Compression != gonats.S2Compression {
		t.Fatalf("DOMAIN_ALARM compression = %v, want S2", domainAlarm.Compression)
	}
	if !subjectCovered("domain.alarm.lifecycle.raised", []StreamDef{*domainAlarm}) {
		t.Fatalf("DOMAIN_ALARM subjects = %v, want domain.alarm.> coverage", domainAlarm.Subjects)
	}
}

func TestDefaultStreams_DomainAlarmDoesNotOverlapLegacyAlarm(t *testing.T) {
	streams := DefaultStreams()
	var legacy, domain *StreamDef
	for i := range streams {
		switch streams[i].Name {
		case "ALARM":
			legacy = &streams[i]
		case "DOMAIN_ALARM":
			domain = &streams[i]
		}
	}
	if legacy == nil || domain == nil {
		t.Fatalf("required streams missing: ALARM=%v DOMAIN_ALARM=%v", legacy != nil, domain != nil)
	}

	if subjectCovered("domain.alarm.lifecycle.raised", []StreamDef{*legacy}) {
		t.Fatalf("legacy ALARM subjects %v overlap the canonical domain alarm namespace", legacy.Subjects)
	}
	if subjectCovered("alarm.raised", []StreamDef{*domain}) {
		t.Fatalf("DOMAIN_ALARM subjects %v overlap the legacy alarm namespace", domain.Subjects)
	}
}

func subjectCovered(subject string, streams []StreamDef) bool {
	for _, s := range streams {
		for _, p := range s.Subjects {
			if subjectMatchesStream(subject, p) {
				return true
			}
		}
	}
	return false
}

// TestDefaultStreams_LogStreamRegistered 守卫 issue #178/#222 的根因修复：
// stationlog 订阅 log.file.received，自主传输桥发布同一 subject；NATS 部署态下
// 必须有一条流吸纳 log.>，否则 publish 报 "no response from stream"、记录永不入库。
func TestDefaultStreams_LogStreamRegistered(t *testing.T) {
	streams := DefaultStreams()

	var logStream *StreamDef
	for i := range streams {
		if streams[i].Name == "LOG" {
			logStream = &streams[i]
			break
		}
	}
	if logStream == nil {
		t.Fatal("LOG stream not registered in DefaultStreams (issue #178/#222 regression)")
	}

	// 必须以 log.> 前缀吸纳所有 log.* 事件
	hasLogPrefix := false
	for _, s := range logStream.Subjects {
		if s == "log.>" {
			hasLogPrefix = true
		}
	}
	if !hasLogPrefix {
		t.Errorf("LOG stream subjects = %v, want to include \"log.>\"", logStream.Subjects)
	}

	// stationlog 是单 consumer fan-in，WorkQueuePolicy（与 PM/MR 同档）
	if logStream.Retention != gonats.WorkQueuePolicy {
		t.Errorf("LOG stream retention = %v, want WorkQueuePolicy", logStream.Retention)
	}
}

// TestDefaultStreams_LogFileReceivedCovered 端到端守卫：实际用到的 subject
// "log.file.received"（event.SubjectLogFileReceived）必须被某条流覆盖。
func TestDefaultStreams_LogFileReceivedCovered(t *testing.T) {
	if !subjectCovered("log.file.received", DefaultStreams()) {
		t.Error("subject log.file.received is not covered by any DefaultStreams stream")
	}
}

// TestDefaultStreams_NamesUnique 守卫不会因复制粘贴造成重复流名。
func TestDefaultStreams_NamesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range DefaultStreams() {
		if seen[s.Name] {
			t.Errorf("duplicate stream name %q in DefaultStreams", s.Name)
		}
		seen[s.Name] = true
	}
}

func TestDefaultStreams_PMAllowsDirectLookupForQueueHealth(t *testing.T) {
	for _, stream := range DefaultStreams() {
		if stream.Name == "PM" {
			if !stream.AllowDirect {
				t.Fatal("PM stream must allow direct subject-filtered lookup for queue health")
			}
			return
		}
	}
	t.Fatal("PM stream is not registered")
}

func TestEnableDirectLookupRunsBeforeRetentionRebuildPolicy(t *testing.T) {
	pm := StreamDef{Name: "PM", Retention: gonats.WorkQueuePolicy, AllowDirect: true}
	info := &gonats.StreamInfo{Config: gonats.StreamConfig{
		Name:        "PM",
		Retention:   gonats.LimitsPolicy,
		AllowDirect: false,
	}}
	updated := false

	err := enableDirectLookup(pm, info, func(config *gonats.StreamConfig) (*gonats.StreamInfo, error) {
		updated = true
		if !config.AllowDirect {
			t.Fatal("AllowDirect must be enabled even when retention rebuild is disabled")
		}
		return &gonats.StreamInfo{Config: *config}, nil
	})

	if err != nil {
		t.Fatalf("enable direct lookup: %v", err)
	}
	if !updated {
		t.Fatal("direct lookup upgrade was not attempted")
	}
}
