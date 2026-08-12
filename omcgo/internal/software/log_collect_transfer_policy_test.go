package software

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type logCollectCapabilityReader struct {
	seenPath string
	values   map[uuid.UUID]string
}

func (r *logCollectCapabilityReader) GetByPath(
	_ context.Context,
	deviceID uuid.UUID,
	path string,
) (*model.DeviceParameter, error) {
	r.seenPath = path
	value, ok := r.values[deviceID]
	if !ok {
		return nil, nil
	}
	return &model.DeviceParameter{
		DeviceID:       deviceID,
		ParameterPath:  path,
		ParameterValue: value,
	}, nil
}

type mutableUploadAddressResolver struct {
	baseURL  string
	protocol transfercfg.TransferProtocol
	reason   transfercfg.AddressDecisionReason
}

func (r *mutableUploadAddressResolver) Resolve(
	_ context.Context,
	_ uuid.UUID,
	direction transfercfg.TransferDirection,
) (transfercfg.AddressDecision, error) {
	return transfercfg.AddressDecision{
		Direction:  direction,
		Protocol:   r.protocol,
		BaseURL:    r.baseURL,
		Capability: transfercfg.HTTPSCapabilityNotRead,
		Reason:     r.reason,
	}, nil
}

func newLogCollectExecutorForTest(
	t *testing.T,
	dev *model.Device,
	onCreate func(*devtask.CreateTaskRequest),
	onStatus func(UpgradeState, string),
) (*UpgradeExecutor, *UpgradeSubTask) {
	t.Helper()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	taskID := uuid.New()
	subTask := &UpgradeSubTask{
		ID:       uuid.New(),
		TaskID:   taskID,
		DeviceID: dev.ID,
		Status:   UpgradePending,
	}
	executor := &UpgradeExecutor{
		taskRepo: &svcMockTaskRepo{},
		subTaskRepo: &svcMockSubTaskRepo{
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, msg string) error {
				subTask.Status = status
				if onStatus != nil {
					onStatus(status, msg)
				}
				return nil
			},
		},
		deviceRepo: &svcMockDeviceRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				if id == dev.ID {
					return dev, nil
				}
				return nil, nil
			},
		},
		cmdQueue: &svcMockCmdQueue{
			createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
				if onCreate != nil {
					onCreate(req)
				}
				return devtask.NewTask(req), nil
			},
		},
		redis:  redisClient,
		logger: zap.NewNop(),
	}
	return executor, subTask
}

func TestExecuteOneUpload_RuntimeLogUsesTransferPolicyHTTPSIPv6(t *testing.T) {
	ctx := context.Background()
	dev := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN RUNTIME+LOG",
		Status:       model.DeviceActive,
	}
	var got *devtask.CreateTaskRequest
	executor, subTask := newLogCollectExecutorForTest(t, dev, func(req *devtask.CreateTaskRequest) {
		got = req
	}, nil)
	reader := &logCollectCapabilityReader{values: map[uuid.UUID]string{
		dev.ID: "true",
	}}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Upload: transfercfg.UploadSettings{
			BaseURL:      "http://upload.example.com:8080",
			HTTPSBaseURL: "https://[2001:db8::10]:8443/omc",
		},
	}, nil)
	executor.SetUploadAddressResolver(transfercfg.NewAddressResolver(
		policy,
		transfercfg.NewDeviceParameterHTTPSCapabilityReader(reader),
	))

	executor.ExecuteOneUpload(
		ctx,
		subTask,
		"Custom Runtime Log File",
		"runtime-{task_id8}-{sn}.tar.gz",
		"/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=",
	)

	require.NotNil(t, got)
	assert.Equal(t, "Upload", got.Method)
	assert.Equal(t, "Collect LOG,"+subTask.ID.String()[:13], got.CommandKey)
	assert.Equal(t, transfercfg.HTTPSCapabilityParameterPath, reader.seenPath)

	var params struct {
		CommandKey     string `json:"command_key"`
		FileType       string `json:"file_type"`
		URL            string `json:"url"`
		TargetFileName string `json:"target_file_name"`
	}
	require.NoError(t, json.Unmarshal(got.Params, &params))
	assert.Equal(t, got.CommandKey, params.CommandKey)
	assert.Equal(t, "Custom Runtime Log File", params.FileType)
	assert.Equal(t, "runtime-"+taskIDPrefix(subTask.TaskID)+"-SN RUNTIME+LOG.tar.gz", params.TargetFileName)
	assert.Equal(t,
		"https://[2001:db8::10]:8443/omc/smallcell/FileUploadService?fileType=LOG&sn=SN+RUNTIME%2BLOG&taskId="+strings.ReplaceAll(subTask.TaskID.String(), "-", "")+"&filename=",
		params.URL,
	)
}

