package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
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

// newSoftwareTransferDefaults 生成 UFTE / 软件模块的传输默认 Snapshot
// （sys_configs 未配置 acs_transfer.* 时兜底，见 transfercfg.Policy）。
// 与 PM 上传模板（pm.upload_url_template）同款：支持 OMC_PUBLIC_HOST 环境变量——
// 部署方用 docker-compose 注入对外可达 host（如 OMC_PUBLIC_HOST=172.19.1.132），
// 新环境无需改 yaml / 配 sys_config 即可让设备连上 ACS 上传/下载服务。
// 优先级：yaml acs_upload_base_url > OMC_PUBLIC_HOST > localhost。
func newSoftwareTransferDefaults(cfg appconfig.UpgradeConfig) transfercfg.Snapshot {
	baseURL := strings.TrimSpace(cfg.ACSUploadBaseURL)
	if baseURL == "" {
		baseURL = transferPublicBaseURL()
	}
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
	if baseURL == "" {
		baseURL = transferPublicBaseURL()
	}
	return transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
		Upload: transfercfg.UploadSettings{
			BaseURL: baseURL,
			Path:    "/smallcell/FileUploadService",
		},
	}
}

// transferPublicBaseURL 推导设备可达的 ACS 上传/下载服务 base URL。
// 优先级：OMC_PUBLIC_HOST 环境变量（docker-compose 可注入对外可达 host）>
// 自动探测宿主机主网卡 IPv4 > localhost。端口取 OMC_PUBLIC_ACS_PORT（默认 7557，
// ACS HTTP 监听端口）。
// 仅作为 yaml/sys_config 均未配置时的多环境免手配兜底；显式配置
// （yaml acs_upload_base_url / sys_config acs_transfer.uploadBaseURL）始终优先。
func transferPublicBaseURL() string {
	host := os.Getenv("OMC_PUBLIC_HOST")
	if host == "" {
		host = detectHostIPv4()
	}
	if host == "" {
		host = "localhost"
	}
	port := 7557
	if p := os.Getenv("OMC_PUBLIC_ACS_PORT"); p != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
			port = n
		}
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// detectHostIPv4 探测本机主网卡的非回环 IPv4 地址（作为设备可达的 ACS host 兜底）。
// 返回空串表示探测失败（调用方回退 localhost）。
func detectHostIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var fallback string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			if ip.IsPrivate() {
				fallback = ip.String()
				// 偏好 docker0 网段外的私网地址（compose host 网络场景常见）
				if !strings.HasPrefix(ip.String(), "172.17.") {
					return ip.String()
				}
			}
		}
	}
	return fallback
}
