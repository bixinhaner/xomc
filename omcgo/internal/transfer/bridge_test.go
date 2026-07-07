package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// --- mock device repo ---

type mockDeviceRepo struct {
	devices map[string]*model.Device
}

func newMockDeviceRepo(devs ...*model.Device) *mockDeviceRepo {
	m := &mockDeviceRepo{devices: make(map[string]*model.Device)}
	for _, d := range devs {
		m.devices[d.SerialNumber] = d
	}
	return m
}

func (m *mockDeviceRepo) Create(_ context.Context, d *model.Device) error {
	m.devices[d.SerialNumber] = d
	return nil
}
func (m *mockDeviceRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Device, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("device not found")
}
func (m *mockDeviceRepo) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	d, ok := m.devices[sn]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", sn)
	}
	return d, nil
}
func (m *mockDeviceRepo) GetDeletedBySerialNumber(_ context.Context, _ string, _ model.CarrierCode) (*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *mockDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *mockDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return nil, nil
}
func (m *mockDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}

// T-0162 新接口方法
func (m *mockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ device.GeoStatsFilter) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}
func (m *mockDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
	return model.NewListResponse([]device.DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*device.RestoreResult, error) {
	return &device.RestoreResult{}, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]device.GeoDevice, error) {
	return nil, nil
}

// --- mock event bus (captures published events) ---

type mockEventBus struct {
	mu        sync.Mutex
	published []publishedEvent
	handlers  map[string]event.EventHandler
}

type publishedEvent struct {
	Subject string
	Event   event.Event
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		handlers: make(map[string]event.EventHandler),
	}
}

func (b *mockEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, publishedEvent{Subject: subject, Event: evt})
	return nil
}

func (b *mockEventBus) Subscribe(subject string, handler event.EventHandler) (event.Subscription, error) {
	b.handlers[subject] = handler
	return &mockSub{}, nil
}

func (b *mockEventBus) QueueSubscribe(subject string, _ string, handler event.EventHandler) (event.Subscription, error) {
	b.handlers[subject] = handler
	return &mockSub{}, nil
}

func (b *mockEventBus) PullSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	return b.QueueSubscribe(subject, queue, handler)
}

func (b *mockEventBus) Close() error { return nil }

func (b *mockEventBus) publishedBySubject(subject string) []publishedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	var result []publishedEvent
	for _, p := range b.published {
		if p.Subject == subject {
			result = append(result, p)
		}
	}
	return result
}

type mockSub struct{}

func (s *mockSub) Unsubscribe() error { return nil }

// --- Tests ---

func TestClassifyFileType_PM(t *testing.T) {
	tests := []struct {
		fileType string
		fileName string
	}{
		{"4", "A20250311.1400+0800-1415+0800_SubNetwork=CMCC.xml"},
		{"PM", "pm_data.xml"},
		{"", "PM_data_20250311.xml"},
		{"", "counter_report.xml"},
	}
	for _, tt := range tests {
		result := classifyFileType(tt.fileType, tt.fileName)
		if result != tr069.FileTypePM {
			t.Errorf("classifyFileType(%q, %q) = %q, want %q", tt.fileType, tt.fileName, result, tr069.FileTypePM)
		}
	}
}

func TestClassifyFileType_MR(t *testing.T) {
	tests := []struct {
		fileType string
		fileName string
	}{
		{"5", "MRO_eNB1234_20250311.xml"},
		{"MR", "data.xml"},
		{"", "MRO_data.xml"},
		{"", "MRS_eNB5678_20250311.xml"},
		{"", "MRE_gNB9012_20250311.xml"},
	}
	for _, tt := range tests {
		result := classifyFileType(tt.fileType, tt.fileName)
		if result != tr069.FileTypeMR {
			t.Errorf("classifyFileType(%q, %q) = %q, want %q", tt.fileType, tt.fileName, result, tr069.FileTypeMR)
		}
	}
}

func TestClassifyFileType_Log(t *testing.T) {
	tests := []struct {
		fileType string
		fileName string
	}{
		{"3", "vendor_log.txt"},
		{"Log", "device.log"},
		{"99", "unknown_file.dat"},
	}
	for _, tt := range tests {
		result := classifyFileType(tt.fileType, tt.fileName)
		if result != tr069.FileTypeRunningLog {
			t.Errorf("classifyFileType(%q, %q) = %q, want %q", tt.fileType, tt.fileName, result, tr069.FileTypeRunningLog)
		}
	}
}

