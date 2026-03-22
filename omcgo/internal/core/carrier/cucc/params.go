package cucc

// buildParamMappings creates bidirectional parameter path mappings
// between CUCC-specific paths and unified internal names.
// CUCC only supports NR (5G), so no LTE parameters are included.
func buildParamMappings() (forward map[string]string, reverse map[string]string) {
	mappings := map[string]string{
		// CUCC 5G NR private extensions (X_CUCC_ prefix)
		"Device.X_CUCC.gNBName":     "gnb_name",
		"Device.X_CUCC.gNBId":       "gnb_id",
		"Device.X_CUCC.SiteName":    "site_name",
		"Device.X_CUCC.SiteId":      "site_id",
		"Device.X_CUCC.Region":      "region",
		"Device.X_CUCC.PMEnable":    "pm_enable",
		"Device.X_CUCC.PMInterval":  "pm_interval",
		"Device.X_CUCC.SubNetwork":  "sub_network",
		"Device.X_CUCC.ManagedBy":   "managed_by",

		// NR RAN parameters
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.gNBId":         "gnb_id",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.gNBIdLength":   "gnb_id_length",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellLocalId":   "nr_cell_local_id",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.NRARFCN":       "nrarfcn",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRPCI":             "nr_pci",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.SSBFrequency":      "ssb_frequency",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.SubcarrierSpacing": "subcarrier_spacing",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ChannelBandwidth":  "channel_bandwidth",

		// NR Core parameters
		"Device.Services.FAPService.1.CellConfig.NR.Core.TAC":               "nr_tac",
		"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID": "nr_plmn_id",
		"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST":     "nr_sst",
		"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SD":      "nr_sd",

		// NR Control parameters
		"Device.Services.FAPService.1.FAPControl.NR.AdminState":           "nr_admin_state",
		"Device.Services.FAPService.1.FAPControl.NR.RFTxStatus":           "nr_rf_tx_status",
		"Device.Services.FAPService.1.FAPControl.NR.OpState":              "nr_op_state",
		"Device.Services.FAPService.1.FAPControl.NR.Gateway.SecGWServer1": "nr_secgw_server1",
		"Device.Services.FAPService.1.FAPControl.NR.Gateway.SecGWServer2": "nr_secgw_server2",

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
