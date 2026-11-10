package report

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
)

// textReporter renders a human-readable report.
type textReporter struct{ color bool }

type entry struct{ key, loc string }

// paint wraps s in the given ANSI color when color output is enabled.
func (t textReporter) paint(code, s string) string {
	if !t.color {
		return s
	}
	return code + s + ansiReset
}

// Report implements Reporter. The whole report is buffered and written once,
// so a writer error is returned as-is.
func (t textReporter) Report(w io.Writer, r diff.Result) error {
	var b bytes.Buffer
	nm, ne, nx, nd := len(r.Missing), len(r.Empty), len(r.Extra), len(r.Duplicates)

	if nm+ne+nx+nd == 0 && r.OK {
		b.WriteString(t.paint(ansiGreen, fmt.Sprintf("✓ %s matches %s (%d keys)", r.Env, r.Example, r.KeyCount)))
		b.WriteByte('\n')
		_, err := w.Write(b.Bytes())
		return err
	}

	fmt.Fprintf(&b, "Comparing %s against %s\n", r.Env, r.Example)

	// Extra keys and duplicates are warnings, or failures in strict mode.
	soft := ansiYellow
	if r.Strict {
		soft = ansiRed
	}

	var es []entry
	for _, it := range r.Missing {
		es = append(es, entry{it.Key, fmt.Sprintf("%s:%d", r.Example, it.Line)})
	}
	t.section(&b, ansiRed, "✗", fmt.Sprintf("Missing in %s", r.Env), es)

	es = nil
	for _, it := range r.Empty {
		es = append(es, entry{it.Key, fmt.Sprintf("%s:%d", r.Env, it.Line)})
	}
	t.section(&b, ansiRed, "✗", fmt.Sprintf("Empty in %s", r.Env), es)

	es = nil
	for _, it := range r.Extra {
		es = append(es, entry{it.Key, fmt.Sprintf("%s:%d", r.Env, it.Line)})
	}
	t.section(&b, soft, "!", fmt.Sprintf("Extra in %s", r.Env), es)

	es = nil
	for _, d := range r.Duplicates {
		es = append(es, entry{d.Key, fmt.Sprintf("%s: lines %s", d.File, joinLines(d.Lines))})
	}
	t.section(&b, soft, "!", "Duplicate keys", es)

	// Count labels are invariant (never pluralized): "1 duplicate", "3 duplicate".
	var parts []string
	for _, c := range []struct {
		n     int
		label string
	}{{nm, "missing"}, {ne, "empty"}, {nx, "extra"}, {nd, "duplicate"}} {
		if c.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c.n, c.label))
		}
	}
	summary := ""
	if len(parts) > 0 {
		summary = " (" + strings.Join(parts, ", ") + ")"
	}
	b.WriteByte('\n')
	if r.OK {
		b.WriteString(t.paint(ansiYellow, "Result: OK with warnings"+summary))
	} else {
		b.WriteString(t.paint(ansiRed, "Result: FAIL"+summary))
	}
	b.WriteByte('\n')

	_, err := w.Write(b.Bytes())
	return err
}

// section writes one titled block; alignment uses plain text widths only.
func (t textReporter) section(b *bytes.Buffer, code, symbol, title string, es []entry) {
	if len(es) == 0 {
		return
	}
	width := 0
	for _, e := range es {
		if n := len([]rune(e.key)); n > width {
			width = n
		}
	}
	b.WriteByte('\n')
	b.WriteString(t.paint(code, fmt.Sprintf("%s %s (%d):", symbol, title, len(es))))
	b.WriteByte('\n')
	for _, e := range es {
		pad := strings.Repeat(" ", width-len([]rune(e.key))+2)
		fmt.Fprintf(b, "  - %s%s(%s)\n", e.key, pad, e.loc)
	}
}
