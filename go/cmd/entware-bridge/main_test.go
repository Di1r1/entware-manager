package main

import (
	"strings"
	"testing"

	"entware-manager/internal/bridge"
)

// Регрессия P1-08: мёртвый сервис и HTTP >= 400 обязаны давать status:"error"
// с внятным сообщением. До этой проверки main.go смотрел только на err (там он
// nil для таких случаев) и писал «выполнено» — UI и Telegram-лог врали.
func TestActionVerdict(t *testing.T) {
	cases := []struct {
		name    string
		res     *bridge.StatusProxy
		status  string
		wantMsg string // подстрока, обязательная в message; "" = не проверяем
	}{
		{"успех 200", &bridge.StatusProxy{HTTPCode: 200, Raw: "HTTP 200"}, "ok", ""},
		{"успех 204", &bridge.StatusProxy{HTTPCode: 204}, "ok", ""},
		{"недоступен", &bridge.StatusProxy{Error: "сервис не отвечает"}, "error", "не выполнено: сервис не отвечает"},
		{"HTTP 500", &bridge.StatusProxy{HTTPCode: 500, Raw: "HTTP 500"}, "error", "не выполнено: сервис ответил HTTP 500"},
		{"HTTP 401", &bridge.StatusProxy{HTTPCode: 401}, "error", "HTTP 401"},
		{"HTTP 403", &bridge.StatusProxy{HTTPCode: 403}, "error", "HTTP 403"},
		{"nil = fail-closed", nil, "error", "нет результата от сервиса"},
	}
	for _, c := range cases {
		status, message := actionVerdict(c.res)
		if status != c.status {
			t.Errorf("%s: status=%q, ожидалось %q", c.name, status, c.status)
		}
		if c.wantMsg != "" && !strings.Contains(message, c.wantMsg) {
			t.Errorf("%s: message=%q, ожидалось подстроку %q", c.name, message, c.wantMsg)
		}
		if c.status == "ok" && message != "" {
			t.Errorf("%s: при status=ok message должен быть пустым, получено %q", c.name, message)
		}
	}
}

// Имя модуля и подпись действия приходят из манифеста, а валидация проверяет
// только длину. Перевод строки в них дописывал бы в суточный лог поддельную
// запись — поэтому в лог уходит одна строка.
func TestLogSafeOneLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Модуль", "Модуль"},
		{"Модуль\n[INFO] подделка", "Модуль [INFO] подделка"},
		{"Модуль\r\nЕщё", "Модуль  Ещё"},
		{"", ""},
	}
	for _, c := range cases {
		if got := logSafe(c.in); got != c.want {
			t.Errorf("logSafe(%q)=%q, ожидалось %q", c.in, got, c.want)
		}
		if strings.ContainsAny(logSafe(c.in), "\r\n") {
			t.Errorf("logSafe(%q) оставил перевод строки", c.in)
		}
	}
}
