// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Self-signed HTTPS-сертификат панели.
//
// При первом старте с включённым TLS генерируется ECDSA P-256 ключ и
// самоподписанный сертификат (10 лет) в /opt/web_entware/ssl/. SAN включают
// домен из конфига, localhost и все LAN-адреса интерфейсов — браузер считает
// сертификат доверенным после единственного ручного исключения, шифрование
// при этом настоящее. Повторные старты переиспользуют файлы; перегенерация —
// удалением пары файлов в каталоге ssl/.
package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// TLSDirEnv — каталог сертификатов (переопределяется для тестов через env
// вместе с остальными путями сервера).
var tlsCertPath, tlsKeyPath string

func init() {
	SetTLSDir(envOr("EWM_WEB_ROOT", "/opt/web_entware"))
}

// SetTLSDir задаёт пути сертификата (для тестов).
func SetTLSDir(webRoot string) {
	dir := filepath.Join(webRoot, "ssl")
	tlsCertPath = filepath.Join(dir, "panel.crt")
	tlsKeyPath = filepath.Join(dir, "panel.key")
}

// CertExists — пара файлов уже сгенерирована.
func CertExists() bool {
	st, errC := os.Stat(tlsCertPath)
	stk, errK := os.Stat(tlsKeyPath)
	return errC == nil && errK == nil && !st.IsDir() && !stk.IsDir()
}

// EnsureCert гарантирует наличие self-signed пары (создаёт при отсутствии)
// и возвращает готовый tls.Certificate.
func EnsureCert(domain string) (tls.Certificate, error) {
	if CertExists() {
		cert, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err == nil {
			return cert, nil
		}
		// битая/устаревшая пара — перегенерируем
		os.Remove(tlsCertPath)
		os.Remove(tlsKeyPath)
	}
	if err := os.MkdirAll(filepath.Dir(tlsCertPath), 0700); err != nil {
		return tls.Certificate{}, fmt.Errorf("ssl dir: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("serial: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: domain, Organization: []string{"Entware Manager"}},
		NotBefore:             now.Add(-24 * time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
		IPAddresses:           append([]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, lanIPs()...),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cert: %w", err)
	}

	keyPEM, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal key: %w", err)
	}
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyPEM})
	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	// Атомарно: temp + rename; ключ строго 0600.
	if err := atomicWrite(tlsKeyPath, keyOut, 0600); err != nil {
		return tls.Certificate{}, fmt.Errorf("write key: %w", err)
	}
	if err := atomicWrite(tlsCertPath, certOut, 0644); err != nil {
		return tls.Certificate{}, fmt.Errorf("write cert: %w", err)
	}

	return tls.X509KeyPair(certOut, keyOut)
}

// lanIPs — адреса интерфейсов роутера для SAN (ошибки игнорируем: worst case
// сертификат без LAN IP, доступ по домену/localhost всё равно работает).
func lanIPs() []net.IP {
	var out []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifc := range ifaces {
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() || ipNet.IP.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, ipNet.IP)
		}
	}
	return out
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// TLSDomain — CN/SAN сертификата: домен из конфига, иначе локальное имя хоста.
func TLSDomain(cfg Config) string {
	if cfg.TLSPort <= 0 {
		return "entware-manager.local"
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "entware-manager.local"
	}
	return host
}
