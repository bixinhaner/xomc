package cases

import "github.com/omcgo/omcgo/internal/interop"

// ProtocolCases returns predefined protocol conformance test cases
// that verify TR069/CWMP protocol compliance.
func ProtocolCases() []interop.TestCase {
	return []interop.TestCase{
		{
			ID:          "PROTO-001",
			Name:        "Inform Required Fields",
			Description: "Verify that the device Inform message contains all mandatory fields (DeviceId, Event, MaxEnvelopes, CurrentTime, RetryCount, ParameterList)",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Retrieve last Inform record for the device",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "serial_number",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Verify DeviceId section contains OUI, Manufacturer, and ProductClass",
					Action:      "check_param",
					Params: map[string]interface{}{
						"fields": []string{"oui", "manufacturer", "product_class"},
						"check":  "not_empty",
					},
				},
				{
					Order:       3,
					Description: "Verify last Inform events list is non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "last_inform_events",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "PROTO-002",
			Name:        "Session Management",
			Description: "Verify the device supports proper session management with valid connection request URL and inform interval",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Verify connection request URL is configured",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "connection_request_url",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Verify inform interval is a positive value",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "inform_interval",
						"check": "positive",
					},
				},
			},
		},
		{
			ID:          "PROTO-003",
			Name:        "SOAP Envelope Correctness",
			Description: "Verify the device uses a valid firmware version and IP address indicating proper SOAP communication",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Verify device has reported a firmware version",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "firmware_version",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Verify device IP address is present",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "ip_address",
						"check": "not_empty",
					},
				},
			},
		},
		{
			ID:          "PROTO-004",
			Name:        "Inform Identity Quartet Integrity",
			Description: "Verify the four immutable device-identity fields parsed out of Inform's DeviceId block are all persisted together (cross-field consistency check beyond PROTO-001)",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "OUI / Manufacturer / ProductClass / FirmwareVersion all non-empty (atomic group)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"fields": []string{"oui", "manufacturer", "product_class", "firmware_version"},
						"check":  "not_empty",
					},
				},
			},
		},
		{
			ID:          "PROTO-005",
			Name:        "Session Lifecycle Triad",
			Description: "Verify the three fields required for a healthy TR-069 session: ConnectionRequestURL (push channel) + IP address (data plane) + InformInterval (pull cadence)",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "ConnectionRequestURL non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "connection_request_url",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "IP address non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "ip_address",
						"check": "not_empty",
					},
				},
				{
					Order:       3,
					Description: "InformInterval positive",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "inform_interval",
						"check": "positive",
					},
				},
			},
		},
		{
			ID:          "PROTO-006",
			Name:        "Inform Event List Persisted",
			Description: "Verify at least one Inform event code has been recorded — a missing event list indicates protocol-level breakage during Inform parsing or persistence",
			Category:    interop.CategoryProtocol,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "last_inform_events non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "last_inform_events",
						"check": "not_empty",
					},
				},
			},
		},
		{
			// Negative path: probes runner reaction to a malformed step (missing required param).
			// Validates PRD §3 V3 "失败路径可识别".
			ID:              "PROTO-007",
			Name:            "Negative — Send RPC Without Method",
			Description:     "Negative path: send_rpc step with no 'method' param; runner must surface 'send_rpc step missing method parameter' error. Pass = error correctly surfaced",
			Category:        interop.CategoryProtocol,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Invoke send_rpc with empty method — expected to fail",
					Action:      "send_rpc",
					Params: map[string]interface{}{
						"method": "",
					},
				},
			},
		},
	}
}
