package pm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
)

// stubTaskCreator 抓取 CreateTask 入参，断言 OnlineSubscriber 发出的 SPV 是否符合预期。
type stubTaskCreator struct {
	mu             sync.Mutex
	captured       []*task.CreateTaskRequest
	err            error
	open           *task.Task
	completed      *task.Task
	completedByKey map[string]*task.Task
	completedKeys  []string
	lookupErr      error
	openStarted    chan struct{}
	openRelease    <-chan struct{}
	openStartOnce  sync.Once
}

func (s *stubTaskCreator) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captured = append(s.captured, req)
	if s.err != nil {
		return nil, s.err
	}
	return &task.Task{ID: "stub-task-" + req.DeviceSN}, nil
}

func (s *stubTaskCreator) LatestOpenTaskByDeviceAndMethod(
	_ context.Context, _, _, _ string,
) (*task.Task, error) {
	if s.openStarted != nil {
		s.openStartOnce.Do(func() { close(s.openStarted) })
		<-s.openRelease
	}
	return s.open, s.lookupErr
}

func (s *stubTaskCreator) LatestCompletedTaskByDeviceAndCommandKey(
	_ context.Context, _, commandKey string,
) (*task.Task, error) {
	s.completedKeys = append(s.completedKeys, commandKey)
	if s.completedByKey != nil {
		return s.completedByKey[commandKey], s.lookupErr
	}
	return s.completed, s.lookupErr
}

type stubPMSetupAdmissionGate struct {
	acquired bool
	err      error
	released int
	renewed  int
}

type stubPMUploadAddressResolver struct {
	decision  transfercfg.AddressDecision
	err       error
	deviceIDs []uuid.UUID
	direction transfercfg.TransferDirection
}

func (r *stubPMUploadAddressResolver) Resolve(
	_ context.Context,
	deviceID uuid.UUID,
	direction transfercfg.TransferDirection,
) (transfercfg.AddressDecision, error) {
	r.deviceIDs = append(r.deviceIDs, deviceID)
	r.direction = direction
	if r.err != nil {
		return transfercfg.AddressDecision{}, r.err
	}
	return r.decision, nil
}

type staticPMTransferProvider struct {
	snapshot transfercfg.Snapshot
}

func (p staticPMTransferProvider) Snapshot(context.Context) transfercfg.Snapshot {
	return p.snapshot
}

type stubPMHTTPSCapabilityReader struct {
	status    transfercfg.HTTPSCapabilityStatus
	deviceIDs []uuid.UUID
}

func (r *stubPMHTTPSCapabilityReader) ReadHTTPSCapability(
	_ context.Context,
	deviceID uuid.UUID,
) transfercfg.HTTPSCapabilityStatus {
	r.deviceIDs = append(r.deviceIDs, deviceID)
	return r.status
}

func (g *stubPMSetupAdmissionGate) Acquire(
	context.Context,
	string,
) (string, bool, error) {
	if !g.acquired {
		return "", false, g.err
	}
	return "lease-token", true, g.err
}

func (g *stubPMSetupAdmissionGate) Release(context.Context, string, string) error {
	g.released++
	return nil
}

func (g *stubPMSetupAdmissionGate) Renew(context.Context, string, string) (bool, error) {
	g.renewed++
	return true, g.err
}

func mustEvent(t *testing.T, payload device.DeviceOnlineEvent) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectDeviceOnline, payload)
	require.NoError(t, err)
	return evt
}

func samplePayload() device.DeviceOnlineEvent {
	return device.DeviceOnlineEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "BLQ-TEST-001",
		ProductClass: "FAP/mBS31001/SC",
		SwVersion:    "BaiBLQ_5.0.16.1_1229",
	}
}

func capturedPMUploadURL(t *testing.T, req *task.CreateTaskRequest) string {
	t.Helper()
	var params spvParams
	require.NoError(t, json.Unmarshal(req.Params, &params))
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			return v.Value
		}
	}
	t.Fatalf("missing PM upload URL SPV parameter")
	return ""
}

