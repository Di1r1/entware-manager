package services

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func HandleDebug() {
	w := os.Stdout
	fmt.Fprintln(w, "Content-type: text/plain; charset=utf-8")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "REQUEST_METHOD="+os.Getenv("REQUEST_METHOD"))
	fmt.Fprintln(w, "CONTENT_LENGTH="+os.Getenv("CONTENT_LENGTH"))

	if os.Getenv("REQUEST_METHOD") == "POST" {
		body, _ := io.ReadAll(os.Stdin)
		bodyStr := string(body)
		fmt.Fprintf(w, "_POST_BODY=[%s]\n", bodyStr)

		params := parseFormBody(bodyStr)
		pkg := params["package"]
		fmt.Fprintf(w, "pkg=[%s]\n", pkg)
		clean := sanitizeAlnum(pkg)
		fmt.Fprintf(w, "clean=[%s]\n", clean)
	}
}

func sanitizeAlnum(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
