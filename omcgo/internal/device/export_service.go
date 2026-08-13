package device

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"go.uber.org/zap"
)

// ExportService provides device list export functionality.
type ExportService struct {
	deviceInfoRepo DeviceInfoRepository
	controlReader  DeviceControlSummaryReader
	logger         *zap.Logger
}

func (s *ExportService) SetControlSummaryReader(reader DeviceControlSummaryReader) {
	s.controlReader = reader
}

// NewExportService creates a new ExportService.
func NewExportService(deviceInfoRepo DeviceInfoRepository, logger *zap.Logger) *ExportService {
	return &ExportService{deviceInfoRepo: deviceInfoRepo, logger: logger.Named("export-service")}
}

// ExportCSV writes device list data as CSV to the given writer.
func (s *ExportService) ExportCSV(ctx context.Context, filter DeviceFilter, w io.Writer) error {
	// Override pagination to fetch up to 10K rows.
	filter.Page = 1
	filter.PageSize = 10000

	result, err := s.deviceInfoRepo.ListDevicesWithInfo(ctx, filter)
	if err != nil {
		return fmt.Errorf("query devices for export: %w", err)
	}
	if s.controlReader != nil && len(result.Items) > 0 {
		deviceIDs := make([]uuid.UUID, 0, len(result.Items))
		for index := range result.Items {
			deviceIDs = append(deviceIDs, result.Items[index].ID)
		}
		summaries, err := s.controlReader.ListCurrentByDeviceIDs(ctx, deviceIDs)
		if err != nil {
			return fmt.Errorf("query device control summaries for export: %w", err)
		}
		for index := range result.Items {
			if summary, ok := summaries[result.Items[index].ID]; ok {
				summaryCopy := summary
				result.Items[index].ControlSummary = &summaryCopy
			}
		}
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header.
	header := []string{
		"序列号", "设备名称", "状态", "运营商", "制式",
		"型号", "厂商", "站点", "IP 地址", "固件版本",
		"射频状态", "小区状态", "GPS 状态", "告警级别", "License 状态",
		"OMC管控来源", "OMC管控对象", "OMC管控阶段", "OMC管控原因码",
		"OMC管控触发时间", "OMC管控错误",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	// Write rows.
	for _, d := range result.Items {
		row := []string{
			d.SerialNumber,
			derefStr(d.InfoDeviceName),
			string(d.Status),
			string(d.Carrier),
			string(d.Technology),
			d.ModelName,
			d.Manufacturer,
			d.DeviceName,
			d.IPAddress,
			d.FirmwareVersion,
			derefStr(d.RFStatus),
			derefStr(d.CellStatus),
			derefStr(d.GPSStatus),
			derefStr(d.AlarmSeverity),
			derefStr(d.LicenseStatus),
		}
		if d.ControlSummary != nil {
			row = append(row,
				d.ControlSummary.SourceType,
				d.ControlSummary.SourceName,
				d.ControlSummary.Phase,
				d.ControlSummary.ReasonCode,
				d.ControlSummary.TriggeredAt.Format(time.RFC3339),
				d.ControlSummary.LastError,
			)
		} else {
			row = append(row, "", "", "", "", "", "")
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}

	return nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
