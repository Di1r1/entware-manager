package stats

import (
	"path/filepath"
	"testing"
)

func TestSafeSymlinkTarget(t *testing.T) {
	tmp := "/tmp/ewm_extract"
	cases := []struct {
		name     string
		linkRel  string
		linkname string
		wantOK   bool
	}{
		{"cgi bin symlink to go.cgi", "cgi-bin/api.cgi", "go.cgi", true},
		{"subdir to ../go.cgi", "logger/rotate.cgi", "../go.cgi", true},
		{"absolute blocked", "cgi-bin/api.cgi", "/etc/passwd", false},
		{"escape via ..", "cgi-bin/a.cgi", "../../../../etc/passwd", false},
		{"empty blocked", "cgi-bin/a.cgi", "", false},
		{"deep inside ok", "a/b/c.cgi", "../../go.cgi", true},
	}
	for _, c := range cases {
		gotPath, ok := safeSymlinkTarget(tmp, c.linkRel, c.linkname)
		if ok != c.wantOK {
			t.Errorf("%s: safeSymlinkTarget(%q, %q) ok=%v, want %v",
				c.name, c.linkRel, c.linkname, ok, c.wantOK)
			continue
		}
		if c.wantOK && filepath.Join(tmp, c.linkRel) != gotPath {
			t.Errorf("%s: path = %q, want %q", c.name, gotPath, filepath.Join(tmp, c.linkRel))
		}
	}
}
