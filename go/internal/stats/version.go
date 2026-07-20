package stats

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed help.html
var helpTemplate string

func HandleVersion() {
	data, err := os.ReadFile("/opt/web_entware/version.json")
	if err != nil {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"version":"unknown","date":"unknown"}`)
		return
	}
	fmt.Println("Content-type: application/json; charset=utf-8\n")
	os.Stdout.Write(data)
}

func HandleHelp() {
	version := "unknown"
	dateStr := "?"
	data, err := os.ReadFile("/opt/web_entware/version.json")
	if err == nil {
		var v struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &v) == nil && v.Version != "" {
			version = v.Version
		}
		if fi, err := os.Stat("/opt/web_entware/version.json"); err == nil {
			dateStr = fi.ModTime().Format("2006-01-02")
		}
	}

	html := helpTemplate
	html = replacePlaceholder(html, "{VERSION}", version)
	html = replacePlaceholder(html, "{DATE}", dateStr)

	fmt.Println("Content-type: text/html; charset=utf-8")
	fmt.Println("Cache-Control: no-cache, no-store, must-revalidate")
	fmt.Println("Pragma: no-cache")
	fmt.Println("Expires: 0")
	fmt.Println()
	fmt.Print(html)
}

func replacePlaceholder(s, placeholder, value string) string {
	b := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if i+len(placeholder) <= len(s) && s[i:i+len(placeholder)] == placeholder {
			b = append(b, value...)
			i += len(placeholder)
		} else {
			b = append(b, s[i])
			i++
		}
	}
	return string(b)
}
