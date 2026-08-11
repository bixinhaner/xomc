// hardware.go — 旧项目硬件绑定校验（MAC / SystemUUID）的 Go 复刻。
//
// 对应旧项目 LicenseVerifyServiceImpl.checkOMCSupportMAC /
// checkOMCSupportSystemUUID：license 携带 MACAddress / systemUUID 时，与本机实际
// 硬件比对；任一不匹配即拒绝授权。两者为空表示不绑定（任意硬件放行）。
package license

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// validateHardwareBinding 校验 license 的 MAC/UUID 绑定是否与本机匹配。
// 字段为空 = 不绑定（跳过）；非空则按旧项目语义比对，不匹配返回 wrap 了
// commonerrors.ErrLicenseHardwareMismatch 的错误（→ service 层翻译为 403）。
func validateHardwareBinding(licMAC, licUUID string) error {
	if mac := strings.TrimSpace(licMAC); mac != "" {
		serverMACs := serverMACAddresses()
		if !macMatches(mac, serverMACs) {
			return fmt.Errorf("license MAC %q does not match server MACs %v: %w",
				mac, serverMACs, commonerrors.ErrLicenseHardwareMismatch)
		}
	}
	if uuid := strings.TrimSpace(licUUID); uuid != "" {
		serverUUID, _ := serverSystemUUID()
		if !uuidMatches(uuid, serverUUID) {
			return fmt.Errorf("license systemUUID %q does not match server UUID %q: %w",
				uuid, serverUUID, commonerrors.ErrLicenseHardwareMismatch)
		}
	}
	return nil
}

// macMatches 复刻旧项目语义：
//   - license 含逗号（多 MAC）：本机任一 MAC 命中 license 列表即通过
//   - 单 MAC：本机 MAC 列表包含该 MAC 即通过
//
// MAC 统一去掉分隔符并大写比较（如 aa:bb:cc:dd:ee:ff → AABBCCDDEEFF）。
func macMatches(licMAC string, serverMACs []string) bool {
	licSet := make(map[string]bool)
	for _, m := range strings.Split(licMAC, ",") {
		if norm := normalizeMAC(m); norm != "" {
			licSet[norm] = true
		}
	}
	if len(licSet) == 0 {
		return true
	}
	if strings.Contains(licMAC, ",") {
		for _, s := range serverMACs {
			if licSet[normalizeMAC(s)] {
				return true
			}
		}
		return false
	}
	// 单 MAC：取唯一元素
	for lic := range licSet {
		for _, s := range serverMACs {
			if normalizeMAC(s) == lic {
				return true
			}
		}
	}
	return false
}

// uuidMatches 复刻旧项目语义：大小写不敏感；多 UUID（逗号分隔）任一命中即通过。
func uuidMatches(licUUID, serverUUID string) bool {
	serverUUID = strings.ToLower(strings.TrimSpace(serverUUID))
	if serverUUID == "" {
		return false // 本机 UUID 无法获取且 license 绑定了 UUID → 不匹配
	}
	if strings.Contains(licUUID, ",") {
		for _, u := range strings.Split(licUUID, ",") {
			if strings.ToLower(strings.TrimSpace(u)) == serverUUID {
				return true
			}
		}
		return false
	}
	return strings.ToLower(strings.TrimSpace(licUUID)) == serverUUID
}

// normalizeMAC 去掉 MAC 的 ":","-" 分隔符并转大写。
func normalizeMAC(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// serverMACAddresses 返回宿主机物理网卡 MAC（大写、无分隔符），复刻旧项目
// getServerMACInfo（收集所有非 loopback）。
//
// docker 部署下：容器 net.Interfaces() 只看得到容器虚拟网卡（每次重建变化，
// 不能用于 license 绑定）。故优先读挂载进来的宿主机 /host/sys/class/net（compose
// 把宿主 /sys 只读挂到 /host/sys）；挂载不存在则回退容器自身网卡（裸机/dev 场景）。
func serverMACAddresses() []string {
	if macs := readMACsFromSysFS("/host/sys/class/net"); len(macs) > 0 {
		return macs
	}
	return netInterfacesMACs()
}

// readMACsFromSysFS 从 sysfs 的 class/net 目录读所有非 loopback MAC。
func readMACsFromSysFS(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name(), "address"))
		if err != nil {
			continue
		}
		if mac := normalizeMAC(strings.TrimSpace(string(b))); mac != "" && mac != "000000000000" {
			out = append(out, mac)
		}
	}
	return out
}

// netInterfacesMACs 回退方案：读本进程所在网络命名空间的网卡 MAC。
func netInterfacesMACs() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		out = append(out, normalizeMAC(iface.HardwareAddr.String()))
	}
	return out
}

// serverSystemUUID 读宿主机硬件 UUID（小写）。优先挂载的 /host/sys/class/dmi/id，
// 回退容器自身；都读不到返回空（license 若绑定 UUID 则会判失配）。
func serverSystemUUID() (string, error) {
	for _, p := range []string{"/host/sys/class/dmi/id/product_uuid", "/sys/class/dmi/id/product_uuid"} {
		if b, err := os.ReadFile(p); err == nil {
			if s := strings.ToLower(strings.TrimSpace(string(b))); s != "" {
				return s, nil
			}
		}
	}
	return "", nil
}