func Test_OnlineSubscriber_EnqueuesSingleSPVWith3Params(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(
		stub,
		"http://1.2.3.4:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1", 900, nil,
	)
	sample := samplePayload()
	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, sample)))

	require.Len(t, stub.captured, 1)
	req := stub.captured[0]
	assert.Equal(t, "BLQ-TEST-001", req.DeviceSN)
	assert.Equal(t, "SetParameterValues", req.Method)
	assert.Equal(t, task.TaskSourceSystem, req.Source)
	assert.Equal(t, "", req.CreatorID, "system task should not have CreatorID (skip notification)")
	assert.Equal(t, "pm_upload_setup_on_online:"+sample.DeviceID.String(), req.CommandKey)
	require.NotNil(t, req.MaxRetries)
	assert.Equal(t, req.ExpiresIn/req.RetryIntervalSeconds, *req.MaxRetries,
		"自动 PM 配置任务的重试预算必须覆盖完整 TTL")
	assert.Equal(t, 30, req.RetryIntervalSeconds)

	var params spvParams
	require.NoError(t, json.Unmarshal(req.Params, &params))
	require.Len(t, params.Values, 3)

	pathMap := map[string]spvParam{}
	for _, v := range params.Values {
		pathMap[v.Name] = v
	}
	enable := pathMap["Device.FAP.PerfMgmt.Config.1.Enable"]
	assert.Equal(t, "1", enable.Value)
	assert.Equal(t, "xsd:boolean", enable.Type)

	url := pathMap["Device.FAP.PerfMgmt.Config.1.URL"]
	assert.Contains(t, url.Value, "http://1.2.3.4:7557/smallcell/FileUploadService")
	assert.Equal(t, "xsd:string", url.Type)

	interval := pathMap["Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval"]
	assert.Equal(t, "900", interval.Value)
	assert.Equal(t, "xsd:unsignedInt", interval.Type)
}

func Test_OnlineSubscriber_EnvSubstitutionDefaultFallback(t *testing.T) {
	stub := &stubTaskCreator{}
	// 故意不设 OMC_PUBLIC_HOST 让 :- default 生效
	require.NoError(t, os.Unsetenv("OMC_PUBLIC_HOST_TEST_KEY"))
	tmpl := "http://${OMC_PUBLIC_HOST_TEST_KEY:-localhost}:7557/x?fileType=PM&filename="
	s := NewOnlineSubscriber(stub, tmpl, "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)

	var params spvParams
	_ = json.Unmarshal(stub.captured[0].Params, &params)
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			assert.Contains(t, v.Value, "http://localhost:7557")
		}
	}
}

func Test_OnlineSubscriber_EnvSubstitutionFromEnv(t *testing.T) {
	stub := &stubTaskCreator{}
	t.Setenv("OMC_PUBLIC_HOST_TEST_KEY2", "172.19.1.132")
	tmpl := "http://${OMC_PUBLIC_HOST_TEST_KEY2:-localhost}:7557/x?fileType=PM&filename="
	s := NewOnlineSubscriber(stub, tmpl, "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)

	var params spvParams
	_ = json.Unmarshal(stub.captured[0].Params, &params)
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			assert.Contains(t, v.Value, "http://172.19.1.132:7557")
		}
	}
}

func Test_OnlineSubscriber_TransientFailureIsRetried(t *testing.T) {
	stub := &stubTaskCreator{err: errors.New("enqueue failed")}
	s := NewOnlineSubscriber(stub, "http://localhost/x", "1", 900, nil)

	err := s.handleOnline(context.Background(), mustEvent(t, samplePayload()))
	assert.Error(t, err)
}

func Test_OnlineSubscriber_EmptySerialIsSkipped(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(stub, "http://localhost/x", "1", 900, nil)

	payload := samplePayload()
	payload.SerialNumber = ""
	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, payload)))

	assert.Empty(t, stub.captured, "empty SN should be skipped without enqueue")
}

func Test_OnlineSubscriber_MalformedURLIsSkipped(t *testing.T) {
	stub := &stubTaskCreator{}
	// 模板不含 :// → 渲染后也不是合法 URL，跳过
	s := NewOnlineSubscriber(stub, "localhost/x", "1", 900, nil)
	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	assert.Empty(t, stub.captured)
}

