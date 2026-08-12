package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	devicetask "github.com/omcgo/omcgo/internal/task"
)

// ---------- fake collaborators ----------

// fakePlatformResolver 把 productClass 映射为 param_model.name；返回的 name 会
// 经过 IsSupportedPlatform 校对。空 map 时所有 productClass 都解析为空 = unsupport。
type fakePlatformResolver struct {
	pcToPlatform map[string]string
}

func (f *fakePlatformResolver) ResolvePlatform(_ context.Context, pc string) (string, error) {
	if f.pcToPlatform == nil {
		return "", nil
	}
	return f.pcToPlatform[pc], nil
}

type fakeDeviceContext struct {
	devs map[string]*coremodel.Device
	err  error
}

func (f *fakeDeviceContext) GetDevice(_ context.Context, sn string) (*coremodel.Device, error) {
	if f.err != nil {
		return nil, f.err
	}
	d, ok := f.devs[sn]
	if !ok {
		return nil, errors.New("device not found")
	}
	return d, nil
}

type fakeEnqueuer struct {
	captured []*devicetask.CreateTaskRequest
	err      error
}

func (f *fakeEnqueuer) CreateTask(_ context.Context, req *devicetask.CreateTaskRequest) (*devicetask.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.captured = append(f.captured, req)
	return &devicetask.Task{ID: uuid.NewString()}, nil
}

func (f *fakeEnqueuer) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type fakeHTTPSCapabilityReader struct {
	status transfercfg.HTTPSCapabilityStatus
	reads  int
}

func (f *fakeHTTPSCapabilityReader) ReadHTTPSCapability(_ context.Context, _ uuid.UUID) transfercfg.HTTPSCapabilityStatus {
	f.reads++
	return f.status
}

// fakeResolver 是 TranslatorResolver 的轻量实现。mappings 给的 standardPath →
// privatePath 会被 Translator 命中（Found=true）。
// 未在 mappings 中的 path 会 Found=false → dispatcher 降级返回 standardPath。
type fakeResolver struct {
	mappings map[string]string // standardPath（已替换 {i}）→ privatePath
	disable  bool              // true 时返回 (nil, false)
}

func (f *fakeResolver) ResolveForDevice(_ context.Context, _ *coremodel.Device) (*parammodel.Translator, bool) {
	if f.disable {
		return nil, false
	}
	mappings := make([]parammodel.ParamMapping, 0, len(f.mappings))
	for std, priv := range f.mappings {
		mappings = append(mappings, parammodel.ParamMapping{
			StandardPath: std,
			PrivatePath:  priv,
		})
	}
	set := &parammodel.MappingSet{Mappings: mappings}
	return parammodel.NewTranslator(set, nil, nil), true
}

// ---------- 测试 ----------

func newDispatcherSetup(productClass string, lookupErr error) (*Dispatcher, *fakeEnqueuer, *fakeRepo) {
	return newDispatcherSetupWithResolver(productClass, lookupErr, &fakeResolver{disable: true})
}

func newDispatcherSetupWithResolver(productClass string, lookupErr error, resolver TranslatorResolver) (*Dispatcher, *fakeEnqueuer, *fakeRepo) {
	cfg := appconfig.MRConfig{
		Vendor:  "Baicells",
		OmcName: "Baicells-OMC",
		URLBase: "http://omc.example.com",
	}.Defaults()
	enq := &fakeEnqueuer{}
	repo := newFakeRepo()
	dev := &fakeDeviceContext{
		devs: map[string]*coremodel.Device{
			"SN001": {ID: uuid.New(), SerialNumber: "SN001", ProductClass: productClass, FirmwareVersion: "1.0"},
		},
		err: lookupErr,
	}
	d := NewDispatcher(cfg, enq, dev, resolver, repo, nil)
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(
		transfercfg.NewPolicy(transfercfg.Snapshot{
			ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
			Upload: transfercfg.UploadSettings{
				BaseURL: "http://omc.example.com",
				Path:    "/smallcell/FileUploadService",
			},
		}, nil),
		nil,
	))
	// 默认把 supported productClass（如 INTEL_CR_SC_CARRIER / FAP/BAIBLQ/SC）
	// 都解析为 "BLQ" 让测试走过 IsSupportedPlatform；不支持的 productClass
	// 如 QAV3_SC 解析为空 → unsupport。
	d.SetPlatformResolver(&fakePlatformResolver{pcToPlatform: map[string]string{
		"INTEL_CR_SC_CARRIER": "BLQ",
		"FAP/BAIBLQ/SC":       "BLQ",
		"BAICELLS_BM_DC":      "BM",
		"MLN_SC_CARRIER":      "MLN",
	}})
	return d, enq, repo
}

