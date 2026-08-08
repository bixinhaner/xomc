package pageconfig

// No built-in delivery targets: delivery destinations are environment-specific and must
// be added manually by the operator. Kept as a function (returning nil) so the no-repo
// fallback and test fakes have a stable empty default.
func defaultDeliveryTargets() []DeliveryTarget {
	return nil
}

func defaultSNMPAlarmFields() []SNMPAlarmField {
	return []SNMPAlarmField{
		{Order: 1, Field: "notificationID", OID: OIDNotificationIDValue, Type: "Integer32(1..2147483647)", Source: "alarm.sequence_id", Required: true},
		{Order: 2, Field: "alarmUniqueId", OID: OIDAlarmUniqueIDValue, Type: "OCTET STRING(5)", Source: "alarm.alarm_identifier", Required: true},
		{Order: 3, Field: "notificationType", OID: OIDNotificationTypeValue, Type: "OCTET STRING(1)", Source: "alarm.status -> 0/1", Required: true},
		{Order: 4, Field: "eventTime", OID: OIDEventTimeValue, Type: "Counter64(13)", Source: "alarm.raised_at / alarm.cleared_at -> epoch_ms", Required: true},
		{Order: 5, Field: "equipmentSDN", OID: OIDEquipmentSDNValue, Type: "OCTET STRING(1..45)", Source: "alarm.device_sn", Required: true},
		{Order: 6, Field: "equipmentName", OID: OIDEquipmentNameValue, Type: "OCTET STRING(0..50)", Source: "alarm.device_name / device_info.device_name", Required: true},
		{Order: 7, Field: "equipmentClass", OID: OIDEquipmentClassValue, Type: "OCTET STRING(0..45)", Source: "device.technology", Required: true},
		{Order: 8, Field: "objectSDN", OID: OIDObjectSDNValue, Type: "OCTET STRING(1..45)", Source: "alarm.alarm_source", Required: true},
		{Order: 9, Field: "objectInstanceName", OID: OIDObjectInstanceNameValue, Type: "OCTET STRING(0..50)", Source: "alarm.network_location", Required: true},
		{Order: 10, Field: "objectClass", OID: OIDObjectClassValue, Type: "OCTET STRING(0..45)", Source: "alarm.event_type", Required: true},
		{Order: 11, Field: "additionalText", OID: OIDAdditionalTextValue, Type: "OCTET STRING(0..256)", Source: "alarm.description", Required: true},
		{Order: 12, Field: "deviceVendorOUI", OID: OIDDeviceVendorOUIValue, Type: "OCTET STRING(0..10)", Source: "device.oui / device.manufacturer", Required: true},
		{Order: 13, Field: "specificProblemID", OID: OIDSpecificProblemIDValue, Type: "OCTET STRING(5)", Source: "alarm.alarm_type / alarm_definition.vendor_code", Required: true},
		{Order: 14, Field: "specificProblem", OID: OIDSpecificProblemValue, Type: "OCTET STRING(0..256)", Source: "alarm.description", Required: true},
		{Order: 15, Field: "alarmType", OID: OIDAlarmTypeValue, Type: "OCTET STRING(5)", Source: "alarm.alarm_type", Required: true},
		{Order: 16, Field: "perceivedSeverity", OID: OIDPerceivedSeverityValue, Type: "OCTET STRING(5..8)", Source: "alarm.severity", Required: true},
		{Order: 17, Field: "probableCause", OID: OIDProbableCauseValue, Type: "OCTET STRING(0..256)", Source: "alarm.probable_cause", Required: true},
		{Order: 18, Field: "additionalInformation", OID: OIDAdditionalInformationValue, Type: "OCTET STRING(0..256)", Source: "alarm.additional_info", Required: true},
	}
}

func defaultSNMPAlarmTargets() []SNMPAlarmTarget {
	return []SNMPAlarmTarget{
		{
			Key:                 "snmp-v2-primary",
			Name:                "SNMP v2 Trap 主用目标",
			Enabled:             false,
			Version:             "v2",
			NotificationType:    "Trap",
			ListenIP:            "0.0.0.0",
			ListenPort:          161,
			TargetHost:          "",
			TargetPort:          162,
			Community:           "",
			ClearSeverityPolicy: "保留原级别",
			MIBQueryEnabled:     true,
			TimeoutSeconds:      5,
			Retries:             1,
			MIBFields:           defaultSNMPAlarmFields(),
		},
		{
			Key:                 "snmp-v3-inform",
			Name:                "SNMP v3 Inform 备用目标",
			Enabled:             false,
			Version:             "v3",
			NotificationType:    "Inform",
			ListenIP:            "0.0.0.0",
			ListenPort:          161,
			TargetHost:          "",
			TargetPort:          163,
			SecurityName:        "",
			AuthProtocol:        "SHA",
			PrivProtocol:        "DES",
			ClearSeverityPolicy: "保留原级别",
			MIBQueryEnabled:     false,
			TimeoutSeconds:      5,
			Retries:             1,
			MIBFields:           defaultSNMPAlarmFields(),
		},
	}
}

