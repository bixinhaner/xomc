package mml

import (
	"strings"
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestBuildLongFormatCSV_DoesNotEmitDeviceSeparatorOrLegacyColumns(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"command_code":   "LST DEVICE_INFO",
			"operation_type": "LST",
			"param_paths":    []interface{}{"Device.DeviceInfo.SerialNumber"},
		},
	}
	rows := []DeviceTaskResultRowView{
		{DeviceSN: "SN001", DeviceTaskID: "task-1", Status: "completed", CommandIndex: 0},
		{DeviceSN: "SN002", DeviceTaskID: "task-2", Status: "completed", CommandIndex: 0},
	}

	data, err := buildLongFormatCSVForLocale(
		[]exportColumn{{standard: "Device.DeviceInfo.SerialNumber", private: "Device.DeviceInfo.SerialNumber"}},
		rows,
		commands,
		map[string]string{"Device.DeviceInfo.SerialNumber": "Serial number"},
		true,
		appcontext.LocaleEN,
	)
	if err != nil {
		t.Fatal(err)
	}

	csv := string(data)
	if strings.Contains(csv, "执行模式") || strings.Contains(csv, "子任务ID") {
		t.Fatalf("legacy aggregate columns must not be exported: %q", csv)
	}
	if strings.Contains(csv, "\r\n\r\n") {
		t.Fatalf("device rows must not be separated by a blank CSV row: %q", csv)
	}
	if !strings.HasPrefix(csv, "\ufeffIndex,Device SN,Command,Operation,") {
		t.Fatalf("English export header is not localized: %q", csv)
	}
}

func TestDeviceTaskRowToResultMap_PreservesCommandName(t *testing.T) {
	task := &MMLTask{Commands: []map[string]interface{}{
		{
			"command_code":   "RAW LST",
			"command_name":   "dxpTest",
			"operation_type": "LST",
		},
	}}
	row := DeviceTaskResultRowView{DeviceSN: "SN001", CommandIndex: 0, Status: "completed"}

	got := deviceTaskRowToResultMap(row, task)
	if got["command_name"] != "dxpTest" {
		t.Fatalf("command_name = %v, want dxpTest", got["command_name"])
	}
}