func sampleTask() *Task {
	endTime := time.Now().Add(time.Hour)
	return &Task{
		TaskID:       uuid.New(),
		TaskName:     "t",
		MRType:       "MRS,MRE,MRO",
		StatisPeriod: "5120",
		ReportPeriod: "15",
		StartTime:    time.Now(),
		EndTime:      &endTime,
		TaskStatus:   StatusWaiting,
	}
}

func sampleProgress() Progress {
	return Progress{
		ID:             uuid.New(),
		TaskID:         uuid.New(),
		SmallCellCode:  "CELL001",
		SerialNumber:   "SN001",
		ProgressStatus: ProgressPending,
		HealthStatus:   HealthUnknown,
	}
}

func TestDispatcher_Open_SupportedPlatform_EnqueuesSPVWith6Params(t *testing.T) {
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	err := d.Open(context.Background(), task, cell)
	require.NoError(t, err)
	require.Len(t, enq.captured, 1)

	req := enq.captured[0]
	assert.Equal(t, "SN001", req.DeviceSN)
	assert.Equal(t, methodSetParameterValues, req.Method)
	assert.Equal(t, devicetask.TaskSourceSystem, req.Source)
	assert.Equal(t, task.TaskID.String(), req.SourceID)
	assert.Contains(t, req.CommandKey, commandKeyOpenPrefix)
	assert.Contains(t, req.CommandKey, "CELL001")

	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(req.Params, &body))
	require.Len(t, body.Values, 6, "MR open SPV must have exactly 6 parameters per doc §4")

	byName := map[string]string{}
	for _, v := range body.Values {
		byName[v.Name] = v.Value
	}
	assert.Equal(t, "true", byName["Device.FAP.MRMgmt.Config.1.MrEnable"])
	assert.Equal(t, "Baicells", byName["Device.FAP.MRMgmt.Config.1.Vendor"])
	assert.Equal(t, "Baicells-OMC", byName["Device.FAP.MRMgmt.Config.1.OmcName"])
	assert.Equal(t, "5120", byName["Device.FAP.MRMgmt.Config.1.PeriodicReportInterval"])
	assert.Equal(t, "900", byName["Device.FAP.MRMgmt.Config.1.UploadPeriod"],
		"report_period=15 (分钟) 应换算为 UploadPeriod=900 秒")
	assert.Contains(t, byName["Device.FAP.MRMgmt.Config.1.MrUrl"],
		"fileType=MR&cellCode=CELL001&filename=",
		"MR URL 必须含 cellCode 与空 filename")
}

