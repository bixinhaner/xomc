package pm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/paramsync"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/redis/go-redis/v9"
)

// stubTaskCreator 抓取 CreateTask 入参，断言 OnlineSubscriber 发出的 SPV 是否符合预期。
type stubTaskCreator struct {
	mu             sync.Mutex
	captured       []*task.CreateTaskRequest
	err            error
	open           *task.Task
	openTasks      []*task.Task
	completed      *task.Task
	completedByKey map[string]*task.Task
	completedKeys  []string
	lookupErr      error
	openStarted    chan struct{}
	openRelease    <-chan struct{}
	openStartOnce  sync.Once
	openLookups    []openTaskLookup
}

type openTaskLookup struct {
	deviceSN    string
	method      string
	description string
}

type recordedQueueSubscription struct {
	subject string
	queue   string
}

type recordingEventBus struct {
	queueSubscriptions []recordedQueueSubscription
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

func (b *recordingEventBus) Publish(context.Context, string, event.Event) error { return nil }

func (b *recordingEventBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return noopSubscription{}, nil
}

func (b *recordingEventBus) QueueSubscribe(
	subject string,
	queue string,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.queueSubscriptions = append(b.queueSubscriptions, recordedQueueSubscription{
		subject: subject,
		queue:   queue,
	})
	return noopSubscription{}, nil
}

func (b *recordingEventBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return noopSubscription{}, nil
}

func (b *recordingEventBus) Close() error { return nil }

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
	_ context.Context, deviceSN, method, description string,
) (*task.Task, error) {
	s.openLookups = append(s.openLookups, openTaskLookup{
		deviceSN:    deviceSN,
		method:      method,
		description: description,
	})
	if s.openStarted != nil {
		s.openStartOnce.Do(func() { close(s.openStarted) })
		<-s.openRelease
	}
	return s.open, s.lookupErr
}

func (s *stubTaskCreator) ListOpenTasksByDeviceAndMethods(
	_ context.Context,
	deviceSN string,
	methods []string,
) ([]*task.Task, error) {
	s.openLookups = append(s.openLookups, openTaskLookup{
		deviceSN: deviceSN,
		method:   strings.Join(methods, ","),
	})
	if s.openStarted != nil {
		s.openStartOnce.Do(func() { close(s.openStarted) })
		<-s.openRelease
	}
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.openTasks != nil {
		return s.openTasks, nil
	}
	if s.open != nil {
		return []*task.Task{s.open}, nil
	}
	return nil, nil
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

type stubPMDeviceLookup struct {
	devices map[uuid.UUID]*model.Device
	err     error
}

func (l stubPMDeviceLookup) GetByID(_ context.Context, deviceID uuid.UUID) (*model.Device, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.devices[deviceID], nil
}

type stubPMParameterReader struct {
	values    map[uuid.UUID]map[string]string
	updatedAt time.Time
	err       error
	reads     []string
}

func (r *stubPMParameterReader) GetByPath(
	_ context.Context,
	deviceID uuid.UUID,
	path string,
) (*model.DeviceParameter, error) {
	r.reads = append(r.reads, path)
	if r.err != nil {
		return nil, r.err
	}
	deviceValues := r.values[deviceID]
	if deviceValues == nil {
		return nil, nil
	}
	value, ok := deviceValues[path]
	if !ok {
		return nil, nil
	}
	return &model.DeviceParameter{
		DeviceID:       deviceID,
		ParameterPath:  path,
		ParameterValue: value,
		ParameterType:  "xsd:string",
		LastUpdatedAt:  r.updatedAt,
		FAPInstance:    1,
		ParamGroup:     "PerfMgmt",
	}, nil
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
	params := capturedSPVParams(t, req)
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			return v.Value
		}
	}
	t.Fatalf("missing PM upload URL SPV parameter")
	return ""
}

func capturedSPVParams(t *testing.T, req *task.CreateTaskRequest) spvParams {
	t.Helper()
	var params spvParams
	require.NoError(t, json.Unmarshal(req.Params, &params))
	return params
}

func mustParamSyncCompletedEvent(t *testing.T, deviceID uuid.UUID, status string) event.Event {
	t.Helper()
	return mustParamSyncCompletedEventForRun(t, deviceID, uuid.New(), status, string(paramsync.SyncScopeFull))
}

