package cucc

import (
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// knownOUIProducts returns known OUI-ProductClass combinations for CUCC.
// CUCC only deploys NR (5G) small cells.
func knownOUIProducts(tech model.Technology) []carrier.OUIProductClassInfo {
	if tech != model.TechNR && tech != "" {
		return nil
	}

	return []carrier.OUIProductClassInfo{
		{OUI: "34CDBE", ProductClass: "SmallCell-NR-CU", ManufacturerName: "Ericsson", Description: "爱立信 5G 小基站（联通）", Technology: model.TechNR},
		{OUI: "A4BA76", ProductClass: "SmallCell-NR-CU", ManufacturerName: "Nokia", Description: "诺基亚 5G 小基站（联通）", Technology: model.TechNR},
		{OUI: "B0B2DC", ProductClass: "SmallCell-NR-CU", ManufacturerName: "Samsung", Description: "三星 5G 小基站（联通）", Technology: model.TechNR},
	}
}

// provisioningTemplates returns default provisioning templates for CUCC.
// CUCC uses PLMN 46001 and only supports NR.
func provisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	if tech != model.TechNR {
		return nil
	}

	return []*carrier.ProvisionTemplate{
		{
			Name:       "cucc_nr_default",
			Technology: model.TechNR,
			Parameters: map[string]interface{}{
				"Device.ManagementServer.PeriodicInformEnable":                              "true",
				"Device.ManagementServer.PeriodicInformInterval":                            "300",
				"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID":        "46001",
				"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.Enable":        "true",
				"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST":             "1",
				"Device.Services.FAPService.1.FAPControl.NR.AdminState":                     "true",
				"Device.X_CUCC.PMEnable":                                                    "true",
				"Device.X_CUCC.PMInterval":                                                  "15",
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
}