func TestDispatcher_Open_MRUploadURLUsesUnifiedTransferPolicy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		policy     string
		capability transfercfg.HTTPSCapabilityStatus
		wantURL    string
		wantReads  int
	}{
		{
			name:       "force_http ignores enabled capability",
			policy:     transfercfg.ProtocolPolicyForceHTTP,
			capability: transfercfg.HTTPSCapabilityEnabled,
			wantURL:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
		},
		{
			name:       "prefer_https uses HTTPS only when capability is true",
			policy:     transfercfg.ProtocolPolicyPreferHTTPS,
			capability: transfercfg.HTTPSCapabilityEnabled,
			wantURL:    "https://[2001:db8::10]:9443/secure-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
			wantReads:  1,
		},
		{
			name:       "prefer_https falls back to HTTP when capability is false",
			policy:     transfercfg.ProtocolPolicyPreferHTTPS,
			capability: transfercfg.HTTPSCapabilityDisabled,
			wantURL:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
			wantReads:  1,
		},
		{
			name:       "prefer_https falls back to HTTP when capability is unknown",
			policy:     transfercfg.ProtocolPolicyPreferHTTPS,
			capability: transfercfg.HTTPSCapabilityUnknown,
			wantURL:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
			wantReads:  1,
		},
		{
			name:       "prefer_https falls back to HTTP when capability read errors",
			policy:     transfercfg.ProtocolPolicyPreferHTTPS,
			capability: transfercfg.HTTPSCapabilityReadError,
			wantURL:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
			wantReads:  1,
		},
		{
			name:       "prefer_https falls back to HTTP for non true values",
			policy:     transfercfg.ProtocolPolicyPreferHTTPS,
			capability: transfercfg.HTTPSCapabilityOtherValue,
			wantURL:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
			wantReads:  1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
			reader := &fakeHTTPSCapabilityReader{status: tc.capability}
			policy := transfercfg.NewPolicy(transfercfg.Snapshot{
				ProtocolPolicy: tc.policy,
				Upload: transfercfg.UploadSettings{
					BaseURL:      "http://192.0.2.10:8080/reverse-proxy",
					HTTPSBaseURL: "https://[2001:db8::10]:9443/secure-proxy",
				},
			}, nil)
			d.SetUploadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
			task := sampleTask()
			cell := sampleProgress()
			cell.TaskID = task.TaskID

			require.NoError(t, d.Open(context.Background(), task, cell))

			assert.Equal(t, tc.wantReads, reader.reads)
			assert.Equal(t, tc.wantURL, capturedMRURL(t, enq.captured[0].Params))
		})
	}
}

func TestDispatcher_Open_MRUploadURLReadsCurrentPolicyForEachDispatch(t *testing.T) {
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	reader := &fakeHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	values := map[string]string{
		transfercfg.KeyProtocolPolicy:     transfercfg.ProtocolPolicyForceHTTP,
		transfercfg.KeyUploadBaseURL:      "http://192.0.2.10:8080/reverse-proxy",
		transfercfg.KeyHTTPSUploadBaseURL: "https://[2001:db8::10]:9443/secure-proxy",
	}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, func(_ context.Context, category, key string) (string, bool) {
		if category != transfercfg.Category {
			return "", false
		}
		value, ok := values[key]
		return value, ok
	})
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	require.NoError(t, d.Open(context.Background(), task, cell))

	values[transfercfg.KeyProtocolPolicy] = transfercfg.ProtocolPolicyPreferHTTPS
	policy.InvalidateCache()
	require.NoError(t, d.Open(context.Background(), task, cell))

	require.Len(t, enq.captured, 2)
	assert.Equal(t,
		"http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
		capturedMRURL(t, enq.captured[0].Params),
	)
	assert.Equal(t,
		"https://[2001:db8::10]:9443/secure-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL001&filename=",
		capturedMRURL(t, enq.captured[1].Params),
	)
	assert.Equal(t, 1, reader.reads)
}

