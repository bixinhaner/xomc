package transfercfg

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrHTTPSBaseURLUnavailable = errors.New("HTTPS transfer base URL is unavailable")

type TransferDirection string

const (
	TransferDirectionUpload   TransferDirection = "upload"
	TransferDirectionDownload TransferDirection = "download"
)

type TransferProtocol string

const (
	TransferProtocolHTTP  TransferProtocol = "http"
	TransferProtocolHTTPS TransferProtocol = "https"
)

type AddressDecisionReason string

const (
	AddressReasonForceHTTP                 AddressDecisionReason = "policy_force_http"
	AddressReasonHTTPSCapabilityEnabled    AddressDecisionReason = "https_capability_enabled"
	AddressReasonHTTPSCapabilityDisabled   AddressDecisionReason = "https_capability_disabled"
	AddressReasonHTTPSCapabilityUnknown    AddressDecisionReason = "https_capability_unknown"
	AddressReasonHTTPSCapabilityReadError  AddressDecisionReason = "https_capability_read_error"
	AddressReasonHTTPSCapabilityOtherValue AddressDecisionReason = "https_capability_other_value"
)

type AddressDecision struct {
	Direction  TransferDirection
	Protocol   TransferProtocol
	BaseURL    string
	Capability HTTPSCapabilityStatus
	Reason     AddressDecisionReason
}

type AddressResolver struct {
	provider         Provider
	capabilityReader HTTPSCapabilityReader
}

func NewAddressResolver(provider Provider, capabilityReader HTTPSCapabilityReader) *AddressResolver {
	return &AddressResolver{provider: provider, capabilityReader: capabilityReader}
}

func (r *AddressResolver) Resolve(
	ctx context.Context,
	deviceID uuid.UUID,
	direction TransferDirection,
) (AddressDecision, error) {
	if r == nil || r.provider == nil {
		return AddressDecision{}, fmt.Errorf("transfer config provider is required")
	}

	snapshot := r.provider.Snapshot(ctx)
	policy := strings.TrimSpace(snapshot.ProtocolPolicy)
	if policy == "" {
		policy = ProtocolPolicyForceHTTP
	}
	if policy != ProtocolPolicyForceHTTP && policy != ProtocolPolicyPreferHTTPS {
		return AddressDecision{}, fmt.Errorf("unsupported transfer protocol policy %q", policy)
	}
	if policy == ProtocolPolicyForceHTTP {
		return r.decision(snapshot, direction, TransferProtocolHTTP, HTTPSCapabilityNotRead, AddressReasonForceHTTP)
	}

	capability := HTTPSCapabilityUnknown
	if r.capabilityReader != nil {
		capability = r.capabilityReader.ReadHTTPSCapability(ctx, deviceID)
	}
	if capability == HTTPSCapabilityEnabled {
		return r.decision(
			snapshot,
			direction,
			TransferProtocolHTTPS,
			capability,
			AddressReasonHTTPSCapabilityEnabled,
		)
	}
	return r.decision(
		snapshot,
		direction,
		TransferProtocolHTTP,
		capability,
		fallbackReason(capability),
	)
}

func (r *AddressResolver) decision(
	snapshot Snapshot,
	direction TransferDirection,
	protocol TransferProtocol,
	capability HTTPSCapabilityStatus,
	reason AddressDecisionReason,
) (AddressDecision, error) {
	baseURL, err := baseURLForDirection(snapshot, direction, protocol)
	if err != nil {
		return AddressDecision{}, err
	}
	return AddressDecision{
		Direction:  direction,
		Protocol:   protocol,
		BaseURL:    baseURL,
		Capability: capability,
		Reason:     reason,
	}, nil
}

func fallbackReason(capability HTTPSCapabilityStatus) AddressDecisionReason {
	switch capability {
	case HTTPSCapabilityDisabled:
		return AddressReasonHTTPSCapabilityDisabled
	case HTTPSCapabilityReadError:
		return AddressReasonHTTPSCapabilityReadError
	case HTTPSCapabilityOtherValue:
		return AddressReasonHTTPSCapabilityOtherValue
	default:
		return AddressReasonHTTPSCapabilityUnknown
	}
}

func baseURLForDirection(
	snapshot Snapshot,
	direction TransferDirection,
	protocol TransferProtocol,
) (string, error) {
	var baseURL string
	switch direction {
	case TransferDirectionUpload:
		if protocol == TransferProtocolHTTPS {
			baseURL = snapshot.Upload.HTTPSBaseURL
		} else {
			baseURL = snapshot.Upload.BaseURL
		}
	case TransferDirectionDownload:
		if protocol == TransferProtocolHTTPS {
			baseURL = snapshot.Download.HTTPSBaseURL
		} else {
			baseURL = snapshot.Download.BaseURL
		}
	default:
		return "", fmt.Errorf("unsupported transfer direction %q", direction)
	}

	if strings.TrimSpace(baseURL) == "" {
		if protocol == TransferProtocolHTTPS {
			return "", fmt.Errorf("%w: %s base URL is missing", ErrHTTPSBaseURLUnavailable, direction)
		}
		return "", fmt.Errorf("%s %s base URL is missing", protocol, direction)
	}
	var err error
	if protocol == TransferProtocolHTTPS {
		err = ValidateHTTPSBaseURL(baseURL)
	} else {
		err = ValidateHTTPBaseURL(baseURL)
	}
	if err != nil {
		if protocol == TransferProtocolHTTPS {
			return "", fmt.Errorf("%w: invalid %s base URL: %v", ErrHTTPSBaseURLUnavailable, direction, err)
		}
		return "", fmt.Errorf("invalid %s %s base URL: %w", protocol, direction, err)
	}
	return baseURL, nil
}