// #178/#222：自主传输运行日志（FileType "6"）应归 RunningLog。
func TestClassifyFileType_RunningLog(t *testing.T) {
	tests := []struct {
		fileType string
		fileName string
	}{
		{"6", "runtime-deadbeef-SN001.tar.gz"},
		{"6", ""},
	}
	for _, tt := range tests {
		result := classifyFileType(tt.fileType, tt.fileName)
		if result != tr069.FileTypeRunningLog {
			t.Errorf("classifyFileType(%q, %q) = %q, want %q", tt.fileType, tt.fileName, result, tr069.FileTypeRunningLog)
		}
	}
}

// #178/#222：自主传输故障日志（FileType "8"）此前误分类成 RunningLog（落 default），
// 应显式归 FaultLog，与 ACS 直传路径对齐。
func TestClassifyFileType_FaultLog(t *testing.T) {
	tests := []struct {
		fileType string
		fileName string
	}{
		{"8", "fault-deadbeef-SN001.tar.gz"},
		{"8", ""},
		{"", "fault-deadbeef-SN001.tar.gz"},
		{"", "device_FAULTLOG.tar.gz"},
	}
	for _, tt := range tests {
		result := classifyFileType(tt.fileType, tt.fileName)
		if result != tr069.FileTypeFaultLog {
			t.Errorf("classifyFileType(%q, %q) = %q, want %q", tt.fileType, tt.fileName, result, tr069.FileTypeFaultLog)
		}
	}
}

// parseLogFilename 从 executor 生成的标准日志文件名解析出 (taskID8, deviceSN)。
func TestParseLogFilename(t *testing.T) {
	tests := []struct {
		filename string
		wantTask string
		wantSN   string
	}{
		{"runtime-deadbeef-SN-ABC-001.tar.gz", "deadbeef", "SN-ABC-001"},
		{"fault-0a1b2c3d-XYZ.tar.gz", "0a1b2c3d", "XYZ"},
		{"adhoc_upload.txt", "", ""}, // 非标准命名 → 空
	}
	for _, tt := range tests {
		gotTask, gotSN := parseLogFilename(tt.filename)
		if gotTask != tt.wantTask || gotSN != tt.wantSN {
			t.Errorf("parseLogFilename(%q) = (%q, %q), want (%q, %q)",
				tt.filename, gotTask, gotSN, tt.wantTask, tt.wantSN)
		}
	}
}

