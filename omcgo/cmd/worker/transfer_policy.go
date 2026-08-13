package main

import (
	"context"
	"net/url"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/pm"
)

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

func newTransferAddressResolver(
	provider transfercfg.Provider,
	capabilityReader transfercfg.DeviceParameterByPathReader,
) *transfercfg.AddressResolver {
	return transfercfg.NewAddressResolver(
		provider,
		transfercfg.NewDeviceParameterHTTPSCapabilityReader(capabilityReader),
	)
}

func newWorkerTransferDefaults(cfg appconfig.PMConfig) transfercfg.Snapshot {
	defaults := transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
	}
	rendered, err := pm.ValidateUploadURLTemplate(cfg.UploadURLTemplate)
	if err != nil {
		return defaults
	}
	parsed, err := url.Parse(rendered)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return defaults
	}
	base := url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
	defaults.Upload.BaseURL = base.String()
	defaults.Upload.Path = parsed.EscapedPath()
	return defaults
}
