package ctcc

// buildParamMappings creates bidirectional parameter path mappings
// between CTCC-specific paths and unified internal names.
// CTCC uses Device. prefix with X_CTCC_ private extensions.
func buildParamMappings() (forward map[string]string, reverse map[string]string) {
	mappings := map[string]string{
		// CTCC LTE extensions
		"Device.X_CTCC.ENBId":      "enb_id",
		"Device.X_CTCC.ENBName":    "enb_name",
		"Device.X_CTCC.SiteName":   "site_name",
		"Device.X_CTCC.SiteId":     "site_id",
		"Device.X_CTCC.Region":     "region",
		"Device.X_CTCC.PMEnable":   "pm_enable",
		"Device.X_CTCC.PMInterval": "pm_interval",

		// CTCC 5G NR extensions
		"Device.X_CTCC.gNBName": "gnb_name",

		// Standard parameters — LTE (same TR-069 paths as other carriers)
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity":          "cell_id",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL":              "earfcn_dl",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNUL":              "earfcn_ul",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID":                 "pci",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower":      "rs_power",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth":               "dl_bandwidth",
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULBandwidth":               "ul_bandwidth",
		"Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC":                          "tac",
		"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID":            "plmn_id",
		"Device.Services.FAPService.1.FAPControl.LTE.AdminState":                        "admin_state",
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus":                        "rf_tx_status",
		"Device.Services.FAPService.1.FAPControl.LTE.OpState":                           "op_state",
		"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer1":              "secgw_server1",
		"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer2":              "secgw_server2",

		// NR parameters
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.gNBId":          "gnb_id",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.gNBIdLength":    "gnb_id_length",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellLocalId":    "nr_cell_local_id",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.NRARFCN":        "nrarfcn",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRPCI":              "nr_pci",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.SSBFrequency":       "ssb_frequency",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.SubcarrierSpacing":  "subcarrier_spacing",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ChannelBandwidth":   "channel_bandwidth",
		"Device.Services.FAPService.1.CellConfig.NR.Core.TAC":                  "nr_tac",
		"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID":    "nr_plmn_id",
		"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST":         "nr_sst",
		"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SD":          "nr_sd",
		"Device.Services.FAPService.1.FAPControl.NR.AdminState":                 "nr_admin_state",
		"Device.Services.FAPService.1.FAPControl.NR.Gateway.SecGWServer1":       "nr_secgw_server1",
		"Device.Services.FAPService.1.FAPControl.NR.Gateway.SecGWServer2":       "nr_secgw_server2",

		// Common device info (standard TR069 paths, same across carriers)
		"Device.DeviceInfo.Manufacturer":                 "manufacturer",
		"Device.DeviceInfo.ManufacturerOUI":              "oui",
		"Device.DeviceInfo.ModelName":                    "model_name",
		"Device.DeviceInfo.SerialNumber":                 "serial_number",
		"Device.DeviceInfo.HardwareVersion":              "hw_version",
		"Device.DeviceInfo.SoftwareVersion":              "sw_version",
		"Device.ManagementServer.URL":                    "acs_url",
		"Device.ManagementServer.PeriodicInformEnable":   "periodic_inform_enable",
		"Device.ManagementServer.PeriodicInformInterval": "periodic_inform_interval",
		"Device.ManagementServer.ConnectionRequestURL":   "conn_req_url",
	}

	forward = make(map[string]string, len(mappings))
	reverse = make(map[string]string, len(mappings))
	for k, v := range mappings {
		forward[k] = v
		reverse[v] = k
	}
	return forward, reverse
}
