package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLinkType(t *testing.T) {
	cases := []struct {
		in       string
		wantTyp  string
		wantMAC  string
	}{
		{"<BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP \\    link/ether d4:9c:53:25:64:d9 brd ff:ff:ff:ff:ff:ff", "ether", "d4:9c:53:25:64:d9"},
		{"<LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN \\    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00", "loopback", "00:00:00:00:00:00"},
		{"<NOARP> mtu 1480 qdisc noop state DOWN \\    link/ipip 0.0.0.0 brd 0.0.0.0", "ipip", ""},
		{"<NOARP> mtu 1452 qdisc noop state DOWN \\    link/tunnel6 :: brd ::", "tunnel6", ""},
		{"<POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1492 \\    link/ppp ", "ppp", ""},
		{"<NO-CARRIER,POINTOPOINT,MULTICAST,NOARP,UP> mtu 1500 \\    link/none ", "none", ""},
	}
	for _, c := range cases {
		typ, mac := parseLinkType(c.in)
		if typ != c.wantTyp {
			t.Errorf("parseLinkType(%q) typ = %q, want %q", c.in, typ, c.wantTyp)
		}
		if mac != c.wantMAC {
			t.Errorf("parseLinkType(%q) mac = %q, want %q", c.in, mac, c.wantMAC)
		}
	}
}

func TestIsMAC(t *testing.T) {
	cases := map[string]bool{
		"d4:9c:53:25:64:d9": true,
		"00:00:00:00:00:00": true,
		"0.0.0.0":           false,
		"::":                false,
		"d4:9c:53":          false,
		"":                  false,
	}
	for in, want := range cases {
		if got := isMAC(in); got != want {
			t.Errorf("isMAC(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFetchRCIWifiSSIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"WifiMaster0/AccessPoint0": {"mac": "d4:9c:53:25:64:d9", "ssid": "DiZyXEL"},
			"WifiMaster0/AccessPoint1": {"mac": "d6:9c:53:15:64:d9"},
			"WifiMaster1/AccessPoint0": {"mac": "d4:9c:53:25:64:dc", "ssid": "DiZyXEL-5G"},
			"Bridge0": {"mac": "d4:9c:53:25:64:d9", "ssid": ""}
		}`))
	}))
	defer server.Close()

	orig := rciIfaceURL
	rciIfaceURL = server.URL
	defer func() { rciIfaceURL = orig }()

	ssids := fetchRCIWifiSSIDs()

	if got := ssids["d4:9c:53:25:64:d9"]; got != "DiZyXEL" {
		t.Errorf("ssid для d4:9c:53:25:64:d9 = %q, want DiZyXEL", got)
	}
	if got := ssids["d4:9c:53:25:64:dc"]; got != "DiZyXEL-5G" {
		t.Errorf("ssid для d4:9c:53:25:64:dc = %q, want DiZyXEL-5G", got)
	}
	// без ssid и с пустым ssid — не попадают
	if _, ok := ssids["d6:9c:53:15:64:d9"]; ok {
		t.Error("запись без ssid не должна попасть в map")
	}
	if _, ok := ssids["Bridge0"]; ok {
		t.Error("пустой ssid не должен попасть в map")
	}
}

func TestFetchRCIWifiSSIDs_Error(t *testing.T) {
	orig := rciIfaceURL
	rciIfaceURL = "http://127.0.0.1:1/rci/show/interface/"
	defer func() { rciIfaceURL = orig }()

	ssids := fetchRCIWifiSSIDs()
	if len(ssids) != 0 {
		t.Errorf("при ошибке RCI ожидали пустую map, got %v", ssids)
	}
}