func defaultSocketAlarmConfigs() []SocketAlarmConfig {
	return []SocketAlarmConfig{
		{
			Key:                 "socket-ctcc-server",
			Name:                "CTCC Socket 告警服务端",
			Enabled:             false,
			Profile:             "CTCC",
			Mode:                "server",
			ListenIP:            "0.0.0.0",
			ListenPort:          31232,
			MaxClients:          20,
			RealtimePushEnabled: true,
			ClientSyncEnabled:   true,
			HeartbeatSeconds:    60,
			HeartbeatTimes:      3,
			IdleTimeoutSeconds:  180,
			Accounts: []SocketAccount{
				{Key: "ctcc-msg", Enabled: false, Channel: "实时/同步账号", Username: "", Type: "msg", Purpose: "实时推送和客户端同步告警"},
			},
		},
		{
			Key:                 "socket-cucc-server",
			Name:                "CUCC Socket 告警服务端",
			Enabled:             false,
			Profile:             "CUCC",
			Mode:                "server",
			ListenIP:            "0.0.0.0",
			ListenPort:          31233,
			MaxClients:          20,
			RealtimePushEnabled: true,
			ClientSyncEnabled:   true,
			HeartbeatSeconds:    60,
			HeartbeatTimes:      3,
			IdleTimeoutSeconds:  180,
			Accounts: []SocketAccount{
				{Key: "cucc-msg", Enabled: false, Channel: "实时/同步账号", Username: "", Type: "msg", Purpose: "实时推送和客户端同步告警"},
				{Key: "cucc-ftp", Enabled: false, Channel: "文件同步账号", Username: "", Type: "ftp", Purpose: "客户端同步历史告警文件"},
			},
		},
	}
}

func defaultAPIConfigs() []APIConfig {
	return []APIConfig{
		{
			Key:                "auth-login",
			Name:               "登录换取 JWT",
			Method:             "POST",
			Path:               "/api/v1/auth/login",
			Kind:               "鉴权管理",
			DataType:           "auth",
			Enabled:            false,
			OldSystemSupported: true,
			CurrentSupported:   true,
			Source:             "omcgo/internal/admin/auth_handler.go",
			ResponseContract:   map[string]any{"fields": []string{"ret", "msg", "data.access_token", "data.refresh_token", "data.expires_at", "data.token_type"}},
		},
		{
			Key:                "nb-sync-full-device",
			Name:               "设备全量同步",
			Method:             "GET",
			Path:               "/api/v1/northbound/sync/full?data_type=device&format=json",
			Kind:               "正式北向",
			DataType:           "device",
			Enabled:            false,
			OldSystemSupported: true,
			CurrentSupported:   true,
			Source:             "omcgo/internal/northbound/sync/service.go",
			ResponseContract:   map[string]any{"fields": []string{"ret", "msg", "data.data_type", "data.items[]", "data.total", "data.synced_at"}},
		},
		{
			Key:                "nb-export-config",
			Name:               "配置快照导出",
			Method:             "GET",
			Path:               "/api/v1/northbound/export/config/{deviceId}",
			Kind:               "正式北向",
			DataType:           "config",
			Enabled:            false,
			OldSystemSupported: true,
			CurrentSupported:   true,
			Source:             "omcgo/internal/northbound/config_handler.go",
			ResponseContract:   map[string]any{"fields": []string{"ret", "msg", "data.device_id", "data.parameters[]", "data.total"}},
		},
	}
}

const (
	OIDNotificationIDValue        = "1.3.6.1.4.1.53058.1.1.1.1.1.1"
	OIDAlarmUniqueIDValue         = "1.3.6.1.4.1.53058.1.1.1.1.1.2"
	OIDNotificationTypeValue      = "1.3.6.1.4.1.53058.1.1.1.1.1.3"
	OIDEventTimeValue             = "1.3.6.1.4.1.53058.1.1.1.1.1.4"
	OIDEquipmentSDNValue          = "1.3.6.1.4.1.53058.1.1.1.1.1.5"
	OIDEquipmentNameValue         = "1.3.6.1.4.1.53058.1.1.1.1.1.6"
	OIDEquipmentClassValue        = "1.3.6.1.4.1.53058.1.1.1.1.1.7"
	OIDObjectSDNValue             = "1.3.6.1.4.1.53058.1.1.1.1.1.8"
	OIDObjectInstanceNameValue    = "1.3.6.1.4.1.53058.1.1.1.1.1.9"
	OIDObjectClassValue           = "1.3.6.1.4.1.53058.1.1.1.1.1.10"
	OIDAdditionalTextValue        = "1.3.6.1.4.1.53058.1.1.1.1.1.11"
	OIDDeviceVendorOUIValue       = "1.3.6.1.4.1.53058.1.1.1.1.1.12"
	OIDSpecificProblemIDValue     = "1.3.6.1.4.1.53058.1.1.1.1.1.13"
	OIDSpecificProblemValue       = "1.3.6.1.4.1.53058.1.1.1.1.1.14"
	OIDAlarmTypeValue             = "1.3.6.1.4.1.53058.1.1.1.1.1.15"
	OIDPerceivedSeverityValue     = "1.3.6.1.4.1.53058.1.1.1.1.1.16"
	OIDProbableCauseValue         = "1.3.6.1.4.1.53058.1.1.1.1.1.17"
	OIDAdditionalInformationValue = "1.3.6.1.4.1.53058.1.1.1.1.1.18"
)