func mustParamSyncCompletedEventForRun(
	t *testing.T,
	deviceID uuid.UUID,
	runID uuid.UUID,
	status string,
	syncScope string,
) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectParamSyncRunCompleted, map[string]any{
		"run_id":     runID.String(),
		"device_id":  deviceID.String(),
		"status":     status,
		"sync_scope": syncScope,
	})
	require.NoError(t, err)
	return evt
}

func newPMHTTPSCompensationSubscriber(
	stub *stubTaskCreator,
	deviceID uuid.UUID,
	protocolPolicy string,
	values map[string]string,
) (*OnlineSubscriber, *stubPMParameterReader) {
	parameterReader := &stubPMParameterReader{values: map[uuid.UUID]map[string]string{
		deviceID: values,
	}, updatedAt: time.Now()}
	resolver := transfercfg.NewAddressResolver(staticPMTransferProvider{snapshot: transfercfg.Snapshot{
		ProtocolPolicy: protocolPolicy,
		Upload: transfercfg.UploadSettings{
			BaseURL:      "http://upload.example.com:8080",
			HTTPSBaseURL: "https://upload.example.com:8443",
		},
	}}, transfercfg.NewDeviceParameterHTTPSCapabilityReader(parameterReader))
	s := NewOnlineSubscriber(
		stub,
		"http://template.example.com:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1",
		900,
		nil,
	)
	s.SetUploadAddressResolver(resolver)
	s.SetParamSyncPMCompensationReaders(
		stubPMDeviceLookup{devices: map[uuid.UUID]*model.Device{
			deviceID: {ID: deviceID, SerialNumber: "SYNC-PM-001"},
		}},
		parameterReader,
	)
	return s, parameterReader
}

func Test_OnlineSubscriber_SubscribeParamSyncCompletedDoesNotSubscribeOnlineSetup(t *testing.T) {
	deviceID := uuid.New()
	stub := &stubTaskCreator{}
	s, _ := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, map[string]string{
		transfercfg.HTTPSCapabilityParameterPath: "true",
		"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
	})
	bus := &recordingEventBus{}

	require.NoError(t, s.SubscribeParamSyncCompleted(bus))

	require.Equal(t, []recordedQueueSubscription{{
		subject: event.SubjectParamSyncRunCompleted,
		queue:   "pm-param-sync-https-compensation",
	}}, bus.queueSubscriptions)
}

func Test_OnlineSubscriber_ParamSyncCompletedCompensatesPMURLToHTTPSOnly(t *testing.T) {
	deviceID := uuid.New()
	runID := uuid.New()
	stub := &stubTaskCreator{}
	s, _ := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, map[string]string{
		transfercfg.HTTPSCapabilityParameterPath: "true",
		"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
	})

	require.NoError(t, s.handleParamSyncCompleted(context.Background(), mustParamSyncCompletedEventForRun(
		t, deviceID, runID, string(paramsync.RunStatusSucceeded), string(paramsync.SyncScopeFull),
	)))

	require.Len(t, stub.captured, 1)
	req := stub.captured[0]
	assert.Equal(t, "SYNC-PM-001", req.DeviceSN)
	assert.Equal(t, string(soap.MethodSetParameterValues), req.Method)
	assert.Equal(t, task.TaskSourceSystem, req.Source)
	assert.Equal(t, "", req.CreatorID)
	assert.Equal(t, "pm_upload_https_compensation:"+deviceID.String()+":"+runID.String(), req.CommandKey)
	assert.Equal(t, "Compensate PM upload URL to HTTPS after parameter sync", req.Description)
	require.NotNil(t, req.MaxRetries)
	assert.Equal(t, req.ExpiresIn/req.RetryIntervalSeconds, *req.MaxRetries)

	params := capturedSPVParams(t, req)
	require.Len(t, params.Values, 1)
	assert.Equal(t, "Device.FAP.PerfMgmt.Config.1.URL", params.Values[0].Name)
	assert.Equal(t, "https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=", params.Values[0].Value)
	assert.Equal(t, "xsd:string", params.Values[0].Type)
}

