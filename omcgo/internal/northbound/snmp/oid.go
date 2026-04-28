package snmp

// OMCPrivateOID is the placeholder OMC enterprise OID sub-tree used by the
// skeleton. T-0017 must replace this with the real enterprise OIDs assigned
// by each operator (IANA private enterprise number registration).
//
// The PRD (F08-oss-protocol §4 carrier matrix) tracks the per-operator
// allocation requirement. Each operator may host the alarm sub-tree under a
// distinct sub-OID; for the skeleton we collapse all three under a synthetic
// "99999" tree and rely on Carrier.MapAlarmToTrapPDU (T-0017) to override.
//
// OID format: 1.3.6.1.4.1.<EnterpriseNumber>.<sub-tree>
const OMCPrivateOID = "1.3.6.1.4.1.99999" // TODO(T-0017): replace with assigned PEN

// Standard SNMPv2c trap envelope OIDs (RFC 3418).
const (
	OIDSysUpTime  = "1.3.6.1.2.1.1.3.0"     // sysUpTime.0
	OIDSnmpTrapID = "1.3.6.1.6.3.1.1.4.1.0" // snmpTrapOID.0
)

// OMC alarm Trap sub-tree (1.3.6.1.4.1.99999.1.1.X).
//
// TODO(T-0017): align numbering with the formal MIB once issued.
var (
	OIDAlarmTrap       = OMCPrivateOID + ".1.1.0" // notification OID for alarm trap
	OIDAlarmIdentifier = OMCPrivateOID + ".1.1.1"
	OIDAlarmSeverity   = OMCPrivateOID + ".1.1.2"
	OIDDeviceSerial    = OMCPrivateOID + ".1.1.3"
	OIDOccurTime       = OMCPrivateOID + ".1.1.4"
	OIDAlarmType       = OMCPrivateOID + ".1.1.5"
	OIDCarrierTag      = OMCPrivateOID + ".1.1.6"
)
