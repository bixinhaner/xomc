package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrAssetOwnershipConflict = errors.New("asset ownership conflict")

type AssetEvidenceSource string

const (
	AssetEvidenceSourceUnknown               AssetEvidenceSource = "unknown"
	AssetEvidenceSourceDevice                AssetEvidenceSource = "device"
	AssetEvidenceSourceRegistration          AssetEvidenceSource = "registration"
	AssetEvidenceSourceDeviceAndRegistration AssetEvidenceSource = "device_and_registration"
)

type AssetDeviceRecord struct {
	ID              uuid.UUID
	Carrier         string
	SerialNumber    string
	OUI             string
	ProductClass    string
	SoftwareVersion string
	GroupID         *uuid.UUID
	SiteID          string
	LifecycleState  string
	IsOnline        bool
	LastInformAt    *time.Time
}

type AssetRegistrationRecord struct {
	ID           uuid.UUID
	Carrier      string
	SerialNumber string
	DeviceID     *uuid.UUID
	GroupID      *uuid.UUID
	SiteName     string
	Status       string
}

type AssetEvidence struct {
	DeviceID                *uuid.UUID
	RegistrationID          *uuid.UUID
	Source                  AssetEvidenceSource
	Carrier                 string
	SerialNumber            string
	GroupID                 *uuid.UUID
	SiteID                  *string
	ExpectedOUI             string
	ExpectedProductClass    string
	ExpectedSoftwareVersion string
	AssetRetired            bool
}

type AssetLookup interface {
	FindDevice(ctx context.Context, carrier, serialNumber string) (*AssetDeviceRecord, error)
	FindRegistration(ctx context.Context, carrier, serialNumber string) (*AssetRegistrationRecord, error)
	ExistsUnderOtherCarrier(ctx context.Context, carrier, serialNumber string) (bool, error)
}

type AssetEvidenceResolver interface {
	Resolve(ctx context.Context, carrier, serialNumber string) (AssetEvidence, error)
}

type assetEvidenceResolver struct {
	lookup AssetLookup
}

func NewAssetEvidenceResolver(lookup AssetLookup) AssetEvidenceResolver {
	return &assetEvidenceResolver{lookup: lookup}
}

func (r *assetEvidenceResolver) Resolve(
	ctx context.Context,
	carrier string,
	serialNumber string,
) (AssetEvidence, error) {
	carrier = strings.TrimSpace(carrier)
	serialNumber = strings.TrimSpace(serialNumber)
	if carrier == "" {
		return AssetEvidence{}, ErrCarrierRequired
	}
	if serialNumber == "" {
		return AssetEvidence{}, ErrSerialNumberRequired
	}

	otherCarrier, err := r.lookup.ExistsUnderOtherCarrier(ctx, carrier, serialNumber)
	if err != nil {
		return AssetEvidence{}, fmt.Errorf("resolve asset evidence: check other carrier ownership: %w", err)
	}
	if otherCarrier {
		return AssetEvidence{}, fmt.Errorf(
			"resolve asset evidence for %s/%s: %w",
			carrier,
			serialNumber,
			ErrAssetOwnershipConflict,
		)
	}

	device, err := r.lookup.FindDevice(ctx, carrier, serialNumber)
	if err != nil {
		return AssetEvidence{}, fmt.Errorf("resolve asset evidence: find device: %w", err)
	}
	registration, err := r.lookup.FindRegistration(ctx, carrier, serialNumber)
	if err != nil {
		return AssetEvidence{}, fmt.Errorf("resolve asset evidence: find registration: %w", err)
	}

	if device != nil && registration != nil {
		if registration.DeviceID != nil && *registration.DeviceID != device.ID {
			return AssetEvidence{}, fmt.Errorf(
				"resolve asset evidence for %s/%s: registration device differs: %w",
				carrier,
				serialNumber,
				ErrAssetOwnershipConflict,
			)
		}
		if device.GroupID != nil && registration.GroupID != nil && *device.GroupID != *registration.GroupID {
			return AssetEvidence{}, fmt.Errorf(
				"resolve asset evidence for %s/%s: group differs: %w",
				carrier,
				serialNumber,
				ErrAssetOwnershipConflict,
			)
		}
	}

	evidence := AssetEvidence{
		Carrier:      carrier,
		SerialNumber: serialNumber,
		Source:       AssetEvidenceSourceUnknown,
	}
	if device != nil {
		evidence.DeviceID = &device.ID
		evidence.GroupID = device.GroupID
		evidence.SiteID = nonEmptyStringPointer(device.SiteID)
		evidence.Source = AssetEvidenceSourceDevice
		evidence.ExpectedOUI = strings.TrimSpace(device.OUI)
		evidence.ExpectedProductClass = strings.TrimSpace(device.ProductClass)
		evidence.ExpectedSoftwareVersion = strings.TrimSpace(device.SoftwareVersion)
		evidence.AssetRetired = strings.EqualFold(strings.TrimSpace(device.LifecycleState), "decommissioned")
	}
	if registration != nil {
		evidence.RegistrationID = &registration.ID
		evidence.AssetRetired = evidence.AssetRetired ||
			strings.EqualFold(strings.TrimSpace(registration.Status), "expired")
		if evidence.GroupID == nil {
			evidence.GroupID = registration.GroupID
		}
		if evidence.SiteID == nil {
			evidence.SiteID = nonEmptyStringPointer(registration.SiteName)
		}
		if device == nil {
			evidence.Source = AssetEvidenceSourceRegistration
		} else {
			evidence.Source = AssetEvidenceSourceDeviceAndRegistration
		}
	}
	return evidence, nil
}

func nonEmptyStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
