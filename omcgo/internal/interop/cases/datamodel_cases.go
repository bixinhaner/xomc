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
	}
}