func Test_OnlineSubscriber_ParamSyncCompletedHTTPSCompensationSkipsIneligibleDevices(t *testing.T) {
	tests := []struct {
		name           string
		policy         string
		values         map[string]string
		completedState string
	}{
		{
			name:   "force_http_policy",
			policy: transfercfg.ProtocolPolicyForceHTTP,
			values: map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "true",
				"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusSucceeded),
		},
		{
			name:   "https_capability_false",
			policy: transfercfg.ProtocolPolicyPreferHTTPS,
			values: map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "false",
				"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusSucceeded),
		},
		{
			name:   "https_capability_missing",
			policy: transfercfg.ProtocolPolicyPreferHTTPS,
			values: map[string]string{
				"Device.FAP.PerfMgmt.Config.1.URL": "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusSucceeded),
		},
		{
			name:   "https_capability_uppercase_true",
			policy: transfercfg.ProtocolPolicyPreferHTTPS,
			values: map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "TRUE",
				"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusSucceeded),
		},
		{
			name:   "current_pm_url_already_https",
			policy: transfercfg.ProtocolPolicyPreferHTTPS,
			values: map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "true",
				"Device.FAP.PerfMgmt.Config.1.URL":       "https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusSucceeded),
		},
		{
			name:   "sync_failed",
			policy: transfercfg.ProtocolPolicyPreferHTTPS,
			values: map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "true",
				"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			},
			completedState: string(paramsync.RunStatusFailed),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID := uuid.New()
			stub := &stubTaskCreator{}
			s, _ := newPMHTTPSCompensationSubscriber(stub, deviceID, tt.policy, tt.values)

			require.NoError(t, s.handleParamSyncCompleted(context.Background(), mustParamSyncCompletedEvent(t, deviceID, tt.completedState)))

			assert.Empty(t, stub.captured)
		})
	}
}

func Test_OnlineSubscriber_ParamSyncCompletedHTTPSCompensationRequiresSucceededFullSyncEvent(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
	}{
		{
			name: "missing_status",
			payload: map[string]any{
				"run_id":     uuid.New().String(),
				"device_id":  uuid.New().String(),
				"sync_scope": string(paramsync.SyncScopeFull),
			},
		},
		{
			name: "missing_sync_scope",
			payload: map[string]any{
				"run_id":    uuid.New().String(),
				"device_id": uuid.New().String(),
				"status":    string(paramsync.RunStatusSucceeded),
			},
		},
		{
			name: "partial_sync",
			payload: map[string]any{
				"run_id":     uuid.New().String(),
				"device_id":  uuid.New().String(),
				"status":     string(paramsync.RunStatusSucceeded),
				"sync_scope": string(paramsync.SyncScopePartial),
			},
		},
		{
			name: "missing_run_id",
			payload: map[string]any{
				"device_id":  uuid.New().String(),
				"status":     string(paramsync.RunStatusSucceeded),
				"sync_scope": string(paramsync.SyncScopeFull),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID := uuid.New()
			stub := &stubTaskCreator{}
			s, _ := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, map[string]string{
				transfercfg.HTTPSCapabilityParameterPath: "true",
				"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
			})
			evt, err := event.NewEvent(event.SubjectParamSyncRunCompleted, tt.payload)
			require.NoError(t, err)

			require.NoError(t, s.handleParamSyncCompleted(context.Background(), evt))

			assert.Empty(t, stub.captured)
		})
	}
}

