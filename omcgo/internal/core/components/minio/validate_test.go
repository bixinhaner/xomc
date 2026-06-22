package minio

import "testing"

func TestValidatePublicEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty 合法（走回退）", "", false},
		{"纯 host", "minio.example.com", false},
		{"host:port", "minio.example.com:9000", false},
		{"IPv4", "10.0.0.5", false},
		{"IPv4:port", "10.0.0.5:9100", false},
		{"port 边界 1", "h:1", false},
		{"port 边界 65535", "h:65535", false},

		{"带 http scheme", "http://minio:9000", true},
		{"带 https scheme", "https://minio.example.com", true},
		{"带 path", "minio:9000/foo", true},
		{"带 query", "minio:9000?x=1", true},
		{"带 fragment", "minio:9000#frag", true},
		{"空 host : port", ":9000", true},
		{"port 非整数", "h:abc", true},
		{"port 0", "h:0", true},
		{"port 越界", "h:65536", true},
		{"port 负数（split 后非整数）", "h:-1", true},

		// issue #548 切片 2 · 回合 2 Explore P11 采纳：IPv6 字面量显式拒绝且错误消息明确。
		{"IPv6 字面量带 [", "[::1]:9000", true},
		{"裸 IPv6 多个 :", "::1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePublicEndpoint(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidatePublicEndpoint(%q) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}
