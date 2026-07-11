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
