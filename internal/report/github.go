package report

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
)

// githubReporter emits GitHub Actions workflow commands, one per finding.
type githubReporter struct{}

// Report implements Reporter. Nothing is written when there are no findings.
func (githubReporter) Report(w io.Writer, r diff.Result) error {
	soft := "warning"
	if r.Strict {
		soft = "error"
	}
	var buf bytes.Buffer
	for _, it := range r.Missing {
		command(&buf, "error", r.Example, it.Line,
			fmt.Sprintf("Missing key %s in %s", it.Key, r.Env))
	}
	for _, it := range r.Empty {
		command(&buf, "error", r.Env, it.Line,
			fmt.Sprintf("Empty value for key %s in %s", it.Key, r.Env))
	}
	for _, it := range r.Extra {
		command(&buf, soft, r.Env, it.Line,
			fmt.Sprintf("Extra key %s not present in %s", it.Key, r.Example))
	}
	for _, d := range r.Duplicates {
		line := 0
		if len(d.Lines) > 0 {
			line = d.Lines[0]
		}
		command(&buf, soft, d.File, line,
			fmt.Sprintf("Duplicate key %s (lines %s)", d.Key, joinLines(d.Lines)))
	}
	if buf.Len() == 0 {
		return nil
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func command(buf *bytes.Buffer, level, file string, line int, msg string) {
	fmt.Fprintf(buf, "::%s file=%s,line=%d::%s\n", level, escapeProperty(file), line, escapeData(msg))
}

func joinLines(lines []int) string {
	s := make([]string, len(lines))
	for i, l := range lines {
		s[i] = strconv.Itoa(l)
	}
	return strings.Join(s, ", ")
}

var (
	dataEscaper     = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	propertyEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
)

// escapeData escapes a workflow command message.
func escapeData(s string) string { return dataEscaper.Replace(s) }

// escapeProperty escapes a workflow command property value.
func escapeProperty(s string) string { return propertyEscaper.Replace(s) }
