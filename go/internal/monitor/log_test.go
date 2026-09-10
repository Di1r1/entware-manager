package monitor

import (
	"encoding/json"
	"strings"
	"testing"
)

// collectLogJSON: уровень по [WARN]/[ERROR], время — первый [...]-блок, текст — строка целиком.
func TestCollectLogJSON(t *testing.T) {
	entries := collectLogJSON([]string{
		`[2026-09-10 10:00:00] [INFO] [__VG_IPV4_17a4e350a1ee__] [1] [monitor] watchdog started`,
		`[2026-09-10 10:01:00] [WARN] [__VG_IPV4_17a4e350a1ee__] [1] [monitor] high temp (58)`,
		`[2026-09-10 10:02:00] [ERROR] [__VG_IPV4_17a4e350a1ee__] [1] [monitor] oom`,
	})
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Level != "INFO" || entries[0].Timestamp != "2026-09-10 10:00:00" {
		t.Errorf("first entry broken: %+v", entries[0])
	}
	if entries[1].Level != "WARN" {
		t.Errorf("expected WARN, got '%s'", entries[1].Level)
	}
	if entries[2].Level != "ERROR" {
		t.Errorf("expected ERROR, got '%s'", entries[2].Level)
	}
	out, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, k := range []string{`"timestamp"`, `"level"`, `"text"`} {
		if !strings.Contains(string(out), k) {
			t.Errorf("key %s missing in %s", k, out)
		}
	}
}
