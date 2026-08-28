package services

import "testing"

func TestIsTTYDOnPort(t *testing.T) {
	valid := []struct {
		cmdline string
		port    int
	}{
		{"ttyd -p 8089 -W -c admin:x -i lo --base-path /htop htop", 8089},
		{"ttyd --port 9089 -c admin:x --base-path /terminal /opt/bin/bash", 9089},
		{"ttyd --port=8089 -c admin:x /opt/bin/bash", 8089},
		{"/opt/sbin/ttyd -p 8089 -c admin:x /opt/bin/bash", 8089},
		{"ttyd 8089", 8089},
	}
	for _, tc := range valid {
		if !isTTYDOnPort(tc.cmdline, tc.port) {
			t.Errorf("expected %q to match port %d", tc.cmdline, tc.port)
		}
	}

	invalid := []struct {
		cmdline string
		port    int
	}{
		// собственный CGI / sh -c — ключевой регрессионный кейс (убивал свою сессию)
		{"sh -c ENDPOINT=ttyd_control REQUEST_METHOD=POST ... port=8089 ...", 8089},
		{"entware-services ttyd_control", 8089},
		{"ttyd -p 9089 -c admin:x /opt/bin/bash", 8089},
		{"ttyd -p 8089 -c admin:x /opt/bin/bash", 9089},
		{"htop", 8089},
		{"nginx -c admin:x -p 8089", 8089},
		{"", 8089},
		{"ttyd -p abc", 8089},
	}
	for _, tc := range invalid {
		if isTTYDOnPort(tc.cmdline, tc.port) {
			t.Errorf("expected %q NOT to match port %d", tc.cmdline, tc.port)
		}
	}
}
