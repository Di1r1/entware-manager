package smart

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/cache"
)

var (
	smartctlBin    = "/opt/sbin/smartctl"
	procPartitions = "/proc/partitions"
	sysBlockDir    = "/sys/block"
	dfBin          = "df"
	sudoBin        = "sudo"
	procDir        = "/proc"
)

type DiskInfo struct {
	Device       string `json:"device"`
	Model        string `json:"model"`
	Serial       string `json:"serial"`
	Size         string `json:"size"`
	Type         string `json:"type"`
	Health       string `json:"health"`
	Temperature  *int   `json:"temperature"`
	PowerOnHours *int   `json:"power_on_hours"`
	AttrHealth   string `json:"attr_health"`
}

var criticalAttrIDs = map[int]bool{5: true, 10: true, 187: true, 196: true, 197: true, 198: true}

var deviceRe = regexp.MustCompile(`^[a-z0-9-]+$`)

type AttrInfo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Value     int    `json:"value"`
	Worst     int    `json:"worst"`
	Threshold int    `json:"threshold"`
	Raw       string `json:"raw"`
}

type PartitionInfo struct {
	Part  string `json:"part"`
	Size  string `json:"size"`
	Used  string `json:"used"`
	Avail string `json:"avail"`
	Pct   int    `json:"pct"`
	Mnt   string `json:"mnt"`
}

func writeJSON(v any) {
	fmt.Print(jsonBody(v))
}

// jsonBody сериализует v в JSON с Content-Type (без вывода в stdout).
func jsonBody(v any) string {
	out, _ := json.Marshal(v)
	return "Content-type: application/json; charset=utf-8\n\n" + string(out)
}

func writeError(msg string) {
	writeJSON(map[string]string{"status": "error", "message": msg})
}

func notAllowed() {
	writeJSON(map[string]string{"error": "Method not allowed"})
}

func isGET() bool {
	return os.Getenv("REQUEST_METHOD") == "GET"
}

func isPOST() bool {
	return os.Getenv("REQUEST_METHOD") == "POST"
}

func getQueryParam(key string) string {
	q := os.Getenv("QUERY_STRING")
	for _, part := range strings.Split(q, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			val := kv[1]
			val = strings.ReplaceAll(val, "+", " ")
			val = urlDecode(val)
			return val
		}
	}
	return ""
}

func urlDecode(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			high := unhex(s[i+1])
			low := unhex(s[i+2])
			if high >= 0 && low >= 0 {
				sb.WriteByte(byte(high<<4 | low))
				i += 2
				continue
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
}

func readPOSTBody() string {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseFormBody(body string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(body, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := urlDecode(strings.ReplaceAll(kv[0], "+", " "))
			val := urlDecode(strings.ReplaceAll(kv[1], "+", " "))
			params[key] = val
		}
	}
	return params
}

// errDeviceBusy — на устройстве уже висит незавершённый smartctl (состояние D).
var errDeviceBusy = fmt.Errorf("smartctl busy on device")

// smartctlBusy проверяет, есть ли живой процесс smartctl на устройстве device
// (процессы в состоянии D от прошлых запросов). Сканирует /proc/<pid>/cmdline.
func smartctlBusy(device string) bool {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid := e.Name()
		if pid[0] < '0' || pid[0] > '9' {
			continue
		}
		cmdline, err := os.ReadFile(procDir + "/" + pid + "/cmdline")
		if err != nil || len(cmdline) == 0 {
			continue
		}
		// Токенизация по NUL (cmdline разделён \0), ищем smartctl и точный device.
		tokens := strings.Split(string(cmdline), "\x00")
		foundSmart := false
		foundDev := false
		for _, t := range tokens {
			if t == "" {
				continue
			}
			if filepath.Base(t) == "smartctl" {
				foundSmart = true
			}
			if t == device {
				foundDev = true
			}
		}
		if foundSmart && foundDev {
			return true
		}
	}
	return false
}

