package snmp

import "time"

// SNMPVersion identifies the SNMP protocol version used for a Trap target.
//
// Supported in skeleton:
//   - VersionV2c — community-based, default for CMCC / CUCC
//   - VersionV3  — USM authPriv, required by CTCC; encryption verified at T-0017.
type SNMPVersion string

const (
	VersionV2c SNMPVersion = "v2c"
	VersionV3  SNMPVersion = "v3"
)

// IsValid reports whether v is a supported SNMP version.
func (v SNMPVersion) IsValid() bool {
	switch v {
	case VersionV2c, VersionV3:
		return true
	default:
		return false
	}
}

// AuthProtocol enumerates SNMPv3 authentication protocols. Skeleton stage only
// validates string identity; the actual cryptographic dispatch is delegated to
// gosnmp at Send time (see sender.go).
type AuthProtocol string

const (
	AuthMD5    AuthProtocol = "MD5"
	AuthSHA    AuthProtocol = "SHA"
	AuthSHA224 AuthProtocol = "SHA224"
	AuthSHA256 AuthProtocol = "SHA256"
	AuthSHA384 AuthProtocol = "SHA384"
	AuthSHA512 AuthProtocol = "SHA512"
	AuthNoAuth AuthProtocol = "" // explicit no-auth marker
)

// PrivProtocol enumerates SNMPv3 privacy (encryption) protocols.
type PrivProtocol string

const (
	PrivDES    PrivProtocol = "DES"
	PrivAES    PrivProtocol = "AES"
	PrivAES128 PrivProtocol = "AES128"
	PrivAES192 PrivProtocol = "AES192"
	PrivAES256 PrivProtocol = "AES256"
	PrivNoPriv PrivProtocol = "" // explicit no-priv marker
)

// TrapTarget is the runtime representation of an OSS Trap receiver.
//
// In T-0017 this struct will be persisted to snmp_trap_targets (see PRD
// F08-oss-protocol §D1). In the skeleton stage it lives in memory only.
//
// Sensitive fields (Community, AuthPassword, PrivPassword) MUST NOT appear in
// logs, Prometheus labels, or API responses. The String / format methods on
// this type intentionally redact them.
type TrapTarget struct {
	ID           string
	OSSName      string
	Description  string
	Host         string
	Port         uint16
	Version      SNMPVersion
	Community    string // v2c
	Username     string // v3
	AuthProtocol AuthProtocol
	AuthPassword string // v3, redacted on String()
	PrivProtocol PrivProtocol
	PrivPassword string // v3, redacted on String()
	Carrier      string // optional: cmcc | ctcc | cucc
	Timeout      time.Duration
	Retries      int // skeleton stage: 0
	Inform       bool
	Enabled      bool
}

// String returns a redacted, log-safe rendering of the target.
func (t *TrapTarget) String() string {
	if t == nil {
		return "<nil TrapTarget>"
	}
	return "TrapTarget{id=" + t.ID +
		", oss=" + t.OSSName +
		", host=" + t.Host +
		", version=" + string(t.Version) +
		", carrier=" + t.Carrier +
		", enabled=" + boolStr(t.Enabled) + "}"
}

// VarType identifies the ASN.1 / SMI variable type used in a SNMP VarBind.
//
// Skeleton stage exposes the subset actually needed for alarm Traps. Mapping
// to gosnmp.Asn1BER happens in sender.go to keep this enum free of any
// gosnmp-specific value.
type VarType string

const (
	VarTypeOctetString VarType = "OctetString"
	VarTypeInteger     VarType = "Integer"
	VarTypeCounter32   VarType = "Counter32"
	VarTypeCounter64   VarType = "Counter64"
	VarTypeTimeTicks   VarType = "TimeTicks"
	VarTypeIPAddress   VarType = "IPAddress"
	VarTypeObjectID    VarType = "ObjectID"
)

// Variable is one VarBind in a SNMP Trap PDU.
type Variable struct {
	OID   string
	Type  VarType
	Value any
}

// SendResult records the outcome of one Trap delivery attempt against a single
// target. Engine.Process returns one SendResult per attempted target.
type SendResult struct {
	TargetID string
	OSSName  string
	Success  bool
	Err      error
	Latency  time.Duration
}

// AlarmEvent is the skeleton-stage shape of an alarm consumable by Engine.
//
// IMPORTANT: this is a private placeholder type. T-0017 will replace it with
// the canonical alarm event from internal/alarm or internal/core/model. We
// intentionally do NOT import those packages here to preserve the dependency
// isolation guarantee documented in doc.go.
type AlarmEvent struct {
	AlarmID      string    // unique alarm identifier
	DeviceSerial string    // CPE serial number
	Severity     string    // critical / major / minor / warning / cleared
	AlarmType    string    // category, e.g. POWER_FAIL
	OccurTime    time.Time // when the alarm fired on the device
	Carrier      string    // cmcc / ctcc / cucc, optional
	Extra        map[string]string
}

// AlarmMapper converts an AlarmEvent into the SNMP VarBind list expected by
// the destination OSS. T-0017 will replace the package-default mapper with
// the per-carrier implementations behind internal/carrier.Carrier.
type AlarmMapper interface {
	MapAlarmToTrapPDU(alarm *AlarmEvent) ([]Variable, error)
}

// boolStr is a tiny helper kept local to avoid pulling strconv just for one
// boolean rendering on a hot logging path.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
