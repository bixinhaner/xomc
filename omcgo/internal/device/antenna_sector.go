package device

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

var antennaInfoPath = regexp.MustCompile(`^(.*\.AntennaInfo\.)([^.]+)$`)
var antennaFieldWithNumber = regexp.MustCompile(`^([A-Za-z]+)([0-9]+)$`)
var cellConfigInstancePath = regexp.MustCompile(`\.CellConfig\.([0-9]+)\.AntennaInfo\.`)

// AntennaSector is the normalized antenna geometry for one sector.
type AntennaSector struct {
	Number              int               `json:"number"`
	CellID              string            `json:"cell_id,omitempty"`
	AntennaHeight       *float64          `json:"antenna_height"`
	MechanicalDowntilt  *float64          `json:"mechanical_downtilt"`
	ElectronicDowntilt  string            `json:"electronic_downtilt,omitempty"`
	VerticalBeamwidth   *float64          `json:"vertical_beamwidth"`
	HorizontalBeamwidth *float64          `json:"horizontal_beamwidth"`
	Azimuth             *float64          `json:"azimuth"`
	NearRadiusMeters    *float64          `json:"near_radius_meters"`
	FarRadiusMeters     *float64          `json:"far_radius_meters"`
	FieldSources        map[string]string `json:"field_sources"`
	DirectionAvailable  bool              `json:"direction_available"`
	CoverageAvailable   bool              `json:"coverage_available"`
	MissingFields       []string          `json:"missing_fields"`
}

type antennaSectorRaw struct {
	sector AntennaSector
}

// AssembleAntennaSectors normalizes antenna geometry from dynamic device parameters.
func AssembleAntennaSectors(params []model.DeviceParameter) []AntennaSector {
	sectors := make(map[int]*antennaSectorRaw)
	for _, param := range params {
		matches := antennaInfoPath.FindStringSubmatch(param.ParameterPath)
		if matches == nil {
			continue
		}

		field, number := antennaFieldNumber(matches[2], param.ParameterPath)
		if field == "" {
			continue
		}
		if sectors[number] == nil {
			sectors[number] = &antennaSectorRaw{sector: AntennaSector{
				Number:       number,
				FieldSources: make(map[string]string),
			}}
		}
		assignAntennaField(&sectors[number].sector, field, param)
	}

	numbers := make([]int, 0, len(sectors))
	for number := range sectors {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)

	result := make([]AntennaSector, 0, len(numbers))
	for _, number := range numbers {
		sector := sectors[number].sector
		validateAntennaSector(&sector)
		result = append(result, sector)
	}
	return result
}

func antennaFieldNumber(field, path string) (string, int) {
	number := 1
	if instance := cellConfigInstancePath.FindStringSubmatch(path); instance != nil {
		parsed, err := strconv.Atoi(instance[1])
		if err == nil && parsed > 0 {
			number = parsed
		}
	}
	if match := antennaFieldWithNumber.FindStringSubmatch(field); match != nil {
		parsed, err := strconv.Atoi(match[2])
		if err != nil || parsed < 1 {
			return "", 0
		}
		return match[1], parsed
	}
	return field, number
}

func assignAntennaField(sector *AntennaSector, field string, param model.DeviceParameter) {
	field = canonicalAntennaField(field)
	sector.FieldSources[field] = param.ParameterPath

	switch field {
	case "cellid", "cellidentifier":
		sector.CellID = param.ParameterValue
	case "height", "antennaheight":
		if value, ok := parseAntennaNumber(param.ParameterValue); ok {
			sector.AntennaHeight = antennaFloat(value)
		} else {
			delete(sector.FieldSources, field)
		}
	case "downtilt", "mechanicaldowntilt":
		if value, ok := parseAntennaNumber(param.ParameterValue); ok {
			sector.MechanicalDowntilt = antennaFloat(value)
		} else {
			delete(sector.FieldSources, field)
		}
	case "electronicdowntilt":
		sector.ElectronicDowntilt = param.ParameterValue
	case "verticalbeamwidth":
		if value, ok := parseAntennaNumber(param.ParameterValue); ok {
			sector.VerticalBeamwidth = antennaFloat(value)
		} else {
			delete(sector.FieldSources, field)
		}
	case "beamwidth", "horizontalbeamwidth":
		if value, ok := parseAntennaNumber(param.ParameterValue); ok {
			sector.HorizontalBeamwidth = antennaFloat(value)
		} else {
			delete(sector.FieldSources, field)
		}
	case "azimuth":
		if value, ok := parseAntennaNumber(param.ParameterValue); ok {
			sector.Azimuth = antennaFloat(value)
		} else {
			delete(sector.FieldSources, field)
		}
	}
}

func canonicalAntennaField(field string) string {
	switch strings.ToLower(field) {
	case "antennaheight":
		return "height"
	case "mechanicaldowntilt":
		return "downtilt"
	case "horizontalbeamwidth":
		return "beamwidth"
	case "cellidentifier":
		return "cellid"
	default:
		return strings.ToLower(field)
	}
}

func parseAntennaNumber(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

func antennaFloat(value float64) *float64 {
	return &value
}

func validateAntennaSector(sector *AntennaSector) {
	sector.DirectionAvailable = false
	sector.CoverageAvailable = false
	sector.NearRadiusMeters = nil
	sector.FarRadiusMeters = nil
	sector.MissingFields = nil

	if sector.Azimuth != nil && *sector.Azimuth >= 0 && *sector.Azimuth < 360 {
		sector.DirectionAvailable = true
	} else {
		sector.MissingFields = append(sector.MissingFields, "azimuth")
	}

	if sector.AntennaHeight == nil || *sector.AntennaHeight <= 0 {
		sector.MissingFields = append(sector.MissingFields, "antennaHeight")
	}
	if sector.MechanicalDowntilt == nil || *sector.MechanicalDowntilt < 0 || *sector.MechanicalDowntilt >= 90 {
		sector.MissingFields = append(sector.MissingFields, "mechanicalDowntilt")
	}
	if sector.HorizontalBeamwidth == nil || *sector.HorizontalBeamwidth <= 0 || *sector.HorizontalBeamwidth >= 180 {
		sector.MissingFields = append(sector.MissingFields, "horizontalBeamwidth")
	}
	if sector.VerticalBeamwidth == nil || *sector.VerticalBeamwidth <= 0 || *sector.VerticalBeamwidth >= 180 {
		sector.MissingFields = append(sector.MissingFields, "verticalBeamwidth")
	}

	if len(sector.MissingFields) > 0 {
		return
	}

	nearAngle := *sector.MechanicalDowntilt + *sector.VerticalBeamwidth/2
	farAngle := *sector.MechanicalDowntilt - *sector.VerticalBeamwidth/2
	if farAngle <= 0 || nearAngle >= 90 {
		sector.MissingFields = append(sector.MissingFields, "coverageGeometry")
		return
	}

	nearRadius := *sector.AntennaHeight / math.Tan(degreesToRadians(nearAngle))
	farRadius := *sector.AntennaHeight / math.Tan(degreesToRadians(farAngle))
	if nearRadius > 0 && nearRadius < farRadius {
		sector.NearRadiusMeters = antennaFloat(nearRadius)
		sector.FarRadiusMeters = antennaFloat(farRadius)
		sector.CoverageAvailable = true
	}
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
