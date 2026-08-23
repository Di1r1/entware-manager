package smart

import (
	"bufio"
	"bytes"
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	attrCacheDir   = "/tmp/entware/cache/disk"
)

// attrCacheTTL — срок жизни кеша атрибутов SMART. Снимает повторные долгие
// опросы спящих дисков (SPINUP 13–60 сек): после первого запроса атрибуты
// отдаются из файла до 5 минут, затем перезаписываются при следующем запросе.
const attrCacheTTL = 5 * time.Minute

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

// importantAttrIDs — «важные, но не критические» атрибуты (для флага importance).
var importantAttrIDs = map[int]bool{1: true, 3: true, 4: true, 7: true, 9: true, 12: true, 184: true, 188: true, 189: true, 190: true, 193: true, 194: true, 199: true}

// attrImportance — важность атрибута для фронтенда ("critical"/"important"/"").
// Единый источник вместо дублей списков в smart.js.
func attrImportance(id int) string {
	if criticalAttrIDs[id] {
		return "critical"
	}
	if importantAttrIDs[id] {
		return "important"
	}
	return ""
}

var deviceRe = regexp.MustCompile(`^[a-z0-9-]+$`)

type AttrInfo struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Threshold  int    `json:"threshold"`
	Raw        string `json:"raw"`
	Importance string `json:"importance,omitempty"` // critical | important
}

type PartitionInfo struct {
	Part  string `json:"part"`
	Size  string `json:"size"`
	Used  string `json:"used"`
	Avail string `json:"avail"`
	Pct   int    `json:"pct"`
	Mnt   string `json:"mnt"`
}

// jsonBody сериализует v в JSON с Content-Type (без вывода в stdout).
func jsonBody(v any) string {
	out, _ := json.Marshal(v)
	return "Content-type: application/json; charset=utf-8\n\n" + string(out)
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
// убивает его (сигнал встаёт в очередь) и возвращает partial-вывод.
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

	outStr, err := waitOutcome(out, done, timeout, func() { cmd.Process.Kill() })
	return outStr, err
}

// waitOutcome читает вывод из out до первого из двух событий: процесс завершился
// (done) или истёк timeout. В ветке timeout вызывается kill() и закрывается
// read-end пайпа (out.Close()) — это разблокирует висящий Read() горутины чтения
// (процесс в D-состоянии не пишет и не умирает). Wait() в этой ветке НЕ
// вызывается: для D-state процесса он заблокировал бы навсегда, поэтому ребёнок
// остаётся зомби — приемлемо, т.к. CGI-процесс короткоживущий, зомби
// переподхватится при его выходе. done — буферизованный канал ёмкости 1,
// readDone закрывается ровно один раз (в горутине чтения).
func waitOutcome(out io.ReadCloser, done <-chan error, timeout time.Duration, kill func()) (string, error) {
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
		kill()
		out.Close()
		<-readDone
		return string(buf), fmt.Errorf("timeout after %v", timeout)
	}
}

// smartctlRun вызывает smartctl с коротким таймаутом (8 сек) — для точечных
// действий (info/attributes/health), где ждать пробуждение диска не нужно.
func smartctlRun(device string, args ...string) (string, error) {
	return smartctlRunTimeout(device, shortSmartctlTimeout, args...)
}