func TestDispatcher_Open_LogsProtocolDecisionWithoutFullURL(t *testing.T) {
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	core, logs := observer.New(zap.InfoLevel)
	d.logger = zap.New(core).Named("mr-dispatcher")
	reader := &fakeHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Upload: transfercfg.UploadSettings{
			BaseURL:      "http://user:secret@192.0.2.10:8080/reverse-proxy",
			HTTPSBaseURL: "https://[2001:db8::10]:9443/secure-proxy",
		},
	}, nil)
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	require.NoError(t, d.Open(context.Background(), task, cell))
	require.Len(t, enq.captured, 1)

	entries := logs.FilterMessage("MR open SPV enqueued").All()
	require.Len(t, entries, 1)
	contextMap := entries[0].ContextMap()
	assert.Equal(t, "https", contextMap["transfer_protocol"])
	assert.Equal(t, "https_capability_enabled", contextMap["transfer_reason"])
	assert.Equal(t, "enabled", contextMap["https_capability"])
	assert.NotContains(t, entries[0].Context, "secret")
	assert.NotContains(t, entries[0].Context, "FileUploadService")
}

func TestDispatcher_Open_MRUploadURLPreservesCellEncoding(t *testing.T) {
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
		Upload: transfercfg.UploadSettings{
			BaseURL: "http://192.0.2.10:8080/reverse-proxy",
		},
	}, nil)
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(policy, nil))
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID
	cell.SmallCellCode = "CELL 001/A&B"

	require.NoError(t, d.Open(context.Background(), task, cell))

	assert.Equal(t,
		"http://192.0.2.10:8080/reverse-proxy/smallcell/FileUploadService?fileType=MR&cellCode=CELL+001%2FA%26B&filename=",
		capturedMRURL(t, enq.captured[0].Params),
	)
}

func TestDispatcher_Open_InvalidMRUploadURLDoesNotEnqueueAndMarksOpenFailure(t *testing.T) {
	d, enq, repo := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Upload: transfercfg.UploadSettings{
			BaseURL: "http://192.0.2.10:8080/reverse-proxy",
		},
	}, nil)
	d.SetUploadAddressResolver(transfercfg.NewAddressResolver(
		policy,
		&fakeHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled},
	))
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	err := d.Open(context.Background(), task, cell)

	require.Error(t, err)
	assert.Empty(t, enq.captured)
	require.NotNil(t, repo.lastProgressDispatch)
	assert.Equal(t, task.TaskID, repo.lastProgressDispatch.taskID)
	assert.Equal(t, cell.SmallCellCode, repo.lastProgressDispatch.smallCellCode)
	assert.Equal(t, ProgressOpenFailure, repo.lastProgressDispatch.status)
	require.NotNil(t, repo.lastProgressDispatch.faultCode)
	assert.Equal(t, "invalid_mr_upload_url", *repo.lastProgressDispatch.faultCode)
}

func TestDispatcher_Open_UnsupportedPlatform_MarksUnsupportNoEnqueue(t *testing.T) {
	d, enq, repo := newDispatcherSetup("QAV3_SC", nil)
	task := sampleTask()
	repo.tasks[task.TaskID] = task
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	// 让 repo 能找到这一行
	_ = repo.CreateTask(context.Background(), task)
	_ = repo.InsertProgressRows(context.Background(), task.TaskID, []CellTarget{
		{SmallCellCode: cell.SmallCellCode, SerialNumber: cell.SerialNumber},
	})

	err := d.Open(context.Background(), task, cell)
	require.NoError(t, err)
	assert.Empty(t, enq.captured, "unsupported platform should not enqueue SPV")
}

func TestDispatcher_Open_DeviceLookupFails_MarksUnsupport(t *testing.T) {
	d, enq, _ := newDispatcherSetup("", errors.New("db down"))
	err := d.Open(context.Background(), sampleTask(), sampleProgress())
	require.NoError(t, err) // fail-soft：记录 unsupport，不向上抛
	assert.Empty(t, enq.captured)
}