// Test_OnlineSubscriber_EmptyHostSkipsSPV 复现 #207：OMC_PUBLIC_HOST 漏配时模板渲染成
// "http://:7557/..."（含 :// 含端口，但 host 段被清空）。旧守卫只查 :// 会放行，
// 把残缺 URL 下发到设备导致 KPI 不上报。新 host 非空守卫必须拦住（不下发）。
func Test_OnlineSubscriber_EmptyHostSkipsSPV(t *testing.T) {
	stub := &stubTaskCreator{}
	// 故意不设 OMC_PUBLIC_HOST_207，模板无 :- 兜底 → 渲染成 host 为空
	require.NoError(t, os.Unsetenv("OMC_PUBLIC_HOST_207"))
	tmpl := "http://${OMC_PUBLIC_HOST_207}:7557/smallcell/FileUploadService?fileType=PM&filename="
	s := NewOnlineSubscriber(stub, tmpl, "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	assert.Empty(t, stub.captured, "empty-host URL (http://:7557) must not be pushed to device")
}

func Test_ValidateUploadURLTemplate(t *testing.T) {
	t.Run("valid host renders ok", func(t *testing.T) {
		t.Setenv("OMC_PUBLIC_HOST_207_OK", "172.19.1.132")
		rendered, err := ValidateUploadURLTemplate("http://${OMC_PUBLIC_HOST_207_OK}:7557/x?fileType=PM&filename=")
		require.NoError(t, err)
		assert.Equal(t, "http://172.19.1.132:7557/x?fileType=PM&filename=", rendered)
	})

	t.Run("empty host is rejected", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("OMC_PUBLIC_HOST_207_EMPTY"))
		rendered, err := ValidateUploadURLTemplate("http://${OMC_PUBLIC_HOST_207_EMPTY}:7557/x")
		require.Error(t, err)
		assert.Equal(t, "http://:7557/x", rendered, "rendered value returned for log visibility")
	})

	t.Run("missing scheme is rejected", func(t *testing.T) {
		_, err := ValidateUploadURLTemplate("localhost:7557/x")
		require.Error(t, err)
	})

	t.Run("default fallback host renders ok", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("OMC_PUBLIC_HOST_207_FB"))
		rendered, err := ValidateUploadURLTemplate("http://${OMC_PUBLIC_HOST_207_FB:-localhost}:7557/x")
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:7557/x", rendered)
	})
}

// Test_OnlineSubscriber_BaseURLResolverOverridesHost 验证：注入 sys_config 解析器后，
// 上传 URL 的 scheme/host 用 acs_transfer.uploadBaseURL，path+query 仍取自模板。
func Test_OnlineSubscriber_BaseURLResolverOverridesHost(t *testing.T) {
	stub := &stubTaskCreator{}
	// 模板 host 故意是不可达的 localhost；解析器给出真实可达基址
	s := NewOnlineSubscriber(stub, "http://localhost:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	s.SetUploadBaseURLResolver(func(_ context.Context) string { return "http://172.19.1.173:8080" })

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)
	var params spvParams
	require.NoError(t, json.Unmarshal(stub.captured[0].Params, &params))
	var gotURL string
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			gotURL = v.Value
		}
	}
	// host 被换成 sys_config 基址，path+query 保留
	assert.Equal(t, "http://172.19.1.173:8080/smallcell/FileUploadService?fileType=PM&filename=", gotURL)
}

// Test_OnlineSubscriber_BaseURLResolverEmptyFallsBackToTemplate 验证：解析器返回空时回退模板。
func Test_OnlineSubscriber_BaseURLResolverEmptyFallsBackToTemplate(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	s.SetUploadBaseURLResolver(func(_ context.Context) string { return "" }) // 未配置 → 回退
	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)
	var params spvParams
	require.NoError(t, json.Unmarshal(stub.captured[0].Params, &params))
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			assert.Contains(t, v.Value, "http://1.2.3.4:8080/smallcell/FileUploadService")
		}
	}
}