// smartctlRunTimeout запускает smartctl с указанным таймаутом. Длинный таймаут
// (diskSmartctlTimeout) используется в diskInfo: спящий диск «просыпается»
// за 13–60 сек, и первый же вызов должен успеть вернуть полные данные.
func smartctlRunTimeout(device string, timeout time.Duration, args ...string) (string, error) {
	if smartctlBusy(device) {
		return "", errDeviceBusy
	}

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, device)

	out, err := runBounded(exec.Command(smartctlBin, allArgs...), timeout)
	outStr := out
	if err == nil {
		return outStr, nil
	}

	// Try with sudo
	sudoArgs := append([]string{smartctlBin}, allArgs...)
	out2, err2 := runBounded(exec.Command(sudoBin, sudoArgs...), timeout)
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
	if !cgiutil.IsGET() && !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}

	var postParams map[string]string
	if cgiutil.IsPOST() {
		postParams = cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	}

	getParam := func(key string) string {
		v := cgiutil.GetQueryParam(key)
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
		cgiutil.WriteStatusError("invalid device")
		return
	}

	switch action {
	case "list":
		handleList(getParam("refresh") == "1")
	case "info":
		handleInfo(device)
	case "attributes":
		handleAttributes(device)
	case "health":
		handleHealth(device)
	case "usage":
		handleUsage(device)
	case "selftest":
		if cgiutil.IsPOST() {
			if auth.IsCrossSiteOrigin() {
				cgiutil.WriteStatusError(auth.CrossSiteDeny)
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
		cgiutil.WriteStatusError("Unknown action")
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

// listDeadline — сколько ждём ответы бодрых дисков при формировании списка.
// Диски, отвечающие быстро (~1 сек), попадают в первый ответ сразу. Диски,
// которые «просыпаются» (SPINUP за 13–60 сек), помечаются attr_health=loading,
// и фронт дозагружает их повторными запросами (см. smart.js loadDisks).
const listDeadline = 5 * time.Second

// listReloadDeadline — дедлайн дозагрузки (refresh=1): ждём проснувшийся диск
// до diskSmartctlTimeout, чтобы полные данные вернулись за один повторный запрос.
const listReloadDeadline = 65 * time.Second

// shortSmartctlTimeout — для точечных действий (info/attributes/health).
const shortSmartctlTimeout = 8 * time.Second

// shortListTimeout — таймаут опроса диска при первичной загрузке списка:
// чуть меньше listDeadline, чтобы спящий диск не порождал долгий осиротевший
// smartctl (в D-состоянии), блокирующий последующие запросы как busy.
const shortListTimeout = 4 * time.Second

// diskSmartctlTimeout — таймаут опроса диска в diskInfo: спящий диск может
// «просыпаться» до 60 сек, прежде чем smartctl вернёт полные данные.
const diskSmartctlTimeout = 60 * time.Second

// handleList отдаёт результат асинхронно: бодрые диски — сразу в первом ответе,
// не успевшие проснуться — со статусом loading. Ответ НЕ кэшируется, пока в
// списке есть loading-диски, чтобы повторные запросы фронта дозагружали их.
func handleList(refresh bool) {
	// Кнопка «Обновить» (refresh=1) сбрасывает кэш и опрашивает диски напрямую,
	// чтобы подхватить диск, который только что «проснулся».
	if refresh {
		cache.Invalidate("smart_list")
	}
	// Кэш 60с: повторные клики «Обновить» отвечают мгновенно и не запускают
	// smartctl на подвешенных дисках (процессы в D-состоянии).
	if data, ok := cache.Get("smart_list", 60*time.Second); ok {
		fmt.Print(string(data))
		return
	}

	disks := discoverDisks()
	// Канал буферизован на все диски: горутины, не успевшие к дедлайну,
	// не блокируются и завершаются вместе с выходом CGI-процесса.
	type diskResult struct {
		idx  int
		info DiskInfo
	}
	// Таймаут опроса диска: первичная загрузка — короткий (спящий диск не
	// должен порождать долгий осиротевший smartctl, блокирующий последующие
	// запросы), дозагрузка (refresh) — длинный, чтобы диск успел проснуться.
	probeTimeout := diskSmartctlTimeout
	if !refresh {
		probeTimeout = shortListTimeout
	}
	ch := make(chan diskResult, len(disks))
	for i, name := range disks {
		go func(idx int, dev string) {
			ch <- diskResult{idx, diskInfo(dev, probeTimeout)}
		}(i, name)
	}

	// Дозагрузка (refresh=1) ждёт проснувшийся диск дольше, чем первичная
	// загрузка, чтобы полные данные вернулись за один повторный запрос.
	deadlineDur := listDeadline
	if refresh {
		deadlineDur = listReloadDeadline
	}
	deadline := time.After(deadlineDur)
	done := make(map[int]DiskInfo)
	for len(done) < len(disks) {
		select {
		case r := <-ch:
			done[r.idx] = r.info
		case <-deadline:
			goto collect
		}
	}
collect:

	result := make([]DiskInfo, len(disks))
	hasLoading := false
	for i, name := range disks {
		if info, ok := done[i]; ok {
			result[i] = info
		} else {
			// Диск не ответил за дедлайн — он «просыпается» или подвешен.
			result[i] = loadingDiskInfo(name, "/dev/"+name, detectType(name))
			hasLoading = true
		}
	}

	out := jsonBody(map[string]any{"disks": result})
	// Не кэшируем список с loading-дисками: повторный запрос фронта (refresh=1)
	// должен дозагрузить проснувшиеся диски, а не получить застрявший снимок.
	if !hasLoading {
		cache.Put("smart_list", []byte(out))
	}
	fmt.Print(out)
}

// loadingDiskInfo — карточка диска, который не успел ответить за дедлайн
// (просыпается). Данные только из /sys (не блокируются), health «Загрузка…».
func loadingDiskInfo(name, devpath, diskType string) DiskInfo {
	info := busyDiskInfo(name, devpath, diskType)
	info.Health = "Загрузка…"
	info.AttrHealth = "loading"
	return info
}

func diskInfo(name string, probeTimeout time.Duration) DiskInfo {
	devpath := "/dev/" + name
	diskType := detectType(name)
	output, err := smartctlRunTimeout(devpath, probeTimeout, "-a", "-d", diskType)
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
		out2, err2 := smartctlRunTimeout(devpath, probeTimeout, "-a", "-d", "scsi")
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
	} else if health == "UNKNOWN" {
		// SMART-строка не получена (диск просыпается/не успел ответить) —
		// помечаем как loading: фронт дозагрузит, когда диск ответит.
		attrHealth = "loading"
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
		cgiutil.WriteStatusError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)
	output, _ := smartctlRun(devpath, "-i", "-d", diskType)
	cgiutil.WriteJSON(map[string]string{"info": output})
}

// readAttrCache возвращает закешированный вывод smartctl -A для device,
// если кеш свежий (< attrCacheTTL). Файл один на диск (имя = device),
// дубликаты не создаются. Stat и чтение идут через один дескриптор —
// без TOCTOU между проверкой mtime и чтением содержимого.
func readAttrCache(device string) (string, bool) {
	path := filepath.Join(attrCacheDir, device)
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", false
	}
	if time.Since(fi.ModTime()) > attrCacheTTL {
		return "", false
	}
	data, err := io.ReadAll(f)
	if err != nil || len(data) == 0 {
		return "", false
	}
	return string(data), true
}

// writeAttrCache атомарно (уникальный tmp + rename) записывает вывод smartctl -A
// в кеш. Каталог создаётся при необходимости; файл перезаписывается, не
// дублируется. Уникальное имя tmp (os.CreateTemp) исключает гонку между
// параллельными запросами одного диска.
func writeAttrCache(device, output string) {
	if output == "" {
		return
	}
	if err := os.MkdirAll(attrCacheDir, 0755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(attrCacheDir, device+".tmp-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(output); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return
	}
	os.Rename(tmpName, filepath.Join(attrCacheDir, device))
}

func handleAttributes(device string) {
	if device == "" {
		cgiutil.WriteStatusError("device required")
		return
	}
	devpath := "/dev/" + device
	diskType := detectType(device)

	// Сначала свежий кеш (5 мин) — не дёргаем smartctl на спящем диске.
	output, ok := readAttrCache(device)
	if !ok {
		// Длинный таймаут (60 сек): спящий диск может «просыпаться».
		output, _ = smartctlRunTimeout(devpath, diskSmartctlTimeout, "-A", "-d", diskType)
		if output != "" {
			writeAttrCache(device, output)
		}
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
		attr.Importance = attrImportance(id)
		attrs = append(attrs, attr)
	}
	if attrs == nil {
		attrs = []AttrInfo{}
	}
	cgiutil.WriteJSON(map[string]any{"attributes": attrs})
}

func isAttrLine(s string) bool {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return false
	}
	if _, err := strconv.Atoi(fields[0]); err != nil {
		return false
	}
	// Второй токен — имя атрибута (буквы/подчёркивания). Если это число/hex,
	// строка относится к SMART Error Log или self-test, а не к таблице атрибутов.
	if _, err := strconv.Atoi(fields[1]); err == nil {
		return false
	}
	return true
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
		cgiutil.WriteStatusError("device required")
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

	cgiutil.WriteJSON(map[string]string{
		"health":  result,
		"message": healthLine,
	})
}

func handleUsage(device string) {
	if device == "" {
		cgiutil.WriteStatusError("device required")
		return
	}

	out, err := runBounded(exec.Command(dfBin, "-h"), 5*time.Second)
	if err != nil && out == "" {
		cgiutil.WriteJSON(map[string]any{"partitions": []PartitionInfo{}})
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
	cgiutil.WriteJSON(map[string]any{"partitions": parts})
}

func handleSelftestStart(device string, testType string) {
	if device == "" {
		cgiutil.WriteStatusError("device required")
		return
	}
	switch testType {
	case "short", "long", "conveyance", "offline":
	default:
		cgiutil.WriteStatusError("invalid test type")
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

	cgiutil.WriteJSON(map[string]string{"status": status, "message": msg})
}

func handleSelftestStatus(device string) {
	if device == "" {
		cgiutil.WriteStatusError("device required")
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

	cgiutil.WriteJSON(map[string]any{"status": status, "progress": progress})
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