// runBounded выполняет команду с жёстким дедлайном. Если процесс не завершился
// вовремя (например, ушёл в D-состояние — непрерываемый сон на подвешенном диске),
// убивает его (сигнал встаёт в очередь) и возвращает partial-вывод. Wait() не
// вызывается для зависшего процесса — иначе заблокируемся навсегда.
func runBounded(cmd *exec.Cmd, timeout time.Duration) (string, error) {
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	buf := make([]byte, 0, 4096)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		tmp := make([]byte, 4096)
		for {
			n, err := out.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
		<-readDone
		return string(buf), nil
	case <-time.After(timeout):
		// Процесс, вероятно, в D-состоянии. Kill() очередит SIGKILL —
		// умрёт, когда диск придёт в себя. Wait() не вызываем.
		cmd.Process.Kill()
		<-readDone
		return string(buf), fmt.Errorf("timeout after %v", timeout)
	}
}

func smartctlRun(device string, args ...string) (string, error) {
	if smartctlBusy(device) {
		return "", errDeviceBusy
	}

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, device)

	out, err := runBounded(exec.Command(smartctlBin, allArgs...), 8*time.Second)
	outStr := out
	if err == nil {
		return outStr, nil
	}

	// Try with sudo
	sudoArgs := append([]string{smartctlBin}, allArgs...)
	out2, err2 := runBounded(exec.Command(sudoBin, sudoArgs...), 8*time.Second)
	if err2 == nil {
		return string(out2), nil
	}

	// Return partial output + error
	return outStr, fmt.Errorf("smartctl failed: %w (sudo: %v)", err, err2)
}

// busyDiskInfo — карточка диска, на котором висит незавершённый smartctl.
// Данные только из /sys (не блокируются), health «—», attr_health «busy».
func busyDiskInfo(name, devpath, diskType string) DiskInfo {
	displayType := diskType
	if isRemovable(name) {
		displayType = "usb"
	}
	model := ""
	if data, err := os.ReadFile(filepath.Join(sysBlockDir, name, "device", "model")); err == nil {
		model = strings.TrimSpace(string(data))
	}
	serial := ""
	if data, err := os.ReadFile(filepath.Join(sysBlockDir, name, "device", "serial")); err == nil {
		serial = strings.TrimSpace(string(data))
	}
	if model == "" {
		model = "\u2014"
	}
	if serial == "" {
		serial = "\u2014"
	}
	return DiskInfo{
		Device:       devpath,
		Model:        model,
		Serial:       serial,
		Size:         diskSize(name),
		Type:         displayType,
		Health:       "\u2014",
		Temperature:  nil,
		PowerOnHours: nil,
		AttrHealth:   "busy",
	}
}

func detectType(device string) string {
	if strings.HasPrefix(device, "nvme") {
		return "nvme"
	}
	return "sat"
}

func discoverDisks() []string {
	data, err := os.ReadFile(procPartitions)
	if err != nil {
		return nil
	}
	var disks []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := fields[3]
			if isDisk(name) {
				disks = append(disks, name)
			}
		}
	}
	return disks
}

func isDisk(name string) bool {
	if len(name) == 3 && name[:2] == "sd" && name[2] >= 'a' && name[2] <= 'z' {
		return true
	}
	// nvme0n1, nvme1n2 — partitions have 'p' (nvme0n1p1)
	if strings.HasPrefix(name, "nvme") && !strings.Contains(name, "p") {
		return true
	}
	return false
}

func diskSize(name string) string {
	data, err := os.ReadFile(procPartitions)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[3] == name && len(fields) >= 3 {
			blocks, _ := strconv.ParseUint(fields[2], 10, 64)
			return fmt.Sprintf("%d", blocks*1024)
		}
	}
	return ""
}

func isRemovable(name string) bool {
	data, err := os.ReadFile(filepath.Join(sysBlockDir, name, "removable"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

func extractField(output, pattern string, field int) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			if field < 0 {
				return line
			}
			fields := strings.Fields(line)
			if field < len(fields) {
				return fields[field]
			}
		}
	}
	return ""
}

func extractFieldAfter(output, pattern string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				return strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return ""
}

