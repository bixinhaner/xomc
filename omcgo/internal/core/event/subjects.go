package event

// Device events
const (
	SubjectDeviceBootstrap                  = "device.inform.bootstrap"
	SubjectDevicePeriodic                   = "device.inform.periodic"
	SubjectDeviceValueChange                = "device.inform.value_change"
	SubjectDeviceAlarm                      = "device.inform.alarm"
	SubjectDeviceTransferComplete           = "device.inform.transfer_complete"
	SubjectDeviceAutonomousTransferComplete = "device.inform.autonomous_transfer_complete"
	SubjectDeviceRebootComplete             = "device.inform.reboot_complete"
	SubjectDeviceConnectionRequest          = "device.inform.connection_request"
	SubjectDeviceConnectionLost             = "device.connection.lost"
	SubjectDeviceRegistered                 = "device.registered"
	SubjectDeviceRebootAbnormal             = "device.reboot.abnormal"
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

// Provisioning events
const (
	SubjectProvisionStarted   = "provision.started"
	SubjectProvisionCompleted = "provision.completed"
	SubjectProvisionFailed    = "provision.failed"
	SubjectProvisionStepDone  = "provision.step.done"
)

// DataModel events
const (
	SubjectDataModelUploadRequested = "datamodel.upload.requested"
	SubjectDataModelUploadCompleted = "datamodel.upload.completed"
	SubjectDataModelUploadFailed    = "datamodel.upload.failed"
	SubjectDataModelFileReceived    = "datamodel.file.received"
)

// Command response events
const (
	SubjectCommandGetParamsResponse      = "command.get_parameters.response"
	SubjectCommandSetParamsResponse      = "command.set_parameters.response"
	SubjectCommandDownloadResponse       = "command.download.response"
	SubjectCommandUploadResponse         = "command.upload.response"
	SubjectCommandGetNamesResponse       = "command.get_names.response"
	SubjectCommandAddObjectResponse      = "command.add_object.response"
	SubjectCommandDeleteObjectResponse   = "command.delete_object.response"
	SubjectCommandRebootResponse         = "command.reboot.response"
	SubjectCommandFactoryResetResponse   = "command.factory_reset.response"
	SubjectCommandGetAttrsResponse       = "command.get_attrs.response"
	SubjectCommandSetAttrsResponse       = "command.set_attrs.response"
)

// Software/Firmware events
const (
	SubjectFirmwareUploaded = "firmware.uploaded"
	SubjectUpgradeStarted   = "upgrade.started"
	SubjectUpgradeCompleted = "upgrade.completed"
	SubjectUpgradeFailed    = "upgrade.failed"
)

// Backup events
const (
	SubjectBackupTaskCreated = "backup.task.created"
	SubjectBackupTaskDone    = "backup.task.done"
)

// Report events
const (
	SubjectReportGenerateRequested = "report.generate.requested"
	SubjectReportGenerateDone      = "report.generate.done"
)

// Northbound/OSS events
const (
	SubjectOSSAlarmForward   = "oss.alarm.forward"
	SubjectOSSPMExport       = "oss.pm.export"
	SubjectOSSConfigSnapshot = "oss.config.snapshot"
)

// NE Direct events
const (
	SubjectNEDirectRegister   = "nedirect.register"
	SubjectNEDirectFault      = "nedirect.fault"
	SubjectNEDirectConnect    = "nedirect.session.connect"
	SubjectNEDirectDisconnect = "nedirect.session.disconnect"
	SubjectNEDirectCommand    = "nedirect.command.sent"
)