func Test_OnlineSubscriber_ParamSyncCompletedHTTPSCompensationSkipsEquivalentOpenOrCompletedTask(t *testing.T) {
	deviceID := uuid.New()
	runID := uuid.New()
	paramUpdatedAt := time.Now()
	paramsJSON, err := json.Marshal(buildURLOnlySPVParams("https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename="))
	require.NoError(t, err)
	paramsWithExtraJSON, err := json.Marshal(spvParams{Values: []spvParam{
		{
			Name:  pmUploadURLParameterPath,
			Value: "https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=",
			Type:  "xsd:string",
		},
		{
			Name:  "Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval",
			Value: "900",
			Type:  "xsd:unsignedInt",
		},
	}})
	require.NoError(t, err)
	values := map[string]string{
		transfercfg.HTTPSCapabilityParameterPath: "true",
		"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
	}

	t.Run("open", func(t *testing.T) {
		stub := &stubTaskCreator{open: &task.Task{
			ID:          "open-equivalent-spv",
			Params:      paramsWithExtraJSON,
			CommandKey:  "manual-pm-url-correction",
			Description: "Manually requested PM URL correction",
		}}
		s, parameterReader := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, values)
		parameterReader.updatedAt = paramUpdatedAt

		require.NoError(t, s.handleParamSyncCompleted(context.Background(), mustParamSyncCompletedEventForRun(
			t, deviceID, runID, string(paramsync.RunStatusSucceeded), string(paramsync.SyncScopeFull),
		)))

		assert.Empty(t, stub.captured)
		require.Len(t, stub.openLookups, 1)
		assert.Equal(t, string(soap.MethodSetParameterValues), stub.openLookups[0].method)
	})

	t.Run("completed", func(t *testing.T) {
		completedAt := paramUpdatedAt.Add(time.Second)
		stub := &stubTaskCreator{completedByKey: map[string]*task.Task{
			"pm_upload_https_compensation:" + deviceID.String() + ":" + runID.String(): {
				ID:          "completed-compensation",
				Params:      paramsJSON,
				CreatedAt:   completedAt,
				CompletedAt: &completedAt,
			},
		}}
		s, parameterReader := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, values)
		parameterReader.updatedAt = paramUpdatedAt

		require.NoError(t, s.handleParamSyncCompleted(context.Background(), mustParamSyncCompletedEventForRun(
			t, deviceID, runID, string(paramsync.RunStatusSucceeded), string(paramsync.SyncScopeFull),
		)))

		assert.Empty(t, stub.captured)
		assert.Equal(t, []string{"pm_upload_https_compensation:" + deviceID.String() + ":" + runID.String()}, stub.completedKeys)
	})
}

func Test_OnlineSubscriber_ParamSyncCompletedHTTPSCompensationReappliesWhenCurrentHTTPIsNewerThanCompletedTask(t *testing.T) {
	deviceID := uuid.New()
	oldRunID := uuid.New()
	newRunID := uuid.New()
	oldCompletedAt := time.Now().Add(-time.Minute)
	currentHTTPUpdatedAt := time.Now()
	paramsJSON, err := json.Marshal(buildURLOnlySPVParams("https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename="))
	require.NoError(t, err)
	stub := &stubTaskCreator{completedByKey: map[string]*task.Task{
		"pm_upload_https_compensation:" + deviceID.String() + ":" + oldRunID.String(): {
			ID:          "old-completed-compensation",
			Params:      paramsJSON,
			CreatedAt:   oldCompletedAt,
			CompletedAt: &oldCompletedAt,
		},
	}}
	s, parameterReader := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, map[string]string{
		transfercfg.HTTPSCapabilityParameterPath: "true",
		"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
	})
	parameterReader.updatedAt = currentHTTPUpdatedAt

	require.NoError(t, s.handleParamSyncCompleted(context.Background(), mustParamSyncCompletedEventForRun(
		t, deviceID, newRunID, string(paramsync.RunStatusSucceeded), string(paramsync.SyncScopeFull),
	)))

	require.Len(t, stub.captured, 1)
	assert.Equal(t, "https://upload.example.com:8443/smallcell/FileUploadService?fileType=PM&filename=", capturedPMUploadURL(t, stub.captured[0]))
	assert.Equal(t, "pm_upload_https_compensation:"+deviceID.String()+":"+newRunID.String(), stub.captured[0].CommandKey)
}

func Test_OnlineSubscriber_ParamSyncCompletedHTTPSCompensationCoalescesConcurrentEvents(t *testing.T) {
	deviceID := uuid.New()
	stub := &stubTaskCreator{}
	s, _ := newPMHTTPSCompensationSubscriber(stub, deviceID, transfercfg.ProtocolPolicyPreferHTTPS, map[string]string{
		transfercfg.HTTPSCapabilityParameterPath: "true",
		"Device.FAP.PerfMgmt.Config.1.URL":       "http://upload.example.com:8080/smallcell/FileUploadService?fileType=PM&filename=",
	})
	s.SetAdmissionGate(&stubPMSetupAdmissionGate{acquired: false})

	require.NoError(t, s.handleParamSyncCompleted(
		context.Background(),
		mustParamSyncCompletedEvent(t, deviceID, string(paramsync.RunStatusSucceeded)),
	))

	assert.Empty(t, stub.captured)
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
	assert.Equal(t, string(soap.MethodSetParameterValues), req.Method)
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
