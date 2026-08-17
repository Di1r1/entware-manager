package monitor

import (
	"entware-manager/internal/cgiutil"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func HandleTemperature() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	temp := readCPUTemp()
	if temp != "" {
		cgiutil.WriteJSON(map[string]interface{}{"temperature": temp})
	} else {
		cgiutil.WriteJSON(map[string]interface{}{"temperature": nil})
	}
}

func readCPUTemp() string {
	zones := []string{
		"/sys/class/thermal/thermal_zone0/temp",
	}
	data, err := os.ReadFile(zones[0])
	if err == nil {
		t := strings.TrimSpace(string(data))
		if len(t) > 3 {
			return t[:len(t)-3]
		}
		return t
	}

	matches, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil || len(matches) == 0 {
		return ""
	}
	data, err = os.ReadFile(matches[0])
	if err != nil {
		return ""
	}
	t := strings.TrimSpace(string(data))
	if len(t) > 3 {
		return t[:len(t)-3]
	}
	return t
}

func HandleWifiTemp() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	temp0 := getWifiTemp("WifiMaster0")
	temp1 := getWifiTemp("WifiMaster1")

	combined := "null"
	if temp0 == "null" && temp1 != "null" {
		combined = "WiFi1: " + temp1 + "°C"
	} else if temp1 == "null" && temp0 != "null" {
		combined = "WiFi0: " + temp0 + "°C"
	} else if temp0 != "null" && temp1 != "null" {
		combined = "WiFi0: " + temp0 + "°C / WiFi1: " + temp1 + "°C"
	}

	cgiutil.WriteJSON(map[string]string{
		"temp0":    temp0,
		"temp1":    temp1,
		"combined": combined,
	})
}

const rciBase = "http://127.0.0.1:79"

func getWifiTemp(iface string) string {
	url := rciBase + "/rci/show/interface/" + iface
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "null"
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "null"
	}
	body := string(data)

	temp := extractJSONInt(body, "temperature")
	if temp == "" {
		temp = extractJSONInt(body, "\u0442\u0435\u043C\u043F\u0435\u0440\u0430\u0442\u0443\u0440\u0430")
	}
	if temp == "" {
		return "null"
	}
	return temp
}

func extractJSONInt(json, key string) string {
	qs := `"` + key + `":`
	idx := strings.Index(json, qs)
	if idx < 0 {
		return ""
	}
	rest := json[idx+len(qs):]
	rest = strings.TrimSpace(rest)
	end := strings.IndexAny(rest, ",\n\r}")
	if end < 0 {
		end = len(rest)
	}
	val := strings.TrimSpace(rest[:end])
	if val == "" || val == "null" {
		return ""
	}
	return val
}

const tempBaseDir = "/tmp/entware/temp_history"
const maxDays = 7

func HandleTempHistory() {
	action := GetParam("action")
	if action == "" {
		action = "history"
	}

	os.MkdirAll(tempBaseDir, 0755)

	switch action {
	case "save":
		cleanupOldTemp("cpu", ".cpu_cleanup")
		temp := GetParam("temp")
		if isValidNum(temp) {
			saveTempPoint("cpu", temp)
		}
		cgiutil.WriteJSON(map[string]string{"status": "ok"})
	default:
		history := readTempHistory("cpu.*")
		result := parseTempHistory(history)
		cgiutil.WriteJSON(result)
	}
}

func HandleWifiTempHistory() {
	action := GetParam("action")
	if action == "" {
		action = "current"
	}

	os.MkdirAll(tempBaseDir, 0755)

	switch action {
	case "save":
		cleanupOldTemp("wifi", ".wifi_cleanup")
		temp0 := GetParam("temp0")
		temp1 := GetParam("temp1")
		if temp0 == "" {
			temp0 = "-"
		}
		if temp1 == "" {
			temp1 = "-"
		}
		saveWifiTempPoint(temp0, temp1)
		cgiutil.WriteJSON(map[string]string{"status": "ok"})
	default:
		history := readTempHistory("wifi.*")
		result := parseWifiTempHistory(history)
		cgiutil.WriteJSON(result)
	}
}

func cleanupOldTemp(prefix, marker string) {
	markerPath := tempBaseDir + "/" + marker
	today := time.Now().Format("2006-01-02")
	if data, err := os.ReadFile(markerPath); err == nil && string(data) == today {
		return
	}

	filepath.Walk(tempBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.HasPrefix(info.Name(), prefix+".") {
			if time.Since(info.ModTime()) > maxDays*24*time.Hour {
				os.Remove(path)
			}
		}
		return nil
	})

	os.WriteFile(markerPath, []byte(today), 0644)
}

func saveTempPoint(prefix, temp string) {
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("15:04:05")
	line := today + " " + now + "|" + temp + "\n"
	f, _ := os.OpenFile(tempBaseDir+"/"+prefix+"."+today, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(line)
		f.Close()
	}
}

func saveWifiTempPoint(temp0, temp1 string) {
	if temp0 == "" || temp0 == "null" || temp0 == "-" {
		temp0 = "-"
	}
	if temp1 == "" || temp1 == "null" || temp1 == "-" {
		temp1 = "-"
	}
	if temp0 == "-" && temp1 == "-" {
		return
	}
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("15:04:05")
	line := today + " " + now + "|" + temp0 + "|" + temp1 + "\n"
	f, _ := os.OpenFile(tempBaseDir+"/wifi."+today, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(line)
		f.Close()
	}
}

func readTempHistory(pattern string) []string {
	matches, err := filepath.Glob(tempBaseDir + "/" + pattern)
	if err != nil {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] > matches[j]
	})

	var lines []string
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func parseTempHistory(lines []string) []map[string]interface{} {
	var result []map[string]interface{}
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		t, val := parts[0], parts[1]
		n, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"time": t,
			"temp": n,
		})
	}
	if result == nil {
		return []map[string]interface{}{}
	}
	return result
}

func parseWifiTempHistory(lines []string) []map[string]interface{} {
	var result []map[string]interface{}
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		t, t0, t1 := parts[0], parts[1], parts[2]
		if t0 == "-" || t0 == "null" {
			t0 = ""
		}
		if t1 == "-" || t1 == "null" {
			t1 = ""
		}
		if t0 == "" && t1 == "" {
			continue
		}
		var t0Val interface{}
		var t1Val interface{}
		if n, err := strconv.Atoi(t0); err == nil {
			t0Val = n
		}
		if n, err := strconv.Atoi(t1); err == nil {
			t1Val = n
		}
		result = append(result, map[string]interface{}{
			"time":  t,
			"temp0": t0Val,
			"temp1": t1Val,
		})
	}
	if result == nil {
		return []map[string]interface{}{}
	}
	return result
}

func isValidNum(s string) bool {
	if s == "" || s == "-" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
