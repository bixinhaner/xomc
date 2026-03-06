package interop

import "time"

// ValidationReport summarizes the comparison between a device's actual parameters
// and the expected parameters from its resolved data model definition.
type ValidationReport struct {
	DeviceID       string           `json:"device_id"`
	DeviceSN       string           `json:"device_sn"`
	ModelVersion   string           `json:"model_version"`
	TotalParams    int              `json:"total_params"`
	MatchedParams  int              `json:"matched_params"`
	MismatchParams []ParamMismatch  `json:"mismatch_params"`
	MissingParams  []string         `json:"missing_params"`
	ExtraParams    []string         `json:"extra_params"`
	Score          float64          `json:"score"`
	CreatedAt      time.Time        `json:"created_at"`
}

// ParamMismatch records a single parameter whose actual value or type on the
// device differs from what the data model definition expects.
type ParamMismatch struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Type     string `json:"type"` // "type_mismatch" or "writable_mismatch"
}
