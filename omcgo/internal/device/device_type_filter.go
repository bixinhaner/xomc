package device

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const (
	DeviceListDeviceTypeAll         = "ALL"
	DeviceListDeviceTypeBaseStation = "BASE_STATION"
	DeviceListDeviceTypeUPS         = "UPS"
)

const upsProductClassPredicate = `COALESCE(d.product_class, '') LIKE 'UPS%'`

func normalizeDeviceListDeviceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return DeviceListDeviceTypeAll
	case "base_station", "basestation", "base-station", "radio", "station":
		return DeviceListDeviceTypeBaseStation
	case "ups":
		return DeviceListDeviceTypeUPS
	default:
		return ""
	}
}

func deviceListDeviceTypeFilterCond(value string) sq.Sqlizer {
	switch normalizeDeviceListDeviceType(value) {
	case DeviceListDeviceTypeUPS:
		return sq.Expr(upsProductClassPredicate)
	case DeviceListDeviceTypeBaseStation:
		return sq.Expr("NOT (" + upsProductClassPredicate + ")")
	default:
		return nil
	}
}

func applyDeviceListDeviceTypeFilter(b sq.SelectBuilder, value string) sq.SelectBuilder {
	if cond := deviceListDeviceTypeFilterCond(value); cond != nil {
		return b.Where(cond)
	}
	return b
}

func deviceListDeviceTypeFromProductClass(productClass string) string {
	if isUPSProductClass(productClass) {
		return DeviceListDeviceTypeUPS
	}
	return DeviceListDeviceTypeBaseStation
}

// IsUPSProductClass is the shared UPS discriminator. UPS is intentionally
// identified only by Inform ProductClass prefix, not by radio technology.
func IsUPSProductClass(productClass string) bool {
	return isUPSProductClass(productClass)
}
