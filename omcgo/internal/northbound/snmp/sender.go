package snmp

import (
	"context"
	"errors"
	"fmt"
	"time"

	g "github.com/gosnmp/gosnmp"
	"go.uber.org/zap"
)

// Sender abstracts SNMP Trap delivery so the Engine and tests can substitute
// a fake implementation. The production binding (GoSNMPSender) wraps the
// gosnmp library; tests use NewMockSender (see sender_test.go).
type Sender interface {
	// Send delivers a single Trap PDU built from vars to target. Implementations
	// MUST honour ctx cancellation (or its embedded deadline) and MUST return a
	// non-nil error on any delivery failure.
	Send(ctx context.Context, target *TrapTarget, vars []Variable) error
}

// GoSNMPSender is the production Sender backed by github.com/gosnmp/gosnmp.
//
// The skeleton stage exercises the full happy-path code (PDU construction,
// connection lifecycle, redacted logging) but is NOT wired into the running
// process — see doc.go and PRD §5 for the isolation contract.
type GoSNMPSender struct {
	logger *zap.Logger
}

// NewGoSNMPSender constructs a production Sender. logger may be nil in tests.
func NewGoSNMPSender(logger *zap.Logger) *GoSNMPSender {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GoSNMPSender{logger: logger}
}

// Send implements Sender. It builds a SNMPv2c TRAP-V2 (or v3 USM TRAP) PDU
// using the supplied variables and dispatches it via gosnmp.SendTrap.
//
// Behaviour:
//   - target nil  → returns ErrNilTarget
//   - target invalid (missing host / port / version) → validation error
//   - vars empty  → returns ErrNoVariables (nothing useful to send)
//   - ctx deadline → applied to the underlying gosnmp Connect / SendTrap
func (s *GoSNMPSender) Send(ctx context.Context, target *TrapTarget, vars []Variable) error {
	if target == nil {
		return ErrNilTarget
	}
	if err := validateTarget(target); err != nil {
		return fmt.Errorf("snmp: target invalid: %w", err)
	}
	if len(vars) == 0 {
		return ErrNoVariables
	}

	timeout := target.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	params := &g.GoSNMP{
		Target:    target.Host,
		Port:      target.Port,
		Transport: "udp",
		Community: target.Community,
		Version:   gosnmpVersion(target.Version),
		Timeout:   timeout,
		Retries:   target.Retries,
	}

	if target.Version == VersionV3 {
		applyV3Params(params, target)
	}

	if err := params.Connect(); err != nil {
		return fmt.Errorf("snmp: connect %s:%d: %w", target.Host, target.Port, err)
	}
	defer func() {
		if params.Conn != nil {
			_ = params.Conn.Close()
		}
	}()

	pdus := buildPDUs(vars)
	trap := g.SnmpTrap{
		Variables: pdus,
		IsInform:  target.Inform,
	}

	if _, err := params.SendTrap(trap); err != nil {
		return fmt.Errorf("snmp: send trap to %s: %w", target.OSSName, err)
	}

	s.logger.Debug("snmp trap sent",
		zap.String("target_id", target.ID),
		zap.String("oss_name", target.OSSName),
		zap.String("host", target.Host),
		zap.Uint16("port", target.Port),
		zap.String("version", string(target.Version)),
		zap.Int("var_count", len(vars)),
	)
	return nil
}

// Errors returned by Send. Engine.Process will surface these via SendResult.Err.
var (
	ErrNilTarget   = errors.New("snmp: nil target")
	ErrNoVariables = errors.New("snmp: no variables to send")
)

// validateTarget ensures a TrapTarget has the minimum fields required to
// attempt a Send. It does NOT validate auth credentials beyond presence —
// gosnmp will surface cryptographic / authn errors at Send time.
func validateTarget(t *TrapTarget) error {
	if t.Host == "" {
		return errors.New("missing Host")
	}
	if t.Port == 0 {
		return errors.New("missing Port")
	}
	if !t.Version.IsValid() {
		return fmt.Errorf("unsupported version %q", t.Version)
	}
	switch t.Version {
	case VersionV2c:
		if t.Community == "" {
			return errors.New("v2c requires Community")
		}
	case VersionV3:
		if t.Username == "" {
			return errors.New("v3 requires Username")
		}
	}
	return nil
}

