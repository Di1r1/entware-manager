package stats

import (
	"archive/tar"
	"compress/gzip"
	"entware-manager/internal/cgiutil"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func HandleOfflinePrepare() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Method not allowed"})
		return
	}

	arch := getRouterArch()
	if arch == "" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Архитектура не определена"})
		return
	}

	version := getLocalVersion()
	if version == "" || version == "unknown" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Версия не определена"})
		return
	}

	cleanupOldTemp("entware-offline-")

	tmpDir, err := os.MkdirTemp("", "entware-offline-")
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось создать временную папку"})
		return
	}
	defer os.RemoveAll(tmpDir)

	err = buildOfflineBundle(tmpDir, arch, version)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	archiveName := fmt.Sprintf("entware-manager-offline-%s-v%s.tar.gz", arch, version)
	archivePath := filepath.Join(tmpDir, archiveName)

	fi, err := os.Stat(archivePath)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Архив не найден"})
		return
	}

	fmt.Printf("Content-Type: application/gzip\n")
	fmt.Printf("Content-Disposition: attachment; filename=\"%s\"\n", archiveName)
	fmt.Printf("Content-Length: %d\n", fi.Size())
	fmt.Println()

	f, err := os.Open(archivePath)
	if err != nil {
		return
	}
	defer f.Close()
	io.Copy(os.Stdout, f)
}

func buildOfflineBundle(tmpDir, arch, version string) error {
	pkgDir := filepath.Join(tmpDir, "packages")
	deployDir := filepath.Join(tmpDir, "deploy")

	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать packages: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}

	url := getDownloadURL(version, arch)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка загрузки Entware Manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub ответил %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка gzip: %w", err)
	}
	defer gzr.Close()

	prefix := "deploy-" + arch
	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка tar: %w", err)
		}

		name := header.Name
		if strings.HasPrefix(name, prefix) {
			name = "deploy" + strings.TrimPrefix(name, prefix)
		} else {
			// Архив должен содержать только содержимое deploy-<arch>/.
			continue
		}

		// Защита от tar-path-traversal: только чистые относительные пути внутри tmpDir.
		rel := filepath.Clean(name)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			return fmt.Errorf("небезопасный путь в архиве: %s", header.Name)
		}
		target := filepath.Join(tmpDir, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("не удалось создать %s: %w", target, err)
			}
			io.Copy(f, tarReader)
			f.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		case tar.TypeSymlink:
			// Симлинки создаём только с безопасной (относительной, внутри дерева)
			// целью — защита от tar-traversal через Linkname.
			if linkTarget, ok := safeSymlinkTarget(tmpDir, rel, header.Linkname); ok {
				os.MkdirAll(filepath.Dir(linkTarget), 0755)
				os.Symlink(header.Linkname, linkTarget)
			}
		}
	}

	if _, err := os.Stat(deployDir); os.IsNotExist(err) {
		return fmt.Errorf("deploy/ не найден в архиве")
	}

	deps := []string{"lighttpd", "lighttpd-mod-cgi", "jq", "curl", "ttyd", "htop", "coreutils", "coreutils-timeout", "procps-ng", "bridge", "ip-full", "sudo", "bash", "smartmontools", "smartmontools-drivedb"}
	for _, pkg := range deps {
		cmd := exec.Command("opkg", "download", pkg)
		cmd.Dir = pkgDir
		_ = cmd.Run()
	}

	script := `#!/bin/sh
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
echo "============================================"
echo " Установка Entware Manager (офлайн)"
echo "============================================"
echo ""
echo "[1] Установка зависимостей..."
echo "--------------------------------------------"
cd "$DIR"
for ipk in packages/*.ipk; do
    [ -f "$ipk" ] && {
        echo "  $(basename "$ipk")..."
        opkg install "$ipk" 2>&1 || echo "  [warn] ошибка установки $(basename "$ipk")"
    }
done
echo ""
echo "[2] Установка Entware Manager..."
echo "--------------------------------------------"
cd "$DIR/deploy" && sh Install/install.sh
echo ""
echo "============================================"
echo " Готово"
echo "============================================"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "install-offline.sh"), []byte(script), 0755); err != nil {
		return fmt.Errorf("не удалось создать install-offline.sh: %w", err)
	}

	archiveName := fmt.Sprintf("entware-manager-offline-%s-v%s.tar.gz", arch, version)
	archivePath := filepath.Join(tmpDir, archiveName)
	cmd := exec.Command("tar", "-czf", archivePath, "-C", tmpDir, "deploy", "packages", "install-offline.sh")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// cleanupOldTemp удаляет временные папки с указанным префиксом старше 24 часов.
// Защита: папки, изменённые за последние 24 часа, не трогаем — операция может быть активной.
func cleanupOldTemp(prefix string) {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), prefix+"*"))
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, d := range dirs {
		fi, err := os.Stat(d)
		if err != nil || !fi.IsDir() {
			continue
		}
		if fi.ModTime().Before(cutoff) {
			os.RemoveAll(d)
		}
	}
}
