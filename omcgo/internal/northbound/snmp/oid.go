package snmp

// OMCPrivateOID is the Baicells enterprise OID from omcAlarmMIB.mib:
// enterprises.53058.
const OMCPrivateOID = "1.3.6.1.4.1.53058"

const (
	OIDOMCAlarmMIB           = OMCPrivateOID + ".1.1"
	OIDOMCAlarmNotifications = OIDOMCAlarmMIB + ".0"
	OIDOMCAlarmObjects       = OIDOMCAlarmMIB + ".1"
	OIDOMCAlarmTable         = OIDOMCAlarmObjects + ".1"
	OIDOMCAlarmEntry         = OIDOMCAlarmTable + ".1"
)

// Standard SNMPv2c trap envelope OIDs (RFC 3418).
const (
	OIDSysUpTime  = "1.3.6.1.2.1.1.3.0"     // sysUpTime.0
	OIDSnmpTrapID = "1.3.6.1.6.3.1.1.4.1.0" // snmpTrapOID.0
)

// OMC alarm Trap sub-tree and VarBind OIDs from omcAlarmMIB.mib.
var (
	OIDAlarmTrap             = OIDOMCAlarmNotifications + ".1"
	OIDNotificationID        = OIDOMCAlarmEntry + ".1"
	OIDAlarmUniqueID         = OIDOMCAlarmEntry + ".2"
	OIDNotificationType      = OIDOMCAlarmEntry + ".3"
	OIDEventTime             = OIDOMCAlarmEntry + ".4"
	OIDEquipmentSDN          = OIDOMCAlarmEntry + ".5"
	OIDEquipmentName         = OIDOMCAlarmEntry + ".6"
	OIDEquipmentClass        = OIDOMCAlarmEntry + ".7"
	OIDObjectSDN             = OIDOMCAlarmEntry + ".8"
	OIDObjectInstanceName    = OIDOMCAlarmEntry + ".9"
	OIDObjectClass           = OIDOMCAlarmEntry + ".10"
	OIDAdditionalText        = OIDOMCAlarmEntry + ".11"
	OIDDeviceVendorOUI       = OIDOMCAlarmEntry + ".12"
	OIDSpecificProblemID     = OIDOMCAlarmEntry + ".13"
	OIDSpecificProblem       = OIDOMCAlarmEntry + ".14"
	OIDAlarmType             = OIDOMCAlarmEntry + ".15"
	OIDPerceivedSeverity     = OIDOMCAlarmEntry + ".16"
	OIDProbableCause         = OIDOMCAlarmEntry + ".17"
	OIDAdditionalInformation = OIDOMCAlarmEntry + ".18"

	// Backward-compatible aliases used by older sender tests and callers.
	OIDAlarmIdentifier = OIDAlarmUniqueID
	OIDAlarmSeverity   = OIDPerceivedSeverity
	OIDDeviceSerial    = OIDEquipmentSDN
	OIDOccurTime       = OIDEventTime
	OIDCarrierTag      = OIDAdditionalInformation
)
