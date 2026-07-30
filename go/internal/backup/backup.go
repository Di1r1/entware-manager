package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const webRoot = "/opt/web_entware"

type configFile struct {
	Path    string // relative to webRoot
	AbsPath string // absolute path (empty = generate)
	GenFn   func() (string, string) // returns (filename, content)
}

var configs = []configFile{
	{Path: "links.json"},
	{Path: "monitor_config.json"},
	{Path: "network_config.json"},
	{Path: "service_config.json"},
	{Path: "logger/config.json"},
}

func HandleCreate() {
	tmpDir, err := os.MkdirTemp("", "entware-backup-*")
	if err != nil {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
		fmt.Println("Error: cannot create temp dir:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	for _, cf := range configs {
		src := filepath.Join(webRoot, cf.Path)
		dst := filepath.Join(tmpDir, strings.ReplaceAll(cf.Path, "/", "_"))
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(dst, data, 0644)
		}
	}

	pkgList, _ := exec.Command("opkg", "list-installed").Output()
	if len(pkgList) > 0 {
		os.WriteFile(filepath.Join(tmpDir, "packages.txt"), pkgList, 0644)
	}

	info := map[string]string{
		"version": "1.06.5",
		"date":    "2026-07-29",
	}
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "backup.json"), infoJSON, 0644)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
		if err != nil {
			continue
		}
		tw.WriteHeader(&tar.Header{
			Name: entry.Name(),
			Size: int64(len(data)),
			Mode: 0644,
		})
		tw.Write(data)
	}

	tw.Close()
	gw.Close()

	fmt.Println("Content-type: application/gzip; charset=utf-8")
	fmt.Println("Content-Disposition: attachment; filename=entware-manager-backup.tar.gz")
	fmt.Println("Content-Length:", buf.Len())
	fmt.Println()
	os.Stdout.Write(buf.Bytes())
}

func HandleRestore() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
		fmt.Println("Error: POST required")
		return
	}

	body, err := io.ReadAll(os.Stdin)
	if err != nil || len(body) < 100 {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Empty or too small file"}`)
		return
	}

	tmpDir, err := os.MkdirTemp("", "entware-restore-*")
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Cannot create temp dir"}`)
		return
	}
	defer os.RemoveAll(tmpDir)

	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Invalid gzip data"}`)
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Invalid tar archive"}`)
			return
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			continue
		}

		os.WriteFile(filepath.Join(tmpDir, header.Name), data, 0644)
	}

	restoreFile := func(tmpName, destRel string) error {
		src := filepath.Join(tmpDir, tmpName)
		if _, err := os.Stat(src); err != nil {
			return err
		}
		dst := filepath.Join(webRoot, destRel)
		os.MkdirAll(filepath.Dir(dst), 0755)
		data, _ := os.ReadFile(src)
		return os.WriteFile(dst, data, 0644)
	}

	type restoreMap struct {
		tmpName string
		dest    string
	}

	restoreList := []restoreMap{
		{"links.json", "links.json"},
		{"monitor_config.json", "monitor_config.json"},
		{"network_config.json", "network_config.json"},
		{"service_config.json", "service_config.json"},
		{"logger_config.json", "logger/config.json"},
	}

	var restored []string
	for _, rm := range restoreList {
		if err := restoreFile(rm.tmpName, rm.dest); err == nil {
			restored = append(restored, rm.dest)
		}
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	if len(restored) > 0 {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "ok",
			"message":  fmt.Sprintf("Restored: %s", strings.Join(restored, ", ")),
			"restored": restored,
		})
	} else {
		fmt.Println(`{"status":"error","message":"No config files found in backup"}`)
	}
}
