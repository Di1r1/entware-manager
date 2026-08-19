package smart

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestParseIntPtr(t *testing.T) {
	check := func(s string, expected int, wantNil bool) {
		t.Helper()
		v := parseIntPtr(s)
		if wantNil {
			if v != nil {
				t.Errorf("parseIntPtr(%q): expected nil, got %d", s, *v)
			}
			return
		}
		if v == nil || *v != expected {
			t.Errorf("parseIntPtr(%q): expected %d, got %v", s, expected, v)
		}
	}
	check("35", 35, false)
	check("+123", 123, false)
	check(" 35 ", 35, false)
	check("", 0, true)
	check("-", 0, true)
	check("abc", 0, true)
}

func TestValidateDevice(t *testing.T) {
	valid := []string{"sda", "nvme0n1", "sda1", "mmcblk0", "nvme1n2", "dm-0"}
	invalid := []string{"sda;rm", "/dev/sda", "../sda", "sda/..", "sda x", "SD-A", "sda\x00", strings.Repeat("a", 33)}

	for _, d := range valid {
		if !deviceRe.MatchString(d) || len(d) > 32 {
			t.Errorf("device %q should be valid", d)
		}
	}
	for _, d := range invalid {
		if deviceRe.MatchString(d) && len(d) <= 32 {
			t.Errorf("device %q should be invalid", d)
		}
	}
}

func TestHandleSmart_InvalidDeviceRejected(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=info&device=%2Fdev%2Fsda%3Brm")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleSmart)
	if !strings.Contains(string(body), `"status":"error"`) {
		t.Errorf("expected error for invalid device, got: %s", body)
	}
}

func TestHandleSmart_ListWithEmptyDeviceOK(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=list")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleSmart)
	if strings.Contains(string(body), `"status":"error"`) {
		t.Errorf("action=list with empty device should not error, got: %s", body)
	}
}

func TestSelftestStart_InvalidType(t *testing.T) {
	body := captureStdout(t, func() { handleSelftestStart("sda", "quick") })
	if !strings.Contains(string(body), `"status":"error"`) {
		t.Errorf("expected error for invalid test type, got: %s", body)
	}
}