func parseIntPtr(s string) *int {
	if s == "" || s == "-" {
		return nil
	}
	if strings.ContainsAny(s, "+-") {
		s = strings.TrimLeft(s, "+")
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &n
}

// parseNvmeValue извлекает числовое значение из NVMe-строки «ключ: значение»:
// «Temperature: 36 Celsius» → "36"; «Power On Hours: 6,671» → "6671".
func parseNvmeValue(output, key string) string {
	v := extractFieldAfter(output, key)
	if v == "" {
		return ""
	}
	fields := strings.Fields(v)
	if len(fields) == 0 {
		return ""
	}
	return strings.ReplaceAll(fields[0], ",", "")
}

func HandleSmart() {
	if !isGET() && !isPOST() {
		notAllowed()
		return
	}

	var postParams map[string]string
	if isPOST() {
		postParams = parseFormBody(readPOSTBody())
	}

	getParam := func(key string) string {
		v := getQueryParam(key)
		if v == "" && postParams != nil {
			v = postParams[key]
		}
		return v
	}

	action := getParam("action")
	if action == "" {
		action = "list"
	}

	device := strings.TrimPrefix(getParam("device"), "/dev/")
	if device != "" && (!deviceRe.MatchString(device) || len(device) > 32) {
		writeError("invalid device")
		return
	}

	switch action {
	case "list":
		handleList()
	case "info":
		handleInfo(device)
	case "attributes":
		handleAttributes(device)
	case "health":
		handleHealth(device)
	case "usage":
		handleUsage(device)
	case "selftest":
		if isPOST() {
			if auth.IsCrossSiteOrigin() {
				writeError(auth.CrossSiteDeny)
				return
			}
			testType := getParam("type")
			if testType == "" {
				testType = "short"
			}
			handleSelftestStart(device, testType)
		} else {
			handleSelftestStatus(device)
		}
	default:
		writeError("Unknown action")
	}
}

func checkAttrHealth(output string) string {
	// Default: PASSED → ok
	if output == "" {
		return "unknown"
	}
	scanner := bufio.NewScanner(strings.NewReader(output))
	worst := "ok"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !isAttrLine(line) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		id, _ := strconv.Atoi(fields[0])
		if !criticalAttrIDs[id] {
			continue
		}
		val := atoiWithDefault(fields[3])
		thresh := atoiWithDefault(fields[5])
		if thresh > 0 && val <= thresh {
			return "critical"
		}
		if thresh > 0 && val-thresh < 10 {
			worst = "warning"
		}
	}
	return worst
}

func handleList() {
	// Кэш 60с: повторные клики «Обновить» отвечают мгновенно и не запускают
	// smartctl на подвешенных дисках (процессы в D-состоянии).
	if data, ok := cache.Get("smart_list", 60*time.Second); ok {
		fmt.Print(string(data))
		return
	}

	disks := discoverDisks()
	result := make([]DiskInfo, len(disks))

	// Опрос дисков параллельно: один зависший smartctl (например, спящий диск)
	// не блокирует весь список. Каждый запрос уже ограничен timeout (см. smartctlRun).
	var wg sync.WaitGroup
	for i, name := range disks {
		wg.Add(1)
		go func(idx int, dev string) {
			defer wg.Done()
			result[idx] = diskInfo(dev)
		}(i, name)
	}
	wg.Wait()

	out := jsonBody(map[string]any{"disks": result})
	cache.Put("smart_list", []byte(out))
	fmt.Print(out)
}