func Test_OnlineSubscriber_UsesUnifiedUploadAddressDecision(t *testing.T) {
	cases := []struct {
		name                string
		policy              string
		capability          transfercfg.HTTPSCapabilityStatus
		wantURL             string
		wantCapabilityReads int
	}{
		{
			name:                "force_http_even_when_capability_enabled",
			policy:              transfercfg.ProtocolPolicyForceHTTP,
			capability:          transfercfg.HTTPSCapabilityEnabled,
			wantURL:             "http://upload-http.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			wantCapabilityReads: 0,
		},
		{
			name:                "prefer_https_enabled",
			policy:              transfercfg.ProtocolPolicyPreferHTTPS,
			capability:          transfercfg.HTTPSCapabilityEnabled,
			wantURL:             "https://upload-https.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=",
			wantCapabilityReads: 1,
		},
		{
			name:                "prefer_https_disabled_falls_back_to_http",
			policy:              transfercfg.ProtocolPolicyPreferHTTPS,
			capability:          transfercfg.HTTPSCapabilityDisabled,
			wantURL:             "http://upload-http.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			wantCapabilityReads: 1,
		},
		{
			name:                "prefer_https_unknown_falls_back_to_http",
			policy:              transfercfg.ProtocolPolicyPreferHTTPS,
			capability:          transfercfg.HTTPSCapabilityUnknown,
			wantURL:             "http://upload-http.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			wantCapabilityReads: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubTaskCreator{}
			capabilityReader := &stubPMHTTPSCapabilityReader{status: tc.capability}
			resolver := transfercfg.NewAddressResolver(staticPMTransferProvider{snapshot: transfercfg.Snapshot{
				ProtocolPolicy: tc.policy,
				Upload: transfercfg.UploadSettings{
					BaseURL:      "http://upload-http.example.com:8080",
					HTTPSBaseURL: "https://upload-https.example.com:8443",
				},
			}}, capabilityReader)
			s := NewOnlineSubscriber(
				stub,
				"http://template.example.com:7557/smallcell/FileUploadService?fileType=PM&filename=",
				"1",
				900,
				nil,
			)
			s.SetUploadAddressResolver(resolver)
			payload := samplePayload()

			require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, payload)))
			require.Len(t, stub.captured, 1)
			assert.Equal(t, tc.wantURL, capturedPMUploadURL(t, stub.captured[0]))
			assert.Len(t, capabilityReader.deviceIDs, tc.wantCapabilityReads)
			if tc.wantCapabilityReads > 0 {
				assert.Equal(t, payload.DeviceID, capabilityReader.deviceIDs[0])
			}
		})
	}
}

func Test_OnlineSubscriber_UploadAddressResolverErrorRetriesEvent(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(
		stub,
		"http://template.example.com:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1",
		900,
		nil,
	)
	s.SetUploadAddressResolver(&stubPMUploadAddressResolver{err: errors.New("https base unavailable")})

	err := s.handleOnline(context.Background(), mustEvent(t, samplePayload()))
	require.Error(t, err)
	assert.Empty(t, stub.captured)
}

func Test_OnlineSubscriber_RegisteredPassesDeviceIDToUploadAddressResolver(t *testing.T) {
	stub := &stubTaskCreator{}
	resolver := &stubPMUploadAddressResolver{decision: transfercfg.AddressDecision{
		Direction: transfercfg.TransferDirectionUpload,
		BaseURL:   "https://upload.example.com:8443",
	}}
	s := NewOnlineSubscriber(
		stub,
		"http://template.example.com:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1",
		900,
		nil,
	)
	s.SetUploadAddressResolver(resolver)
	deviceID := uuid.New()
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     deviceID.String(),
		"serial_number": "REG-ADDR-001",
		"created":       true,
	})
	require.NoError(t, err)

	require.NoError(t, s.handleRegistered(context.Background(), evt))
	require.Len(t, stub.captured, 1)
	require.Len(t, resolver.deviceIDs, 1)
	assert.Equal(t, deviceID, resolver.deviceIDs[0])
	assert.Equal(t, "https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=", capturedPMUploadURL(t, stub.captured[0]))
}

func Test_OnlineSubscriber_UnifiedUploadAddressPreservesBaseProxyPrefix(t *testing.T) {
	stub := &stubTaskCreator{}
	resolver := &stubPMUploadAddressResolver{decision: transfercfg.AddressDecision{
		Direction: transfercfg.TransferDirectionUpload,
		BaseURL:   "https://upload.example.com:8443/reverse-proxy",
	}}
	s := NewOnlineSubscriber(
		stub,
		"http://template.example.com:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1",
		900,
		nil,
	)
	s.SetUploadAddressResolver(resolver)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)
	assert.Equal(
		t,
		"https://upload.example.com:8443/reverse-proxy/smallcell/FileUploadService?fileType=PM&filename=",
		capturedPMUploadURL(t, stub.captured[0]),
	)
}

// Test_OnlineSubscriber_RegisteredTriggersSPV 验证：首次 onboard 的 device.registered
// 事件（map payload）也会下发 PM 上报配置（覆盖「首次上线」，不只重连）。
func Test_OnlineSubscriber_RegisteredTriggersSPV(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	deviceID := uuid.New().String()
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     deviceID,
		"serial_number": "REG-TEST-001",
		"product_class": "FAP/MLQ/SC",
	})
	require.NoError(t, err)
	require.NoError(t, s.handleRegistered(context.Background(), evt))
	require.Len(t, stub.captured, 1)
	assert.Equal(t, "REG-TEST-001", stub.captured[0].DeviceSN)
	assert.Equal(t, "pm_upload_setup_on_online:"+deviceID, stub.captured[0].CommandKey)
}

