package device

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
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
