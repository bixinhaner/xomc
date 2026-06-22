package minio

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidatePublicEndpoint 校验 MinIO public_endpoint 的字符串格式（issue #548 切片 2 D 后端）。
//
// 合法形式：
//   - "" （空字符串：表示走兜底回退到 env / YAML default，由 PresignBridge 处理）
//   - "host"        如 "minio.example.com" / "10.0.0.5"
//   - "host:port"   如 "minio.example.com:9100" / "10.0.0.5:9000"（port 必须 1..65535）
//
// 禁止：
//   - 带 scheme（出现 "://"）
//   - 带 path / query / fragment（出现 "/" "?" "#"）
//   - port 非整数或越界
//
// 暴露给 admin.SysConfigService 的 Validator 注册（cmd/app/provider/minio_presign_bridge.go）
// 在 BatchUpsert 阶段拒绝运维填入非法格式，HTTP 翻 400。
//
// 设计取舍：不做 DNS / 连通性校验，只做语法形态校验——保持纯函数 / 单测无依赖，
// 同时 PresignBridge 在 SetPublicEndpoint 时会再次构造 minio.Client 暴露连通性问题。
func ValidatePublicEndpoint(s string) error {
	if s == "" {
		return nil // 空 = 走回退，合法
	}
	if strings.Contains(s, "://") {
		return fmt.Errorf("minio public_endpoint 不允许带 scheme（http://、https://），请填 host[:port] 形式，收到 %q", s)
	}
	if strings.ContainsAny(s, "/?#") {
		return fmt.Errorf("minio public_endpoint 不允许带 path / query / fragment，请填 host[:port] 形式，收到 %q", s)
	}
	// 显式拒绝 IPv6 字面量（含 [ 或 ]，或裸 IPv6 即多个 :）——issue #548 切片 2 范围
	// 只支持 host / IPv4 / DNS 名（OMC 部署 MinIO public endpoint 默认走 DNS / IPv4）。
	// 这里给明确错误，比让 splitHostPort 把 host="::" port="1" 误通过更安全。
	// 回合 1 检查方 Explore P11 采纳。
	if strings.ContainsAny(s, "[]") || strings.Count(s, ":") > 1 {
		return fmt.Errorf("minio public_endpoint 暂不支持 IPv6 字面量，请使用 DNS 名或 IPv4，收到 %q", s)
	}
	// 拆 host : port
	host, port, hasPort := splitHostPort(s)
	if host == "" {
		return fmt.Errorf("minio public_endpoint host 部分为空，收到 %q", s)
	}
	if hasPort {
		p, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("minio public_endpoint 端口非整数 %q（来自 %q）", port, s)
		}
		if p < 1 || p > 65535 {
			return fmt.Errorf("minio public_endpoint 端口越界 %d（合法 1..65535，来自 %q）", p, s)
		}
	}
	return nil
}

// splitHostPort 是 net.SplitHostPort 的轻量替代：原版要求 host:port 必须含 :，
// 而本函数对纯 host（无 :）也返回 host + 空 port + hasPort=false，便于校验函数统一处理。
//
// 注：IPv6 字面量（含多个 :）当前不在 issue #548 切片 2 支持范围（OMC 部署里 MinIO public
// endpoint 默认走 DNS / IPv4），如未来扩展再补 [::1]:9000 形式解析。
func splitHostPort(s string) (host, port string, hasPort bool) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return s, "", false
	}
	// 含 : 的视作 host:port；如果是 IPv6 字面量（多个 :），按当前不支持处理——
	// 校验时 strconv.Atoi 会失败，给出明确错误。
	return s[:idx], s[idx+1:], true
}
