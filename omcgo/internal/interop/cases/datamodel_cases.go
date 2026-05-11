package cases

import "github.com/omcgo/omcgo/internal/interop"

// DataModelCases returns predefined data model conformance test cases that
// verify the device's parameter tree matches the expected data model definition.
func DataModelCases() []interop.TestCase {
	return []interop.TestCase{
		{
			ID:          "DM-001",
			Name:        "Parameter Path Existence",
			Description: "Verify that the device reports parameters defined in the resolved data model",
			Category:    interop.CategoryDataModel,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Resolve the data model for the device (carrier + tech + OUI + product class)",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"operation": "resolve_datamodel",
						"check":     "model_exists",
					},
				},
				{
					Order:       2,
					Description: "Compare device parameters against data model parameter tree paths",
					Action:      "check_param",
					Params: map[string]interface{}{
						"operation": "check_param_paths",
						"check":     "paths_present",
					},
				},
			},
		},
		{
			ID:          "DM-002",
			Name:        "Parameter Type Validation",
			Description: "Verify that device parameter types match the types defined in the data model",
			Category:    interop.CategoryDataModel,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Resolve the data model for the device",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"operation": "resolve_datamodel",
						"check":     "model_exists",
					},
				},
				{
					Order:       2,
					Description: "Compare device parameter types against data model definitions",
					Action:      "check_param",
					Params: map[string]interface{}{
						"operation": "check_param_types",
						"check":     "types_match",
					},
				},
			},
		},
		{
			ID:          "DM-003",
			Name:        "Data Model Source Resolution",
			Description: "Verify the OMC can resolve a data model for this device's product class (productRegistry → paramRegistry lookup chain succeeds)",
			Category:    interop.CategoryDataModel,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Device firmware version is persisted (param-mapping resolution input)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"field": "firmware_version",
						"check": "not_empty",
					},
				},
				{
					Order:       2,
					Description: "Resolved data model exists for product class + firmware",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"operation": "resolve_datamodel",
						"check":     "model_exists",
					},
				},
			},
		},
		{
			ID:          "DM-004",
			Name:        "Parameter Path Inventory Consistency",
			Description: "Verify the device-reported parameter set is not empty after resolving the data model — covers the boundary case where Inform succeeds but parameter list is empty",
			Category:    interop.CategoryDataModel,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Data model resolves successfully",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"operation": "resolve_datamodel",
						"check":     "model_exists",
					},
				},
				{
					Order:       2,
					Description: "Reported parameter paths exist (cross-check with model)",
					Action:      "check_param",
					Params: map[string]interface{}{
						"operation": "check_param_paths",
						"check":     "paths_present",
					},
				},
			},
		},
		{
			ID:          "DM-005",
			Name:        "Device Identifier Triplet Persistence",
			Description: "Verify OUI / Manufacturer / ProductClass are jointly persisted — these are required for ProductRegistry route-by-regex lookup",
			Category:    interop.CategoryDataModel,
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "OUI / Manufacturer / ProductClass all non-empty",
					Action:      "check_param",
					Params: map[string]interface{}{
						"fields": []string{"oui", "manufacturer", "product_class"},
						"check":  "not_empty",
					},
				},
			},
		},
		{
			// Negative path: probes runner reaction to an unknown check keyword.
			// Validates PRD §3 V3 "失败路径可识别".
			ID:              "DM-006",
			Name:            "Negative — Unknown DataModel Check Keyword",
			Description:     "Negative path: invoke an unknown check keyword on check_param; runner must surface 'unknown check type' error. Pass = error correctly surfaced",
			Category:        interop.CategoryDataModel,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Invoke check_param with check=no_such_keyword — expected to fail",
					Action:      "check_param",
					Params: map[string]interface{}{
						"check": "no_such_keyword",
					},
				},
			},
		},
		{
			// Negative path 2 (Phase 2): unknown verify_response check inside datamodel context.
			ID:              "DM-007",
			Name:            "Negative — Unknown verify_response Check",
			Description:     "Negative path: verify_response with unknown check keyword (e.g. 'model_does_not_exist_check'); runner must surface 'unknown verify_response check' error",
			Category:        interop.CategoryDataModel,
			ExpectedOutcome: "fail",
			Steps: []interop.TestStep{
				{
					Order:       1,
					Description: "Invoke verify_response with unrecognized check — expected to fail",
					Action:      "verify_response",
					Params: map[string]interface{}{
						"check": "model_does_not_exist_check",
					},
				},
			},
		},
	}
}
