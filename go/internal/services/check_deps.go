package services

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	opkgBin         = "/opt/bin/opkg"
	lighttpdPidFile = "/opt/var/run/lighttpd.pid"
	cronPidFile     = "/opt/var/run/cron.pid"
)

type DepsBase struct {
	Opkg            bool `json:"opkg"`
	LighttpdRunning bool `json:"lighttpd_running"`
	Sed             bool `json:"sed"`
	Awk             bool `json:"awk"`
	Grep            bool `json:"grep"`
	Ps              bool `json:"ps"`
}

type DepsDeps struct {
	CronInstalled  bool   `json:"cron_installed"`
	CronRunning    bool   `json:"cron_running"`
	Jq             bool   `json:"jq"`
	Ip             bool   `json:"ip"`
	IpPath         string `json:"ip_path"`
	IpPkgInstalled bool   `json:"ip_pkg_installed"`
	Curl           bool   `json:"curl"`
	Bash           bool   `json:"bash"`
	Brctl          bool   `json:"brctl"`
	BrctlPath      string `json:"brctl_path"`
}

type DepsSections struct {
	Packages   string `json:"packages"`
	Services   string `json:"services"`
	Monitoring string `json:"monitoring"`
	Network    string `json:"network"`
	Logger     string `json:"logger"`
	Smart      string `json:"smart"`
}

type DepsResult struct {
	Base          DepsBase     `json:"base"`
	Deps          DepsDeps     `json:"deps"`
	Sections      DepsSections `json:"sections"`
	OverallStatus string       `json:"overall_status"`
	Timestamp     string       `json:"timestamp"`
}

func HandleCheckDeps() {
	if !IsGET() {
		NotAllowed()
		return
	}

	r := DepsResult{}

	r.Base.Sed = lookPath("sed")
	r.Base.Awk = lookPath("awk")
	r.Base.Grep = lookPath("grep")
	r.Base.Ps = lookPath("ps")

	_, err := exec.Command(opkgBin, "--version").CombinedOutput()
	r.Base.Opkg = err == nil

	lighttpdPid := readPid(lighttpdPidFile)
	r.Base.LighttpdRunning = lighttpdPid > 0 && pidIsAlive(lighttpdPid)

	r.Deps.CronInstalled = opkgListInstalled("cron")
	cronPid := readPid(cronPidFile)
	r.Deps.CronRunning = cronPid > 0 && pidIsAlive(cronPid)

	r.Deps.Jq = lookPath("/opt/bin/jq")

	ipPath, ipOk := lookPathWithPath("ip")
	r.Deps.Ip = ipOk
	r.Deps.IpPath = ipPath
	r.Deps.IpPkgInstalled = opkgListInstalled("ip-full")

	r.Deps.Curl = lookPath("curl") || lookPath("/opt/bin/curl")
	r.Deps.Bash = lookPath("/opt/bin/bash") || lookPath("bash")
	brctlPath, brctlOk := lookPathWithPath("brctl")
	r.Deps.Brctl = brctlOk
	r.Deps.BrctlPath = brctlPath

	r.Sections.Packages = statusOk(r.Base.Opkg)
	r.Sections.Services = statusOk(r.Deps.CronInstalled && r.Deps.Jq)
	r.Sections.Monitoring = statusPartial(r.Deps.CronInstalled && r.Deps.Jq)
	r.Sections.Network = statusOk(r.Deps.Ip && r.Deps.Brctl)
	r.Sections.Logger = statusOk(r.Deps.Jq)
	r.Sections.Smart = statusSmart()

	r.OverallStatus = "ok"
	if !r.Base.Opkg || !r.Base.LighttpdRunning {
		r.OverallStatus = "critical"
	} else if r.Sections.Services == "missing" || r.Sections.Network == "missing" || r.Sections.Logger == "missing" || r.Sections.Smart == "missing" {
		r.OverallStatus = "partial"
	}

	r.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	WriteJSON(r)
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func lookPathWithPath(name string) (string, bool) {
	p, err := exec.LookPath(name)
	return p, err == nil
}

func pidIsAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func readPid(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	return pid
}

func opkgListInstalled(pkg string) bool {
	out, err := exec.Command(opkgBin, "list-installed").Output()
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, pkg+" ") {
			return true
		}
	}
	return false
}

func statusOk(cond bool) string {
	if cond {
		return "ok"
	}
	return "missing"
}

func statusPartial(cond bool) string {
	if cond {
		return "ok"
	}
	return "partial"
}

func statusSmart() string {
	if lookPath("/opt/sbin/smartctl") || lookPath("smartctl") {
		return "ok"
	}
	return "missing"
}

type SyntaxFile struct {
	File    string `json:"file"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SyntaxResult struct {
	Results     []SyntaxFile `json:"results"`
	TotalErrors int          `json:"total_errors"`
	Timestamp   string       `json:"timestamp"`
}

var (
	webEntwareDir = "/opt/web_entware"
)

func HandleCheckSyntax() {
	if !IsGET() {
		NotAllowed()
		return
	}

	var files []string
	cgiDir := filepath.Join(webEntwareDir, "cgi-bin")
	if entries, err := os.ReadDir(cgiDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				sub, _ := os.ReadDir(filepath.Join(cgiDir, e.Name()))
				for _, f := range sub {
					if !f.IsDir() && (strings.HasSuffix(f.Name(), ".cgi") || strings.HasSuffix(f.Name(), ".sh")) {
						files = append(files, filepath.Join(cgiDir, e.Name(), f.Name()))
					}
				}
			} else if strings.HasSuffix(e.Name(), ".cgi") {
				files = append(files, filepath.Join(cgiDir, e.Name()))
			}
		}
	}
	libDir := filepath.Join(webEntwareDir, "lib")
	if entries, err := os.ReadDir(libDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
				files = append(files, filepath.Join(libDir, e.Name()))
			}
		}
	}

	r := SyntaxResult{Results: []SyntaxFile{}}
	for _, f := range files {
		rel, _ := filepath.Rel(webEntwareDir, f)
		sf := SyntaxFile{File: rel, Status: "ok"}
		out, err := exec.Command("sh", "-n", f).CombinedOutput()
		if err != nil {
			sf.Status = "error"
			sf.Message = strings.TrimSpace(string(out))
			r.TotalErrors++
		}
		r.Results = append(r.Results, sf)
	}
	r.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	WriteJSON(r)
}
