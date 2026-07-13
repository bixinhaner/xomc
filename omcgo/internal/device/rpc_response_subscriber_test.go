package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
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