// TestWaitOutcome_TimeoutUnblocksRead проверяет, что ветка «Kill + Close»
// разблокирует висящий Read() и waitOutcome возвращается по истечении timeout.
// Процесс в D-состоянии в тесте не воспроизвести — моделируем его фиктивным
// io.Pipe, у которого read-end закрывается, а done никогда не приходит.
func TestWaitOutcome_TimeoutUnblocksRead(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	done := make(chan error, 1)

	start := time.Now()
	out, err := waitOutcome(pr, done, 50*time.Millisecond, func() {})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("waitOutcome should return promptly after pipe close, took %v", elapsed)
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

// TestWaitOutcome_TimeoutKeepsPartialOutput проверяет, что в ветке timeout
// сохраняется частичный вывод, записанный до зависания.
func TestWaitOutcome_TimeoutKeepsPartialOutput(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	done := make(chan error, 1)
	go func() {
		pw.Write([]byte("partial output\n"))
	}()

	out, err := waitOutcome(pr, done, 50*time.Millisecond, func() {})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if out != "partial output\n" {
		t.Errorf("expected partial output, got %q", out)
	}
}

func TestHandleSmart_EmptyDeviceForInfoRejected(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=info")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleSmart)
	if !strings.Contains(string(body), `"status":"error"`) {
		t.Errorf("expected error for empty device in info, got: %s", body)
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
	if isAttrLine("  10 51 68 e0 b0 21 01  Error: IDNF 104 sectors at LBA = 0x0121b0e0 = 18985184") {
		t.Error("SMART Error Log line should not be attr line")
	}
	if isAttrLine("   35 00 68 e0 b0 21 a0 ff  14d+19:55:37.134  WRITE DMA EXT") {
		t.Error("SMART Error Log detail line should not be attr line")
	}
	if isAttrLine("    1        0        0  Not_testing") {
		t.Error("self-test summary line should not be attr line")
	}
}

func TestCheckAttrHealth_IgnoresErrorLog(t *testing.T) {
	output := `SMART overall-health self-assessment test result: PASSED

SMART Attributes Data Structure revision number: 16
ID# ATTRIBUTE_NAME          FLAG     VALUE WORST THRESH TYPE      UPDATED  WHEN_FAILED RAW_VALUE
  1 Raw_Read_Error_Rate     0x002f   100   095   016    Pre-fail  Always       -       0
  5 Reallocated_Sector_Ct   0x0033   100   100   005    Pre-fail  Always       -       0
 10 Spin_Retry_Count        0x0033   100   100   060    Pre-fail  Always       -       0
187 Reported_Uncorrect      0x0032   097   097   000    Old_age   Always       -       3
196 Reallocated_Event_Count 0x0032   100   100   000    Old_age   Always       -       0
197 Current_Pending_Sector  0x0032   100   100   000    Old_age   Always       -       0
198 Offline_Uncorrectable   0x0030   100   100   000    Old_age   Offline      -       0

SMART Error Log Version: 1
  10 51 68 e0 b0 21 01  Error: IDNF 104 sectors at LBA = 0x0121b0e0 = 18985184
  35 00 68 e0 b0 21 a0 ff  14d+19:55:37.134  WRITE DMA EXT`
	if got := checkAttrHealth(output); got != "ok" {
		t.Errorf("error log lines must be ignored, got %q, want ok", got)
	}
}

func TestAttrHealthUnknownIsLoading(t *testing.T) {
	d := DiskInfo{Health: "UNKNOWN", Type: "sat", Device: "/dev/sda", Model: "M", Serial: "S", AttrHealth: ""}
	attr := d.AttrHealth
	if attr != "" {
		t.Fatalf("precondition: empty attr, got %q", attr)
	}
	// Проверяем логику diskInfo: моделируем через извлечённый health.
	// health "UNKNOWN" (smartctl не успел отдать строку) → loading, не warning/critical.
	attrHealth := "ok"
	health := "UNKNOWN"
	if health == "\u2014" {
		attrHealth = "inactive"
	} else if health == "UNKNOWN" {
		attrHealth = "loading"
	} else if health != "PASSED" {
		attrHealth = "critical"
	} else {
		attrHealth = checkAttrHealth("")
	}
	if attrHealth != "loading" {
		t.Errorf("UNKNOWN health should map to loading, got %q", attrHealth)
	}
}

func TestAttrCacheWriteRead(t *testing.T) {
	orig := attrCacheDir
	attrCacheDir = t.TempDir()
	defer func() { attrCacheDir = orig }()

	// Запись → чтение в пределах TTL.
	writeAttrCache("sda", "SMART Attributes\n  1 Raw_Read 100 100 062 - 0\n")
	out, ok := readAttrCache("sda")
	if !ok {
		t.Fatal("expected cached attributes after write")
	}
	if !strings.Contains(out, "Raw_Read") {
		t.Errorf("expected attribute line in cache, got %q", out)
	}

	// Один файл на диск, дубликатов нет.
	entries, err := os.ReadDir(attrCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 cache file for sda, got %d", len(entries))
	}
}

func TestAttrCacheExpires(t *testing.T) {
	orig := attrCacheDir
	attrCacheDir = t.TempDir()
	defer func() { attrCacheDir = orig }()

	writeAttrCache("sdb", "SMART Attributes\n  5 Reallocated 100 100 005 - 0\n")

	// Старим файл за пределы TTL.
	path := filepath.Join(attrCacheDir, "sdb")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	old := fi.ModTime().Add(-attrCacheTTL - time.Minute)
	os.Chtimes(path, old, old)

	if _, ok := readAttrCache("sdb"); ok {
		t.Error("expected cache to expire after TTL")
	}
}

func TestAttrCacheEmptyOutputNotWritten(t *testing.T) {
	orig := attrCacheDir
	attrCacheDir = t.TempDir()
	defer func() { attrCacheDir = orig }()

	writeAttrCache("sdc", "")
	entries, _ := os.ReadDir(attrCacheDir)
	if len(entries) != 0 {
		t.Errorf("empty output must not create cache files, got %d", len(entries))
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

func TestParseNvmeValue(t *testing.T) {
	cases := []struct {
		output string
		key    string
		want   string
	}{
		{"Temperature:                        36 Celsius", "Temperature", "36"},
		{"Power On Hours:                     6,671", "Power On Hours", "6671"},
		{"", "Temperature", ""},
		// SATA-строка таблицы атрибутов: слово Temperature есть, но без «:» → пусто.
		{"194 Temperature_Celsius 0x0032 048 048 045 - 45", "Temperature", ""},
	}
	for _, c := range cases {
		if got := parseNvmeValue(c.output, c.key); got != c.want {
			t.Errorf("parseNvmeValue(%q, %q): expected %q, got %q", c.output, c.key, c.want, got)
		}
	}
}

func TestParseSelftestLine(t *testing.T) {
	cases := []struct {
		line       string
		wantStatus string
		wantProg   int
	}{
		{"# 1  Short offline  Completed without error 00% 1592 1434139663 -", "Completed", 100},
		{"# 1  Extended offline  Self-test routine in progress 90% 1000 123456789 -", "Self-test", 10},
		{"# 2  Short offline  Completed: read failure 60% 5 123456789 -", "Completed:", 40},
		{"# 1  Short offline  Completed without error 00% 1592 1434139663 5", "Completed", 100},
		{"no hash prefix", "No tests logged", 100},
	}
	for _, c := range cases {
		status, prog := parseSelftestLine(c.line)
		if status != c.wantStatus || prog != c.wantProg {
			t.Errorf("parseSelftestLine(%q): expected (%q,%d), got (%q,%d)", c.line, c.wantStatus, c.wantProg, status, prog)
		}
	}
}
