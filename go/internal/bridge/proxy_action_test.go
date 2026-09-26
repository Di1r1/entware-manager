package bridge

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Закрытый loopback-порт: соединение отклоняется мгновенно.
func closedPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// Регрессия P1-08: RunAction кладёт transport-ошибку и код ответа в StatusProxy,
// а не в error. Раньше main.go проверял только err и писал в лог «выполнено»
// для мёртвого сервиса, отдавая UI status:"ok".
func TestRunActionTransportErrorIsFailure(t *testing.T) {
	port := closedPort(t)
	m := &Manifest{ID: "dead", Name: "Мёртвый", Base: "http://" + port}
	a := &Action{ID: "restart", Label: "Перезапуск", URL: "http://" + port + "/restart"}

	sp, err := RunAction(m, a)
	if err != nil {
		t.Fatalf("RunAction вернул err=%v; контракт: transport-ошибка лежит в StatusProxy", err)
	}
	if sp.Error == "" {
		t.Fatal("StatusProxy.Error пуст при недоступном сервисе")
	}
	if !sp.Failed() {
		t.Errorf("недоступный сервис не помечен как сбой: %+v", sp)
	}
	if !strings.Contains(sp.FailMessage(), "не отвечает") {
		t.Errorf("FailMessage=%q, ожидалось «не отвечает»", sp.FailMessage())
	}
}

func TestStatusProxyFailed(t *testing.T) {
	cases := []struct {
		name    string
		sp      *StatusProxy
		failed  bool
		message string
	}{
		{"успех 200", &StatusProxy{HTTPCode: 200, Raw: "HTTP 200"}, false, ""},
		{"успех 204 без тела", &StatusProxy{HTTPCode: 204}, false, ""},
		{"редирект не ошибка", &StatusProxy{HTTPCode: 302}, false, ""},
		{"ошибка 500", &StatusProxy{HTTPCode: 500, Raw: "HTTP 500"}, true, "HTTP 500"},
		{"ошибка 401", &StatusProxy{HTTPCode: 401}, true, "HTTP 401"},
		{"ошибка 503", &StatusProxy{HTTPCode: 503}, true, "HTTP 503"},
		{"transport-ошибка", &StatusProxy{HTTPCode: 0, Error: "сервис не отвечает"}, true, "сервис не отвечает"},
		{"nil не паникует", nil, false, "неизвестная ошибка"},
	}
	for _, c := range cases {
		if got := c.sp.Failed(); got != c.failed {
			t.Errorf("%s: Failed()=%v, ожидалось %v (%+v)", c.name, got, c.failed, c.sp)
		}
		msg := c.sp.FailMessage()
		if c.failed && !strings.Contains(msg, c.message) {
			t.Errorf("%s: FailMessage()=%q, ожидалось подстроку %q", c.name, msg, c.message)
		}
	}
}

// Сервис ответил 500: err==nil, но действие выполнено не было -- Failed() обязан быть true.
func TestRunActionHTTP500IsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := &Manifest{ID: "broken", Name: "Сломанный", Base: srv.URL}
	a := &Action{ID: "apply", Label: "Применить", URL: srv.URL + "/apply"}

	sp, err := RunAction(m, a)
	if err != nil {
		t.Fatalf("RunAction вернул err=%v на HTTP 500", err)
	}
	if sp.HTTPCode != http.StatusInternalServerError {
		t.Fatalf("HTTPCode=%d, ожидалось 500", sp.HTTPCode)
	}
	if !sp.Failed() {
		t.Errorf("HTTP 500 не помечен как сбой: %+v", sp)
	}
}

// Успешный путь менять нельзя: 2xx остаётся успехом.
func TestRunActionOKNotFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := &Manifest{ID: "good", Name: "Рабочий", Base: srv.URL}
	a := &Action{ID: "apply", Label: "Применить", URL: srv.URL + "/apply"}

	sp, err := RunAction(m, a)
	if err != nil {
		t.Fatalf("RunAction вернул err=%v на HTTP 200", err)
	}
	if sp.Failed() {
		t.Errorf("HTTP 200 помечен как сбой: %+v", sp)
	}
}

// Невалидный URL остаётся ошибкой Go (этот путь не менялся).
func TestRunActionBadURLStillError(t *testing.T) {
	m := &Manifest{ID: "x", Name: "X", Base: "http://127.0.0.1:8080"}
	a := &Action{ID: "go", Label: "Уйти", URL: "http://127.0.0.1:8080.evil.tld/act"}

	if _, err := RunAction(m, a); err == nil {
		t.Fatal("RunAction принял невалидный URL без ошибки")
	}
}
