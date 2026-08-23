// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package cgiutil

import "testing"

func TestClientIP(t *testing.T) {
	cases := []struct {
		name      string
		remote    string
		xff, xrip string
		want      string
	}{
		{"прямой публичный", "203.0.113.5:44312", "6.6.6.6", "", "203.0.113.5"},
		{"loopback + XFF", "127.0.0.1:8080", "198.51.100.7", "", "198.51.100.7"},
		{"loopback + X-Real-IP", "[::1]:9000", "", "198.51.100.9", "198.51.100.9"},
		{"приватная подсеть + XFF", "192.168.3.10:5000", "203.0.113.99", "", "203.0.113.99"},
		{"XFF с цепочкой прокси", "127.0.0.1:80", "203.0.113.1, 10.0.0.2", "", "203.0.113.1"},
		{"подделка XFF без доверенного источника", "8.8.8.8:1234", "1.2.3.4", "", "8.8.8.8"},
		{"битый XFF остаётся remote", "127.0.0.1:80", "not-an-ip", "", "127.0.0.1"},
		{"пустой XFF", "127.0.0.1:80", "", "", "127.0.0.1"},
		{"IPv6 клиент", "[2001:db8::5]:443", "", "", "2001:db8::5"},
	}
	for _, c := range cases {
		if got := ClientIP(c.remote, c.xff, c.xrip); got != c.want {
			t.Errorf("%s: ClientIP(%q,%q,%q) = %q, want %q", c.name, c.remote, c.xff, c.xrip, got, c.want)
		}
	}
}

func TestClientIPEmpty(t *testing.T) {
	if got := ClientIP("", "", ""); got != "" {
		t.Errorf("пустой remote = %q, want пусто", got)
	}
}