func Test_OnlineSubscriber_SkipsExistingOpenPMSetup(t *testing.T) {
	const uploadURL = "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename="
	params, err := json.Marshal(buildSPVParams("1", uploadURL, 900))
	require.NoError(t, err)
	payload := samplePayload()
	stub := &stubTaskCreator{open: &task.Task{
		ID:         "already-open",
		Params:     params,
		CommandKey: pmSetupCommandKey + ":" + payload.DeviceID.String(),
	}}
	s := NewOnlineSubscriber(stub, uploadURL, "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, payload)))
	require.Empty(t, stub.captured)
}

func Test_OnlineSubscriber_DoesNotLetOldOpenTaskSwallowChangedConfig(t *testing.T) {
	stub := &stubTaskCreator{open: &task.Task{
		ID: "old-open", Params: json.RawMessage(`{"values":[]}`),
	}}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)
}

func Test_OnlineSubscriber_SkipsCompletedMatchingPMSetupOnReconnect(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	params, err := json.Marshal(buildSPVParams(
		"1",
		"http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=",
		900,
	))
	require.NoError(t, err)
	stub.completed = &task.Task{ID: "already-applied", Params: params}

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Empty(t, stub.captured)
}

func Test_OnlineSubscriber_NewRegistrationDoesNotTrustOldCompletedState(t *testing.T) {
	const uploadURL = "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename="
	params, err := json.Marshal(buildSPVParams("1", uploadURL, 900))
	require.NoError(t, err)
	stub := &stubTaskCreator{completedByKey: map[string]*task.Task{
		pmSetupCommandKey: {
			ID:     "old-device-state",
			Params: params,
		},
	}}
	s := NewOnlineSubscriber(stub, uploadURL, "1", 900, nil)
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id":     uuid.New().String(),
		"serial_number": "REG-NEW-001",
		"created":       true,
	})
	require.NoError(t, err)

	require.NoError(t, s.handleRegistered(context.Background(), evt))
	require.Len(t, stub.captured, 1)
}

func Test_OnlineSubscriber_NewRegistrationDoesNotTrustOldOpenDevice(t *testing.T) {
	const uploadURL = "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename="
	params, err := json.Marshal(buildSPVParams("1", uploadURL, 900))
	require.NoError(t, err)
	newDeviceID := uuid.New().String()
	stub := &stubTaskCreator{open: &task.Task{
		ID:         "old-device-open",
		Params:     params,
		CommandKey: pmSetupCommandKey + ":" + uuid.New().String(),
	}}
	s := NewOnlineSubscriber(stub, uploadURL, "1", 900, nil)
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id": newDeviceID, "serial_number": "REG-REUSED-SN", "created": true,
	})
	require.NoError(t, err)

	require.NoError(t, s.handleRegistered(context.Background(), evt))
	require.Len(t, stub.captured, 1)
	require.Equal(t, pmSetupCommandKey+":"+newDeviceID, stub.captured[0].CommandKey)
}

func Test_OnlineSubscriber_NewRegistrationReplayUsesDeviceIdentity(t *testing.T) {
	const uploadURL = "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename="
	deviceID := uuid.New().String()
	params, err := json.Marshal(buildSPVParams("1", uploadURL, 900))
	require.NoError(t, err)
	commandKey := pmSetupCommandKey + ":" + deviceID
	stub := &stubTaskCreator{completedByKey: map[string]*task.Task{
		commandKey: {ID: "same-registration", Params: params},
	}}
	s := NewOnlineSubscriber(stub, uploadURL, "1", 900, nil)
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]interface{}{
		"device_id": deviceID, "serial_number": "REG-REPLAY-001", "created": true,
	})
	require.NoError(t, err)

	require.NoError(t, s.handleRegistered(context.Background(), evt))
	require.Empty(t, stub.captured)
	require.Equal(t, []string{commandKey}, stub.completedKeys)
}

func Test_OnlineSubscriber_ReappliesChangedDesiredConfiguration(t *testing.T) {
	stub := &stubTaskCreator{completed: &task.Task{
		ID:     "old-config",
		Params: json.RawMessage(`{"values":[]}`),
	}}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)

	require.NoError(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)
}

