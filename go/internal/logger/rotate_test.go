package logger

import "testing"

func TestParseRotated(t *testing.T) {
	files := parseRotated("ROTATED|/opt/var/log/entware/2026-08-15.log|12345\nROTATED|/opt/var/log/entware/service_events.log|42\n\njunk line\n")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	if files[0].Path != "/opt/var/log/entware/2026-08-15.log" || files[0].Size != 12345 {
		t.Errorf("unexpected first file: %+v", files[0])
	}
	if files[1].Path != "/opt/var/log/entware/service_events.log" || files[1].Size != 42 {
		t.Errorf("unexpected second file: %+v", files[1])
	}
}

func TestParseRotated_Empty(t *testing.T) {
	if files := parseRotated(""); len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
	if files := parseRotated("Ротация выполнена\nsome output\n"); len(files) != 0 {
		t.Errorf("expected no files for non-ROTATED output, got %v", files)
	}
}

func TestParseRotated_BadSize(t *testing.T) {
	files := parseRotated("ROTATED|/opt/var/log/entware/x.log|abc\nROTATED|/opt/var/log/entware/y.log|-7\n")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Size != 0 || files[1].Size != -7 {
		t.Errorf("unexpected sizes: %+v %+v", files[0], files[1])
	}
}

func TestRotationSummary(t *testing.T) {
	files := []RotatedFile{
		{Path: "/opt/var/log/entware/service_events.log", Size: 226},
		{Path: "/opt/var/log/entware/network_events.log", Size: 1536},
	}
	want := "/opt/var/log/entware/service_events.log (226B), /opt/var/log/entware/network_events.log (1K)"
	if got := rotationSummary(files); got != want {
		t.Errorf("rotationSummary: expected %q, got %q", want, got)
	}
}

func TestRotationSummary_Empty(t *testing.T) {
	if got := rotationSummary(nil); got != "файлов для ротации нет" {
		t.Errorf("expected empty message, got %q", got)
	}
}