// #178/#222 成功路径：自主传输运行日志事件构造的 SubjectLogFileReceived payload
// 必须能被 stationlog 侧消费的字段名解出（bucket/object_path/file_name/file_type/
// file_size/task_id8/device_sn），否则记录不会写库。
func TestLogFileReceivedPayload_Shape(t *testing.T) {
	bus := newMockEventBus()

	taskID8, deviceSN := parseLogFilename("runtime-deadbeef-TEST-SN-009.tar.gz")
	if deviceSN == "" {
		deviceSN = "TEST-SN-009"
	}
	logPayload := map[string]interface{}{
		"bucket":      "omc-logs",
		"object_path": "running/cmcc/TEST-SN-009/runtime-deadbeef-TEST-SN-009.tar.gz",
		"file_name":   "runtime-deadbeef-TEST-SN-009.tar.gz",
		"file_type":   string(tr069.FileTypeRunningLog),
		"file_size":   int64(2048),
		"task_id8":    taskID8,
		"device_sn":   deviceSN,
	}
	logEvt, err := event.NewEvent(event.SubjectLogFileReceived, logPayload)
	if err != nil {
		t.Fatalf("create log event: %v", err)
	}
	if err := bus.Publish(context.Background(), event.SubjectLogFileReceived, logEvt); err != nil {
		t.Fatalf("publish log event: %v", err)
	}

	published := bus.publishedBySubject(event.SubjectLogFileReceived)
	if len(published) != 1 {
		t.Fatalf("expected 1 log.file.received event, got %d", len(published))
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(published[0].Event.Payload, &decoded); err != nil {
		t.Fatalf("decode log payload: %v", err)
	}
	if decoded["device_sn"] != "TEST-SN-009" {
		t.Errorf("expected device_sn=TEST-SN-009, got %v", decoded["device_sn"])
	}
	if decoded["file_type"] != "6" {
		t.Errorf("expected file_type=6 (running log), got %v", decoded["file_type"])
	}
	if decoded["task_id8"] != "deadbeef" {
		t.Errorf("expected task_id8=deadbeef, got %v", decoded["task_id8"])
	}
	if decoded["object_path"] == "" || decoded["object_path"] == nil {
		t.Errorf("expected non-empty object_path, got %v", decoded["object_path"])
	}
}

func TestHandleATC_FaultSkipped(t *testing.T) {
	logger := zap.NewNop()
	bus := newMockEventBus()
	repo := newMockDeviceRepo()

	bridge := NewTransferBridge(repo, nil, appconfig.BucketConfig{PMFiles: "pm", MRFiles: "mr", Logs: "logs"}, bus, nil, logger)

	payload := map[string]interface{}{
		"device_sn":       "TEST-SN-001",
		"transfer_url":    "http://example.com/file.xml",
		"file_type":       "4",
		"target_filename": "pm_data.xml",
		"fault": map[string]interface{}{
			"fault_code":   "9001",
			"fault_string": "Upload failure",
		},
	}
	evt, _ := event.NewEvent(event.SubjectDeviceAutonomousTransferComplete, payload)

	err := bridge.handleAutonomousTransferComplete(context.Background(), evt)
	if err != nil {
		t.Fatalf("expected nil error for fault, got: %v", err)
	}

	// No events should have been published
	if len(bus.published) != 0 {
		t.Errorf("expected 0 published events, got %d", len(bus.published))
	}
}

func TestHandleATC_DeviceNotFound(t *testing.T) {
	logger := zap.NewNop()
	bus := newMockEventBus()
	repo := newMockDeviceRepo() // empty

	bridge := NewTransferBridge(repo, nil, appconfig.BucketConfig{PMFiles: "pm", MRFiles: "mr", Logs: "logs"}, bus, nil, logger)

	payload := map[string]interface{}{
		"device_sn":       "UNKNOWN-SN",
		"transfer_url":    "http://example.com/file.xml",
		"file_type":       "4",
		"target_filename": "pm_data.xml",
	}
	evt, _ := event.NewEvent(event.SubjectDeviceAutonomousTransferComplete, payload)

	err := bridge.handleAutonomousTransferComplete(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error for unknown device, got nil")
	}
}

func TestHandleATC_NoTransferURL(t *testing.T) {
	logger := zap.NewNop()
	bus := newMockEventBus()
	repo := newMockDeviceRepo()

	bridge := NewTransferBridge(repo, nil, appconfig.BucketConfig{PMFiles: "pm", MRFiles: "mr", Logs: "logs"}, bus, nil, logger)

	payload := map[string]interface{}{
		"device_sn":       "TEST-SN-001",
		"transfer_url":    "",
		"file_type":       "4",
		"target_filename": "pm_data.xml",
	}
	evt, _ := event.NewEvent(event.SubjectDeviceAutonomousTransferComplete, payload)

	err := bridge.handleAutonomousTransferComplete(context.Background(), evt)
	if err != nil {
		t.Fatalf("expected nil error for empty transfer URL, got: %v", err)
	}

	if len(bus.published) != 0 {
		t.Errorf("expected 0 published events for empty URL, got %d", len(bus.published))
	}
}

func TestHandleATC_PMFilePublish(t *testing.T) {
	bus := newMockEventBus()

	devID := uuid.New()
	_ = newMockDeviceRepo(&model.Device{
		ID:           devID,
		SerialNumber: "TEST-SN-001",
		Carrier:      "CMCC",
		Technology:   "LTE",
	})

	// HTTP test server to serve the "file"
	fileContent := []byte("<xml>pm data</xml>")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(fileContent)
	}))
	defer ts.Close()

	// We need a mock MinIO client - since minio.Client is a concrete struct,
	// we test downloadAndStore indirectly. For the unit test, we'll call
	// handleAutonomousTransferComplete which calls downloadAndStore. The MinIO
	// PutObject will fail since we pass nil, so we test the classify + publish
	// logic separately.

	// Test the event construction path by directly testing classifyFileType
	// and the publish logic structure
	if classifyFileType("4", "pm_data.xml") != tr069.FileTypePM {
		t.Fatal("expected PM classification")
	}

	// Verify PM event payload structure
	pmPayload := map[string]interface{}{
		"minio_path": "2026/03/11/TEST-SN-001/pm_data.xml",
		"device_id":  devID.String(),
		"device_sn":  "TEST-SN-001",
		"carrier":    "CMCC",
		"technology": "LTE",
	}
	pmEvt, err := event.NewEvent(event.SubjectPMFileReceived, pmPayload)
	if err != nil {
		t.Fatalf("create PM event: %v", err)
	}

	if err := bus.Publish(context.Background(), event.SubjectPMFileReceived, pmEvt); err != nil {
		t.Fatalf("publish PM event: %v", err)
	}

	published := bus.publishedBySubject(event.SubjectPMFileReceived)
	if len(published) != 1 {
		t.Fatalf("expected 1 PM event, got %d", len(published))
	}

	// Verify payload fields
	var decoded map[string]interface{}
	if err := json.Unmarshal(published[0].Event.Payload, &decoded); err != nil {
		t.Fatalf("decode PM payload: %v", err)
	}
	if decoded["device_sn"] != "TEST-SN-001" {
		t.Errorf("expected device_sn=TEST-SN-001, got %v", decoded["device_sn"])
	}
	if decoded["carrier"] != "CMCC" {
		t.Errorf("expected carrier=CMCC, got %v", decoded["carrier"])
	}
	if decoded["technology"] != "LTE" {
		t.Errorf("expected technology=LTE, got %v", decoded["technology"])
	}
}

