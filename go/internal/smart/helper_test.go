package smart

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.Bytes()
}

func setStdin(t *testing.T, content string) func() {
	t.Helper()
	old := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte(content))
	w.Close()
	os.Stdin = r
	return func() { os.Stdin = old }
}