func Test_OnlineSubscriber_DistributedAdmissionCoalescesConcurrentEvents(t *testing.T) {
	stub := &stubTaskCreator{}
	gate := &stubPMSetupAdmissionGate{acquired: false}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	s.SetAdmissionGate(gate)

	require.Error(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Empty(t, stub.captured)
	require.Zero(t, gate.released)
}

func Test_OnlineSubscriber_ReleasesAdmissionWhenCreateFails(t *testing.T) {
	stub := &stubTaskCreator{err: errors.New("insert failed")}
	gate := &stubPMSetupAdmissionGate{acquired: true}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	s.SetAdmissionGate(gate)

	require.Error(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	require.Equal(t, 1, gate.released)
}

func Test_OnlineSubscriber_RetriesAdmissionAndLookupFailures(t *testing.T) {
	t.Run("admission", func(t *testing.T) {
		stub := &stubTaskCreator{}
		s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
		s.SetAdmissionGate(&stubPMSetupAdmissionGate{err: errors.New("redis unavailable")})
		require.Error(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	})
	t.Run("lookup", func(t *testing.T) {
		stub := &stubTaskCreator{lookupErr: errors.New("postgres unavailable")}
		s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
		require.Error(t, s.handleOnline(context.Background(), mustEvent(t, samplePayload())))
	})
}

func TestRedisPMSetupAdmissionGate_CoalescesAndReleases(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	gate := NewRedisPMSetupAdmissionGate(client, time.Minute)

	token, acquired, err := gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.True(t, acquired)
	_, acquired, err = gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.False(t, acquired)

	require.NoError(t, gate.Release(context.Background(), "SN-001", token))
	_, acquired, err = gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestRedisPMSetupAdmissionGate_StaleOwnerCannotReleaseNewLease(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	gate := NewRedisPMSetupAdmissionGate(client, time.Minute)

	staleToken, acquired, err := gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.True(t, acquired)
	server.FastForward(2 * time.Minute)

	currentToken, acquired, err := gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, gate.Release(context.Background(), "SN-001", staleToken))

	_, acquired, err = gate.Acquire(context.Background(), "SN-001")
	require.NoError(t, err)
	require.False(t, acquired, "stale release must not delete the current owner's lease")
	require.NoError(t, gate.Release(context.Background(), "SN-001", currentToken))
}

func TestRedisPMSetupAdmissionGate_RenewPreventsLeaseExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	gate := NewRedisPMSetupAdmissionGate(client, time.Minute)

	token, acquired, err := gate.Acquire(context.Background(), "SN-RENEW")
	require.NoError(t, err)
	require.True(t, acquired)
	server.FastForward(45 * time.Second)
	renewed, err := gate.Renew(context.Background(), "SN-RENEW", token)
	require.NoError(t, err)
	require.True(t, renewed)
	server.FastForward(45 * time.Second)

	_, acquired, err = gate.Acquire(context.Background(), "SN-RENEW")
	require.NoError(t, err)
	require.False(t, acquired)
}

func Test_OnlineSubscriber_RenewsLeaseAcrossSlowLookup(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	const ttl = 60 * time.Millisecond
	gate := NewRedisPMSetupAdmissionGate(client, ttl)
	openStarted := make(chan struct{})
	openRelease := make(chan struct{})
	stub := &stubTaskCreator{
		openStarted: openStarted,
		openRelease: openRelease,
	}
	s := NewOnlineSubscriber(stub, "http://1.2.3.4:8080/smallcell/FileUploadService?fileType=PM&filename=", "1", 900, nil)
	s.SetAdmissionGate(gate)
	payload := samplePayload()
	evt := mustEvent(t, payload)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- s.handleOnline(context.Background(), evt)
	}()
	<-openStarted
	time.Sleep(3 * ttl)

	_, acquired, err := gate.Acquire(context.Background(), payload.SerialNumber)
	require.NoError(t, err)
	require.False(t, acquired,
		"a second owner must not enter after the original TTL while renewal is active")
	close(openRelease)
	require.NoError(t, <-firstDone)
	require.Len(t, stub.captured, 1)
}

func Test_expandEnv_DefaultSyntax(t *testing.T) {
	t.Setenv("MY_VAR_X1", "value-x1")
	cases := []struct {
		in   string
		want string
	}{
		{"plain text", "plain text"},
		{"${MY_VAR_X1}", "value-x1"},
		{"${UNSET_VAR_QQQ:-fallback}", "fallback"},
		{"${MY_VAR_X1:-fallback}", "value-x1"}, // env wins over default
		{"prefix-${MY_VAR_X1}-suffix", "prefix-value-x1-suffix"},
		{"${UNSET_VAR_QQQ}", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, expandEnv(c.in), "input: %s", c.in)
	}
}
