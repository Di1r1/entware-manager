package localtime

import (
	"testing"
	"time"
)

func TestPosixLocation(t *testing.T) {
	cases := []struct {
		in   string
		want int // сдвиг от UTC в секундах
	}{
		{"MSK-3", 3 * 3600},     // UTC+3
		{"EST5", -5 * 3600},     // UTC-5
		{"UTC0", 0},             // UTC
		{"JST-9", 9 * 3600},     // UTC+9
		{"CET-1CEST", 1 * 3600}, // UTC+1, DST-часть игнорируется
		{"UTC", 0},              // нет offset → считаем UTC
		{"", 0},
		{"foo", 0},
	}
	for _, c := range cases {
		loc := posixLocation(c.in)
		if loc == nil {
			if c.want != 0 {
				t.Errorf("%q: expected offset %d, got nil", c.in, c.want)
			}
			continue
		}
		_, off := time.Date(2026, 1, 1, 12, 0, 0, 0, loc).Zone()
		if off != c.want {
			t.Errorf("%q: offset = %d, want %d", c.in, off, c.want)
		}
	}
}
