package smart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectType_NVMe(t *testing.T) {
	if detectType("nvme0n1") != "nvme" {
		t.Error("expected nvme for nvme device")
	}
}

func TestDetectType_SATA(t *testing.T) {
	if detectType("sda") != "sat" {
		t.Error("expected sat for sata device")
	}
}

func TestIsDisk_SDA(t *testing.T) {
	if !isDisk("sda") {
		t.Error("sda should be a disk")
	}
	if !isDisk("sdz") {
		t.Error("sdz should be a disk")
	}
	if isDisk("sda1") {
		t.Error("sda1 should not be a disk")
	}
	if isDisk("sdaa") {
		t.Error("sdaa should not be a disk (too long)")
	}
}

func TestIsDisk_NVMe(t *testing.T) {
	if !isDisk("nvme0n1") {
		t.Error("nvme0n1 should be a disk")
	}
	if !isDisk("nvme1n2") {
		t.Error("nvme1n2 should be a disk")
	}
	if isDisk("nvme0n1p1") {
		t.Error("nvme partition should not be a disk")
	}
}

func TestParseIntOrNull(t *testing.T) {
	if v := parseIntOrNull("35"); v != 35 {
		t.Errorf("expected 35, got %v", v)
	}
	if v := parseIntOrNull("+123"); v != 123 {
		t.Errorf("expected 123, got %v", v)
	}
	if v := parseIntOrNull(""); v != nil {
		t.Errorf("expected nil, got %v", v)
	}
	if v := parseIntOrNull("-"); v != nil {
		t.Errorf("expected nil for '-', got %v", v)
	}
}

func TestExtractField(t *testing.T) {
	output := "SMART overall-health self-assessment test result: PASSED\nSome other line"
	result := extractField(output, "SMART overall-health", -1)
	if !strings.Contains(result, "PASSED") {
		t.Errorf("expected PASSED in result, got %q", result)
	}
}

func TestExtractFieldAfter(t *testing.T) {
	output := "Device Model:     ST1000DM003-1CH162\nSerial Number:    W1D12345"
	model := extractFieldAfter(output, "Device Model")
	if model != "ST1000DM003-1CH162" {
		t.Errorf("expected ST1000DM003-1CH162, got %q", model)
	}
}

func TestDiscoverDisks_WithTempFile(t *testing.T) {
	orig := procPartitions
	procPartitions = filepath.Join(t.TempDir(), "partitions")
	defer func() { procPartitions = orig }()

	content := `major minor  #blocks  name

   8        0  976762584 sda
   8        1  976760832 sda1
   8       16  488386584 sdb
 259        0  500118192 nvme0n1
 259        1  500118192 nvme0n1p1
`
	os.WriteFile(procPartitions, []byte(content), 0644)

	disks := discoverDisks()
	if len(disks) != 3 {
		t.Fatalf("expected 3 disks (sda, sdb, nvme0n1), got %d: %v", len(disks), disks)
	}
	if disks[0] != "sda" {
		t.Errorf("expected sda, got %s", disks[0])
	}
	if disks[1] != "sdb" {
		t.Errorf("expected sdb, got %s", disks[1])
	}
	if disks[2] != "nvme0n1" {
		t.Errorf("expected nvme0n1, got %s", disks[2])
	}
}

func TestDiskSize(t *testing.T) {
	orig := procPartitions
	procPartitions = filepath.Join(t.TempDir(), "partitions")
	defer func() { procPartitions = orig }()

	content := `major minor  #blocks  name
   8        0  1000000 sda
`
	os.WriteFile(procPartitions, []byte(content), 0644)

	size := diskSize("sda")
	expected := "1024000000" // 1000000 * 1024
	if size != expected {
		t.Errorf("expected %s, got %s", expected, size)
	}
}

func TestIsRemovable(t *testing.T) {
	orig := sysBlockDir
	sysBlockDir = t.TempDir()
	defer func() { sysBlockDir = orig }()

	os.Mkdir(filepath.Join(sysBlockDir, "sda"), 0755)
	os.WriteFile(filepath.Join(sysBlockDir, "sda", "removable"), []byte("1"), 0644)

	if !isRemovable("sda") {
		t.Error("expected removable")
	}

	os.WriteFile(filepath.Join(sysBlockDir, "sda", "removable"), []byte("0"), 0644)
	if isRemovable("sda") {
		t.Error("expected not removable")
	}
}

func TestIsAttrLine(t *testing.T) {
	if !isAttrLine("  1 Raw_Read_Error_Rate 0x000f 100 100 062 Pre-fail Always - 0") {
		t.Error("should be attr line")
	}
	if !isAttrLine("5 Reallocated_Sector_Ct 0x0033 100 100 036 Pre-fail Always - 0") {
		t.Error("should be attr line")
	}
	if isAttrLine("ID# ATTRIBUTE_NAME FLAG VALUE WORST THRESH TYPE UPDATED WHEN_FAILED RAW_VALUE") {
		t.Error("should not be attr line (header)")
	}
	if isAttrLine("") {
		t.Error("empty should not be attr line")
	}
}

func TestAtoiWithDefault(t *testing.T) {
	if v := atoiWithDefault("100"); v != 100 {
		t.Errorf("expected 100, got %d", v)
	}
	if v := atoiWithDefault("036"); v != 36 {
		t.Errorf("expected 36 (stripped leading 0), got %d", v)
	}
	if v := atoiWithDefault(""); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
	if v := atoiWithDefault("-"); v != 0 {
		t.Errorf("expected 0 for '-', got %d", v)
	}
}

func TestHandleSmart_NoActionDefaultsToList(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	// This should work without panicking, and return a disk list
	// Since /proc/partitions is real, this may return empty disks
	body := captureStdout(t, HandleSmart)
	if len(body) == 0 {
		t.Fatal("expected output")
	}
}

func TestHandleSmart_POSTMethodNotAllowed(t *testing.T) {
	// Actually smart handles both GET and POST, so this should work
	// Let's test a POST with action in body
	os.Setenv("REQUEST_METHOD", "POST")
	os.Setenv("QUERY_STRING", "")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	// This should not panic
	body := captureStdout(t, HandleSmart)
	if len(body) == 0 {
		t.Fatal("expected output")
	}
}
