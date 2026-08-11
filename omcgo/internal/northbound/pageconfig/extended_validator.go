package pageconfig

import (
	"fmt"
	"strings"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func validateDeliveryTargets(req ReplaceDeliveryTargetsRequest) error {
	if req.Scope == "" {
		return fmt.Errorf("%w: delivery scope is required", commonerrors.ErrInvalidInput)
	}
	switch req.Scope {
	case DeliveryScopeFile, DeliveryScopeInventory, DeliveryScopeSocket:
	default:
		return fmt.Errorf("%w: unsupported delivery scope %s", commonerrors.ErrInvalidInput, req.Scope)
	}
	seen := map[string]struct{}{}
	for _, item := range req.Items {
		if strings.TrimSpace(item.Key) == "" {
			return fmt.Errorf("%w: delivery target key is required", commonerrors.ErrInvalidInput)
		}
		if _, ok := seen[item.Key]; ok {
			return fmt.Errorf("%w: duplicate delivery target key %s", commonerrors.ErrInvalidInput, item.Key)
		}
		seen[item.Key] = struct{}{}
		if item.Protocol != DeliveryProtocolFTP && item.Protocol != DeliveryProtocolSFTP {
			return fmt.Errorf("%w: unsupported delivery protocol %s", commonerrors.ErrInvalidInput, item.Protocol)
		}
		if strings.TrimSpace(item.Host) == "" {
			return fmt.Errorf("%w: delivery target host is required", commonerrors.ErrInvalidInput)
		}
		if strings.TrimSpace(item.Username) == "" {
			return fmt.Errorf("%w: delivery target username is required", commonerrors.ErrInvalidInput)
		}
		if item.Port < 1 || item.Port > 65535 {
			return fmt.Errorf("%w: delivery target port must be between 1 and 65535", commonerrors.ErrInvalidInput)
		}
		// Username + password only; private-key auth is not supported.
		if item.AuthMode != "" && item.AuthMode != DeliveryAuthPassword {
			return fmt.Errorf("%w: unsupported delivery auth mode %s (only PASSWORD is supported)", commonerrors.ErrInvalidInput, item.AuthMode)
		}
		if item.RetryTimes < 0 || item.RetryTimes > 20 {
			return fmt.Errorf("%w: delivery retry_times must be between 0 and 20", commonerrors.ErrInvalidInput)
		}
		if item.TimeoutSeconds < 1 || item.TimeoutSeconds > 300 {
			return fmt.Errorf("%w: delivery timeout_seconds must be between 1 and 300", commonerrors.ErrInvalidInput)
		}
		if strings.Contains(item.RemoteRoot, "..") {
			return fmt.Errorf("%w: delivery remote_root must not contain '..'", commonerrors.ErrInvalidInput)
		}
	}
	return nil
}

func validateSNMPAlarmTarget(target SNMPAlarmTarget) error {
	switch target.Version {
	case "v2", "v3":
	default:
		return fmt.Errorf("%w: unsupported SNMP version %s", commonerrors.ErrInvalidInput, target.Version)
	}
	switch target.NotificationType {
	case "Trap", "Inform":
	default:
		return fmt.Errorf("%w: unsupported SNMP notification type %s", commonerrors.ErrInvalidInput, target.NotificationType)
	}
	if target.ListenPort < 1 || target.ListenPort > 65535 || target.TargetPort < 1 || target.TargetPort > 65535 {
		return fmt.Errorf("%w: SNMP ports must be between 1 and 65535", commonerrors.ErrInvalidInput)
	}
	if target.Enabled {
		if strings.TrimSpace(target.TargetHost) == "" {
			return fmt.Errorf("%w: SNMP target_host is required when enabled", commonerrors.ErrInvalidInput)
		}
		if strings.EqualFold(target.Version, "v2") && strings.TrimSpace(target.Community) == "" && !target.CommunitySet {
			return fmt.Errorf("%w: SNMP v2 community is required when enabled", commonerrors.ErrInvalidInput)
		}
	}
	authProtocol := strings.TrimSpace(target.AuthProtocol)
	privProtocol := strings.TrimSpace(target.PrivProtocol)
	if authProtocol != "" && !allowedSNMPAuthProtocol(authProtocol) {
		return fmt.Errorf("%w: unsupported SNMP auth_protocol %s", commonerrors.ErrInvalidInput, target.AuthProtocol)
	}
	if privProtocol != "" && !allowedSNMPPrivProtocol(privProtocol) {
		return fmt.Errorf("%w: unsupported SNMP priv_protocol %s", commonerrors.ErrInvalidInput, target.PrivProtocol)
	}
	if strings.EqualFold(target.Version, "v3") && privProtocol != "" && authProtocol == "" {
		return fmt.Errorf("%w: SNMP v3 privacy requires auth_protocol", commonerrors.ErrInvalidInput)
	}
	if strings.EqualFold(target.Version, "v3") && (target.Enabled || target.MIBQueryEnabled) {
		if strings.TrimSpace(target.SecurityName) == "" {
			return fmt.Errorf("%w: SNMP v3 security_name is required when v3 alarm report or MIB query is enabled", commonerrors.ErrInvalidInput)
		}
		if authProtocol != "" && strings.TrimSpace(target.AuthCredential) == "" && !target.AuthCredentialSet {
			return fmt.Errorf("%w: SNMP v3 auth credential is required when v3 alarm report or MIB query is enabled", commonerrors.ErrInvalidInput)
		}
		if authProtocol != "" && !target.AuthCredentialSet && len(strings.TrimSpace(target.AuthCredential)) < 8 {
			return fmt.Errorf("%w: SNMP v3 auth credential must be at least 8 characters", commonerrors.ErrInvalidInput)
		}
		if privProtocol != "" && strings.TrimSpace(target.PrivCredential) == "" && !target.PrivCredentialSet {
			return fmt.Errorf("%w: SNMP v3 privacy credential is required when v3 alarm report or MIB query is enabled", commonerrors.ErrInvalidInput)
		}
		if privProtocol != "" && !target.PrivCredentialSet && len(strings.TrimSpace(target.PrivCredential)) < 8 {
			return fmt.Errorf("%w: SNMP v3 privacy credential must be at least 8 characters", commonerrors.ErrInvalidInput)
		}
	}
	if target.TimeoutSeconds < 1 || target.TimeoutSeconds > 300 {
		return fmt.Errorf("%w: SNMP timeout_seconds must be between 1 and 300", commonerrors.ErrInvalidInput)
	}
	if target.Retries < 0 || target.Retries > 20 {
		return fmt.Errorf("%w: SNMP retries must be between 0 and 20", commonerrors.ErrInvalidInput)
	}
	return nil
}

func allowedSNMPAuthProtocol(protocol string) bool {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "", "MD5", "SHA", "SHA1", "SHA224", "SHA256", "SHA384", "SHA512":
		return true
	default:
		return false
	}
}

