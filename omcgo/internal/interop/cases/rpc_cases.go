package cases

import "github.com/omcgo/omcgo/internal/interop"

// RPCCases returns predefined RPC method conformance test cases for
// the nine standard TR069 RPC methods (ACS -> CPE direction).
func RPCCases() []interop.TestCase {
	return []interop.TestCase{
		{
			ID:          "RPC-001",
			Name:        "GetParameterValues",
			Description: "Verify the device supports the GetParameterValues RPC by enqueueing a read command",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a GetParameterValues command for a well-known parameter path",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "GetParameterValues",
						"paths":  []string{"Device.DeviceInfo."},
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-002",
			Name:        "SetParameterValues",
			Description: "Verify the device supports the SetParameterValues RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a SetParameterValues command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "SetParameterValues",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-003",
			Name:        "GetParameterNames",
			Description: "Verify the device supports the GetParameterNames RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a GetParameterNames command for the root object",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method":     "GetParameterNames",
						"path":       "Device.",
						"next_level": true,
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-004",
			Name:        "GetParameterAttributes",
			Description: "Verify the device supports the GetParameterAttributes RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a GetParameterAttributes command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "GetParameterAttributes",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-005",
			Name:        "SetParameterAttributes",
			Description: "Verify the device supports the SetParameterAttributes RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a SetParameterAttributes command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "SetParameterAttributes",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-006",
			Name:        "Download",
			Description: "Verify the device supports the Download RPC method for firmware updates",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a Download command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method":    "Download",
						"file_type": "1 Firmware Upgrade Image",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-007",
			Name:        "Upload",
			Description: "Verify the device supports the Upload RPC method for configuration backups",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue an Upload command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method":    "Upload",
						"file_type": "1 Vendor Configuration File",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-008",
			Name:        "Reboot",
			Description: "Verify the device supports the Reboot RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a Reboot command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "Reboot",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-009",
			Name:        "FactoryReset",
			Description: "Verify the device supports the FactoryReset RPC method",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a FactoryReset command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "FactoryReset",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-010",
			Name:        "Upload",
			Description: "Verify the device supports the Upload RPC method (used for log / config / PM file collection from device → OMC)",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue an Upload command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "Upload",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-011",
			Name:        "ScheduleInform",
			Description: "Verify the device supports the ScheduleInform RPC method (used to trigger an on-demand Inform after a delay)",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a ScheduleInform command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "ScheduleInform",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			ID:          "RPC-012",
			Name:        "GetRPCMethods",
			Description: "Verify the device supports the GetRPCMethods RPC (interoperability discovery — lets OMC enumerate what the device implements)",
			Category:    interop.CategoryRPC,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a GetRPCMethods command",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "GetRPCMethods",
					},
				},
				{
					Order:       2,
					Description: "Verify command was enqueued successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "command_queued",
					},
				},
			},
		},
		{
			// Negative path: probes runner reaction to an unknown verify_response check.
			// Validates PRD §3 V3 "失败路径可识别".
			ID:              "RPC-013",
			Name:            "Negative — Verify Response With Unknown Check",
			Description:     "Negative path: verify_response with an unrecognized check keyword; runner must surface 'unknown verify_response check' error. Pass = error correctly surfaced",
			Category:        interop.CategoryRPC,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Enqueue a known-good RPC (Reboot) — step expected to pass",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "Reboot",
					},
				},
				{
					Order:       2,
					Description: "Invoke verify_response with check=no_such_check — expected to fail",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "no_such_check",
					},
				},
			},
		},
	}
}
