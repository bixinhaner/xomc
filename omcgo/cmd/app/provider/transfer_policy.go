package provider

import (
	"context"
	"strings"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
)

type sysConfigSavedHookRegistrar interface {
	RegisterSavedHook(admin.SysConfigSavedHook)
}

func newTransferSysConfigLookup(repo interface {
	GetByKey(context.Context, string, string) (*admin.SysConfig, error)
}) transfercfg.SysConfigLookup {
	return func(ctx context.Context, category, key string) (string, bool) {
		cfg, err := repo.GetByKey(ctx, category, key)
		if err != nil || cfg == nil {
			return "", false
		}
		return cfg.Value, true
	}
}

func registerTransferPolicyInvalidation(
	registrar sysConfigSavedHookRegistrar,
	policy interface{ InvalidateCache() },
) {
	if registrar == nil || policy == nil {
		return
	}
	registrar.RegisterSavedHook(func(_ context.Context, category string) {
		if category == transfercfg.Category {
			policy.InvalidateCache()
		}
	})
}

func newTransferAddressResolver(
	provider transfercfg.Provider,
	capabilityReader transfercfg.DeviceParameterByPathReader,
) *transfercfg.AddressResolver {
	return transfercfg.NewAddressResolver(
		provider,
		transfercfg.NewDeviceParameterHTTPSCapabilityReader(capabilityReader),
	)
}

func newSoftwareTransferDefaults(cfg appconfig.UpgradeConfig) transfercfg.Snapshot {
	baseURL := strings.TrimSpace(cfg.ACSUploadBaseURL)
	return transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
		Upload: transfercfg.UploadSettings{
			BaseURL: baseURL,
			Path:    "/smallcell/FileUploadService",
		},
		Download: transfercfg.DownloadSettings{
			BaseURL: baseURL,
			Path:    "/smallcell/FileDownloadService",
		},
	}
}

func newMRTransferDefaults(cfg appconfig.UpgradeConfig) transfercfg.Snapshot {
	baseURL := strings.TrimSpace(cfg.ACSUploadBaseURL)
	return transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
		Upload: transfercfg.UploadSettings{
			BaseURL: baseURL,
			Path:    "/smallcell/FileUploadService",
		},
	}
}
