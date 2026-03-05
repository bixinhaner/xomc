package event

// Device events
const (
	SubjectDeviceBootstrap        = "device.inform.bootstrap"
	SubjectDevicePeriodic         = "device.inform.periodic"
	SubjectDeviceValueChange      = "device.inform.value_change"
	SubjectDeviceAlarm            = "device.inform.alarm"
	SubjectDeviceTransferComplete = "device.inform.transfer_complete"
	SubjectDeviceConnectionLost   = "device.connection.lost"
)

// Command events
const (
	SubjectCommandGetParams  = "command.get_parameters"
	SubjectCommandSetParams  = "command.set_parameters"
	SubjectCommandDownload   = "command.download"
	SubjectCommandUpload     = "command.upload"
	SubjectCommandReboot     = "command.reboot"
	SubjectCommandReset      = "command.factory_reset"
)

// PM events
const (
	SubjectPMFileReceived = "pm.file.received"
	SubjectPMFileParsed   = "pm.file.parsed"
)

// MR events
const (
	SubjectMRFileReceived = "mr.file.received"
	SubjectMRFileParsed   = "mr.file.parsed"
)

// Alarm events
const (
	SubjectAlarmRaised       = "alarm.raised"
	SubjectAlarmCleared      = "alarm.cleared"
	SubjectAlarmAcknowledged = "alarm.acknowledged"
)

// Northbound/OSS events
const (
	SubjectOSSAlarmForward   = "oss.alarm.forward"
	SubjectOSSPMExport       = "oss.pm.export"
	SubjectOSSConfigSnapshot = "oss.config.snapshot"
)
