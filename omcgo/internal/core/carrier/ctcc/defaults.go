package ctcc

import (
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// knownOUIProducts returns known OUI-ProductClass combinations for CTCC.
// CTCC shares many of the same hardware vendors as CMCC but has its own
// product class naming conventions.
func knownOUIProducts(tech model.Technology) []carrier.OUIProductClassInfo {
	var results []carrier.OUIProductClassInfo

	if tech == model.TechLTE || tech == "" {
		results = append(results, []carrier.OUIProductClassInfo{
			{OUI: "00E0FC", ProductClass: "eSmallCell-LTE", ManufacturerName: "Huawei", Description: "华为 LTE 小基站（电信）", Technology: model.TechLTE},
			{OUI: "001E7E", ProductClass: "FemtoCell-LTE-CT", ManufacturerName: "ZTE", Description: "中兴 LTE 飞基站（电信）", Technology: model.TechLTE},
			{OUI: "58FB96", ProductClass: "Pico-LTE-CT", ManufacturerName: "Comba", Description: "京信 LTE 皮基站（电信）", Technology: model.TechLTE},
		}...)
	}

	if tech == model.TechNR || tech == "" {
		results = append(results, []carrier.OUIProductClassInfo{
			{OUI: "00E0FC", ProductClass: "eSmallCell-NR", ManufacturerName: "Huawei", Description: "华为 5G 小基站（电信）", Technology: model.TechNR},
			{OUI: "001E7E", ProductClass: "SmallCell-NR-CT", ManufacturerName: "ZTE", Description: "中兴 5G 小基站（电信）", Technology: model.TechNR},
			{OUI: "58FB96", ProductClass: "Pico-NR-CT", ManufacturerName: "Comba", Description: "京信 5G 皮基站（电信）", Technology: model.TechNR},
		}...)
	}

	return results
}

// provisioningTemplates returns default provisioning templates for CTCC.
// CTCC uses PLMNID 46011 and X_CTCC private extensions.
func provisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	switch tech {
	case model.TechLTE:
		return []*carrier.ProvisionTemplate{
			{
				Name:       "ctcc_lte_default",
				Technology: model.TechLTE,
				Parameters: map[string]interface{}{
					"Device.ManagementServer.PeriodicInformEnable":                             "true",
					"Device.ManagementServer.PeriodicInformInterval":                           "300",
					"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID":       "46011",
					"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.Enable":       "true",
					"Device.Services.FAPService.1.FAPControl.LTE.AdminState":                   "true",
					"Device.X_CTCC.PMEnable":                                                   "true",
					"Device.X_CTCC.PMInterval":                                                 "15",
				},
				Required: []string{
					"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
					"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL",
					"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID",
					"Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
					"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer1",
				},
			},
		}
	case model.TechNR:
		return []*carrier.ProvisionTemplate{
			{
				Name:       "ctcc_nr_default",
				Technology: model.TechNR,
				Parameters: map[string]interface{}{
					"Device.ManagementServer.PeriodicInformEnable":                            "true",
					"Device.ManagementServer.PeriodicInformInterval":                          "300",
					"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID":      "46011",
					"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.Enable":      "true",
					"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST":           "1",
					"Device.Services.FAPService.1.FAPControl.NR.AdminState":                    "true",
					"Device.X_CTCC.PMEnable":                                                   "true",
					"Device.X_CTCC.PMInterval":                                                 "15",
				},
				Required: []string{
					"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.gNBId",
					"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.NRARFCN",
					"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRPCI",
					"Device.Services.FAPService.1.CellConfig.NR.Core.TAC",
					"Device.Services.FAPService.1.FAPControl.NR.Gateway.SecGWServer1",
				},
			},
		}
	default:
		return nil
	}
}
