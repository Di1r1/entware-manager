package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFilemgrAuth(t *testing.T) {
	oldCfg, oldMarker := authConfigFile, authMarkerFile
	defer func() { authConfigFile, authMarkerFile = oldCfg, oldMarker }()

	writeCfg := func(t *testing.T, dir, content string) {
		t.Helper()
		authConfigFile = filepath.Join(dir, "auth_config.json")
		authMarkerFile = filepath.Join(dir, ".auth_configured")
		if err := os.WriteFile(authConfigFile, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("no config no marker allow", func(t *testing.T) {
		dir := t.TempDir()
		authConfigFile = filepath.Join(dir, "auth_config.json")
		authMarkerFile = filepath.Join(dir, ".auth_configured")
		if !checkFilemgrAuth("") {
			t.Error("expected allow: no config and no marker")
		}
	})

	t.Run("no config with marker deny", func(t *testing.T) {
		dir := t.TempDir()
		authConfigFile = filepath.Join(dir, "auth_config.json")
		authMarkerFile = filepath.Join(dir, ".auth_configured")
		if err := os.WriteFile(authMarkerFile, []byte("1"), 0600); err != nil {
			t.Fatal(err)
		}
		if checkFilemgrAuth("") {
			t.Error("expected deny: no config but marker present")
		}
	})

	t.Run("broken json deny", func(t *testing.T) {
		dir := t.TempDir()
		writeCfg(t, dir, "{broken")
		if checkFilemgrAuth("") {
			t.Error("expected deny: broken JSON config")
		}
	})

	t.Run("enabled true empty credentials deny", func(t *testing.T) {
		dir := t.TempDir()
		writeCfg(t, dir, `{"enabled":true}`)
		if checkFilemgrAuth("") {
			t.Error("expected deny: enabled=true with empty hash/password")
		}
		if checkFilemgrAuth("secret") {
			t.Error("expected deny: enabled=true with empty hash/password regardless of password")
		}
	})

	t.Run("enabled false valid allow", func(t *testing.T) {
		dir := t.TempDir()
		writeCfg(t, dir, `{"enabled":false}`)
		if !checkFilemgrAuth("") {
			t.Error("expected allow: enabled=false in valid config")
		}
	})

	t.Run("enabled true correct hash allow", func(t *testing.T) {
		dir := t.TempDir()
		writeCfg(t, dir, fmt.Sprintf(`{"enabled":true,"password_hash":%q}`, sha256Hex("secret")))
		if !checkFilemgrAuth("secret") {
			t.Error("expected allow: correct hash")
		}
		if checkFilemgrAuth("wrong") {
			t.Error("expected deny: wrong password")
		}
	})

	t.Run("enabled true plain password allow", func(t *testing.T) {
		dir := t.TempDir()
		writeCfg(t, dir, `{"enabled":true,"password":"secret"}`)
		if !checkFilemgrAuth("secret") {
			t.Error("expected allow: correct plain password")
		}
		if checkFilemgrAuth("wrong") {
			t.Error("expected deny: wrong plain password")
		}
	})
}