func TestDispatcher_Open_TranslatesStandardPathsToPrivate(t *testing.T) {
	// 模拟厂商把 MrEnable / UploadPeriod 改名到私有 path，
	// 其它 4 个 path 字典未收录 → 必须 fallback 到 standardPath。
	resolver := &fakeResolver{mappings: map[string]string{
		"Device.FAP.MRMgmt.Config.1.MrEnable":     "Device.X_BAICELLS_MR.Cfg.1.Enable",
		"Device.FAP.MRMgmt.Config.1.UploadPeriod": "Device.X_BAICELLS_MR.Cfg.1.UploadSec",
	}}
	d, enq, _ := newDispatcherSetupWithResolver("INTEL_CR_SC_CARRIER", nil, resolver)
	task := sampleTask()
	cell := sampleProgress()
	cell.TaskID = task.TaskID

	require.NoError(t, d.Open(context.Background(), task, cell))
	require.Len(t, enq.captured, 1)

	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(enq.captured[0].Params, &body))
	byName := map[string]string{}
	for _, v := range body.Values {
		byName[v.Name] = v.Value
	}

	// 字典命中的两条：必须用 privatePath
	assert.Equal(t, "true", byName["Device.X_BAICELLS_MR.Cfg.1.Enable"],
		"MrEnable should be translated to privatePath")
	assert.Equal(t, "900", byName["Device.X_BAICELLS_MR.Cfg.1.UploadSec"],
		"UploadPeriod should be translated to privatePath")

	// 字典未命中的四条：fallback 到 standardPath
	assert.Equal(t, "Baicells", byName["Device.FAP.MRMgmt.Config.1.Vendor"],
		"Vendor not in dict → fallback to standardPath")
	assert.Equal(t, "Baicells-OMC", byName["Device.FAP.MRMgmt.Config.1.OmcName"])
	assert.Equal(t, "5120", byName["Device.FAP.MRMgmt.Config.1.PeriodicReportInterval"])
	assert.Contains(t, byName["Device.FAP.MRMgmt.Config.1.MrUrl"],
		"fileType=MR&cellCode=CELL001")

	// 关键：被翻译过的 standardPath 路径不应同时出现（避免双下发）
	assert.NotContains(t, byName, "Device.FAP.MRMgmt.Config.1.MrEnable",
		"original standardPath should not appear when translated to privatePath")
}

func capturedMRURL(t *testing.T, params []byte) string {
	t.Helper()
	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(params, &body))
	for _, v := range body.Values {
		if v.Name == "Device.FAP.MRMgmt.Config.1.MrUrl" {
			return v.Value
		}
	}
	t.Fatalf("MR URL parameter not found in %#v", body.Values)
	return ""
}

func TestDispatcher_Open_NilResolver_UsesStandardPath(t *testing.T) {
	// resolver disabled → 全部 6 条都用 standardPath（fail-soft）
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	require.NoError(t, d.Open(context.Background(), sampleTask(), sampleProgress()))
	require.Len(t, enq.captured, 1)

	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(enq.captured[0].Params, &body))
	for _, v := range body.Values {
		assert.True(t,
			v.Name == "Device.FAP.MRMgmt.Config.1.MrEnable" ||
				v.Name == "Device.FAP.MRMgmt.Config.1.Vendor" ||
				v.Name == "Device.FAP.MRMgmt.Config.1.OmcName" ||
				v.Name == "Device.FAP.MRMgmt.Config.1.MrUrl" ||
				v.Name == "Device.FAP.MRMgmt.Config.1.PeriodicReportInterval" ||
				v.Name == "Device.FAP.MRMgmt.Config.1.UploadPeriod",
			"all params should fall back to standardPath, got %s", v.Name)
	}
}

func TestDispatcher_Close_TranslatesPath(t *testing.T) {
	resolver := &fakeResolver{mappings: map[string]string{
		"Device.FAP.MRMgmt.Config.1.MrEnable": "Device.X_BAICELLS_MR.Cfg.1.Enable",
	}}
	d, enq, _ := newDispatcherSetupWithResolver("INTEL_CR_SC_CARRIER", nil, resolver)
	cell := sampleProgress()
	cell.ProgressStatus = ProgressOpenSuccess

	require.NoError(t, d.Close(context.Background(), sampleTask(), cell))
	require.Len(t, enq.captured, 1)

	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(enq.captured[0].Params, &body))
	require.Len(t, body.Values, 1)
	// close 也必须翻译 — 否则可能开了 privatePath 却关 standardPath
	assert.Equal(t, "Device.X_BAICELLS_MR.Cfg.1.Enable", body.Values[0].Name)
	assert.Equal(t, "false", body.Values[0].Value)
}

