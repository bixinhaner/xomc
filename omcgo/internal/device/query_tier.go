package device

// QueryTier defines the frequency at which ACS queries specific parameter groups.
// This classification reduces unnecessary RPC overhead by only querying parameters
// at the frequency appropriate for their rate of change.
type QueryTier int

const (
	// TierEveryInform queries parameters on every Inform (heartbeat).
	// Used for rapidly changing operational status parameters.
	TierEveryInform QueryTier = iota

	// TierDaily queries parameters once per day via scheduled task.
	// Used for slowly changing configuration parameters.
	TierDaily

	// TierOnChange queries parameters only on Bootstrap or ValueChange events.
	// Used for static or rarely changing parameters.
	TierOnChange
)

// QueryGroup defines a set of TR069 parameter paths that share the same query frequency.
type QueryGroup struct {
	Tier  QueryTier
	Label string
	Paths []string
}

// DefaultQueryGroups defines the standard parameter groups and their query tiers.
// ACS scheduler uses this to determine which GetParameterValues RPCs to issue.
var DefaultQueryGroups = []QueryGroup{
	{
		Tier:  TierEveryInform,
		Label: "operational-status",
		Paths: []string{
			"Device.DeviceInfo.X_COM_GPS_Status",
			"Device.ManagementServer.tfcsSyncState",
			"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState",
			// ENB_DEFAULT_098/181 expose MME connectivity through Gateway.MmeStatus.
			"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus",
			"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool1Status",
			"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool2Status",
			"Device.Services.FAPService.1.FAPControl.NR.Gateway.X_COM_tfcsSyncState",
			"Device.DeviceInfo.X_COM_BDS_Status",
			"Device.DeviceInfo.X_COM_1588_Status",
			"Device.Services.FAPService.1.FAPControl.LTE.AdminState",
			"Device.Services.FAPService.1.FAPControl.NR.AdminState",
			"Device.Services.FAPService.1.FAPControl.LTE.OpState",
			"Device.Services.FAPService.1.FAPControl.NR.OpState",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.X_COM_RadioEnable",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.RFTxStatus",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellOpState",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellOpState",
			"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus",
		},
	},
	{
		Tier:  TierDaily,
		Label: "license-mme-antenna",
		Paths: []string{
			// License (prefix query in practice)
			"Device.DeviceInfo.X_COM_LICENSE.",
			// MME pool (prefix query in practice)
			"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.",
			// Antenna info
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.AntennaInfo.",
		},
	},
	{
		Tier:  TierOnChange,
		Label: "static-config",
		Paths: []string{
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
			"Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells",
			"Device.FAP.GPS.LockedLatitude",
			"Device.FAP.GPS.LockedLongitude",
		},
	},
}
