package device

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

type radioFrequencyProjection struct {
	dlValue    string
	ulValue    string
	dlObserved bool
	ulObserved bool
	complete   bool
	reason     string
}

// RadioFrequencyProjection exposes the device-list DL/UL frequency projection
// to compatibility facades that need the same physical-cell semantics.
type RadioFrequencyProjection struct {
	DLValue    string
	ULValue    string
	DLObserved bool
	ULObserved bool
	Complete   bool
	Reason     string
}

func ProjectRadioFrequencyValues(
	params map[string]string,
	tech model.Technology,
	productClass string,
) RadioFrequencyProjection {
	projected := projectRadioFrequencyFields(params, tech, productClass)
	return RadioFrequencyProjection{
		DLValue:    projected.dlValue,
		ULValue:    projected.ulValue,
		DLObserved: projected.dlObserved,
		ULObserved: projected.ulObserved,
		Complete:   projected.complete,
		Reason:     projected.reason,
	}
}

var (
	lteRFDLEARFCNPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.RAN\.RF\.EARFCNDL$`,
	)
	lteCommonDLEARFCNPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.RAN\.Common\.EARFCNDL$`,
	)
	lteULEARFCNPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.RAN\.RF\.EARFCNUL$`,
	)
	nrDLEARFCNPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.1\.CellConfig\.(\d+)\.NR\.RAN\.RF\.NRARFCNDL$`,
	)
	nrULEARFCNPattern = regexp.MustCompile(
		`^Device\.Services\.FAPService\.1\.CellConfig\.(\d+)\.NR\.RAN\.RF\.NRARFCNUL$`,
	)
)

// projectRadioFrequencyFields projects one positional DL/UL value per physical
// LTE/NR cell. The output intentionally preserves duplicate values across cells.
func projectRadioFrequencyFields(
	params map[string]string,
	tech model.Technology,
	productClass string,
) radioFrequencyProjection {
	dlValues, ulValues, dlObserved, ulObserved := collectRadioFrequencyValues(params, tech)
	result := radioFrequencyProjection{
		dlObserved: dlObserved,
		ulObserved: ulObserved,
		complete:   true,
	}
	if !dlObserved && !ulObserved {
		return result
	}

	expectedCount, countKnown, countReason, countInconsistent :=
		resolveExpectedPhysicalCarrierCount(params, productClass)
	_, _, carrierAggregation := productClassCarrierCount(productClass)
	if countInconsistent || (carrierAggregation && !countKnown) {
		result.complete = false
		result.reason = countReason
		return result
	}

	var dlComplete, ulComplete bool
	var dlReason, ulReason string
	result.dlValue, dlComplete, dlReason = projectFrequencyValues(
		"DL", dlValues, dlObserved, expectedCount, countKnown,
	)
	result.ulValue, ulComplete, ulReason = projectFrequencyValues(
		"UL", ulValues, ulObserved, expectedCount, countKnown,
	)
	result.complete = dlComplete && ulComplete
	result.reason = strings.Join(nonEmptyStrings(dlReason, ulReason), "; ")
	return result
}

func collectRadioFrequencyValues(
	params map[string]string,
	tech model.Technology,
) (dlValues, ulValues map[int]string, dlObserved, ulObserved bool) {
	dlValues = make(map[int]string)
	ulValues = make(map[int]string)

	switch tech {
	case model.TechLTE:
		rfDLValues := make(map[int]string)
		commonDLValues := make(map[int]string)
		for path, rawValue := range params {
			switch {
			case lteRFDLEARFCNPattern.MatchString(path):
				dlObserved = true
				storeIndexedFrequencyValue(rfDLValues, lteRFDLEARFCNPattern, path, rawValue)
			case lteCommonDLEARFCNPattern.MatchString(path):
				dlObserved = true
				storeIndexedFrequencyValue(commonDLValues, lteCommonDLEARFCNPattern, path, rawValue)
			case lteULEARFCNPattern.MatchString(path):
				ulObserved = true
				storeIndexedFrequencyValue(ulValues, lteULEARFCNPattern, path, rawValue)
			}
		}
		for index, value := range commonDLValues {
			dlValues[index] = value
		}
		for index, value := range rfDLValues {
			if value != "" || dlValues[index] == "" {
				dlValues[index] = value
			}
		}
	case model.TechNR:
		for path, rawValue := range params {
			switch {
			case nrDLEARFCNPattern.MatchString(path):
				dlObserved = true
				storeIndexedFrequencyValue(dlValues, nrDLEARFCNPattern, path, rawValue)
			case nrULEARFCNPattern.MatchString(path):
				ulObserved = true
				storeIndexedFrequencyValue(ulValues, nrULEARFCNPattern, path, rawValue)
			}
		}
	}

	return dlValues, ulValues, dlObserved, ulObserved
}

func storeIndexedFrequencyValue(values map[int]string, pattern *regexp.Regexp, path, rawValue string) {
	matches := pattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return
	}
	index, err := strconv.Atoi(matches[1])
	if err != nil || index < 1 {
		return
	}
	values[index] = strings.TrimSpace(rawValue)
}

func projectFrequencyValues(
	label string,
	values map[int]string,
	observed bool,
	expectedCount int,
	countKnown bool,
) (value string, complete bool, reason string) {
	if !observed {
		return "", true, ""
	}

	if countKnown {
		if expectedCount == 1 {
			if singleValue, ok := singleObservedFrequencyValue(values); ok {
				return singleValue, true, ""
			}
		}
		projected := make([]string, 0, expectedCount)
		for index := 1; index <= expectedCount; index++ {
			cellValue := values[index]
			if cellValue == "" {
				return "", false, fmt.Sprintf("%s index %d is missing", label, index)
			}
			projected = append(projected, cellValue)
		}
		return strings.Join(projected, ","), true, ""
	}

	indices := make([]int, 0, len(values))
	for index, cellValue := range values {
		if cellValue != "" {
			indices = append(indices, index)
		}
	}
	sort.Ints(indices)
	if len(indices) == 0 {
		return "", false, fmt.Sprintf("%s has no non-empty indexed value", label)
	}

	projected := make([]string, 0, len(indices))
	for _, index := range indices {
		projected = append(projected, values[index])
	}
	return strings.Join(projected, ","), true, ""
}

func singleObservedFrequencyValue(values map[int]string) (string, bool) {
	value := ""
	for _, rawValue := range values {
		cellValue := strings.TrimSpace(rawValue)
		if cellValue == "" {
			continue
		}
		if value != "" {
			return "", false
		}
		value = cellValue
	}
	return value, value != ""
}

func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
