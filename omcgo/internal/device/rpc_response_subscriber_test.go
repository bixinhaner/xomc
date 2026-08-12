package device

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

type rpcRespTestMatcher struct {
	result *product.MatchResult
	err    error
}

func (m rpcRespTestMatcher) MatchProductClass(context.Context, string) (*product.MatchResult, error) {
	return m.result, m.err
}

type rpcRespTestTranslatorFactory struct{}

func (rpcRespTestTranslatorFactory) Translator(context.Context, uuid.UUID, string) (*parammodel.Translator, error) {
	return nil, nil
}

func TestNewRPCResponseSubscriberKeepsGPVConsumerConfig(t *testing.T) {
	config := appconfig.GPVResponseConsumerConfig{
		RPCDurable:       "handoff-consumer",
		RPCStartSequence: 416825,
		RPCConcurrency:   3,
	}
	s := NewRPCResponseSubscriber(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		config,
		zap.NewNop(),
	)

	require.Equal(t, config, s.gpvConsumer)
}

type rpcRespRecordingTranslatorFactory struct {
	productID uuid.UUID
	tr        *parammodel.Translator
}

func (f *rpcRespRecordingTranslatorFactory) Translator(
	_ context.Context,
	productID uuid.UUID,
	_ string,
) (*parammodel.Translator, error) {
	f.productID = productID
	return f.tr, nil
}

func TestRPCResponseSubscriberResolveTranslatorUsesProductID(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	tr := parammodel.NewTranslator(&parammodel.MappingSet{}, nil, zap.NewNop())
	factory := &rpcRespRecordingTranslatorFactory{tr: tr}
	s := &RPCResponseSubscriber{
		productMatcher: rpcRespTestMatcher{result: &product.MatchResult{Product: &product.Product{
			ID:           productID,
			ParamModelID: &paramModelID,
		}}},
		translatorFactory: factory,
		logger:            zap.NewNop(),
	}

	got := s.resolveTranslator(context.Background(), &model.Device{
		SerialNumber:    "SN-220",
		ProductClass:    "X-MLN",
		FirmwareVersion: "1.0.0",
	})

	require.Same(t, tr, got)
	require.Equal(t, productID, factory.productID)
}

type rpcRespDeviceLookupStub struct {
	device *model.Device
	err    error
}

func (s rpcRespDeviceLookupStub) GetBySerialNumber(context.Context, string) (*model.Device, error) {
	return s.device, s.err
}

type rpcRespSubscriptionStub struct{}

func (rpcRespSubscriptionStub) Unsubscribe() error { return nil }

type rpcRespBusCall struct {
	subject string
	queue   string
}

type rpcRespKeyedCall struct {
	subject string
	config  event.KeyedQueueConfig
}

type rpcRespRecordingBus struct {
	subscribeCalls []rpcRespBusCall
	queueCalls     []rpcRespBusCall
	keyedCalls     []rpcRespKeyedCall
}

func (b *rpcRespRecordingBus) Publish(context.Context, string, event.Event) error { return nil }

func (b *rpcRespRecordingBus) Subscribe(
	subject string,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.subscribeCalls = append(b.subscribeCalls, rpcRespBusCall{subject: subject})
	return rpcRespSubscriptionStub{}, nil
}

func (b *rpcRespRecordingBus) QueueSubscribe(
	subject, queue string,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.queueCalls = append(b.queueCalls, rpcRespBusCall{subject: subject, queue: queue})
	return rpcRespSubscriptionStub{}, nil
}

func (b *rpcRespRecordingBus) KeyedQueueSubscribe(
	subject string,
	config event.KeyedQueueConfig,
	_ event.EventKeyFunc,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.keyedCalls = append(b.keyedCalls, rpcRespKeyedCall{subject: subject, config: config})
	return rpcRespSubscriptionStub{}, nil
}

func (b *rpcRespRecordingBus) PullSubscribe(
	string,
	string,
	event.EventHandler,
) (event.Subscription, error) {
	return rpcRespSubscriptionStub{}, nil
}

func (b *rpcRespRecordingBus) Close() error { return nil }

func TestRPCResponseSubscriberStartUsesConfiguredLosslessKeyedGPVConsumer(t *testing.T) {
	bus := &rpcRespRecordingBus{}
	s := &RPCResponseSubscriber{bus: bus, logger: zap.NewNop()}
	s.gpvConsumer = appconfig.GPVResponseConsumerConfig{
		RPCDurable:       "existing-rpc-consumer",
		RPCStartSequence: 416825,
		RPCConcurrency:   2,
		RPCQueueDepth:    1000,
		AckWait:          30 * time.Second,
		MaxDeliver:       5,
		MaxAckPending:    2000,
	}

	require.NoError(t, s.Start())
	require.Empty(t, bus.queueCalls)
	require.Equal(t, []rpcRespKeyedCall{{
		subject: event.SubjectCommandGetParamsResponse,
		config: event.KeyedQueueConfig{
			Durable:       "existing-rpc-consumer",
			StartSequence: 416825,
			Concurrency:   2,
			QueueDepth:    1000,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			MaxAckPending: 2000,
		},
	}}, bus.keyedCalls)
	require.Equal(t, []rpcRespBusCall{
		{subject: event.SubjectCommandDeleteObjectResponse},
		{subject: event.SubjectTaskFailed},
	}, bus.subscribeCalls)
}

