package stats

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UpdateCheckResponse struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"has_update"`
	Error     string `json:"error,omitempty"`
}

type UpdateStatusResponse struct {
	Status string   `json:"status"` // running / done / error
	PID    int      `json:"pid"`
	Lines  []string `json:"lines"`
	Error  string   `json:"error,omitempty"`
}

var (
	updateLogFile = "/tmp/entware/update.log"
	updateLock    sync.Mutex
)

func HandleUpdateCheck() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		writeJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	current := getLocalVersion()
	latest, err := getLatestVersion()
	if err != nil {
		writeJSON(UpdateCheckResponse{
			Current:   current,
			Latest:    current,
			HasUpdate: false,
			Error:     err.Error(),
		})
		return
	}

	writeJSON(UpdateCheckResponse{
		Current:   current,
		Latest:    latest,
		HasUpdate: semverGreater(latest, current),
	})
}

func HandleUpdateRun() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		writeJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	if !updateLock.TryLock() {
		writeJSON(map[string]string{"status": "error", "message": "Обновление уже запущено"})
		return
	}
	defer updateLock.Unlock()

	arch := getRouterArch()
	if arch == "" {
		writeJSON(map[string]string{"status": "error", "message": "Архитектура не определена. Установите обновление через install.sh"})
		return
	}

	latest, err := getLatestVersion()
	if err != nil {
		writeJSON(map[string]string{"status": "error", "message": "Не удалось проверить версию: " + err.Error()})
		return
	}

	os.MkdirAll("/tmp/entware", 0755)
	os.WriteFile(updateLogFile, []byte("[INIT] Запуск обновления\n"), 0644)

	go runUpdate(latest, arch)

	writeJSON(map[string]string{"status": "ok", "message": "Обновление запущено", "version": latest})
}

func HandleUpdateStatus() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		writeJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	resp := UpdateStatusResponse{Status: "done", PID: 0}

	data, err := os.ReadFile(updateLogFile)
	if err != nil {
		resp.Status = "error"
		resp.Error = "Лог не найден"
		writeJSON(resp)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	resp.Lines = lines

	for _, line := range lines {
		if strings.Contains(line, "[RUNNING]") {
			resp.Status = "running"
			break
		}
		if strings.Contains(line, "[ERROR]") || strings.Contains(line, "[FAIL]") {
			resp.Status = "error"
			resp.Error = line
			break
		}
		if strings.Contains(line, "[DONE]") {
			resp.Status = "done"
			break
		}
	}

	writeJSON(resp)
}

// --- helpers ---

func writeJSON(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Println("Content-type: application/json; charset=utf-8\n")
	os.Stdout.Write(data)
}

func getLocalVersion() string {
	data, err := os.ReadFile("/opt/web_entware/version.json")
	if err != nil {
		return "0.0.0"
	}
	var v struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &v) != nil || v.Version == "" {
		return "0.0.0"
	}
	return v.Version
}

func getRouterArch() string {
	data, err := os.ReadFile("/opt/web_entware/.arch")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return archMap[runtime.GOARCH]
}

var archMap = map[string]string{
	"arm64":  "arm64",
	"arm":    "arm",
	"mips":   "mips",
	"mipsle": "mipsel",
	"amd64":  "amd64",
	"386":    "386",
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Di1r1/entware-manager/releases/latest")
	if err != nil {
		return "", fmt.Errorf("GitHub API недоступен: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API ответил %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("Ошибка парсинга ответа: %w", err)
	}

	return strings.TrimPrefix(rel.TagName, "v"), nil
}

func getDownloadURL(latestVersion, arch string) string {
	return fmt.Sprintf("https://github.com/Di1r1/entware-manager/releases/download/v%s/entware-manager-%s.tar.gz", latestVersion, arch)
}

func getDownloadURLIPK(latestVersion, arch string) string {
	return fmt.Sprintf("https://github.com/Di1r1/entware-manager/releases/download/v%s/entware-manager_%s.ipk", latestVersion, arch)
}

func isInstalledViaOpkg() bool {
	out, err := exec.Command("opkg", "list-installed").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "entware-manager")
}

func semverGreater(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var res [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(strings.SplitN(p, "-", 2)[0])
		if err != nil {
			break
		}
		res[i] = n
	}
	return res
}

func runUpdate(version, arch string) {
	log := func(line string) {
		f, _ := os.OpenFile(updateLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), line)
			f.Close()
		}
	}

	tmpDir := "/tmp/entware/update"
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	client := &http.Client{Timeout: 120 * time.Second}

	if isInstalledViaOpkg() {
		log("[RUNNING] Обновление через ipk до v" + version)
		url := getDownloadURLIPK(version, arch)
		log("Загрузка " + url)
		resp, err := client.Get(url)
		if err != nil {
			log("[ERROR] Ошибка загрузки: " + err.Error())
			return
		}
		if resp.StatusCode != 200 {
			log(fmt.Sprintf("[ERROR] GitHub ответил %d", resp.StatusCode))
			resp.Body.Close()
			return
		}
		ipkPath := filepath.Join(tmpDir, "entware-manager.ipk")
		f, _ := os.Create(ipkPath)
		if f == nil {
			log("[ERROR] Не удалось создать временный файл")
			resp.Body.Close()
			return
		}
		io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		log("Установка ipk...")
		var outBuf bytes.Buffer
		cmd := exec.Command("opkg", "install", "--force-reinstall", ipkPath)
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			log("[FAIL] opkg install: " + err.Error())
			for _, line := range strings.Split(outBuf.String(), "\n") {
				if line != "" {
					log("[FAIL] " + line)
				}
			}
		} else {
			log("[DONE] Обновление до v" + version + " завершено")
		}
		os.RemoveAll(tmpDir)
		log("Временные файлы удалены")
		return
	}

	log("[RUNNING] Обновление через tar.gz до v" + version)

	url := getDownloadURL(version, arch)
	log("Загрузка " + url)

	resp, err := client.Get(url)
	if err != nil {
		log("[ERROR] Ошибка загрузки: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log(fmt.Sprintf("[ERROR] GitHub ответил %d", resp.StatusCode))
		return
	}

	log("Распаковка...")

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log("[ERROR] Ошибка распаковки gzip: " + err.Error())
		return
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log("[ERROR] Ошибка tar: " + err.Error())
			return
		}

		name := strings.TrimPrefix(header.Name, "deploy/")
		if name == "" || name == header.Name {
			continue
		}

		target := filepath.Join(tmpDir, name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.Create(target)
			if err != nil {
				log("[ERROR] " + err.Error())
				continue
			}
			io.Copy(f, tarReader)
			f.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		}
	}

	log("Установка...")

	installScript := filepath.Join(tmpDir, "Install", "install.sh")
	if _, err := os.Stat(installScript); os.IsNotExist(err) {
		log("[ERROR] install.sh не найден")
		os.RemoveAll(tmpDir)
		return
	}

	cmd := exec.Command("/bin/sh", installScript)
	cmd.Dir = tmpDir
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Run(); err != nil {
		log("[FAIL] Установка завершилась с ошибкой: " + err.Error())
		for _, line := range strings.Split(outBuf.String(), "\n") {
			if line != "" {
				log("[FAIL] " + line)
			}
		}
	} else {
		log("[DONE] Обновление до v" + version + " завершено")
	}

	os.RemoveAll(tmpDir)
	log("Временные файлы удалены")
}