// gosnmpVersion translates our SNMPVersion enum to gosnmp.SnmpVersion.
func gosnmpVersion(v SNMPVersion) g.SnmpVersion {
	switch v {
	case VersionV2c:
		return g.Version2c
	case VersionV3:
		return g.Version3
	default:
		return g.Version2c
	}
}

// applyV3Params populates SNMPv3 USM parameters on a gosnmp client.
//
// Skeleton stage maps the protocol enums; T-0017 adds full credential
// rotation and engineID discovery wiring.
func applyV3Params(params *g.GoSNMP, t *TrapTarget) {
	params.SecurityModel = g.UserSecurityModel
	params.MsgFlags = msgFlags(t.AuthProtocol, t.PrivProtocol)
	params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:                 t.Username,
		AuthenticationProtocol:   gosnmpAuthProtocol(t.AuthProtocol),
		AuthenticationPassphrase: t.AuthPassword,
		PrivacyProtocol:          gosnmpPrivProtocol(t.PrivProtocol),
		PrivacyPassphrase:        t.PrivPassword,
	}
}

func msgFlags(auth AuthProtocol, priv PrivProtocol) g.SnmpV3MsgFlags {
	switch {
	case auth != AuthNoAuth && priv != PrivNoPriv:
		return g.AuthPriv
	case auth != AuthNoAuth:
		return g.AuthNoPriv
	default:
		return g.NoAuthNoPriv
	}
}

func gosnmpAuthProtocol(p AuthProtocol) g.SnmpV3AuthProtocol {
	switch p {
	case AuthMD5:
		return g.MD5
	case AuthSHA:
		return g.SHA
	case AuthSHA224:
		return g.SHA224
	case AuthSHA256:
		return g.SHA256
	case AuthSHA384:
		return g.SHA384
	case AuthSHA512:
		return g.SHA512
	default:
		return g.NoAuth
	}
}

func gosnmpPrivProtocol(p PrivProtocol) g.SnmpV3PrivProtocol {
	switch p {
	case PrivDES:
		return g.DES
	case PrivAES, PrivAES128:
		return g.AES
	case PrivAES192:
		return g.AES192
	case PrivAES256:
		return g.AES256
	default:
		return g.NoPriv
	}
}

// buildPDUs converts our internal Variable list into gosnmp.SnmpPDU values.
// Unknown / unsupported variable types degrade to OctetString so that the
// trap still goes out (with reduced fidelity) instead of being dropped.
func buildPDUs(vars []Variable) []g.SnmpPDU {
	pdus := make([]g.SnmpPDU, 0, len(vars)+2)
	// SNMPv2c trap envelope: sysUpTime + snmpTrapOID first, per RFC 3416.
	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSysUpTime,
		Type:  g.TimeTicks,
		Value: uint32(0),
	})
	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSnmpTrapID,
		Type:  g.ObjectIdentifier,
		Value: OIDAlarmTrap,
	})

	for _, v := range vars {
		pdus = append(pdus, g.SnmpPDU{
			Name:  v.OID,
			Type:  toAsn1BER(v.Type),
			Value: v.Value,
		})
	}
	return pdus
}

func toAsn1BER(t VarType) g.Asn1BER {
	switch t {
	case VarTypeOctetString:
		return g.OctetString
	case VarTypeInteger:
		return g.Integer
	case VarTypeCounter32:
		return g.Counter32
	case VarTypeCounter64:
		return g.Counter64
	case VarTypeTimeTicks:
		return g.TimeTicks
	case VarTypeIPAddress:
		return g.IPAddress
	case VarTypeObjectID:
		return g.ObjectIdentifier
	default:
		return g.OctetString
	}
}