type rpcRespTrackingParamRepo struct {
	stubDeviceParamRepo
	rows []model.DeviceParameter
}

func (r *rpcRespTrackingParamRepo) BatchUpsert(
	_ context.Context,
	_ uuid.UUID,
	rows []model.DeviceParameter,
) error {
	r.rows = append(r.rows, rows...)
	return nil
}

type rpcRespTrackingInfoRefresher struct {
	calls int
	err   error
}

func (r *rpcRespTrackingInfoRefresher) SyncFromParameters(
	context.Context,
	uuid.UUID,
	model.CarrierCode,
	model.Technology,
	string,
) ([]string, error) {
	r.calls++
	return nil, r.err
}

func TestRPCResponseSubscriber_UECountResponsePersistsStandardPathsAndRefreshesInfo(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	deviceID := uuid.New()
	tr := parammodel.NewTranslator(&parammodel.MappingSet{
		Mappings: []parammodel.ParamMapping{
			{
				StandardPath: "Device.DeviceInfo.UE_Count",
				PrivatePath:  "Device.DeviceInfo.X_COM_UE_Count",
			},
			{
				StandardPath: "Device.DeviceInfo.2.UE_Count",
				PrivatePath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.Status.LteUECount",
			},
		},
	}, nil, zap.NewNop())
	paramRepo := &rpcRespTrackingParamRepo{}
	infoRefresher := &rpcRespTrackingInfoRefresher{}
	s := &RPCResponseSubscriber{
		productMatcher: rpcRespTestMatcher{result: &product.MatchResult{Product: &product.Product{
			ID:           productID,
			ParamModelID: &paramModelID,
		}}},
		translatorFactory: &rpcRespRecordingTranslatorFactory{tr: tr},
		deviceLookup: rpcRespDeviceLookupStub{device: &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-220",
			ProductClass: "X-MLN",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
		}},
		paramRepo:     paramRepo,
		infoRefresher: infoRefresher,
		logger:        zap.NewNop(),
	}
	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn": "SN-220",
		"method":    "GetParameterValuesResponse",
		"parameter_values": []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.X_COM_UE_Count", Value: "2"},
			{Name: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Status.LteUECount", Value: "3"},
		},
	})
	require.NoError(t, err)

	require.NoError(t, s.handleGPVResponse(context.Background(), evt))
	require.Len(t, paramRepo.rows, 2)
	require.Equal(t, "Device.DeviceInfo.UE_Count", paramRepo.rows[0].ParameterPath)
	require.Equal(t, "Device.DeviceInfo.2.UE_Count", paramRepo.rows[1].ParameterPath)
	require.Equal(t, 1, infoRefresher.calls)
}

func TestRPCResponseSubscriber_GeofenceRFReadbackPersistsCanonicalValueAndRequiresSummaryRefresh(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	deviceID := uuid.New()
	privatePath := "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable"
	standardPath := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	tr := parammodel.NewTranslator(&parammodel.MappingSet{
		Mappings: []parammodel.ParamMapping{{
			StandardPath: standardPath,
			PrivatePath:  privatePath,
		}},
	}, nil, zap.NewNop())
	paramRepo := &rpcRespTrackingParamRepo{}
	refreshErr := errors.New("device_info update unavailable")
	infoRefresher := &rpcRespTrackingInfoRefresher{err: refreshErr}
	s := &RPCResponseSubscriber{
		productMatcher: rpcRespTestMatcher{result: &product.MatchResult{Product: &product.Product{
			ID: productID, ParamModelID: &paramModelID,
		}}},
		translatorFactory: &rpcRespRecordingTranslatorFactory{tr: tr},
		deviceLookup: rpcRespDeviceLookupStub{device: &model.Device{
			ID: deviceID, SerialNumber: "SN-GEOFENCE-RF", ProductClass: "FAP/BAIBLQ/SC",
			Carrier: model.CarrierCMCC, Technology: model.TechLTE,
		}},
		paramRepo: paramRepo, infoRefresher: infoRefresher, logger: zap.NewNop(),
	}
	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn":  "SN-GEOFENCE-RF",
		"method":     "GetParameterValuesResponse",
		"command_key": "geofence:device:12:deactivate:verify:0",
		"parameter_values": []tr069.ParameterValueStruct{{Name: privatePath, Value: "false"}},
	})
	require.NoError(t, err)

	err = s.handleGPVResponse(context.Background(), evt)

	require.ErrorIs(t, err, refreshErr)
	require.Len(t, paramRepo.rows, 1)
	require.Equal(t, standardPath, paramRepo.rows[0].ParameterPath)
	require.Equal(t, "false", paramRepo.rows[0].ParameterValue)
	require.False(t, paramRepo.rows[0].LastUpdatedAt.IsZero())
	require.Equal(t, 1, infoRefresher.calls)
}

