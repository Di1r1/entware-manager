package packages

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var reSanitize = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizePkg(name string) string {
	return reSanitize.ReplaceAllString(name, "")
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func writeHTML(html string) {
	fmt.Print("Content-type: text/html; charset=utf-8\n\n")
	fmt.Print(html)
}

func logPackageChange(action, pkg, status string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	os.MkdirAll("/opt/var/log", 0755)
	f, err := os.OpenFile("/opt/var/log/package_changes.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s | %s | %s | %s\n", ts, action, pkg, status)
}

func runOpkg(args ...string) (string, int) {
	return runCmd("/opt/bin/opkg", args...)
}

// runOpkgTimed выполняет opkg с таймаутом 60с через coreutils-timeout (если
// доступен), иначе напрямую. Защита от зависания на недоступном/медленном
// feed — аналог opkgWithTimeout() в stats/update.go.
func runOpkgTimed(args ...string) (string, int) {
	if _, err := exec.LookPath("timeout"); err == nil {
		cmd := append([]string{"60", "/opt/bin/opkg"}, args...)
		return runCmd("timeout", cmd...)
	}
	return runOpkg(args...)
}

func runCmd(cmd string, args ...string) (string, int) {
	c := exec.Command(cmd, args...)
	out, err := c.CombinedOutput()
	if err != nil {
		return string(out), 1
	}
	return string(out), 0
}
