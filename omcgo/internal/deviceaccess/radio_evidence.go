package deviceaccess

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/pkg/tr069"
)

const (
	radioEvidenceLifetime   = time.Hour
	maxServingCellInstances = 16
	maxLTETAC               = 0xffff
)

var (
	lteTACPathPattern      = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.EPC\.TAC$`)
	ltePLMNPathPattern     = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.EPC\.PLMNList\.1\.PLMNID$`)
	ltePLMNPrimaryPattern  = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.EPC\.PLMNList\.1\.IsPrimary$`)
	lteServingPLMNsPattern = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.FAPControl\.LTE\.Gateway\.ExistPlmnidList$`)
	lteECIPathPattern      = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.RAN\.Common\.CellIdentity$`)
)

type radioPathKind string

const (
	radioPathTAC          radioPathKind = "tac"
	radioPathPLMN         radioPathKind = "plmn"
	radioPathPLMNPrimary  radioPathKind = "plmn_primary"
	radioPathServingPLMNs radioPathKind = "serving_plmns"
	radioPathECI          radioPathKind = "eci"
)

type RadioParameterPath struct {
	StandardPath string        `json:"standard_path"`
	PrivatePath  string        `json:"private_path"`
	FAPInstance  int           `json:"fap_instance"`
	Kind         radioPathKind `json:"kind"`
}

type radioEvidenceValue struct {
	TACs      []string `json:"tacs,omitempty"`
	ECGIs     []string `json:"ecgis,omitempty"`
	Instances []int    `json:"instances,omitempty"`
}

type radioEvidencePayload struct {
	Values    []string `json:"values"`
	Instances []int    `json:"fap_instances"`
}

type servingCellValues struct {
	tac          string
	plmn         string
	plmnPrimary  string
	servingPLMNs string
	eci          string
}

func normalizeGPVRadioEvidence(
	paths []RadioParameterPath,
	parameters []tr069.ParameterValueStruct,
	needTAC bool,
	needECGI bool,
) (radioEvidenceValue, error) {
	values := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		values[strings.TrimSpace(parameter.Name)] = strings.TrimSpace(parameter.Value)
	}
	cells := make(map[int]*servingCellValues)
	for _, path := range paths {
		value := values[path.PrivatePath]
		if value == "" {
			value = values[path.StandardPath]
		}
		if value == "" {
			continue
		}
		cell := cells[path.FAPInstance]
		if cell == nil {
			cell = &servingCellValues{}
			cells[path.FAPInstance] = cell
		}
		switch path.Kind {
		case radioPathTAC:
			cell.tac = value
		case radioPathPLMN:
			cell.plmn = value
		case radioPathPLMNPrimary:
			cell.plmnPrimary = value
		case radioPathServingPLMNs:
			cell.servingPLMNs = value
		case radioPathECI:
			cell.eci = value
		}
	}
	normalized := normalizeServingCells(cells)
	expected := make(map[int]map[radioPathKind]bool)
	for _, path := range paths {
		if expected[path.FAPInstance] == nil {
			expected[path.FAPInstance] = make(map[radioPathKind]bool)
		}
		expected[path.FAPInstance][path.Kind] = true
	}
	for instance, kinds := range expected {
		cell := cells[instance]
		if needTAC && kinds[radioPathTAC] {
			if cell == nil {
				return radioEvidenceValue{}, fmt.Errorf("TAC for FAPService.%d is missing: %w", instance, ErrAccessEvidenceUnavailable)
			}
			if _, ok := normalizeUnsigned(cell.tac, maxLTETAC); !ok {
				return radioEvidenceValue{}, fmt.Errorf("TAC for FAPService.%d is missing or invalid: %w", instance, ErrAccessEvidenceUnavailable)
			}
		}
		if needECGI && (kinds[radioPathPLMN] || kinds[radioPathPLMNPrimary] || kinds[radioPathServingPLMNs] || kinds[radioPathECI]) {
			if cell == nil {
				return radioEvidenceValue{}, fmt.Errorf("ECGI for FAPService.%d is missing: %w", instance, ErrAccessEvidenceUnavailable)
			}
			if kinds[radioPathServingPLMNs] {
				if _, ok := normalizeServingPLMNs(cell.servingPLMNs); !ok {
					return radioEvidenceValue{}, fmt.Errorf("serving PLMN list for FAPService.%d is missing or invalid: %w", instance, ErrAccessEvidenceUnavailable)
				}
			} else if !normalizedBool(cell.plmnPrimary) {
				return radioEvidenceValue{}, fmt.Errorf("primary PLMN marker for FAPService.%d is missing or false: %w", instance, ErrAccessEvidenceUnavailable)
			}
			if strings.TrimSpace(cell.eci) == "" {
				return radioEvidenceValue{}, fmt.Errorf("ECGI for FAPService.%d is missing or invalid: %w", instance, ErrAccessEvidenceUnavailable)
			}
			if kinds[radioPathServingPLMNs] {
				plmns, _ := normalizeServingPLMNs(cell.servingPLMNs)
				for _, plmn := range plmns {
					if _, ok := normalizeECGI(plmn, cell.eci); !ok {
						return radioEvidenceValue{}, fmt.Errorf("ECGI for FAPService.%d is missing or invalid: %w", instance, ErrAccessEvidenceUnavailable)
					}
				}
			} else if _, ok := normalizeECGI(cell.plmn, cell.eci); !ok {
				return radioEvidenceValue{}, fmt.Errorf("ECGI for FAPService.%d is missing or invalid: %w", instance, ErrAccessEvidenceUnavailable)
			}
		}
	}
	if needTAC && len(normalized.TACs) == 0 {
		return radioEvidenceValue{}, fmt.Errorf("TAC is missing or invalid: %w", ErrAccessEvidenceUnavailable)
	}
	if needECGI && len(normalized.ECGIs) == 0 {
		return radioEvidenceValue{}, fmt.Errorf("ECGI is missing or invalid: %w", ErrAccessEvidenceUnavailable)
	}
	return normalized, nil
}

func normalizeServingCells(cells map[int]*servingCellValues) radioEvidenceValue {
	var result radioEvidenceValue
	instances := make([]int, 0, len(cells))
	for instance, cell := range cells {
		instances = append(instances, instance)
		if tac, ok := normalizeUnsigned(cell.tac, maxLTETAC); ok {
			result.TACs = append(result.TACs, tac)
		}
		if plmns, ok := normalizeServingPLMNs(cell.servingPLMNs); ok {
			for _, plmn := range plmns {
				if ecgi, valid := normalizeECGI(plmn, cell.eci); valid {
					result.ECGIs = append(result.ECGIs, ecgi)
				}
			}
		} else if ecgi, ok := normalizeECGI(cell.plmn, cell.eci); ok && normalizedBool(cell.plmnPrimary) {
			result.ECGIs = append(result.ECGIs, ecgi)
		}
	}
	result.TACs = sortedUnique(result.TACs)
	result.ECGIs = sortedUnique(result.ECGIs)
	result.Instances = sortedPositiveInstances(instances)
	return result
}

func normalizedBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func normalizeECGI(plmn, eci string) (string, bool) {
	plmn, ok := normalizePLMN(plmn)
	if !ok {
		return "", false
	}
	normalizedECI, ok := normalizeUnsigned(eci, 0x0fffffff)
	if !ok {
		return "", false
	}
	return plmn[:3] + "-" + plmn[3:] + "-" + normalizedECI, true
}

func normalizeServingPLMNs(value string) ([]string, bool) {
	parts := strings.FieldsFunc(value, func(char rune) bool {
		return char == ',' || char == ';' || char == '\n' || char == '\r'
	})
	if len(parts) == 0 {
		return nil, false
	}
	plmns := make([]string, 0, len(parts))
	for _, part := range parts {
		plmn, ok := normalizePLMN(part)
		if !ok {
			return nil, false
		}
		plmns = append(plmns, plmn)
	}
	return sortedUnique(plmns), true
}

func normalizePLMN(value string) (string, bool) {
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.TrimSpace(value))
	if (len(value) != 5 && len(value) != 6) || !allDecimal(value) {
		return "", false
	}
	return value, true
}

func normalizeUnsigned(value string, maximum uint64) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 0
	}
	parsed, err := strconv.ParseUint(value, base, 64)
	if err != nil || parsed > maximum {
		return "", false
	}
	return strconv.FormatUint(parsed, 10), true
}

func allDecimal(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return value != ""
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func radioEvidenceRecords(value radioEvidenceValue, source string, observedAt time.Time) ([]EvidenceRecord, error) {
	expiresAt := observedAt.Add(radioEvidenceLifetime)
	records := make([]EvidenceRecord, 0, 2)
	for _, item := range []struct {
		typeName ConditionType
		values   []string
	}{
		{typeName: ConditionTypeTAC, values: value.TACs},
		{typeName: ConditionTypeECGI, values: value.ECGIs},
	} {
		if len(item.values) == 0 {
			continue
		}
		raw, err := json.Marshal(radioEvidencePayload{Values: item.values, Instances: value.Instances})
		if err != nil {
			return nil, fmt.Errorf("encode %s access evidence: %w", item.typeName, err)
		}
		records = append(records, EvidenceRecord{
			Type:            item.typeName,
			NormalizedValue: raw,
			ValueHash:       evidenceHash(raw),
			Source:          source,
			ObservedAt:      observedAt,
			ExpiresAt:       &expiresAt,
		})
	}
	return records, nil
}

func mergeEvidenceRecords(current []EvidenceRecord, replacements []EvidenceRecord) []EvidenceRecord {
	byType := make(map[ConditionType]EvidenceRecord, len(current)+len(replacements))
	for _, record := range current {
		byType[record.Type] = record
	}
	for _, record := range replacements {
		byType[record.Type] = record
	}
	types := make([]ConditionType, 0, len(byType))
	for evidenceType := range byType {
		types = append(types, evidenceType)
	}
	slices.Sort(types)
	merged := make([]EvidenceRecord, 0, len(types))
	for _, evidenceType := range types {
		merged = append(merged, byType[evidenceType])
	}
	return merged
}
