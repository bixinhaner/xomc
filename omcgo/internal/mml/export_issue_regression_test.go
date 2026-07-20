package mml

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func mustResultJSON(t *testing.T, standardValues []map[string]string, raw string) json.RawMessage {
	t.Helper()
	result, err := json.Marshal(map[string]interface{}{
		"standard_parameter_values": standardValues,
		"raw_response":              raw,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustCSVRecords(t *testing.T, data []byte) [][]string {
	t.Helper()
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func TestBuildLongFormatCSV_ExpandsStandardDescendantsByCommandInFirstSeenOrder(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"command_code":   "LST A",
			"operation_type": "LST",
			"param_paths":    []interface{}{"Device.A.", "Device.A.1."},
		},
		{
			"command_code":   "LST B",
			"operation_type": "LST",
			"param_paths":    []interface{}{"Device.B."},
		},
	}
	rawMustNotBeUsed := `<Envelope><Body><GetParameterValuesResponse><ParameterList>` +
		`<ParameterValueStruct><Name>Device.A.1.Foo</Name><Value>raw-wrong</Value></ParameterValueStruct>` +
		`<ParameterValueStruct><Name>Device.A.1.RawOnly</Name><Value>raw-only</Value></ParameterValueStruct>` +
		`</ParameterList></GetParameterValuesResponse></Body></Envelope>`
	rows := []DeviceTaskResultRowView{
		{
			DeviceSN: "SN001", Status: "completed", CommandIndex: 0,
			Result: mustResultJSON(t, []map[string]string{
				{"name": "Device.A.1.Foo", "value": "foo-1"},
				{"name": "Device.A.1.Bar", "value": "bar-1"},
			}, rawMustNotBeUsed),
		},
		{
			DeviceSN: "SN001", Status: "completed", CommandIndex: 1,
			Result: mustResultJSON(t, []map[string]string{
				{"name": "Device.B.1.Qux", "value": "qux-1"},
			}, ""),
		},
		{
			DeviceSN: "SN002", Status: "completed", CommandIndex: 0,
			Result: mustResultJSON(t, []map[string]string{
				{"name": "Device.A.1.Bar", "value": "bar-2"},
				{"name": "Device.A.1.Baz", "value": "baz-2"},
			}, ""),
		},
		{
			DeviceSN: "SN002", Status: "completed", CommandIndex: 1,
			Result: mustResultJSON(t, []map[string]string{
				{"name": "Device.B.1.Qux", "value": "qux-2"},
			}, ""),
		},
	}

	data, err := buildLongFormatCSVForLocale(
		[]exportColumn{
			{standard: "Device.A.", private: "Device.A."},
			{standard: "Device.A.1.", private: "Device.A.1."},
			{standard: "Device.B.", private: "Device.B."},
		},
		rows,
		commands,
		nil,
		true,
		appcontext.LocaleEN,
	)
	if err != nil {
		t.Fatal(err)
	}

	records := mustCSVRecords(t, data)
	got := make([]string, 0, len(records)-1)
	for _, record := range records[1:] {
		got = append(got, record[4]+"="+record[7])
	}
	want := strings.Join([]string{
		"Device.A.1.Foo=foo-1",
		"Device.A.1.Bar=bar-1",
		"Device.A.1.Baz=",
		"Device.B.1.Qux=qux-1",
		"Device.A.1.Foo=",
		"Device.A.1.Bar=bar-2",
		"Device.A.1.Baz=baz-2",
		"Device.B.1.Qux=qux-2",
	}, "\n")
	if strings.Join(got, "\n") != want {
		t.Fatalf("PATH/value rows =\n%s\nwant:\n%s", strings.Join(got, "\n"), want)
	}
}

func TestBuildDeviceCSVMulti_FallsBackToRawDescendantsAndPreservesExactLeaf(t *testing.T) {
	commands := []map[string]interface{}{
		{
			"command_code":   "LST A",
			"operation_type": "LST",
			"param_paths":    []interface{}{"Device.A.", "Device.C.Leaf"},
		},
	}
	raw := `<Envelope><Body><GetParameterValuesResponse><ParameterList>` +
		`<ParameterValueStruct><Name>InternetGatewayDevice.X.A.2.Bar</Name><Value>private-bar</Value></ParameterValueStruct>` +
		`<ParameterValueStruct><Name>Device.A.1.Foo</Name><Value>standard-foo</Value></ParameterValueStruct>` +
		`<ParameterValueStruct><Name>InternetGatewayDevice.X.C.Leaf</Name><Value>exact-leaf</Value></ParameterValueStruct>` +
		`<ParameterValueStruct><Name>InternetGatewayDevice.X.Unrelated</Name><Value>ignored</Value></ParameterValueStruct>` +
		`</ParameterList></GetParameterValuesResponse></Body></Envelope>`
	rows := []DeviceTaskResultRowView{
		{
			DeviceSN: "SN001", DeviceTaskID: "task-1", Status: "completed", CommandIndex: 0,
			Result: mustResultJSON(t, []map[string]string{}, raw),
		},
	}

	data, err := buildDeviceCSVMultiForLocale(
		[]exportColumn{
			{standard: "Device.A.", private: "InternetGatewayDevice.X.A."},
			{standard: "Device.C.Leaf", private: "InternetGatewayDevice.X.C.Leaf"},
		},
		rows,
		commands,
		nil,
		true,
		appcontext.LocaleEN,
	)
	if err != nil {
		t.Fatal(err)
	}

	records := mustCSVRecords(t, data)
	got := make([]string, 0)
	for _, record := range records {
		if len(record) == 10 && record[0] != "Parameter Path" {
			got = append(got, record[0]+"="+record[3])
		}
	}
	want := strings.Join([]string{
		"Device.A.2.Bar=private-bar",
		"Device.A.1.Foo=standard-foo",
		"Device.C.Leaf=exact-leaf",
	}, "\n")
	if strings.Join(got, "\n") != want {
		t.Fatalf("PATH/value rows =\n%s\nwant:\n%s", strings.Join(got, "\n"), want)
	}
}

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