func diskInfo(name string) DiskInfo {
	devpath := "/dev/" + name
	diskType := detectType(name)
	output, err := smartctlRun(devpath, "-a", "-d", diskType)
	if errors.Is(err, errDeviceBusy) {
		// На диске висит незавершённый smartctl (подвешенное устройство) —
		// отдаём базовые данные из /sys, чтобы не блокировать список.
		return busyDiskInfo(name, devpath, diskType)
	}

	displayType := diskType
	if isRemovable(name) {
		displayType = "usb"
	}

	// If USB and error/keywords, try -d scsi
	if displayType == "usb" && (output == "" || strings.Contains(strings.ToLower(output), "unknown usb bridge") ||
		strings.Contains(strings.ToLower(output), "unsupported scsi opcode") ||
		strings.Contains(strings.ToLower(output), "device lacks smart")) {
		out2, err2 := smartctlRun(devpath, "-a", "-d", "scsi")
		if err2 == nil {
			output = out2
		}
	}

	model := extractFieldAfter(output, "Device Model")
	if model == "" {
		model = extractFieldAfter(output, "Model Number")
	}
	if model == "" {
		model = extractFieldAfter(output, "Product")
	}
	if model == "" {
		modelData, _ := os.ReadFile(filepath.Join(sysBlockDir, name, "device", "model"))
		if len(modelData) > 0 {
			model = strings.TrimSpace(string(modelData))
		}
	}
	if model == "" {
		model = "Unknown"
	}

	serial := extractFieldAfter(output, "Serial Number")
	if serial == "" {
		serial = extractFieldAfter(output, "Serial")
	}
	if serial == "" {
		serialData, _ := os.ReadFile(filepath.Join(sysBlockDir, name, "device", "serial"))
		if len(serialData) > 0 {
			serial = strings.TrimSpace(string(serialData))
		}
	}
	if serial == "" {
		serial = "\u2014"
	}

	health := extractField(output, "SMART overall-health", -1)
	if health == "" {
		health = extractField(output, "SMART Health Status", -1)
	}
	if health == "" {
		health = "UNKNOWN"
	} else {
		// Get last word
		parts := strings.Fields(health)
		if len(parts) > 0 {
			health = parts[len(parts)-1]
		}
	}

	// USB flash drives don't have SMART — suppress UNKNOWN
	if health == "UNKNOWN" && displayType == "usb" {
		health = "\u2014"
	}

	temperature := extractField(output, "Temperature_Celsius", 9)
	if temperature == "" {
		temperature = extractField(output, "Current Temperature", 9)
	}
	if temperature == "" {
		temperature = extractField(output, "194", 9)
	}

	powerOn := extractField(output, "Power_On_Hours", 9)
	if powerOn == "" {
		powerOn = extractField(output, "Power On Hours", 9)
	}
	if powerOn == "" {
		powerOn = extractField(output, "9", 9)
	}
	powerOn = strings.TrimLeft(powerOn, "+")

	// NVMe: «Temperature: 36 Celsius» / «Power On Hours: 6,671» — формат «ключ: значение»,
	// таблицы атрибутов SATA нет. SATA-приоритет сохранён: ветка только для nvme.
	if displayType == "nvme" {
		if temperature == "" {
			temperature = parseNvmeValue(output, "Temperature")
		}
		if powerOn == "" {
			powerOn = parseNvmeValue(output, "Power On Hours")
		}
	}

	attrHealth := "ok"
	if health == "\u2014" {
		attrHealth = "inactive"
	} else if health != "PASSED" {
		attrHealth = "critical"
	} else {
		attrHealth = checkAttrHealth(output)
	}

	return DiskInfo{
		Device:       devpath,
		Model:        model,
		Serial:       serial,
		Size:         diskSize(name),
		Type:         displayType,
		Health:       health,
		Temperature:  parseIntPtr(temperature),
		PowerOnHours: parseIntPtr(powerOn),
		AttrHealth:   attrHealth,
	}
}

func handleInfo(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-i", "-d", diskType)
	writeJSON(map[string]string{"info": output})
}

func handleAttributes(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-A", "-d", diskType)

	var attrs []AttrInfo
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip header, match lines starting with optional spaces + digits
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !isAttrLine(trimmed) {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 9 {
			continue
		}

		id, _ := strconv.Atoi(fields[0])
		if id == 0 {
			continue
		}

		attr := AttrInfo{
			ID:        id,
			Name:      fields[1],
			Value:     atoiWithDefault(fields[3]),
			Worst:     atoiWithDefault(fields[4]),
			Threshold: atoiWithDefault(fields[5]),
			Raw:       strings.Join(fields[9:], " "),
		}
		if attr.Raw == "" || attr.Raw == "-" {
			attr.Raw = "0"
		}
		attrs = append(attrs, attr)
	}
	if attrs == nil {
		attrs = []AttrInfo{}
	}
	writeJSON(map[string]any{"attributes": attrs})
}

