package alarm

import (
	"context"
	"errors"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// EmailDispatcher remains only as the migration boundary for legacy
// notify_email rules. SMTP transport is implemented by notification.
type EmailDispatcher interface {
	Dispatch(context.Context, []string, string, string) error
}

type noopEmailDispatcher struct{}

func (noopEmailDispatcher) Dispatch(context.Context, []string, string, string) error {
	return errors.New("legacy alarm email dispatcher is unavailable")
}

type EmailMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

func NewEmailMetrics(reg prometheus.Registerer) *EmailMetrics {
	metrics := &EmailMetrics{DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alarm_email_dispatches_total",
		Help: "Total legacy alarm email dispatches by result (success/failure/timeout)",
	}, []string{"result"})}
	if reg != nil {
		reg.MustRegister(metrics.DispatchTotal)
	}
	return metrics
}

func validateEmailParams(from string, to []string, subject string) error {
	if strings.TrimSpace(from) == "" {
		return errors.New("smtp dispatch: from address empty")
	}
	if len(to) == 0 {
		return errors.New("smtp dispatch: recipient list empty")
	}
	for _, address := range to {
		if strings.TrimSpace(address) == "" {
			return errors.New("smtp dispatch: empty recipient in list")
		}
	}
	if strings.TrimSpace(subject) == "" {
		return errors.New("smtp dispatch: subject empty")
	}
	return nil
}
