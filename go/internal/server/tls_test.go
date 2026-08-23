// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func timeNow() time.Time           { return time.Now() }
func ellipticP256() elliptic.Curve { return elliptic.P256() }

func TestEnsureCertGeneratesAndReuses(t *testing.T) {
	root := t.TempDir()
	SetTLSDir(root)
	t.Cleanup(func() { SetTLSDir("/opt/web_entware") })

	if CertExists() {
		t.Fatal("свежий каталог не должен содержать сертификат")
	}
	cert, err := EnsureCert("myrouter")
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("пустой сертификат")
	}
	if !CertExists() {
		t.Fatal("файлы crt/key не созданы")
	}

	// права ключа 0600
	st, _ := os.Stat(tlsKeyPath)
	if st.Mode().Perm() != 0600 {
		t.Errorf("права ключа = %v, хочу 0600", st.Mode().Perm())
	}

	// повторный вызов переиспользует (тот же лист сертфикатов)
	cert2, err := EnsureCert("myrouter")
	if err != nil {
		t.Fatal(err)
	}
	if string(cert2.Certificate[0]) != string(cert.Certificate[0]) {
		t.Error("повторный вызов перегенерировал сертификат вместо переиспользования")
	}
}

func TestEnsureCertContent(t *testing.T) {
	root := t.TempDir()
	SetTLSDir(root)
	t.Cleanup(func() { SetTLSDir("/opt/web_entware") })

	if _, err := EnsureCert("myrouter"); err != nil {
		t.Fatal(err)
	}
	pemBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("сертификат не PEM")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if x509Cert.Subject.CommonName != "myrouter" {
		t.Errorf("CN = %q, хочу myrouter", x509Cert.Subject.CommonName)
	}
	if !x509Cert.NotAfter.After(timeNow().AddDate(9, 0, 0)) {
		t.Error("срок действия меньше 10 лет")
	}
	foundLocalhost := false
	for _, ip := range x509Cert.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			foundLocalhost = true
		}
	}
	if !foundLocalhost {
		t.Error("SAN не содержит 127.0.0.1")
	}
	if x509Cert.PublicKeyAlgorithm != x509.ECDSA {
		t.Errorf("алгоритм ключа = %v, хочу ECDSA", x509Cert.PublicKeyAlgorithm)
	}
	pub, ok := x509Cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || pub.Curve != ellipticP256() {
		t.Error("ключ не P-256")
	}
	_ = tls.VersionTLS12
}

func TestLoadConfigTLSFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "server_config.json")
	old := serverConfig
	serverConfig = cfgPath
	defer func() { serverConfig = old }()

	os.WriteFile(cfgPath, []byte(`{"port":9090,"tls":true,"tls_port":9443}`), 0644)
	cfg := LoadConfig()
	if !cfg.TLS || cfg.TLSPort != 9443 || cfg.Port != 9090 {
		t.Errorf("cfg = %+v", cfg)
	}

	// дефолты при отсутствии файла
	os.Remove(cfgPath)
	cfg = LoadConfig()
	if cfg.TLS || cfg.TLSPort != defaultTLSPort {
		t.Errorf("дефолты: %+v", cfg)
	}

	// битый JSON → дефолты
	os.WriteFile(cfgPath, []byte("{broken"), 0644)
	cfg = LoadConfig()
	if cfg.TLS {
		t.Error("битый конфиг не должен включать TLS")
	}
}
