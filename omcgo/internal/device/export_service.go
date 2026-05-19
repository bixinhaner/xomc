package device

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"go.uber.org/zap"
)

// ExportService provides device list export functionality.
type ExportService struct {
	deviceInfoRepo DeviceInfoRepository
	logger         *zap.Logger
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

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header.
	header := []string{
		"序列号", "设备名称", "状态", "运营商", "制式",
		"型号", "厂商", "站点", "IP 地址", "固件版本",
		"射频状态", "��区状态", "GPS 状态", "告警级别", "License 状态",
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
