package provision

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PlugAndPlayPolicy struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	Enabled           bool            `json:"enabled"`
	ProductClass      string          `json:"product_class"`
	ProductClasses    []string        `json:"product_classes"`
	ExecuteType       string          `json:"execute_type"`
	Priority          int             `json:"priority"`
	UpgradeEnabled    bool            `json:"upgrade_enabled"`
	TargetVersion     string          `json:"target_version,omitempty"`
	LicenseEnabled    bool            `json:"license_enabled"`
	SelfConfigEnabled bool            `json:"self_config_enabled"`
	Config            json.RawMessage `json:"config"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func normalizePolicyProductClasses(policy *PlugAndPlayPolicy) {
	if policy == nil {
		return
	}
	classes := make([]string, 0, len(policy.ProductClasses)+1)
	seen := make(map[string]struct{}, len(policy.ProductClasses)+1)
	for _, productClass := range policy.ProductClasses {
		productClass = strings.TrimSpace(productClass)
		if productClass == "" {
			continue
		}
		if _, exists := seen[productClass]; exists {
			continue
		}
		seen[productClass] = struct{}{}
		classes = append(classes, productClass)
	}
	legacy := strings.TrimSpace(policy.ProductClass)
	if len(classes) == 0 && legacy != "" {
		classes = append(classes, legacy)
	}
	policy.ProductClasses = classes
	if len(classes) > 0 {
		policy.ProductClass = classes[0]
	} else {
		policy.ProductClass = ""
	}
}

func policySupportsProductClass(policy *PlugAndPlayPolicy, productClass string) bool {
	normalizePolicyProductClasses(policy)
	productClass = strings.TrimSpace(productClass)
	for _, supported := range policy.ProductClasses {
		if supported == productClass {
			return true
		}
	}
	return false
}

type policyModule string

const (
	policyModuleUpgrade    policyModule = "software_upgrade"
	policyModuleLicense    policyModule = "license"
	policyModuleSelfConfig policyModule = "self_config"
)

// enabledPolicyModules returns the device-dispatch order. Disabled modules are
// omitted without changing the relative order of the selected modules.
func enabledPolicyModules(policy *PlugAndPlayPolicy) []policyModule {
	modules := make([]policyModule, 0, 3)
	if policy == nil {
		return modules
	}
	if policy.UpgradeEnabled {
		modules = append(modules, policyModuleUpgrade)
	}
	if policy.LicenseEnabled {
		modules = append(modules, policyModuleLicense)
	}
	if policy.SelfConfigEnabled {
		modules = append(modules, policyModuleSelfConfig)
	}
	return modules
}

func policyModuleFromStepName(stepName string) (policyModule, bool) {
	stepName = strings.TrimSpace(stepName)
	for _, module := range []policyModule{policyModuleUpgrade, policyModuleLicense, policyModuleSelfConfig} {
		if strings.HasPrefix(stepName, string(module)+"_") {
			return module, true
		}
	}
	return "", false
}

type PolicyFilter struct {
	ProductClass string
	Search       string
	Page         int
	PageSize     int
}

type PlugAndPlayPolicyRepository interface {
	CreatePolicy(context.Context, *PlugAndPlayPolicy) error
	GetPolicy(context.Context, uuid.UUID) (*PlugAndPlayPolicy, error)
	ListPolicies(context.Context, PolicyFilter) ([]PlugAndPlayPolicy, int64, error)
	UpdatePolicy(context.Context, *PlugAndPlayPolicy) error
	DeletePolicy(context.Context, uuid.UUID) error
}

type ProvisioningXMLFile struct {
	ID            uuid.UUID `json:"id"`
	PolicyID      uuid.UUID `json:"policy_id"`
	DeviceID      uuid.UUID `json:"device_id"`
	FileName      string    `json:"file_name"`
	Content       string    `json:"content,omitempty"`
	Checksum      string    `json:"checksum"`
	DownloadToken uuid.UUID `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProvisioningXMLRepository interface {
	CreateXML(context.Context, *ProvisioningXMLFile) error
	GetXML(context.Context, uuid.UUID) (*ProvisioningXMLFile, error)
	RotateXMLDownloadToken(context.Context, uuid.UUID) (uuid.UUID, error)
}
