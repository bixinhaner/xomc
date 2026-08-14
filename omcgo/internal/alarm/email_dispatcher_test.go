package alarm

import (
	"context"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/notification"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type alarmEmailTransportStub struct {
	message notification.EmailMessage
	err     error
	calls   int
}

func (s *alarmEmailTransportStub) Send(_ context.Context, message notification.EmailMessage) error {
	s.calls++
	s.message = message
	return s.err
}

func TestNotificationEmailDispatcher_Dispatch(t *testing.T) {
	t.Parallel()
	transport := &alarmEmailTransportStub{}
	metrics := NewEmailMetrics(nil)
	dispatcher := NewNotificationEmailDispatcher(transport, nil, metrics)

	require.NoError(t, dispatcher.Dispatch(context.Background(), []string{"ops@example.com"}, "告警", "正文"))
	require.Equal(t, 1, transport.calls)
	require.Equal(t, []string{"ops@example.com"}, transport.message.To)
	require.Equal(t, "告警", transport.message.Subject)
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DispatchTotal.WithLabelValues("success")))
}

func TestNotificationEmailDispatcher_ClassifiesFailure(t *testing.T) {
	t.Parallel()
	transport := &alarmEmailTransportStub{err: context.DeadlineExceeded}
	metrics := NewEmailMetrics(nil)
	dispatcher := NewNotificationEmailDispatcher(transport, nil, metrics)
	require.Error(t, dispatcher.Dispatch(context.Background(), []string{"ops@example.com"}, "告警", "正文"))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DispatchTotal.WithLabelValues("timeout")))

	transport.err = errors.New("smtp unavailable")
	require.Error(t, dispatcher.Dispatch(context.Background(), []string{"ops@example.com"}, "告警", "正文"))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DispatchTotal.WithLabelValues("failure")))
}
