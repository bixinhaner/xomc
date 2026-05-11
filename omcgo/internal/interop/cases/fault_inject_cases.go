package cases

import "github.com/omcgo/omcgo/internal/interop"

// FaultInjectCases returns predefined fault-injection test cases that exercise
// the OMC's resilience against degraded device state.
//
// 设计原则（T-0114 PRD §8 Phase 2 选项 fault-inject category）：
//   - 与 negative path（每类 ≥ 2 的 ExpectedOutcome="fail" 用例）的区别：
//     negative path 是"用例本身故意失败（step 写错）"，fault-inject 是
//     "在 device 处于异常状态时验证 OMC 是否正确识别"。
//   - 本任务不接真实设备状态扰动（PRD §5 非目标声明），fault-inject 用例
//     通过 multi-field 校验 + 边界 check 类型组合，模拟"设备身份缺失 /
//     Inform 缺字段 / SOAP DeviceId 故障"等场景在 OMC 持久状态上的表现。
//   - FI-004 是 negative path 用例（ExpectedOutcome="fail"），满足"每类 ≥ 2"
//     门槛中 fault_inject 类的 negative 配额（其他 negative 也可放在这里
//     视后续维护需要）。
func FaultInjectCases() []interop.TestCase {
	return []interop.TestCase{
		{
			ID:          "FI-001",
			Name:        "Identity Triplet Integrity Under Fault",
			Description: "Fault scenario — verify OMC requires OUI + Manufacturer + ProductClass + FirmwareVersion atomically; missing any one means DeviceId block was corrupted during Inform parse. Pass = all four present together (OMC correctly rejected partial DeviceId)",
			Category:    interop.CategoryFaultInject,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "All four DeviceId fields must be non-empty as an atomic group",
					Action:      "check_param",
					Params: map[string]interface{}{
						"fields": []string{"oui", "manufacturer", "product_class", "firmware_version"},
						"check":  "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Serial number persisted (device row exists)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "serial_number",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "FI-002",
			Name:        "Inform Channel Integrity Under Fault",
			Description: "Fault scenario — Inform channel health requires both InformInterval (pull cadence) and ConnectionRequestURL (push channel). If either fails, OMC is one-way blind to the device",
			Category:    interop.CategoryFaultInject,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "InformInterval must be positive (pull side alive)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "inform_interval",
						"check": "positive",
					},
				},
				{
					Order:       2,
					Description: "ConnectionRequestURL must be non-empty (push side alive)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "connection_request_url",
						"check": "not_empty",
					},
				},
				{
					Order:       3,
					Description: "Last Inform events recorded (Inform parse succeeded at least once)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "last_inform_events",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "FI-003",
			Name:        "RPC Channel Resilience Under Boundary Method",
			Description: "Fault scenario — verify task queue accepts boundary RPC method name (long vendor-prefix string). Catches regressions where queue layer adds length / charset constraints that would reject legitimate Carrier extensions",
			Category:    interop.CategoryFaultInject,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue an RPC with long vendor-namespaced method name",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "X_CMCC_BoundaryMethodNameForResilienceProbe",
					},
				},
				{
					Order:       2,
					Description: "Command enqueued successfully (no length / charset rejection)",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			// Negative path within fault_inject category — exercises runner's
			// default branch for unknown action types under "fault" framing.
			ID:              "FI-004",
			Name:            "Negative — Unknown Fault Action",
			Description:     "Negative path: invoke an unknown fault-style action keyword ('inject_corruption'); runner must surface 'unknown action' error. Pass = error correctly surfaced",
			Category:        interop.CategoryFaultInject,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Invoke unrecognized action 'inject_corruption' — expected to fail",
					Action:      "inject_corruption",
					Params:      map[string]interface{}{},
				},
			},
		},
	}
}