func allowedSNMPPrivProtocol(protocol string) bool {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "", "DES", "AES", "AES128", "AES192", "AES256":
		return true
	default:
		return false
	}
}

func validateSocketAlarmConfig(config SocketAlarmConfig) error {
	switch config.Profile {
	case "CTCC", "CUCC":
	default:
		return fmt.Errorf("%w: unsupported socket profile %s", commonerrors.ErrInvalidInput, config.Profile)
	}
	if config.Mode != "" && config.Mode != "server" {
		return fmt.Errorf("%w: socket mode must be server", commonerrors.ErrInvalidInput)
	}
	if config.ListenPort < 1 || config.ListenPort > 65535 {
		return fmt.Errorf("%w: socket listen_port must be between 1 and 65535", commonerrors.ErrInvalidInput)
	}
	if config.MaxClients < 1 || config.MaxClients > 10000 {
		return fmt.Errorf("%w: socket max_clients must be between 1 and 10000", commonerrors.ErrInvalidInput)
	}
	if config.HeartbeatSeconds < 5 || config.HeartbeatSeconds > 3600 {
		return fmt.Errorf("%w: socket heartbeat_seconds must be between 5 and 3600", commonerrors.ErrInvalidInput)
	}
	if config.HeartbeatTimes < 1 || config.HeartbeatTimes > 100 {
		return fmt.Errorf("%w: socket heartbeat_times must be between 1 and 100", commonerrors.ErrInvalidInput)
	}
	if config.IdleTimeoutSeconds < 10 || config.IdleTimeoutSeconds > 86400 {
		return fmt.Errorf("%w: socket idle_timeout_seconds must be between 10 and 86400", commonerrors.ErrInvalidInput)
	}
	seen := map[string]struct{}{}
	for _, account := range config.Accounts {
		if strings.TrimSpace(account.Key) == "" {
			return fmt.Errorf("%w: socket account key is required", commonerrors.ErrInvalidInput)
		}
		if _, ok := seen[account.Key]; ok {
			return fmt.Errorf("%w: duplicate socket account key %s", commonerrors.ErrInvalidInput, account.Key)
		}
		seen[account.Key] = struct{}{}
		if account.Type != "msg" && account.Type != "ftp" {
			return fmt.Errorf("%w: unsupported socket account type %s", commonerrors.ErrInvalidInput, account.Type)
		}
	}
	return nil
}
