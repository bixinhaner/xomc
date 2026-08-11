package mml

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDeviceTaskRowToResultMap_DeviceBoundPlanTrace(t *testing.T) {
	task := &MMLTask{
		ExecuteMode: TaskExecuteModeDeviceBound,
		PlanItems: []MMLPlanItem{
			{
				LineNo:   7,
				DeviceSN: "SN001",
				Order:    2,
				RawLine:  "MOD CELL:CELL_INDEX={1};SN001",
				Command: map[string]interface{}{
					"command_code":   "MOD CELL",
					"operation_type": "MOD",
				},
			},
		},
	}
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-1",
		DeviceSN:     "SN001",
		Status:       "completed",
		CommandIndex: 0,
	}

	got := deviceTaskRowToResultMap(row, task)
	if got["plan_line_no"] != 7 {
		t.Fatalf("plan_line_no = %v, want 7", got["plan_line_no"])
	}
	if got["plan_device_sn"] != "SN001" {
		t.Fatalf("plan_device_sn = %v, want SN001", got["plan_device_sn"])
	}
	if got["plan_order"] != 2 {
		t.Fatalf("plan_order = %v, want 2", got["plan_order"])
	}
	if got["command_code"] != "MOD CELL" {
		t.Fatalf("command_code = %v, want MOD CELL", got["command_code"])
	}
	if got["mml_script"] != "MOD CELL:CELL_INDEX={1}" {
		t.Fatalf("mml_script = %v, want MOD CELL:CELL_INDEX={1}", got["mml_script"])
	}
}

func TestDeviceTaskRowToResultMap_MultiSNRawLineOmitsDeviceList(t *testing.T) {
	task := &MMLTask{
		ExecuteMode: TaskExecuteModeDeviceBound,
		PlanItems: []MMLPlanItem{
			{
				LineNo:   1,
				DeviceSN: "SN001",
				Order:    1,
				RawLine:  "LST Device.DeviceInfo.SoftwareVersion;SN001,SN002",
				Command: map[string]interface{}{
					"command_code":   "RAW LST",
					"operation_type": "LST",
				},
			},
		},
	}
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-1",
		DeviceSN:     "SN001",
		Status:       "completed",
		CommandIndex: 0,
	}

	got := deviceTaskRowToResultMap(row, task)
	if got["mml_script"] != "LST Device.DeviceInfo.SoftwareVersion" {
		t.Fatalf("mml_script = %v, want LST Device.DeviceInfo.SoftwareVersion", got["mml_script"])
	}
}

func TestDeviceTaskRowToResultMap_DeviceBoundExpandedCommandTrace(t *testing.T) {
	task := &MMLTask{
		ExecuteMode: TaskExecuteModeDeviceBound,
		PlanItems: []MMLPlanItem{
			{
				LineNo:   4,
				DeviceSN: "SN001",
				Order:    3,
				RawLine:  "ADD Device.X.:NAME=foo;SN001",
				Command: map[string]interface{}{
					"command_code":   "RAW ADD",
					"operation_type": "ADD",
				},
			},
			{
				LineNo:   5,
				DeviceSN: "SN001",
				Order:    4,
				RawLine:  "RMV Device.X.9999.;SN001",
				Command: map[string]interface{}{
					"command_code":   "RAW RMV",
					"operation_type": "RMV",
				},
			},
		},
		Commands: []map[string]interface{}{
			{
				"command_code":   "RAW ADD",
				"operation_type": "ADD",
				"plan_line_no":   float64(4),
				"plan_order":     float64(3),
				"plan_raw_line":  "ADD Device.X.:NAME=foo;SN001",
				"plan_device_sn": "SN001",
			},
			{
				"command_code":    "RAW MOD",
				"operation_type":  "MOD",
				"compound_phase":  "spv_after_add",
				"compound_parent": "RAW ADD",
				"plan_line_no":    float64(4),
				"plan_order":      float64(3),
				"plan_raw_line":   "ADD Device.X.:NAME=foo;SN001",
				"plan_device_sn":  "SN001",
				"parameters":      map[string]interface{}{"Device.X.{NEW}.NAME": "foo"},
				"param_paths":     []interface{}{"Device.X.{NEW}.NAME"},
				"plan_sort_order": float64(3001),
			},
			{
				"command_code":   "RAW RMV",
				"operation_type": "RMV",
				"plan_line_no":   float64(5),
				"plan_order":     float64(4),
				"plan_raw_line":  "RMV Device.X.9999.;SN001",
				"plan_device_sn": "SN001",
			},
		},
	}
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-spv-after-add",
		DeviceSN:     "SN001",
		Status:       "completed",
		CommandIndex: 1,
	}

	got := deviceTaskRowToResultMap(row, task)
	if got["command_code"] != "RAW MOD" {
		t.Fatalf("command_code = %v, want RAW MOD", got["command_code"])
	}
	if got["operation_type"] != "MOD" {
		t.Fatalf("operation_type = %v, want MOD", got["operation_type"])
	}
	if got["plan_line_no"] != 4 {
		t.Fatalf("plan_line_no = %v, want source ADD line 4", got["plan_line_no"])
	}
	if got["plan_raw_line"] != "ADD Device.X.:NAME=foo;SN001" {
		t.Fatalf("plan_raw_line = %v, want ADD raw line", got["plan_raw_line"])
	}
	if got["mml_script"] != "ADD Device.X.:NAME=foo" {
		t.Fatalf("mml_script = %v, want original ADD script line", got["mml_script"])
	}
}

