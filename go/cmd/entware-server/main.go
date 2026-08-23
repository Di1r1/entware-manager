// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// entware-server — собственный HTTP-сервер Entware Manager.
//
// Заменяет lighttpd на роутерах со сторонним lighttpd (nfqws/zapret).
// Порт берётся из /opt/web_entware/server_config.json ({"port": 8087}).
package main

import (
	"context"
	"crypto/tls"
	_ "entware-manager/internal/buildinfo"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "entware-manager/internal/localtime"
	"entware-manager/internal/server"
)

func main() {
	cfg := server.LoadConfig()

	if f := server.SetupLogging(); f != nil {
		defer f.Close()
	}
	server.WritePID()

	addr := ":8087"
	if cfg.Port > 0 {
		addr = ":" + itoa(cfg.Port)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewHandler(),
		ReadHeaderTimeout: 30 * time.Second,
	}

	// Graceful shutdown по SIGTERM/SIGINT (init-скрипт и watchdog шлют TERM).
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("entware-server listening on %s (timeout=%ds)", addr, cfg.Timeout)

	// Дополнительный HTTPS-листенер (self-signed): HTTP остаётся для
	// совместимости (lighttpd-прокси, LAN-ссылки), HTTPS — для защиты
	// пароля/сессии при доступе по IP.
	if cfg.TLS {
		go func() {
			cert, err := server.EnsureCert(server.TLSDomain(cfg))
			if err != nil {
				log.Printf("tls: сертификат не создан, HTTPS отключён: %v", err)
				return
			}
			tlsSrv := &http.Server{
				Addr:              ":" + itoa(cfg.TLSPort),
				Handler:           server.NewHandler(),
				ReadHeaderTimeout: 30 * time.Second,
				TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
			}
			go func() {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
				<-sig
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = tlsSrv.Shutdown(ctx)
			}()
			log.Printf("entware-server TLS listening on :%d", cfg.TLSPort)
			if err := tlsSrv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("tls error: %v", err)
			}
		}()
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
	os.Remove(server.PidFile())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
