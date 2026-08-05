package alarm

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type sharedEmailTransportStub struct {
	recipients []string
	err        error
}

func (s *sharedEmailTransportStub) Send(_ context.Context, recipients []string, _, _ string) error {
	s.recipients = append([]string(nil), recipients...)
	return s.err
}

func TestSharedEmailDispatcher_ReusesTransportAndPreservesMetrics(t *testing.T) {
	transport := &sharedEmailTransportStub{}
	metrics := NewEmailMetrics(nil)
	dispatcher := NewSharedEmailDispatcher(transport, nil, metrics)

	require.NoError(t, dispatcher.Dispatch(context.Background(), []string{"noc@example.com"}, "subject", "body"))
	require.Equal(t, []string{"noc@example.com"}, transport.recipients)
	require.Equal(t, 1.0, testutil.ToFloat64(metrics.DispatchTotal.WithLabelValues("success")))

	transport.err = errors.New("unavailable")
	require.Error(t, dispatcher.Dispatch(context.Background(), []string{"noc@example.com"}, "subject", "body"))
	require.Equal(t, 1.0, testutil.ToFloat64(metrics.DispatchTotal.WithLabelValues("failure")))
}
