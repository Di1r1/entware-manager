package logger

import (
	"os"
	"path/filepath"
	"strings"
)

type FoundFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func HandleFind() {
	if !IsGET() {
		NotAllowed()
		return
	}

	q := strings.TrimSpace(getQueryParam("q"))
	if q == "" {
		WriteJSON([]FoundFile{})
		return
	}

	q = strings.ToLower(q)
	var result []FoundFile
	root := "/tmp"
	skipPrefix := "/tmp/mnt"

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(path, skipPrefix) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(info.Name()), q) {
			result = append(result, FoundFile{
				Name: info.Name(),
				Path: path,
			})
		}
		return nil
	})

	if result == nil {
		result = []FoundFile{}
	}
	WriteJSON(result)
}
