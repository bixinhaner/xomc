// Package netutil 提供 ACS / Device 共用的网络地址校验工具。
//
// 历史上 `internal/device/device_service.go` 与 `internal/acs/handler.go` 各自
// 持有一份 `isUnspecifiedUDPAddress` / `isUnspecifiedHost` 副本，逻辑完全相同。
// 一旦未来要扩展守卫规则（如 IPv6 link-local / loopback）必然漏一处。统一到这里。
package netutil

import (
	"net"
	"strings"
)

// IsUnspecifiedUDPAddress 判断一个 UDP 地址字符串是否属于"无效/未指定"。
//
// 规则（与 TR-069 STUN/UDP CR 红线一致）：
//   - 空串、纯空白 → true
//   - 含端口但端口为空或 "0" → true
//   - host 解析为 unspecified（0.0.0.0 / ::）→ true
//   - 不含端口时，把整个字符串当作 host 走 IsUnspecifiedHost
//
// 不区分 IPv4/IPv6；不做 DNS 解析（host 是域名时按非 unspecified 处理）。
func IsUnspecifiedUDPAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return IsUnspecifiedHost(addr)
	}
	if port == "" || port == "0" {
		return true
	}
	return IsUnspecifiedHost(host)
}

// IsUnspecifiedHost 判断 host 字符串是否属于"无效/未指定 IP"。
// 空串视为 true；非 IP 字符串（域名）视为 false。
func IsUnspecifiedHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}
