package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempHistoryDir подменяет tempBaseDir на каталог теста и возвращает его.
func tempHistoryDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := tempBaseDir
	tempBaseDir = dir
	t.Cleanup(func() { tempBaseDir = old })
	return dir
}

// getTempHistory вызывает HandleTempHistory с методом и query.
func getTempHistory(t *testing.T, method, query string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", method)
	t.Setenv("QUERY_STRING", query)
	return captureMonitorOut(t, HandleTempHistory)
}

// postWifiTempHistory вызывает HandleWifiTempHistory с методом и query.
func postWifiTempHistory(t *testing.T, method, query string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", method)
	t.Setenv("QUERY_STRING", query)
	return captureMonitorOut(t, HandleWifiTempHistory)
}

// filesIn — файлы данных. Служебные маркеры cleanupOldTemp начинаются
// с точки (".cpu_cleanup"), поэтому шаблон "cpu.*"/"wifi.*" их не берёт.
func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	got := []string{}
	for _, pat := range []string{"cpu.*", "wifi.*"} {
		m, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, m...)
	}
	return got
}

// withHost задаёт Origin/Host: без HTTP_HOST IsCrossSiteOrigin() считает
// запрос доверенным (на реальном сервере Host приходит всегда).
func withHost(t *testing.T, origin string) {
	t.Helper()
	t.Setenv("HTTP_ORIGIN", origin)
	t.Setenv("HTTP_HOST", "192.168.3.1:8087")
}

// Регрессия CSRF/G3: action=save мутировал состояние по GET.
// Гейт обязателен, иначе внешний сайт пишет в историю без ведома владельца.
func TestTempHistorySaveRejectsGET(t *testing.T) {
	dir := tempHistoryDir(t)
	out := getTempHistory(t, "GET", "action=save&temp=50")

	if !strings.Contains(out, "Method not allowed") {
		t.Fatalf("GET на save должен отказывать, получено: %q", out)
	}
	if n := len(filesIn(t, dir)); n != 0 {
		t.Fatalf("GET не должен создавать файлов, создано: %d", n)
	}
	if _, err := os.Stat(dir); err == nil {
		if n := len(filesIn(t, dir)); n != 0 {
			t.Fatalf("в каталоге появились записи: %v", filesIn(t, dir))
		}
	}
}

// Регрессия G3: POST с чужим Origin должен отказывать (CSRF).
func TestTempHistorySaveRejectsCrossSiteOrigin(t *testing.T) {
	dir := tempHistoryDir(t)
	withHost(t, "http://evil.example")
	out := getTempHistory(t, "POST", "action=save&temp=50")

	if !strings.Contains(out, "недоверенного") {
		t.Fatalf("POST с чужим Origin должен отказывать, получено: %q", out)
	}
	if n := len(filesIn(t, dir)); n != 0 {
		t.Fatalf("отказ не должен оставлять файлов, создано: %d", n)
	}
}

// Счастливый путь: POST с тем же Origin пишет точку.
func TestTempHistorySaveWritesOnSameOriginPOST(t *testing.T) {
	dir := tempHistoryDir(t)
	withHost(t, "http://192.168.3.1:8087")
	out := getTempHistory(t, "POST", "action=save&temp=42")

	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("валидный POST должен записать точку, получено: %q", out)
	}
	if n := len(filesIn(t, dir)); n != 1 {
		t.Fatalf("ожидался 1 файл, создано: %d (%v)", n, filesIn(t, dir))
	}
}

// Чтение по GET продолжает работать и не создаёт каталог (write-on-GET не возвращается).
func TestTempHistoryHistoryGETStillWorks(t *testing.T) {
	dir := tempHistoryDir(t)
	os.Remove(dir)

	out := getTempHistory(t, "GET", "action=history")
	if out == "" {
		t.Fatal("чтение истории вернуло пустой ответ")
	}
	if strings.Contains(out, "null") && !strings.Contains(out, "[]") {
		t.Fatalf("ожидался пустой массив истории, получено: %q", out)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatalf("чтение не должно создавать каталог: %s", dir)
	}

	// После записи точка читается обратно.
	withHost(t, "http://192.168.3.1:8087")
	getTempHistory(t, "POST", "action=save&temp=42")
	if out := getTempHistory(t, "GET", "action=history"); !strings.Contains(out, "42") {
		t.Fatalf("записанная точка не читается: %q", out)
	}
}

// Регрессия DoS: wifi-температура не валидировалась, строка произвольной
// длины писалась в tmpfs (RAM) на каждый запрос.
func TestWifiTempHistoryRejectsNonNumeric(t *testing.T) {
	dir := tempHistoryDir(t)
	withHost(t, "http://192.168.3.1:8087")

	long := strings.Repeat("A", 4096)
	postWifiTempHistory(t, "POST", "action=save&temp0="+long+"&temp1=45")

	for _, f := range filesIn(t, dir) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), long) {
			t.Fatalf("нечисловое значение записано в %s (%d байт)", f, len(data))
		}
	}
}

// Гейты wifi_history — те же три рубежа, что и у cpu.
func TestWifiTempHistoryGates(t *testing.T) {
	dir := tempHistoryDir(t)

	if out := postWifiTempHistory(t, "GET", "action=save&temp0=40&temp1=45"); !strings.Contains(out, "Method not allowed") {
		t.Fatalf("GET на wifi save должен отказывать, получено: %q", out)
	}
	if n := len(filesIn(t, dir)); n != 0 {
		t.Fatalf("GET не должен создавать файлов, создано: %d", n)
	}

	withHost(t, "http://evil.example")
	if out := postWifiTempHistory(t, "POST", "action=save&temp0=40&temp1=45"); !strings.Contains(out, "недоверенного") {
		t.Fatalf("POST с чужим Origin должен отказывать, получено: %q", out)
	}

	withHost(t, "http://192.168.3.1:8087")
	if out := postWifiTempHistory(t, "POST", "action=save&temp0=40&temp1=45"); !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("валидный POST должен записать, получено: %q", out)
	}
	if n := len(filesIn(t, dir)); n != 1 {
		t.Fatalf("ожидался 1 файл, создано: %d (%v)", n, filesIn(t, dir))
	}
}
