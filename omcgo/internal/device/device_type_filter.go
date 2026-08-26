package device

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const (
	DeviceListDeviceTypeAll         = "ALL"
	DeviceListDeviceTypeBaseStation = "BASE_STATION"
	DeviceListDeviceTypeCoreNetwork = "CORE_NETWORK"
	DeviceListDeviceTypeUPS         = "UPS"
)

const upsProductClassPredicate = `COALESCE(d.product_class, '') LIKE 'UPS%'`
// coreNetworkProductClassPredicate 与 IsCoreNetworkProductClass 保持同一语义：
// ProductClass 包含 ImsCore（大小写不敏感）即核心网设备。核心网上报的 ProductClass
// 前后可能带其他字符串（如 CoreNetwork/ImsCore），不能做全等匹配。
const coreNetworkProductClassPredicate = `UPPER(COALESCE(d.product_class, '')) LIKE '%IMSCORE%'`

func normalizeDeviceListDeviceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return DeviceListDeviceTypeAll
	case "base_station", "basestation", "base-station", "radio", "station":
		return DeviceListDeviceTypeBaseStation
	case "core_network", "corenetwork", "core-network", "imscore", "core":
		return DeviceListDeviceTypeCoreNetwork
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
	case DeviceListDeviceTypeCoreNetwork:
		return sq.Expr(coreNetworkProductClassPredicate)
	case DeviceListDeviceTypeBaseStation:
		return sq.Expr("NOT (" + upsProductClassPredicate + ") AND NOT (" + coreNetworkProductClassPredicate + ")")
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
	if IsCoreNetworkProductClass(productClass) {
		return DeviceListDeviceTypeCoreNetwork
	}
	if isUPSProductClass(productClass) {
		return DeviceListDeviceTypeUPS
	}
	return DeviceListDeviceTypeBaseStation
}

// IsCoreNetworkProductClass is the shared ImsCore discriminator:
// case-insensitive containment, so prefixed/suffixed classes such as
// "CoreNetwork/ImsCore" still classify as core-network devices.
func IsCoreNetworkProductClass(productClass string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(productClass)), "IMSCORE")
}

// IsUPSProductClass is the shared UPS discriminator. UPS is intentionally
// identified only by Inform ProductClass prefix, not by radio technology.
func IsUPSProductClass(productClass string) bool {
	return isUPSProductClass(productClass)
}
