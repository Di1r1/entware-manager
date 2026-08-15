package stats

import "testing"

func TestIsSafeLinkURL(t *testing.T) {
	valid := []string{
		"http://example.com",
		"https://example.com/path?q=1",
		"https://example.com:8080/",
		"/terminal/",
		"/htop/",
		"/",
	}
	for _, u := range valid {
		if !isSafeLinkURL(u) {
			t.Errorf("expected valid URL %q", u)
		}
	}

	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"//evil.com/x",
		"ftp://example.com",
		"file:///etc/passwd",
		"http://",
		"http:///path",
		"  ",
	}
	for _, u := range invalid {
		if isSafeLinkURL(u) {
			t.Errorf("expected invalid URL %q", u)
		}
	}
}

func TestIsSafeLinkIcon(t *testing.T) {
	valid := []string{"", "link", "router", "chart-2", "a_b", "Terminal"}
	for _, s := range valid {
		if !isSafeLinkIcon(s) {
			t.Errorf("expected valid icon %q", s)
		}
	}

	invalid := []string{
		"x\"><script>alert(1)</script>",
		"icon with space",
		"a/b",
		"a\x00b",
	}
	for _, s := range invalid {
		if isSafeLinkIcon(s) {
			t.Errorf("expected invalid icon %q", s)
		}
	}
}
