package cases

import "github.com/omcgo/omcgo/internal/interop"

// InformCases returns predefined Inform-related test cases that verify the
// OMC's persisted view of CPE → ACS Inform messages.
//
// 设计原则（T-0030 PRD §9.4）：
//   - 本任务不做端到端 SOAP/CWMP 实测，所有用例基于 device 持久状态
//     （LastInformEvents / InformInterval / ConnectionRequestURL 等字段）。
//   - 真实 Inform 处理 / 解析正确性由 internal/acs 单测 + cpe_simulator.py
//     E2E 覆盖，不在 F10 用例库范围。
//   - INF-004 是 negative path 用例（ExpectedOutcome="fail"），用于验证
//     runner 能识别"不存在字段→失败"，参见 PRD §3 V3 验收。
func InformCases() []interop.TestCase {
	return []interop.TestCase{
		{
			ID:          "INF-001",
			Name:        "Bootstrap Inform Acknowledged",
			Description: "Verify the OMC has recorded at least one Inform from the device — confirms initial registration (event code 0 BOOTSTRAP or 1 BOOT) was processed end-to-end",
			Category:    interop.CategoryInform,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Device serial number is persisted (device row exists)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "serial_number",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Device has at least one recorded Inform event code",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "last_inform_events",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "INF-002",
			Name:        "Periodic Inform Channel Active",
			Description: "Verify the Periodic Inform mechanism (event code 2 PERIODIC) is active — InformInterval must be configured positive so the device keeps reporting",
			Category:    interop.CategoryInform,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "InformInterval is set to a positive value",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "inform_interval",
						"check": "positive",
					},
				},
				{
					Order:       2,
					Description: "Device has been reported at least once (last_inform_events present)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "last_inform_events",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "INF-003",
			Name:        "Connection Request URL Available for Push",
			Description: "Verify the device exposed a ConnectionRequestURL — required for OMC to trigger event code 6 CONNECTION REQUEST and event code 4 VALUE CHANGE round-trips",
			Category:    interop.CategoryInform,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "ConnectionRequestURL is non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "connection_request_url",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Manufacturer / OUI / ProductClass identifiers are persisted (Inform DeviceId block was parsed)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"fields": []string{"oui", "manufacturer", "product_class"},
						"check":  "not_empty",
					},
				},
			},
		},
		{
			// Negative path: tests runner's ability to flag unknown fields as failures.
			// Validates PRD §3 V3 "失败路径可识别".
			ID:              "INF-004",
			Name:            "Negative — Unknown Inform Field Probe",
			Description:     "Negative path: probe a non-existent device field; runner must surface 'field is empty' or 'unknown' error. Pass = error correctly surfaced; fail = runner silently swallowed bad input",
			Category:        interop.CategoryInform,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Probe non-existent field 'some_unknown_inform_field' — expected to fail",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "some_unknown_inform_field",
						"check": "not_empty",
					},
				},
			},
		},
	}
}
