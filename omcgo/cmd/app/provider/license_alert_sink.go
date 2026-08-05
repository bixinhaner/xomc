package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/license"
	"go.uber.org/zap"
)

const systemLicenseAlarmSN = "OMC-SYSTEM"

type systemLicenseAlertSink struct {
	engine *alarm.AlarmEngine
	logger *zap.Logger
}

func newSystemLicenseAlertSink(engine *alarm.AlarmEngine, logger *zap.Logger) license.AlertSink {
	if engine == nil {
		return license.NoopAlertSink{}
	}
	return &systemLicenseAlertSink{engine: engine, logger: logger.Named("license-alert-sink")}
}

func (s *systemLicenseAlertSink) Send(ctx context.Context, alert license.Alert) error {
	now := time.Now()
	additionalInfo := make(map[string]string, len(alert.Details))
	for key, value := range alert.Details {
		additionalInfo[key] = fmt.Sprint(value)
	}

	alarmRecord := &model.Alarm{
		DeviceID:        uuid.Nil,
		DeviceSN:        systemLicenseAlarmSN,
		Carrier:         model.CarrierCMCC,
		Severity:        mapLicenseAlertSeverity(alert.Severity),
		AlarmType:       "system_license",
		AlarmIdentifier: alert.Identifier,
		Description:     alert.Summary,
		AlarmSource:     stringPtr("omc-license"),
		EventType:       stringPtr("system_license"),
		ProbableCause:   stringPtr(alert.Identifier),
		AdditionalInfo:  additionalInfo,
		RaisedAt:        now,
		FirstRaisedAt:   now,
		LastUpdatedAt:   now,
	}
	if err := s.engine.Process(ctx, alarmRecord); err != nil {
		return fmt.Errorf("persist system license alert %q: %w", alert.Identifier, err)
	}
	return nil
}

func mapLicenseAlertSeverity(severity license.AlertSeverity) model.AlarmSeverity {
	switch severity {
	case license.AlertSeverityCritical:
		return model.AlarmCritical
	case license.AlertSeverityMajor:
		return model.AlarmMajor
	default:
		return model.AlarmWarning
	}
}

func stringPtr(value string) *string { return &value }