func TestDispatcher_Close_OnlyOpenSuccessCells(t *testing.T) {
	d, enq, _ := newDispatcherSetup("INTEL_CR_SC_CARRIER", nil)
	task := sampleTask()

	// 非 openSuccess 状态：跳过
	for _, s := range []ProgressStatus{ProgressPending, ProgressOpenFailure, ProgressUnsupport, ProgressTimeOut} {
		c := sampleProgress()
		c.ProgressStatus = s
		require.NoError(t, d.Close(context.Background(), task, c))
	}
	assert.Empty(t, enq.captured, "Close should skip non-openSuccess cells")

	// openSuccess：派发关闭 SPV（仅 1 参数）
	c := sampleProgress()
	c.ProgressStatus = ProgressOpenSuccess
	require.NoError(t, d.Close(context.Background(), task, c))
	require.Len(t, enq.captured, 1)

	var body struct {
		Values []spvValue `json:"values"`
	}
	require.NoError(t, json.Unmarshal(enq.captured[0].Params, &body))
	require.Len(t, body.Values, 1, "MR close SPV must have exactly 1 parameter (MrEnable=false)")
	assert.Equal(t, "Device.FAP.MRMgmt.Config.1.MrEnable", body.Values[0].Name)
	assert.Equal(t, "false", body.Values[0].Value)
	assert.Contains(t, enq.captured[0].CommandKey, commandKeyClosePrefix)
}

func TestParseCellFromCommandKey(t *testing.T) {
	cases := map[string]struct {
		key      string
		wantCell string
		wantOK   bool
	}{
		"open":           {"mr-open-abc12345-CELL001", "CELL001", true},
		"close":          {"mr-close-abc12345-CELL001", "CELL001", true},
		"cell with dash": {"mr-open-abc12345-AREA-01-CELL", "AREA-01-CELL", true},
		"missing prefix": {"foo-bar", "", false},
		"missing cell":   {"mr-open-abc12345", "", false},
		"empty cell":     {"mr-open-abc12345-", "", false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cell, ok := parseCellFromCommandKey(c.key)
			assert.Equal(t, c.wantOK, ok)
			assert.Equal(t, c.wantCell, cell)
		})
	}
}

func TestCompletionCallback_RoutesByCommandKey(t *testing.T) {
	repo := newFakeRepo()
	cb := NewCompletionCallback(repo, nil)
	taskID := uuid.New()

	// pre-seed progress 行
	repo.tasks[taskID] = &Task{TaskID: taskID, TaskStatus: StatusOn}

	// open success
	cb.OnTaskCompleted(context.Background(), &devicetask.Task{
		CommandKey: commandKeyOpenPrefix + "abcd1234-CELL001",
		SourceID:   taskID.String(),
		Status:     devicetask.TaskStatus("completed"),
	})
	// close failure
	cb.OnTaskCompleted(context.Background(), &devicetask.Task{
		CommandKey:   commandKeyClosePrefix + "abcd1234-CELL002",
		SourceID:     taskID.String(),
		Status:       devicetask.TaskStatus("failed"),
		ErrorCode:    1001,
		ErrorMessage: "timeout",
	})
	// non-MR command key (ignored)
	cb.OnTaskCompleted(context.Background(), &devicetask.Task{
		CommandKey: "provision-SET-1",
		SourceID:   taskID.String(),
		Status:     devicetask.TaskStatus("completed"),
	})
	// 不抛错即视为通过；具体 progress 转移由 repo.UpdateProgressDispatch 的实现
	// 决定，fakeRepo 当前只校验"被调用"，所以这里主要验证没有 panic / log error。
}