func TestDeviceTaskRowToResultMap_CommonCommandScriptText(t *testing.T) {
	task := &MMLTask{
		ExecuteMode: TaskExecuteModeCommon,
		Commands: []map[string]interface{}{
			{
				"command_code":   "MOD EUTRANNFREQ",
				"operation_type": "MOD",
				"parameters": map[string]interface{}{
					"DL_EARFCN": 1231312,
				},
			},
		},
	}
	rawResult, err := json.Marshal(map[string]interface{}{
		"method":       "GetParameterValuesResponse",
		"raw_response": "<xml/>",
	})
	if err != nil {
		t.Fatal(err)
	}
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-1",
		DeviceSN:     "SN001",
		Status:       "completed",
		CommandIndex: 0,
		Result:       rawResult,
	}

	got := deviceTaskRowToResultMap(row, task)
	if got["mml_script"] != "MOD EUTRANNFREQ:DL_EARFCN={1231312}" {
		t.Fatalf("mml_script = %v, want MOD EUTRANNFREQ:DL_EARFCN={1231312}", got["mml_script"])
	}
	if got["mml_script"] == "GetParameterValuesResponse" {
		t.Fatalf("mml_script should not use SOAP method name")
	}
}

func TestDeviceTaskRowToResultMap_IncludesRequestAndResponseMessages(t *testing.T) {
	task := &MMLTask{
		ExecuteMode: TaskExecuteModeCommon,
		Commands: []map[string]interface{}{
			{
				"command_code":   "MOD MANAGEMENT_SERVER",
				"operation_type": "MOD",
			},
		},
	}
	rawResult, err := json.Marshal(map[string]interface{}{
		"method":       "SetParameterValuesResponse",
		"raw_response": "<soap:Envelope><soap:Body><cwmp:SetParameterValuesResponse/></soap:Body></soap:Envelope>",
	})
	if err != nil {
		t.Fatal(err)
	}
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-1",
		DeviceSN:     "SN001",
		Method:       "SetParameterValues",
		Params:       json.RawMessage(`{"values":[{"name":"Device.ManagementServer.URL","value":"http://localhost:8080/smallcell/AcsService","type":"xsd:string"}]}`),
		CommandKey:   "mml-device-task-1",
		CWMPID:       "ID:intrnl.unset.id.SetParameterValues1780000000.123456",
		Status:       "completed",
		CommandIndex: 0,
		Result:       rawResult,
	}

	got := deviceTaskRowToResultMap(row, task)
	if got["request_method"] != "SetParameterValues" {
		t.Fatalf("request_method = %v, want SetParameterValues", got["request_method"])
	}
	if got["request_cwmp_id"] != "ID:intrnl.unset.id.SetParameterValues1780000000.123456" {
		t.Fatalf("request_cwmp_id = %v", got["request_cwmp_id"])
	}
	rawRequest, _ := got["raw_request"].(string)
	if !strings.Contains(rawRequest, "<cwmp:SetParameterValues>") {
		t.Fatalf("raw_request = %q, want SetParameterValues SOAP", rawRequest)
	}
	if !strings.Contains(rawRequest, "Device.ManagementServer.URL") {
		t.Fatalf("raw_request = %q, want rendered parameter", rawRequest)
	}
	if got["raw_output"] != "<soap:Envelope><soap:Body><cwmp:SetParameterValuesResponse/></soap:Body></soap:Envelope>" {
		t.Fatalf("raw_output = %v, want raw response", got["raw_output"])
	}
}

func TestDeviceTaskRowToResultMap_FallsBackToRPCOperationWhenCommandSnapshotMissing(t *testing.T) {
	row := DeviceTaskResultRowView{
		DeviceTaskID: "device-task-mod-fallback",
		DeviceSN:     "SN001",
		Method:       "SetParameterValues",
		Params:       json.RawMessage(`{"values":[{"name":"Device.DeviceInfo.SoftwareVersion","value":"1.2.3","type":"xsd:string"}]}`),
		Status:       "failed",
	}

	got := deviceTaskRowToResultMap(row, nil)
	if got["operation_type"] != "MOD" {
		t.Fatalf("operation_type = %v, want MOD", got["operation_type"])
	}
	if got["command_code"] != "MOD" {
		t.Fatalf("command_code = %v, want MOD", got["command_code"])
	}
}

func TestBuildDeviceBoundPlanCSV(t *testing.T) {
	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	task := &MMLTask{
		ID:          taskID,
		TaskName:    "device-bound",
		ExecuteMode: TaskExecuteModeDeviceBound,
		PlanItems: []MMLPlanItem{
			{
				LineNo:   3,
				DeviceSN: "SN100",
				Order:    1,
				RawLine:  "LST CELL;SN100",
				Command: map[string]interface{}{
					"command_code":   "LST CELL",
					"rpc_method":     "GetParameterValues",
					"operation_type": "LST",
					"parameters": map[string]interface{}{
						"CELL_INDEX": "{1}",
					},
				},
			},
		},
	}
	result := map[string]interface{}{"raw_response": "<xml/>"}
	rawResult, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	rows := []DeviceTaskResultRowView{
		{
			DeviceTaskID: "dt-1",
			DeviceSN:     "SN100",
			Status:       "completed",
			Result:       rawResult,
			CommandIndex: 0,
		},
	}

	data, err := buildDeviceBoundPlanCSV(task, rows, "")
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), utf8BOM)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}
	if records[0][0] != "任务ID" || records[0][2] != "行号" {
		t.Fatalf("unexpected header: %#v", records[0])
	}
	row := records[1]
	if row[0] != taskID.String() || row[2] != "3" || row[3] != "SN100" || row[6] != "LST CELL" {
		t.Fatalf("unexpected csv row: %#v", row)
	}
	if !strings.Contains(row[9], "CELL_INDEX") {
		t.Fatalf("parameter summary = %q, want CELL_INDEX", row[9])
	}
}