func TestRPCResponseSubscriber_DeviceLookupFailureIsRetried(t *testing.T) {
	lookupErr := errors.New("database unavailable")
	s := &RPCResponseSubscriber{
		deviceLookup: rpcRespDeviceLookupStub{err: lookupErr},
		logger:       zap.NewNop(),
	}
	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn": "SN-DB-ERROR",
		"parameter_values": []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.X_COM_UE_Count", Value: "2"},
		},
	})
	require.NoError(t, err)

	require.ErrorIs(t, s.handleGPVResponse(context.Background(), evt), lookupErr)
}

func TestRPCResponseSubscriber_MissingDeviceIsTerminal(t *testing.T) {
	s := &RPCResponseSubscriber{
		deviceLookup: rpcRespDeviceLookupStub{},
		logger:       zap.NewNop(),
	}
	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn": "SN-NOT-FOUND",
		"parameter_values": []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.X_COM_UE_Count", Value: "2"},
		},
	})
	require.NoError(t, err)

	require.NoError(t, s.handleGPVResponse(context.Background(), evt))
}

func TestDecodeParameterValues_NullPayloadIsAnEmptyResult(t *testing.T) {
	values, err := decodeParameterValues(nil)

	require.NoError(t, err)
	require.Empty(t, values)
}

func TestDecodeParameterValues_RejectsMalformedScalar(t *testing.T) {
	_, err := decodeParameterValues("not-an-array")

	require.ErrorContains(t, err, "unexpected type string")
}

func TestRPCResponseSubscriberResolveTranslator_OrphanFallbackIsWarnNotError(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	s := &RPCResponseSubscriber{
		productMatcher:    rpcRespTestMatcher{err: errors.New("productClass matched no pattern (orphan device)")},
		translatorFactory: rpcRespTestTranslatorFactory{},
		logger:            zap.New(core).Named("device-rpc-resp-sub"),
	}

	translator := s.resolveTranslator(context.Background(), &model.Device{
		SerialNumber: "SN-ORPHAN",
		ProductClass: "FAP/pCRB2000/SC",
	})

	require.Nil(t, translator)
	require.Equal(t, 0, logs.FilterLevelExact(zapcore.ErrorLevel).Len())
	require.Equal(t, 1, logs.FilterMessage("uplink path translation fallback: product/param_model unresolved").FilterLevelExact(zapcore.WarnLevel).Len())
}

func TestRPCResponseSubscriber_PasswordResetFailureQueuesCommonFallback(t *testing.T) {
	enq := &stubSuccessTaskSvc{taskID: "fallback-task-1"}
	s := &RPCResponseSubscriber{
		taskEnqueuer: enq,
		logger:       zap.NewNop(),
	}
	failed := &task.Task{
		ID:           "primary-task-1",
		DeviceSN:     "CELL1123",
		Method:       resetLMTPasswordMethod,
		CommandKey:   "reset-lmt-pwd-abcd1234",
		Status:       task.TaskStatusFailed,
		Priority:     5,
		Source:       task.TaskSourceAPI,
		CreatorID:    "operator-1",
		ErrorMessage: "device fault",
	}
	evt, err := event.NewEvent(event.SubjectTaskFailed, failed)
	require.NoError(t, err)

	require.NoError(t, s.handleSyncTaskFailed(context.Background(), evt))

	require.NotNil(t, enq.lastReq)
	require.Equal(t, "CELL1123", enq.lastReq.DeviceSN)
	require.Equal(t, resetLMTPasswordFallbackMethod, enq.lastReq.Method)
	require.Equal(t, task.TaskSourceAPI, enq.lastReq.Source)
	require.Equal(t, "operator-1", enq.lastReq.CreatorID)
	require.Contains(t, enq.lastReq.CommandKey, resetLMTPasswordFallbackPrefix)

	var params map[string]string
	require.NoError(t, json.Unmarshal(enq.lastReq.Params, &params))
	require.Equal(t, resetLMTPasswordFallbackMethod, params["message_type"])
	require.Equal(t, "primary-task-1", params["fallback_of"])
}

func TestRPCResponseSubscriber_PasswordResetFallbackFailureDoesNotLoop(t *testing.T) {
	enq := &stubSuccessTaskSvc{taskID: "unexpected-task"}
	s := &RPCResponseSubscriber{
		taskEnqueuer: enq,
		logger:       zap.NewNop(),
	}
	failed := &task.Task{
		ID:         "fallback-task-1",
		DeviceSN:   "CELL1123",
		Method:     resetLMTPasswordFallbackMethod,
		CommandKey: resetLMTPasswordFallbackPrefix + "abcd1234",
		Status:     task.TaskStatusFailed,
	}
	evt, err := event.NewEvent(event.SubjectTaskFailed, failed)
	require.NoError(t, err)

	require.NoError(t, s.handleSyncTaskFailed(context.Background(), evt))
	require.Nil(t, enq.lastReq)
}
