package network

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	fn()
	w.Close()
	<-done
	os.Stdout = old

	return trimJSONHeader(buf.Bytes())
}

func trimJSONHeader(out []byte) []byte {
	idx := bytes.Index(out, []byte("\n\n"))
	if idx < 0 {
		return out
	}
	return out[idx+2:]
}

func setStdin(t *testing.T, data string) {
	t.Helper()
	r, w, _ := os.Pipe()
	w.Write([]byte(data))
	w.Close()
	os.Stdin = r
}
