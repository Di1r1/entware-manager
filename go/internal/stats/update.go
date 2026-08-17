package stats

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"entware-manager/internal/cgiutil"
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

	"entware-manager/internal/auth"
	"entware-manager/internal/cache"
)

type UpdateCheckResponse struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"has_update"`
	Error     string `json:"error,omitempty"`
}

type UpdateStatusResponse struct {
	Status   string   `json:"status"` // running / done / error
	PID      int      `json:"pid"`
	Lines    []string `json:"lines"`
	Progress string   `json:"progress,omitempty"` // «Этап 2/5: установка...»
	Elapsed  string   `json:"elapsed,omitempty"`  // «2 мин» — сколько идёт обновление
	Error    string   `json:"error,omitempty"`
}

var (
	updateLogFile = "/tmp/entware/update.log"
	updateLock    sync.Mutex
)

func HandleUpdateCheck() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		cgiutil.WriteJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	current := getLocalVersion()
	latest, err := getLatestVersionCached()
	if err != nil {
		cgiutil.WriteJSON(UpdateCheckResponse{
			Current:   current,
			Latest:    current,
			HasUpdate: false,
			Error:     err.Error(),
		})
		return
	}

	cgiutil.WriteJSON(UpdateCheckResponse{
		Current:   current,
		Latest:    latest,
		HasUpdate: semverGreater(latest, current),
	})
}

func HandleUpdateRun() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		cgiutil.WriteJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}

	// updateLock (sync.Mutex) живёт в процессе CGI, а реальная работа — в
	// отдельном воркере. Поэтому дополнительно проверяем pidfile: если
	// воркер жив (например, пользователь обновил страницу и нажал снова) —
	// отказываем. Иначе воркеры побежали бы параллельно на opkg-lock.
	if updateWorkerRunning() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Обновление уже запущено"})
		return
	}
	if !updateLock.TryLock() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Обновление уже запущено"})
		return
	}
	defer updateLock.Unlock()

	arch := getRouterArch()
	if arch == "" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Архитектура не определена. Установите обновление через install.sh"})
		return
	}

	body := cgiutil.ReadPOSTBody()
	params := cgiutil.ParseFormBody(body)
	reinstall := params["mode"] == "reinstall"

	version := ""
	if reinstall {
		version = getLocalVersion()
		if version == "0.0.0" || version == "" {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось определить установленную версию. Переустановка невозможна."})
			return
		}
	} else {
		var err error
		version, err = getLatestVersion()
		if err != nil {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось проверить версию: " + err.Error()})
			return
		}
	}

	label := "Обновление"
	if reinstall {
		label = "Переустановка"
	}
	os.MkdirAll("/tmp/entware", 0755)
	os.WriteFile(updateLogFile, []byte("[INIT] "+label+" до v"+version+"\n"), 0644)

	vars := fmt.Sprintf(`{"version":"%s","arch":"%s"}`, version, arch)
	os.WriteFile("/tmp/entware/update_vars", []byte(vars), 0644)

	script := "#!/bin/sh\nENDPOINT=update_worker /opt/web_entware/cgi-bin/go/entware-stats >/dev/null 2>&1 &\n"
	os.WriteFile("/tmp/entware/update.sh", []byte(script), 0755)
	exec.Command("/bin/sh", "/tmp/entware/update.sh").Start()

	cgiutil.WriteJSON(map[string]string{"status": "ok", "message": label + " запущено", "version": version})
}

// updateWorkerRunning — жив ли процесс-воркер обновления (по pidfile).
// Путь к воркеру — /opt/web_entware/cgi-bin/go/entware-stats (см. update.sh).
func updateWorkerRunning() bool {
	pidData, err := os.ReadFile("/tmp/entware/update.pid")
	if err != nil {
		return false
	}
	pid := strings.TrimSpace(string(pidData))
	if pid == "" {
		return false
	}
	var p int
	if _, err := fmt.Sscanf(pid, "%d", &p); err != nil || p <= 0 {
		// pidfile битый (не число) — считаем, что воркера нет
		os.Remove("/tmp/entware/update.pid")
		return false
	}
	cmdline, err := os.ReadFile("/proc/" + pid + "/cmdline")
	if err != nil {
		// процесс умер — битый pidfile, забираем
		os.Remove("/tmp/entware/update.pid")
		return false
	}
	return strings.Contains(string(cmdline), "entware-stats")
}

func HandleUpdateWorker() {
	data, err := os.ReadFile("/tmp/entware/update_vars")
	if err != nil {
		return
	}
	var v struct {
		Version string `json:"version"`
		Arch    string `json:"arch"`
	}
	if json.Unmarshal(data, &v) != nil {
		return
	}
	os.Remove("/tmp/entware/update_vars")
	os.Remove("/tmp/entware/update.sh")
	os.WriteFile("/tmp/entware/update.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
	defer os.Remove("/tmp/entware/update.pid")
	runUpdate(v.Version, v.Arch)
}

func HandleUpdateStatus() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		cgiutil.WriteJSON(map[string]string{"error": "Method not allowed"})
		return
	}

	resp := UpdateStatusResponse{Status: "running", PID: 0}

	data, err := os.ReadFile(updateLogFile)
	if err != nil {
		resp.Status = "error"
		resp.Error = "Лог не найден"
		cgiutil.WriteJSON(resp)
		return
	}

	allLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Статус ищем по всему логу: [DONE] → done, иначе [ERROR]/[FAIL] → error.
	status := "running"
	var firstErr string
	for _, line := range allLines {
		if strings.Contains(line, "[DONE]") {
			status = "done"
			break
		}
		if strings.Contains(line, "[ERROR]") || strings.Contains(line, "[FAIL]") {
			if firstErr == "" {
				firstErr = strings.TrimSpace(line)
			}
			status = "error"
		}
	}
	resp.Status = status
	resp.Error = firstErr

	// Фронту отдаём последние 20 строк.
	lines := allLines
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	resp.Lines = lines

	// Прогресс: последняя строка вида «[STEP N/M] описание» или последняя строка.
	for i := len(lines) - 1; i >= 0; i-- {
		idx := strings.Index(lines[i], "[STEP")
		if idx >= 0 {
			// «[STEP 2/5] описание» → «Этап 2/5: описание»
			stepPart := lines[i][idx:]
			stepPart = strings.TrimPrefix(stepPart, "[")
			if closeIdx := strings.Index(stepPart, "]"); closeIdx >= 0 {
				stepNo := strings.TrimSpace(stepPart[:closeIdx])
				desc := strings.TrimSpace(stepPart[closeIdx+1:])
				if strings.HasPrefix(stepNo, "STEP ") {
					stepNo = "Этап " + stepNo[len("STEP "):]
				}
				if desc != "" {
					stepPart = stepNo + ": " + desc
				} else {
					stepPart = stepNo
				}
			}
			resp.Progress = stepPart
			break
		}
	}
	if resp.Progress == "" && len(lines) > 0 {
		last := lines[len(lines)-1]
		if len(last) >= 10 && last[0] == '[' && last[9] == ']' {
			last = strings.TrimSpace(last[10:])
		}
		resp.Progress = last
	}

	cgiutil.WriteJSON(resp)
}

// --- helpers ---

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

const updateCheckCacheTTL = 60 * time.Second

type updateCheckCacheEntry struct {
	Latest string `json:"latest"`
	Error  string `json:"error,omitempty"`
}

// getLatestVersionCached кэширует результат GitHub API на 60с (вызывается
// при каждой загрузке главной страницы).
func getLatestVersionCached() (string, error) {
	if data, ok := cache.Get("update_check", updateCheckCacheTTL); ok {
		var e updateCheckCacheEntry
		if json.Unmarshal(data, &e) == nil {
			if e.Error != "" {
				return "", fmt.Errorf("%s", e.Error)
			}
			return e.Latest, nil
		}
	}
	latest, err := getLatestVersion()
	entry := updateCheckCacheEntry{}
	if err != nil {
		entry.Error = err.Error()
	} else {
		entry.Latest = latest
	}
	if body, merr := json.Marshal(entry); merr == nil {
		cache.Put("update_check", body)
	}
	return latest, err
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

// opkgWithTimeout выполняет opkg с таймаутом 60с через coreutils-timeout
// (в Depends), если он доступен — иначе напрямую. Защита от зависания на
// недоступном/медленном feed: в v1.09.5 opkg update/install без таймаута
// «висел несколько минут» (инцидент).
func opkgWithTimeout(args ...string) error {
	if _, err := exec.LookPath("timeout"); err == nil {
		cmd := append([]string{"60", "opkg"}, args...)
		return exec.Command("timeout", cmd...).Run()
	}
	return exec.Command("opkg", args...).Run()
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
	defer func() {
		if r := recover(); r != nil {
			f, _ := os.OpenFile(updateLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if f != nil {
				fmt.Fprintf(f, "[PANIC] %v\n", r)
				f.Close()
			}
		}
	}()

	log := func(line string) {
		f, _ := os.OpenFile(updateLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), line)
			f.Close()
		}
	}

	// stepLog выводит этап вида «[STEP N/M] описание» — фронт показывает его
	// как понятный прогресс («Этап 2/5: установка ipk...»).
	totalSteps := 5
	stepNum := 0
	stepLog := func(desc string) {
		stepNum++
		log(fmt.Sprintf("[STEP %d/%d] %s", stepNum, totalSteps, desc))
	}

	log("[RUNNING] Начало обновления до v" + version)

	tmpDir := "/tmp/entware/update"
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	client := &http.Client{Timeout: 120 * time.Second}

	// Выбор ветки: opkg-база и диск — независимые источники истины.
	// Если opkg говорит "installed", но панель на диске отсутствует —
	// это БИТАЯ запись (инцидент v1.09.5: убитый opkg оставил "installed"
	// при пустом каталоге). Ставить по такой записи opkg install нельзя
	// (снова prerm по живому пакету) — лечим tar.gz-восстановлением.
	viaOpkg := isInstalledViaOpkg()
	panelPresent := false
	if vd, err := os.ReadFile("/opt/web_entware/version.json"); err == nil {
		panelPresent = len(vd) > 0
	}
	if viaOpkg && !panelPresent {
		log("[WARN] opkg считает пакет установленным, но /opt/web_entware/version.json отсутствует — битая запись/пустой каталог. Восстанавливаю через tar.gz")
		viaOpkg = false
	}

	if viaOpkg {
		stepLog("скачивание ipk-файла")
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
		f, err := os.Create(ipkPath)
		if err != nil {
			log("[ERROR] Не удалось создать временный файл: " + err.Error())
			resp.Body.Close()
			return
		}
		written, cerr := io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if cerr != nil {
			log("[ERROR] Ошибка загрузки (файл неполный): " + cerr.Error())
			os.RemoveAll(tmpDir)
			return
		}
		// Проверка целостности: ipk должен быть gzip-архивом >1КБ.
		// Обрезанная загрузка приводила к «Malformed package file» у opkg.
		if fi, perr := os.Stat(ipkPath); perr != nil || fi.Size() < 1024 || written < fi.Size() {
			log("[ERROR] Загруженный файл повреждён или обрезан (" + humanSize(written) + ")")
			os.RemoveAll(tmpDir)
			return
		}
		stepLog("обновление списков пакетов (opkg update)")
		// Свежие списки пакетов нужны для резолва Depends (модули lighttpd
		// и т.п.). Внешний вызов — НЕ в postinst, lock свободен. Таймаут 60с
		// через coreutils-timeout (уже в Depends), чтобы не висеть на feed.
		log("opkg update (таймаут 60с)...")
		if updErr := opkgWithTimeout("update"); updErr != nil {
			log("[WARN] opkg update не удался: " + updErr.Error() + " — пробуем продолжить")
		} else {
			log("Списки пакетов обновлены")
		}
		stepLog("установка ipk (opkg install)")
		var outBuf bytes.Buffer
		cmd := exec.Command("opkg", "install", "--force-reinstall", ipkPath)
		if _, err := exec.LookPath("timeout"); err == nil {
			cmd = exec.Command("timeout", "60", "opkg", "install", "--force-reinstall", ipkPath)
		}
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		installErr := cmd.Run()
		// Проверка результата по факту на диске, а не только по exit-коду:
		// убитый/оборванный opkg может оставить статус "installed" при пустом
		// каталоге (инцидент v1.09.5: повторный opkg install после обрыва).
		verData, verErr := os.ReadFile("/opt/web_entware/version.json")
		installed := verErr == nil && strings.Contains(string(verData), version)
		if installErr != nil {
			log("[FAIL] opkg install: " + installErr.Error())
			for _, line := range strings.Split(outBuf.String(), "\n") {
				if line != "" {
					log("[FAIL] " + line)
				}
			}
		}
		if installed {
			stepLog("проверка и перезапуск веб-сервера")
			log("[DONE] Обновление до v" + version + " завершено")
			restartWebServer(log)
		} else {
			log("[ERROR] Файлы панели не установлены (/opt/web_entware/version.json отсутствует или старая версия)")
			for _, line := range strings.Split(outBuf.String(), "\n") {
				if line != "" {
					log("[FAIL] " + line)
				}
			}
			log("[ERROR] Запустите восстановление вручную: распакуйте entware-manager-" + arch + ".tar.gz и выполните Install/install.sh")
		}
		os.RemoveAll(tmpDir)
		log("Временные файлы удалены")
		return
	}

	stepLog("скачивание архива tar.gz")
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

	stepLog("распаковка архива")

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

		// Защита от tar-path-traversal: только чистые относительные пути внутри tmpDir.
		rel := filepath.Clean(name)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			log("[ERROR] Пропуск небезопасного пути в архиве: " + header.Name)
			continue
		}

		target := filepath.Join(tmpDir, rel)
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
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Symlink(header.Linkname, target)
		}
	}

	stepLog("установка через install.sh")

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

	installErr := cmd.Run()
	// Проверка по факту на диске (как в ipk-ветке): install.sh завершается
	// с 0, даже если в конце ошибки — надёжнее проверить version.json.
	verData, verErr := os.ReadFile("/opt/web_entware/version.json")
	installed := verErr == nil && strings.Contains(string(verData), version)

	if installErr != nil {
		log("[FAIL] Установка завершилась с ошибкой: " + installErr.Error())
		for _, line := range strings.Split(outBuf.String(), "\n") {
			if line != "" {
				log("[FAIL] " + line)
			}
		}
	}
	if installed {
		stepLog("проверка и перезапуск веб-сервера")
		log("[DONE] Обновление до v" + version + " завершено")
		restartWebServer(log)
	} else {
		log("[ERROR] Файлы панели не установлены (/opt/web_entware/version.json отсутствует или старая версия)")
		for _, line := range strings.Split(outBuf.String(), "\n") {
			if line != "" {
				log("[FAIL] " + line)
			}
		}
		log("[ERROR] Восстановление вручную: распакуйте entware-manager-" + arch + ".tar.gz и выполните Install/install.sh")
	}

	os.RemoveAll(tmpDir)
	log("Временные файлы удалены")
}

// restartWebServer перезапускает entware-server (go-режим), если он работает.
// При обновлении через кнопку install.sh не останавливает работающий
// entware-server (S80entware-server start идемпотентен) — процесс продолжает
// работать со старым бинарником, и новые пути (menu.json, session.cgi и т.п.)
// отдают 404. Перезапуск заставляет процесс подхватить обновлённый бинарник.
func restartWebServer(log func(string)) {
	init := "/opt/etc/init.d/S80entware-server"
	if _, err := os.Stat(init); err != nil {
		return
	}
	// Перезапускаем только если entware-server реально работает (go-режим).
	// В lighttpd-режиме его нет — порт 8087 занимает lighttpd, и запуск
	// entware-server приведёт к падению процесса (address in use).
	if data, err := os.ReadFile("/opt/var/run/entware-server.pid"); err == nil {
		pid := strings.TrimSpace(string(data))
		if pid != "" {
			if _, err := os.Stat("/proc/" + pid); err == nil {
				cmd := exec.Command(init, "restart")
				if _, err := cmd.CombinedOutput(); err != nil {
					log("[WARN] Не удалось перезапустить entware-server: " + err.Error())
					return
				}
				log("entware-server перезапущен (новый бинарник подхвачен)")
			}
		}
	}
}