func isAttrLine(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := strings.Fields(s)
	if len(first) == 0 {
		return false
	}
	_, err := strconv.Atoi(first[0])
	return err == nil
}

func atoiWithDefault(s string) int {
	s = strings.TrimLeft(s, "0")
	if s == "" || s == "-" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func handleHealth(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-H", "-d", diskType)

	healthLine := extractField(output, "SMART overall-health", -1)
	if healthLine == "" {
		healthLine = extractField(output, "SMART Health Status", -1)
	}
	if healthLine == "" {
		healthLine = "SMART: Unable to determine health status"
	}

	parts := strings.Fields(healthLine)
	var result string
	if len(parts) > 0 {
		result = parts[len(parts)-1]
	}

	writeJSON(map[string]string{
		"health":  result,
		"message": healthLine,
	})
}

func handleUsage(device string) {
	if device == "" {
		writeError("device required")
		return
	}

	out, err := runBounded(exec.Command(dfBin, "-h"), 5*time.Second)
	if err != nil && out == "" {
		writeJSON(map[string]any{"partitions": []PartitionInfo{}})
		return
	}
	rawOut := []byte(out)

	var parts []PartitionInfo
	prefix := "/dev/" + device
	scanner := bufio.NewScanner(bytes.NewReader(rawOut))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix) || len(line) <= len(prefix) {
			continue
		}
		// Must be followed by a digit (e.g., /dev/sda1)
		if line[len(prefix)] < '0' || line[len(prefix)] > '9' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.Atoi(pctStr)

		parts = append(parts, PartitionInfo{
			Part:  filepath.Base(fields[0]),
			Size:  fields[1],
			Used:  fields[2],
			Avail: fields[3],
			Pct:   pct,
			Mnt:   fields[5],
		})
	}
	if parts == nil {
		parts = []PartitionInfo{}
	}
	writeJSON(map[string]any{"partitions": parts})
}

func handleSelftestStart(device string, testType string) {
	if device == "" {
		writeError("device required")
		return
	}
	switch testType {
	case "short", "long", "conveyance", "offline":
	default:
		writeError("invalid test type")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-t", testType, "-d", diskType)

	status := "error"
	msg := ""

	if strings.Contains(output, "START") {
		status = "ok"
		msg = "Тест " + testType + " запущен"
	} else if strings.Contains(output, "already") {
		status = "error"
		msg = "Тест уже выполняется"
	} else {
		lines := strings.SplitN(output, "\n", 6)
		msg = strings.Join(lines[:min(5, len(lines))], " ")
		msg = strings.TrimSpace(msg)
	}

	writeJSON(map[string]string{"status": status, "message": msg})
}

func handleSelftestStatus(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-l", "selftest", "-d", diskType)

	status := "No tests logged"
	progress := 100

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			status, progress = parseSelftestLine(line)
			break
		}
	}

	writeJSON(map[string]any{"status": status, "progress": progress})
}

// parseSelftestLine разбирает строку журнала самотеста:
// «# 1  Short offline  Completed without error 00% …» → status="Completed", progress=100;
// «# 1  Extended offline  Self-test routine in progress 90% …» → status="Self-test", progress=10.
// Remaining — % оставшегося времени, индекс поля плавает (7 vs 8) — ищем токен с «%».
func parseSelftestLine(line string) (string, int) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return "No tests logged", 100
	}
	status := fields[4]
	progress := 100
	for _, f := range fields {
		if strings.HasSuffix(f, "%") {
			if r, err := strconv.Atoi(strings.TrimSuffix(f, "%")); err == nil && r >= 0 && r <= 100 {
				progress = 100 - r
			}
			break
		}
	}
	return status, progress
}