func TestExecuteOneSetParamCollect_FaultLogForceHTTPUsesTransferPolicy(t *testing.T) {
	ctx := context.Background()
	dev := &model.Device{
		ID:              uuid.New(),
		SerialNumber:    "SN-FAULT-LOG",
		Status:          model.DeviceActive,
		ProductClass:    "FAP/BAIBLQ/SC",
		FirmwareVersion: "1.0",
	}
	var got *devtask.CreateTaskRequest
	executor, subTask := newLogCollectExecutorForTest(t, dev, func(req *devtask.CreateTaskRequest) {
		got = req
	}, nil)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
		Upload: transfercfg.UploadSettings{
			BaseURL:      "http://logs.example.com:8080/base",
			HTTPSBaseURL: "https://logs.example.com:8443/base",
		},
	}, nil)
	executor.SetUploadAddressResolver(transfercfg.NewAddressResolver(policy, nil))

	executor.ExecuteOneSetParamCollect(
		ctx,
		subTask,
		"Device.DeviceInfo.FaultLogURL",
		"/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=",
	)

	require.NotNil(t, got)
	assert.Equal(t, "SetParameterValues", got.Method)
	assert.Equal(t, subTask.ID.String(), got.CommandKey)
	var params struct {
		CommandKey string `json:"command_key"`
		Values     []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(got.Params, &params))
	require.Len(t, params.Values, 1)
	assert.Equal(t, "Device.DeviceInfo.FaultLogURL", params.Values[0].Name)
	assert.Equal(t, "xsd:string", params.Values[0].Type)
	assert.Equal(t,
		"http://logs.example.com:8080/base/smallcell/FileUploadService?fileType=RL&id="+subTask.TaskID.String()+"&sn=SN-FAULT-LOG&fileName=",
		params.Values[0].Value,
	)
}

func TestExecuteOneUpload_RuntimeLogResumeUsesCurrentTransferDecision(t *testing.T) {
	ctx := context.Background()
	dev := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN-RESUME-LOG",
		Status:       model.DeviceOffline,
	}
	var got *devtask.CreateTaskRequest
	var statuses []UpgradeState
	executor, subTask := newLogCollectExecutorForTest(t, dev, func(req *devtask.CreateTaskRequest) {
		got = req
	}, func(status UpgradeState, _ string) {
		statuses = append(statuses, status)
	})
	resolver := &mutableUploadAddressResolver{
		baseURL:  "http://old.example.com:8080",
		protocol: transfercfg.TransferProtocolHTTP,
		reason:   transfercfg.AddressReasonForceHTTP,
	}
	executor.SetUploadAddressResolver(resolver)

	executor.ExecuteOneUpload(
		ctx,
		subTask,
		"4 Vendor Log File 1,2,3,4",
		"runtime-{task_id8}-{sn}.tar.gz",
		"/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=",
	)
	require.Nil(t, got)
	require.Contains(t, statuses, UpgradeSuspended)

	dev.Status = model.DeviceActive
	resolver.baseURL = "https://current.example.com:8443"
	resolver.protocol = transfercfg.TransferProtocolHTTPS
	resolver.reason = transfercfg.AddressReasonHTTPSCapabilityEnabled
	subTask.Status = UpgradeSuspended
	time.Sleep(time.Millisecond)

	executor.ExecuteOneUpload(
		ctx,
		subTask,
		"4 Vendor Log File 1,2,3,4",
		"runtime-{task_id8}-{sn}.tar.gz",
		"/smallcell/FileUploadService?fileType=LOG&sn={sn}&taskId={taskId32}&filename=",
	)

	require.NotNil(t, got)
	var params struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(got.Params, &params))
	assert.True(t, strings.HasPrefix(params.URL, "https://current.example.com:8443/"),
		"resumed dispatch must use the resolver decision from actual dispatch time")
}

func TestExecuteOneSetParamCollect_FaultLogResumeUsesCurrentTransferDecision(t *testing.T) {
	ctx := context.Background()
	dev := &model.Device{
		ID:              uuid.New(),
		SerialNumber:    "SN FAULT+RESUME",
		Status:          model.DeviceOffline,
		ProductClass:    "FAP/BAIBLQ/SC",
		FirmwareVersion: "1.0",
	}
	var got *devtask.CreateTaskRequest
	var statuses []UpgradeState
	executor, subTask := newLogCollectExecutorForTest(t, dev, func(req *devtask.CreateTaskRequest) {
		got = req
	}, func(status UpgradeState, _ string) {
		statuses = append(statuses, status)
	})
	resolver := &mutableUploadAddressResolver{
		baseURL:  "http://old-fault.example.com:8080",
		protocol: transfercfg.TransferProtocolHTTP,
		reason:   transfercfg.AddressReasonForceHTTP,
	}
	executor.SetUploadAddressResolver(resolver)

	executor.ExecuteOneSetParamCollect(
		ctx,
		subTask,
		"Device.DeviceInfo.FaultLogURL",
		"/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=",
	)
	require.Nil(t, got)
	require.Contains(t, statuses, UpgradeSuspended)

	dev.Status = model.DeviceActive
	resolver.baseURL = "https://current-fault.example.com:8443/base"
	resolver.protocol = transfercfg.TransferProtocolHTTPS
	resolver.reason = transfercfg.AddressReasonHTTPSCapabilityEnabled
	subTask.Status = UpgradeSuspended

	executor.ExecuteOneSetParamCollect(
		ctx,
		subTask,
		"Device.DeviceInfo.FaultLogURL",
		"/smallcell/FileUploadService?fileType=RL&id={id}&sn={sn}&fileName=",
	)

	require.NotNil(t, got)
	var params struct {
		Values []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(got.Params, &params))
	require.Len(t, params.Values, 1)
	assert.Equal(t,
		"https://current-fault.example.com:8443/base/smallcell/FileUploadService?fileType=RL&id="+subTask.TaskID.String()+"&sn=SN+FAULT%2BRESUME&fileName=",
		params.Values[0].Value,
	)
}