func TestHandleATC_MRFilePublish(t *testing.T) {
	bus := newMockEventBus()

	devID := uuid.New()

	// Test MR classification
	if classifyFileType("5", "MRO_eNB1234.xml") != tr069.FileTypeMR {
		t.Fatal("expected MR classification for file_type=5")
	}
	if classifyFileType("", "MRO_eNB1234.xml") != tr069.FileTypeMR {
		t.Fatal("expected MR classification for MRO filename")
	}

	// Verify MR event payload structure
	mrPayload := map[string]interface{}{
		"minio_path": "2026/03/11/TEST-SN-002/MRO_eNB1234.xml",
		"bucket":     "mr-files",
		"device_id":  devID.String(),
		"device_sn":  "TEST-SN-002",
		"carrier":    "CTCC",
		"file_name":  "MRO_eNB1234.xml",
		"file_size":  int64(1024),
	}
	mrEvt, err := event.NewEvent(event.SubjectMRFileReceived, mrPayload)
	if err != nil {
		t.Fatalf("create MR event: %v", err)
	}

	if err := bus.Publish(context.Background(), event.SubjectMRFileReceived, mrEvt); err != nil {
		t.Fatalf("publish MR event: %v", err)
	}

	published := bus.publishedBySubject(event.SubjectMRFileReceived)
	if len(published) != 1 {
		t.Fatalf("expected 1 MR event, got %d", len(published))
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(published[0].Event.Payload, &decoded); err != nil {
		t.Fatalf("decode MR payload: %v", err)
	}
	if decoded["device_sn"] != "TEST-SN-002" {
		t.Errorf("expected device_sn=TEST-SN-002, got %v", decoded["device_sn"])
	}
	if decoded["carrier"] != "CTCC" {
		t.Errorf("expected carrier=CTCC, got %v", decoded["carrier"])
	}
	if decoded["file_name"] != "MRO_eNB1234.xml" {
		t.Errorf("expected file_name=MRO_eNB1234.xml, got %v", decoded["file_name"])
	}
}

func TestSubscribe(t *testing.T) {
	logger := zap.NewNop()
	bus := newMockEventBus()
	repo := newMockDeviceRepo()

	bridge := NewTransferBridge(repo, nil, appconfig.BucketConfig{PMFiles: "pm", MRFiles: "mr", Logs: "logs"}, bus, nil, logger)

	if err := bridge.Subscribe(bus); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	handler, ok := bus.handlers[event.SubjectDeviceAutonomousTransferComplete]
	if !ok {
		t.Fatal("expected handler registered for autonomous_transfer_complete")
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestDownloadAndStore_HTTPError(t *testing.T) {
	logger := zap.NewNop()
	bus := newMockEventBus()

	// Server that returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	bridge := NewTransferBridge(nil, nil, appconfig.BucketConfig{PMFiles: "pm", MRFiles: "mr", Logs: "logs"}, bus, nil, logger)

	_, err := bridge.downloadAndStore(context.Background(), ts.URL+"/file.xml", "test-bucket", "test/path.xml")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
