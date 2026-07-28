package dashboard

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// TopAlarmDevice is a current active-alarm aggregation for one device.
type TopAlarmDevice struct {
	DeviceSN   string `json:"device_sn"`
	Technology string `json:"technology"`
	AlarmCount int64  `json:"alarm_count"`
	Critical   int64  `json:"critical"`
	Major      int64  `json:"major"`
	Minor      int64  `json:"minor"`
	Warning    int64  `json:"warning"`
}

func buildTopAlarmDevicesQuery(visibleGroups []uuid.UUID) (string, []any, error) {
	builder := storage.Psql.Select(
		"device_sn",
		"COALESCE(technology, '') AS technology",
		"COUNT(*) AS alarm_count",
		"COUNT(*) FILTER (WHERE severity IN (1,31001)) AS critical",
		"COUNT(*) FILTER (WHERE severity IN (2,31002)) AS major",
		"COUNT(*) FILTER (WHERE severity IN (3,31003)) AS minor",
		"COUNT(*) FILTER (WHERE severity IN (4,31004)) AS warning",
	).
		From("alarms_active").
		Where(sq.NotEq{"device_sn": ""}).
		GroupBy("device_sn", "technology").
		OrderBy("alarm_count DESC", "device_sn ASC").
		Limit(10)
	builder = authz.ApplyDeviceVisibilityFilter(builder, "alarms_active.device_id", visibleGroups)
	return builder.ToSql()
}

// GetTopAlarmDevices returns the ten devices with the most current uncleared alarms.
func (s *Service) GetTopAlarmDevices(ctx context.Context, visibleGroups []uuid.UUID) ([]TopAlarmDevice, error) {
	query, args, err := buildTopAlarmDevicesQuery(visibleGroups)
	if err != nil {
		return nil, fmt.Errorf("build top alarm devices query: %w", err)
	}
	rows, err := s.pgPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query top alarm devices: %w", err)
	}
	defer rows.Close()

	result := make([]TopAlarmDevice, 0, 10)
	for rows.Next() {
		var entry TopAlarmDevice
		if err := rows.Scan(
			&entry.DeviceSN,
			&entry.Technology,
			&entry.AlarmCount,
			&entry.Critical,
			&entry.Major,
			&entry.Minor,
			&entry.Warning,
		); err != nil {
			return nil, fmt.Errorf("scan top alarm device row: %w", err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top alarm device rows: %w", err)
	}
	return result, nil
}
