package logger

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func HandleLoggerDebug() {
	w := os.Stdout
	fmt.Fprintln(w, "Content-type: text/plain; charset=utf-8")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "--- DEBUG ---")
	fmt.Fprintln(w, "SYSTEM_LOG="+systemLogFile)
	info, err := os.Stat(systemLogFile)
	if err != nil {
		fmt.Fprintln(w, "FILE NOT FOUND")
	} else {
		fmt.Fprintln(w, "FILE EXISTS")
		if info.Size() > 0 {
			fmt.Fprintln(w, "FILE NOT EMPTY")
		} else {
			fmt.Fprintln(w, "FILE EMPTY OR NOT EXIST")
		}
	}
	out, _ := exec.Command("ls", "-la", systemLogFile).CombinedOutput()
	fmt.Fprint(w, string(out))
	fmt.Fprintln(w, "--- END ---")
}

func HandleLoggerDebugPath() {
	w := os.Stdout
	fmt.Fprintln(w, "Content-type: text/plain; charset=utf-8")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "PATH="+os.Getenv("PATH"))
	fmt.Fprintln(w, "---")
	catPath, err := exec.LookPath("cat")
	if err != nil {
		fmt.Fprintln(w, "cat not found")
	} else {
		fmt.Fprintln(w, catPath)
	}
	sedPath, err := exec.LookPath("sed")
	if err != nil {
		fmt.Fprintln(w, "sed not found")
	} else {
		fmt.Fprintln(w, sedPath)
	}
	data, err := os.ReadFile(systemLogFile)
	if err != nil {
		fmt.Fprintln(w, "cat failed: "+err.Error())
	} else {
		lines := bytes.Split(data, []byte("\n"))
		start := 0
		if len(lines) > 50 {
			start = len(lines) - 50
			fmt.Fprintf(w, "... (showing last 50 of %d lines)\n", len(lines))
		}
		for _, l := range lines[start:] {
			fmt.Fprintln(w, string(l))
		}
	}
}
