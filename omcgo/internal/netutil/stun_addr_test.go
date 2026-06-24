package netutil

import "testing"

func TestIsUnspecifiedUDPAddress(t *testing.T) {
	cases := []struct {
		name string
		addr string
		want bool
	}{
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"v4_unspecified_no_port", "0.0.0.0", true},
		{"v4_unspecified_with_port", "0.0.0.0:3478", true},
		{"v6_unspecified", "::", true},
		{"v6_unspecified_with_port", "[::]:3478", true},
		{"host_no_port", "192.168.1.10", false},
		{"host_with_port", "192.168.1.10:3478", false},
		{"host_zero_port", "192.168.1.10:0", true},
		{"host_empty_port", "192.168.1.10:", true},
		{"domain_no_port", "cpe.example.com", false},
		{"domain_with_port", "cpe.example.com:3478", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := IsUnspecifiedUDPAddress(c.addr)
			if got != c.want {
				t.Fatalf("IsUnspecifiedUDPAddress(%q) = %v, want %v", c.addr, got, c.want)
			}
		})
	}
}

func TestIsUnspecifiedHost(t *testing.T) {
	cases := []struct {
		name string
		host string
		want bool
	}{
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"v4_unspecified", "0.0.0.0", true},
		{"v6_unspecified", "::", true},
		{"v4_loopback", "127.0.0.1", false},
		{"v4_private", "192.168.1.1", false},
		{"v6_loopback", "::1", false},
		{"domain", "cpe.example.com", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := IsUnspecifiedHost(c.host)
			if got != c.want {
				t.Fatalf("IsUnspecifiedHost(%q) = %v, want %v", c.host, got, c.want)
			}
		})
	}
}
