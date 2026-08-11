package license

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LegacyLicenseClaims is the normalized, typed representation of the legacy
// LicenseContent fields. LegacyExtra keeps fields that the current OMC model
// does not understand yet.
type LegacyLicenseClaims struct {
	LicenseID        string
	LicenseType      SystemLicenseType
	IssuedAt         time.Time
	StandardNotAfter *time.Time
	OMCNotAfter      *time.Time
	TimeLimitHours   int
	DevicesSupport   DevicesSupport
	MACAddress       string
	SystemUUID       string
	HardwareLimit    string
	IsCloud          bool
	FeatureIDs       []string
	FeatureCodes     []string
	LegacyExtra      map[string]LegacyXMLValue
	UnknownFields    map[string]LegacyXMLValue
	RawSHA256        string
}

// MapLegacyTrueLicense converts a decoded TrueLicense artifact without applying
// host binding or time policy. Those checks belong to their own validators.
func MapLegacyTrueLicense(artifact *LegacyTrueLicenseArtifact) (*LegacyLicenseClaims, error) {
	if artifact == nil {
		return nil, fmt.Errorf("legacy license artifact is nil")
	}
	claims := &LegacyLicenseClaims{
		DevicesSupport: make(DevicesSupport),
		LegacyExtra:    cloneLegacyValues(artifact.Extra),
		UnknownFields:  cloneLegacyValues(artifact.UnknownFields),
		RawSHA256:      artifact.SHA256,
	}
	claims.LicenseID = legacyString(artifact.Extra, "licenseNum")
	if claims.LicenseID == "" {
		return nil, fmt.Errorf("legacy extra.licenseNum is required")
	}

	productType := legacyString(artifact.Extra, "productType")
	claims.IsCloud = productType == "1"
	useFor := legacyString(artifact.Extra, "useFor")
	switch {
	case useFor == "1":
		claims.LicenseType = SystemLicenseTypeCommercial
	case productType == "1" || productType == "2":
		claims.LicenseType = SystemLicenseTypeTrial
	default:
		return nil, fmt.Errorf("legacy productType %q/useFor %q cannot determine license type", productType, useFor)
	}

	var err error
	claims.IssuedAt, err = legacyStandardTime(artifact.StandardFields, "issued")
	if err != nil {
		return nil, err
	}
	if value := legacyStandardTimeOptional(artifact.StandardFields, "notAfter"); value != nil {
		claims.StandardNotAfter = value
	}
	if raw := legacyString(artifact.Extra, "omcNotAfter"); raw != "" {
		parsed, parseErr := time.ParseInLocation("2006-01-02 15:04:05", raw, time.UTC)
		if parseErr != nil {
			return nil, fmt.Errorf("parse legacy omcNotAfter %q: %w", raw, parseErr)
		}
		claims.OMCNotAfter = &parsed
	}

	claims.TimeLimitHours, err = legacyInt(artifact.Extra, "timeLimit")
	if err != nil {
		return nil, err
	}
	for field, key := range map[string]string{
		"eNB": "maxDeviceCapability",
		"gNB": "gnbMaxNum",
		"EPC": "epcMaxNum",
		"CPE": "cpeMaxNum",
		"EGW": "egwMaxNum",
		"UPS": "upsMaxNum",
	} {
		value, valueErr := legacyIntOptional(artifact.Extra, key)
		if valueErr != nil {
			return nil, valueErr
		}
		if value != nil {
			claims.DevicesSupport[field] = *value
		}
	}
	if claims.DevicesSupport["gNB"] == 0 && containsLegacyCode(artifact.Extra, "CODE_GNB") {
		claims.DevicesSupport["gNB"] = claims.DevicesSupport["eNB"]
	}

	claims.MACAddress = legacyString(artifact.Extra, "MACAddress")
	claims.SystemUUID = legacyString(artifact.Extra, "systemUUID")
	claims.HardwareLimit = legacyString(artifact.Extra, "hardwareLimit")
	claims.FeatureIDs = splitLegacyList(legacyString(artifact.Extra, "supportFeatureIdStr"))
	claims.FeatureCodes = splitLegacyList(legacyString(artifact.Extra, "supportFeatureCode"))
	return claims, nil
}

func legacyString(values map[string]LegacyXMLValue, key string) string {
	return strings.TrimSpace(values[key].Value)
}

func legacyInt(values map[string]LegacyXMLValue, key string) (int, error) {
	value := legacyString(values, key)
	if value == "" {
		return 0, fmt.Errorf("legacy extra.%s is required", key)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse legacy extra.%s %q: %w", key, value, err)
	}
	return parsed, nil
}

func legacyIntOptional(values map[string]LegacyXMLValue, key string) (*int, error) {
	value := legacyString(values, key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("parse legacy extra.%s %q: %w", key, value, err)
	}
	return &parsed, nil
}

func legacyStandardTime(values map[string]LegacyXMLValue, key string) (time.Time, error) {
	value := strings.TrimSpace(values[key].Value)
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse legacy standard field %s %q: %w", key, value, err)
	}
	return time.UnixMilli(millis).UTC(), nil
}

func legacyStandardTimeOptional(values map[string]LegacyXMLValue, key string) *time.Time {
	value := strings.TrimSpace(values[key].Value)
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil || value == "" {
		return nil
	}
	parsed := time.UnixMilli(millis).UTC()
	return &parsed
}

func splitLegacyList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func containsLegacyCode(values map[string]LegacyXMLValue, code string) bool {
	for _, item := range splitLegacyList(legacyString(values, "supportFeatureCode")) {
		if item == code {
			return true
		}
	}
	return false
}

func cloneLegacyValues(values map[string]LegacyXMLValue) map[string]LegacyXMLValue {
	clone := make(map[string]LegacyXMLValue, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

// LegacyClaimsJSON returns a diagnostic-safe JSON representation for tests and
// audit tooling; it does not include the sensitive encryptKey field.
func (c *LegacyLicenseClaims) LegacyClaimsJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	extra := cloneLegacyValues(c.LegacyExtra)
	delete(extra, "encryptKey")
	return json.Marshal(struct {
		LicenseID      string                    `json:"license_id"`
		LicenseType    SystemLicenseType         `json:"license_type"`
		IssuedAt       time.Time                 `json:"issued_at"`
		OMCNotAfter    *time.Time                `json:"omc_not_after,omitempty"`
		DevicesSupport DevicesSupport            `json:"devices_support"`
		FeatureIDs     []string                  `json:"feature_ids,omitempty"`
		FeatureCodes   []string                  `json:"feature_codes,omitempty"`
		LegacyExtra    map[string]LegacyXMLValue `json:"legacy_extra"`
	}{c.LicenseID, c.LicenseType, c.IssuedAt, c.OMCNotAfter, c.DevicesSupport, c.FeatureIDs, c.FeatureCodes, extra})
}
