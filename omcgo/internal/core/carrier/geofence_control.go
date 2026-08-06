package carrier

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

type GeofenceControlParameter struct {
	Path  string
	Value string
}

type GeofenceControlParameterInstanceResolver interface {
	GeofenceControlParametersForInstances(
		productClass string,
		tech model.Technology,
		enabled bool,
		instances []int,
	) ([]GeofenceControlParameter, error)
}

func BuildGeofenceControlParametersForInstances(
	productClass string,
	tech model.Technology,
	enabled bool,
	instances []int,
) ([]GeofenceControlParameter, error) {
	productClass = strings.ToUpper(strings.TrimSpace(productClass))
	if len(instances) == 0 {
		return nil, fmt.Errorf("FAPService instances are required")
	}
	value := "0"
	if enabled {
		value = "1"
	}

	const rfPathTemplate = "Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus"
	switch {
	case productClass == "BM" && tech == model.TechGSM:
		return nil, fmt.Errorf("BM GSM RfState is read-only and cannot be used for control")
	case tech == model.TechLTE && (productClass == "BLQ" || productClass == "MLQ" ||
		productClass == "BLN" || productClass == "MLN" || productClass == "BM"):
		// Always enqueue the standard model path. The ACS translator resolves it
		// to the product-private X_COM_RadioEnable or AdminCellState path.
	default:
		return nil, fmt.Errorf("no geofence RF control path for product class %q and technology %q", productClass, tech)
	}

	parameters := []GeofenceControlParameter{
		{Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: value},
	}
	for _, instance := range instances {
		if instance <= 0 {
			return nil, fmt.Errorf("FAPService instance must be positive")
		}
		parameters = append(parameters, GeofenceControlParameter{
			Path:  fmt.Sprintf(rfPathTemplate, instance),
			Value: value,
		})
	}
	return parameters, nil
}

func DetectGeofenceControlInstances(
	parameters []model.DeviceParameter,
	productClass string,
	tech model.Technology,
) ([]int, error) {
	productClass = strings.ToUpper(strings.TrimSpace(productClass))
	var supported bool
	switch {
	case tech == model.TechLTE && (productClass == "BLQ" || productClass == "MLQ" ||
		productClass == "BLN" || productClass == "MLN" || productClass == "BM"):
		supported = true
	default:
		return nil, fmt.Errorf("no geofence RF control parameter for product class %q and technology %q", productClass, tech)
	}
	if !supported {
		return nil, fmt.Errorf("no geofence RF control parameter for product class %q and technology %q", productClass, tech)
	}
	pattern := regexp.MustCompile(
		`^Device\.Services\.FAPService\.([0-9]+)\.(?:` +
			`FAPControl\.LTE\.RFTxStatus|` +
			`CellConfig\.LTE\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)` +
			`)$`,
	)
	instances := make([]int, 0)
	seen := make(map[int]struct{})
	for _, parameter := range parameters {
		if !parameter.Writable {
			continue
		}
		matches := pattern.FindStringSubmatch(parameter.ParameterPath)
		if len(matches) != 2 {
			continue
		}
		var instance int
		if _, err := fmt.Sscanf(matches[1], "%d", &instance); err != nil || instance <= 0 {
			continue
		}
		if _, ok := seen[instance]; ok {
			continue
		}
		seen[instance] = struct{}{}
		instances = append(instances, instance)
	}
	sort.Ints(instances)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no writable geofence RF instances found for product class %q", productClass)
	}
	return instances, nil
}
