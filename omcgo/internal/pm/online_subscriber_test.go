package pm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
)

// stubTaskCreator 抓取 CreateTask 入参，断言 OnlineSubscriber 发出的 SPV 是否符合预期。
type stubTaskCreator struct {
	mu       sync.Mutex
	captured []*task.CreateTaskRequest
	err      error
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

func Test_OnlineSubscriber_EnqueuesSingleSPVWith3Params(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(
		stub,
		"http://1.2.3.4:7557/smallcell/FileUploadService?fileType=PM&filename=",
		"1", 900, nil,
	)
	require.NoError(t, s.handle(context.Background(), mustEvent(t, samplePayload())))

	require.Len(t, stub.captured, 1)
	req := stub.captured[0]
	assert.Equal(t, "BLQ-TEST-001", req.DeviceSN)
	assert.Equal(t, "SetParameterValues", req.Method)
	assert.Equal(t, task.TaskSourceSystem, req.Source)
	assert.Equal(t, "", req.CreatorID, "system task should not have CreatorID (skip notification)")
	assert.Equal(t, "pm_upload_setup_on_online", req.CommandKey)

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

	require.NoError(t, s.handle(context.Background(), mustEvent(t, samplePayload())))
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

	require.NoError(t, s.handle(context.Background(), mustEvent(t, samplePayload())))
	require.Len(t, stub.captured, 1)

	var params spvParams
	_ = json.Unmarshal(stub.captured[0].Params, &params)
	for _, v := range params.Values {
		if v.Name == "Device.FAP.PerfMgmt.Config.1.URL" {
			assert.Contains(t, v.Value, "http://172.19.1.132:7557")
		}
	}
}

func Test_OnlineSubscriber_FailureIsLoggedNotPropagated(t *testing.T) {
	stub := &stubTaskCreator{err: errors.New("enqueue failed")}
	s := NewOnlineSubscriber(stub, "http://localhost/x", "1", 900, nil)

	err := s.handle(context.Background(), mustEvent(t, samplePayload()))
	// 失败 log only，不返 error 让 EventBus 重试
	assert.NoError(t, err)
}

func Test_OnlineSubscriber_EmptySerialIsSkipped(t *testing.T) {
	stub := &stubTaskCreator{}
	s := NewOnlineSubscriber(stub, "http://localhost/x", "1", 900, nil)

	payload := samplePayload()
	payload.SerialNumber = ""
	require.NoError(t, s.handle(context.Background(), mustEvent(t, payload)))

	assert.Empty(t, stub.captured, "empty SN should be skipped without enqueue")
}

func Test_OnlineSubscriber_MalformedURLIsSkipped(t *testing.T) {
	stub := &stubTaskCreator{}
	// 模板不含 :// → 渲染后也不是合法 URL，跳过
	s := NewOnlineSubscriber(stub, "localhost/x", "1", 900, nil)
	require.NoError(t, s.handle(context.Background(), mustEvent(t, samplePayload())))
	assert.Empty(t, stub.captured)
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
		{"${MY_VAR_X1:-fallback}", "value-x1"},  // env wins over default
		{"prefix-${MY_VAR_X1}-suffix", "prefix-value-x1-suffix"},
		{"${UNSET_VAR_QQQ}", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, expandEnv(c.in), "input: %s", c.in)
	}
}
