package cmcc

import (
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/model"
)

// knownOUIProducts returns known OUI-ProductClass combinations for CMCC.
func knownOUIProducts(tech model.Technology) []carrier.OUIProductClassInfo {
	var results []carrier.OUIProductClassInfo

	if tech == model.TechLTE || tech == "" {
		results = append(results, []carrier.OUIProductClassInfo{
			{OUI: "00E0FC", ProductClass: "SmallCell-LTE", ManufacturerName: "Huawei", Description: "华�� LTE 小基站", Technology: model.TechLTE},
			{OUI: "001E7E", ProductClass: "FemtoCell-LTE", ManufacturerName: "ZTE", Description: "中兴 LTE 飞基站", Technology: model.TechLTE},
			{OUI: "58FB96", ProductClass: "Pico-LTE", ManufacturerName: "Comba", Description: "京信 LTE 皮基站", Technology: model.TechLTE},
			{OUI: "D4612E", ProductClass: "SmallCell-LTE", ManufacturerName: "Datang", Description: "大唐 LTE 小基站", Technology: model.TechLTE},
		}...)
	}

	if tech == model.TechNR || tech == "" {
		results = append(results, []carrier.OUIProductClassInfo{
			{OUI: "00E0FC", ProductClass: "SmallCell-NR", ManufacturerName: "Huawei", Description: "华为 5G 小基站", Technology: model.TechNR},
			{OUI: "001E7E", ProductClass: "SmallCell-NR", ManufacturerName: "ZTE", Description: "中兴 5G 小基站", Technology: model.TechNR},
			{OUI: "58FB96", ProductClass: "Pico-NR", ManufacturerName: "Comba", Description: "京信 5G 皮基站", Technology: model.TechNR},
		}...)
	}

	return results
}

// provisioningTemplates returns default provisioning templates for CMCC.
func provisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	switch tech {
	case model.TechLTE:
		return []*carrier.ProvisionTemplate{
			{
				Name:       "cmcc_lte_default",
				Technology: model.TechLTE,
				Parameters: map[string]interface{}{
					"Device.ManagementServer.PeriodicInformEnable":   "true",
					"Device.ManagementServer.PeriodicInformInterval": "300",
					"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID":  "46000",
					"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.Enable":   "true",
					"Device.Services.FAPService.1.FAPControl.LTE.AdminState":               "true",
					"Device.X_CMCC.PMEnable":  "true",
					"Device.X_CMCC.PMInterval": "15",
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
				Name:       "cmcc_nr_default",
				Technology: model.TechNR,
				Parameters: map[string]interface{}{
					"Device.ManagementServer.PeriodicInformEnable":   "true",
					"Device.ManagementServer.PeriodicInformInterval": "300",
					"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID": "46000",
					"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.Enable": "true",
					"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST":     "1",
					"Device.Services.FAPService.1.FAPControl.NR.AdminState":              "true",
					"Device.X_CMCC.PMEnable":  "true",
					"Device.X_CMCC.PMInterval": "15",
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
