package smart

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	smartctlBin    = "/opt/sbin/smartctl"
	procPartitions = "/proc/partitions"
	sysBlockDir    = "/sys/block"
	dfBin          = "df"
	sudoBin        = "sudo"
	timeoutBin     = "timeout"
)

type DiskInfo struct {
	Device        string `json:"device"`
	Model         string `json:"model"`
	Serial        string `json:"serial"`
	Size          string `json:"size"`
	Type          string `json:"type"`
	Health        string `json:"health"`
	Temperature   any    `json:"temperature"`
	PowerOnHours  any    `json:"power_on_hours"`
}

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
	out, _ := json.Marshal(v)
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Print(string(out))
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

func smartctlRun(device string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, device)

	cmd := exec.CommandContext(ctx, smartctlBin, allArgs...)
	out, err := cmd.Output()
	if err == nil {
		return string(out), nil
	}

	// Try with sudo
	sudoArgs := append([]string{smartctlBin}, allArgs...)
	cmd2 := exec.CommandContext(ctx, sudoBin, sudoArgs...)
	out2, err2 := cmd2.Output()
	if err2 == nil {
		return string(out2), nil
	}

	return "", fmt.Errorf("smartctl failed: %w (sudo: %v)", err, err2)
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

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
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

func parseIntOrNull(s string) any {
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
	return n
}

func HandleSmart() {
	if !isGET() && !isPOST() {
		notAllowed()
		return
	}

	action := getQueryParam("action")
	if action == "" {
		// Try POST body for action
		if isPOST() {
			body := readPOSTBody()
			params := parseFormBody(body)
			if a, ok := params["action"]; ok {
				action = a
			}
		}
	}
	if action == "" {
		action = "list"
	}

	device := getQueryParam("device")
	if device == "" && isPOST() {
		body := readPOSTBody()
		params := parseFormBody(body)
		if d, ok := params["device"]; ok {
			device = d
		}
	}
	device = strings.TrimPrefix(device, "/dev/")

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
			testType := getQueryParam("type")
			if testType == "" {
				body := readPOSTBody()
				params := parseFormBody(body)
				if t, ok := params["type"]; ok {
					testType = t
				}
			}
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

func handleList() {
	disks := discoverDisks()
	var result []DiskInfo
	for _, name := range disks {
		d := diskInfo(name)
		result = append(result, d)
	}
	if result == nil {
		result = []DiskInfo{}
	}
	writeJSON(map[string]any{"disks": result})
}

func diskInfo(name string) DiskInfo {
	devpath := "/dev/" + name
	diskType := detectType(name)
	output, err := smartctlRun(devpath, "-a", "-d", diskType)
	if err != nil {
		output = ""
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
		health = parts[len(parts)-1]
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

	return DiskInfo{
		Device:       devpath,
		Model:        escapeJSON(model),
		Serial:       escapeJSON(serial),
		Size:         diskSize(name),
		Type:         displayType,
		Health:       health,
		Temperature:  parseIntOrNull(temperature),
		PowerOnHours: parseIntOrNull(powerOn),
	}
}

func handleInfo(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, err := smartctlRun(devpath, "-i", "-d", diskType)
	if err != nil {
		writeError(fmt.Sprintf("smartctl failed: %v", err))
		return
	}
	writeJSON(map[string]string{"info": escapeJSON(output)})
}

func handleAttributes(device string) {
	if device == "" {
		writeError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, err := smartctlRun(devpath, "-A", "-d", diskType)
	if err != nil {
		writeError(fmt.Sprintf("smartctl failed: %v", err))
		return
	}

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
	output, err := smartctlRun(devpath, "-H", "-d", diskType)
	if err != nil {
		writeError(fmt.Sprintf("smartctl failed: %v", err))
		return
	}

	healthLine := extractField(output, "SMART overall-health", -1)
	if healthLine == "" {
		healthLine = extractField(output, "SMART Health Status", -1)
	}
	if healthLine == "" {
		healthLine = "SMART: Unable to determine health status"
	}

	parts := strings.Fields(healthLine)
	result := parts[len(parts)-1]

	writeJSON(map[string]string{
		"health":  result,
		"message": escapeJSON(healthLine),
	})
}

func handleUsage(device string) {
	if device == "" {
		writeError("device required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dfBin, "-h")
	out, err := cmd.Output()
	if err != nil {
		writeJSON(map[string]any{"partitions": []PartitionInfo{}})
		return
	}

	var parts []PartitionInfo
	prefix := "/dev/" + device
	scanner := bufio.NewScanner(bytes.NewReader(out))
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
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, err := smartctlRun(devpath, "-t", testType, "-d", diskType)
	if err != nil {
		writeError(fmt.Sprintf("smartctl failed: %v", err))
		return
	}

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
	output, err := smartctlRun(devpath, "-l", "selftest", "-d", diskType)
	if err != nil {
		writeError(fmt.Sprintf("smartctl failed: %v", err))
		return
	}

	status := "No tests logged"
	progress := 100

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				status = fields[4]
				last := fields[len(fields)-1]
				if last != "-" {
					progress, _ = strconv.Atoi(last)
				}
				break
			}
		}
	}

	writeJSON(map[string]any{"status": status, "progress": progress})
}
