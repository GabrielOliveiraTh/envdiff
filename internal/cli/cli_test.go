package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GabrielOliveiraTh/envdiff/internal/cli"
)

const sentinel = "S3CR3T_SENTINEL"

func fx(name string) string { return filepath.Join("..", "..", "testdata", name) }

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var o, e bytes.Buffer
	code = cli.Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestSuccess(t *testing.T) {
	code, out, errOut := run(t, "-e", fx("ok.env.example"), "-f", fx("ok.env"))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	if !strings.Contains(out, "✓") || !strings.Contains(out, fx("ok.env")) {
		t.Errorf("unexpected stdout: %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("ANSI codes in non-terminal output: %q", out)
	}
	if errOut != "" {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestFailure(t *testing.T) {
	code, out, _ := run(t, "-e", fx("fail.env.example"), "-f", fx("fail.env"))
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	for _, want := range []string{"API_TOKEN", "STRAY_TOKEN", "FAIL"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q: %q", want, out)
		}
	}
}

func TestLongFlags(t *testing.T) {
	code, _, _ := run(t, "--example", fx("ok.env.example"), "--env", fx("ok.env"))
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
}

func TestDefaultsMissingFile(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	code, out, errOut := run(t)
	if code != 2 || out != "" || !strings.Contains(errOut, ".env.example") {
		t.Errorf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestStrict(t *testing.T) {
	args := []string{"-e", fx("ok.env.example"), "-f", fx("warn.env")}
	if code, _, _ := run(t, args...); code != 0 {
		t.Errorf("without --strict: code = %d", code)
	}
	if code, _, _ := run(t, append(args, "--strict")...); code != 1 {
		t.Errorf("with --strict: code = %d", code)
	}
	dup := []string{"-e", fx("ok.env.example"), "-f", fx("dup.env")}
	if code, _, _ := run(t, dup...); code != 0 {
		t.Errorf("dup without --strict: code = %d", code)
	}
	if code, _, _ := run(t, append(dup, "--strict")...); code != 1 {
		t.Errorf("dup with --strict: code = %d", code)
	}
}

func TestQuiet(t *testing.T) {
	code, out, errOut := run(t, "-q", "-e", fx("ok.env.example"), "-f", fx("ok.env"))
	if code != 0 || out != "" || errOut != "" {
		t.Errorf("ok: %d %q %q", code, out, errOut)
	}
	code, out, _ = run(t, "--quiet", "-e", fx("fail.env.example"), "-f", fx("fail.env"))
	if code != 1 || out != "" {
		t.Errorf("fail: %d %q", code, out)
	}
	code, out, errOut = run(t, "-q", "-e", fx("ok.env.example"), "-f", fx("nope.env"))
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "envdiff: ") {
		t.Errorf("error: %d %q %q", code, out, errOut)
	}
}

func TestMissingFile(t *testing.T) {
	missing := fx("does-not-exist.env")
	code, out, errOut := run(t, "-e", fx("ok.env.example"), "-f", missing)
	if code != 2 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	want := "envdiff: cannot read " + missing + ": "
	if !strings.HasPrefix(errOut, want) {
		t.Errorf("stderr = %q, want prefix %q", errOut, want)
	}
	if strings.Count(errOut, missing) != 1 {
		t.Errorf("path repeated: %q", errOut)
	}
	// The example file is checked first.
	_, _, errOut = run(t, "-e", fx("nope1.env"), "-f", fx("nope2.env"))
	if !strings.Contains(errOut, "nope1.env") || strings.Contains(errOut, "nope2.env") {
		t.Errorf("only the first problem should be reported: %q", errOut)
	}
}

func TestDirectoryAsFile(t *testing.T) {
	code, out, errOut := run(t, "-e", t.TempDir(), "-f", fx("ok.env"))
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "envdiff: cannot read ") {
		t.Errorf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestSyntaxErrors(t *testing.T) {
	cases := []struct{ file, msg string }{
		{"bad.env", ":2: "},
		{"bad_unclosed.env", ":2: "},
	}
	for _, c := range cases {
		code, out, errOut := run(t, "-e", fx("ok.env.example"), "-f", fx(c.file))
		if code != 2 || out != "" {
			t.Errorf("%s: code=%d out=%q", c.file, code, out)
		}
		if !strings.HasPrefix(errOut, "envdiff: "+fx(c.file)+c.msg) {
			t.Errorf("%s: stderr = %q", c.file, errOut)
		}
		if strings.Count(errOut, "\n") != 1 {
			t.Errorf("%s: expected a single line: %q", c.file, errOut)
		}
	}
	// Syntax error in the example file is reported before the env file.
	_, _, errOut := run(t, "-e", fx("bad.env"), "-f", fx("bad_unclosed.env"))
	if !strings.Contains(errOut, "bad.env:") || strings.Contains(errOut, "bad_unclosed") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestInvalidFormat(t *testing.T) {
	code, out, errOut := run(t, "--format", "xml", "-e", fx("ok.env.example"), "-f", fx("ok.env"))
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "envdiff: ") {
		t.Errorf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestInvalidGlob(t *testing.T) {
	for _, g := range []string{"[", "[a-", `\`, "a[", "[]a]x["} {
		code, out, errOut := run(t, "--ignore", g, "-e", fx("ok.env.example"), "-f", fx("ok.env"))
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "envdiff: ") {
			t.Errorf("glob %q: code=%d out=%q err=%q", g, code, out, errOut)
		}
	}
	for _, g := range []string{"LOCAL_*", "A?B", "[A-C]*", `\*`} {
		code, _, errOut := run(t, "--ignore", g, "-e", fx("ok.env.example"), "-f", fx("ok.env"))
		if code != 0 {
			t.Errorf("glob %q rejected: %d %q", g, code, errOut)
		}
	}
}

func TestIgnore(t *testing.T) {
	args := []string{"-e", fx("fail.env.example"), "-f", fx("fail.env")}
	if code, _, _ := run(t, args...); code != 1 {
		t.Fatalf("baseline code = %d", code)
	}
	args = append(args, "--ignore", "API_*", "--ignore", "REDIS_URL", "--ignore", "LOCAL_DEBUG",
		"--ignore", "SMTP_*", "--ignore", "STRAY_*")
	code, out, errOut := run(t, args...)
	if code != 0 {
		t.Errorf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCheckEmpty(t *testing.T) {
	args := []string{"-e", fx("ok.env.example"), "-f", fx("empty.env")}
	if code, _, _ := run(t, args...); code != 0 {
		t.Errorf("without flag: code = %d", code)
	}
	code, out, _ := run(t, append(args, "--check-empty")...)
	if code != 1 || !strings.Contains(out, "APP_ENV") || !strings.Contains(out, "API_TOKEN") {
		t.Errorf("with flag: code=%d out=%q", code, out)
	}
}

func TestVersion(t *testing.T) {
	code, out, errOut := run(t, "--version")
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if !strings.HasPrefix(out, "envdiff version ") || !strings.Contains(out, "commit") || !strings.Contains(out, "built") {
		t.Errorf("stdout = %q", out)
	}
	// --quiet only affects the report.
	_, out, _ = run(t, "--version", "-q")
	if out == "" {
		t.Error("--version must print even with --quiet")
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--bogus"},
		{"-z"},
		{"positional"},
		{"--format"},
		{"-e", fx("ok.env.example"), "-f", fx("ok.env"), "extra"},
	} {
		code, out, errOut := run(t, args...)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "envdiff: ") {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errOut)
		}
	}
}

func TestJSON(t *testing.T) {
	code, out, _ := run(t, "--format", "json", "-e", fx("fail.env.example"), "-f", fx("fail.env"))
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	for _, k := range []string{"example", "env", "missing", "extra", "empty", "duplicates", "key_count", "ok"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing JSON key %q", k)
		}
	}
	if string(m["ok"]) != "false" {
		t.Errorf("ok = %s", m["ok"])
	}
}

func TestGitHub(t *testing.T) {
	code, out, _ := run(t, "--format", "github", "-e", fx("fail.env.example"), "-f", fx("fail.env"))
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines: %q", len(lines), out)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "::error ") && !strings.HasPrefix(l, "::warning ") {
			t.Errorf("unexpected line %q", l)
		}
	}
}

func TestNoColorOnBuffer(t *testing.T) {
	_, out, _ := run(t, "-e", fx("fail.env.example"), "-f", fx("fail.env"))
	if strings.Contains(out, "\x1b") {
		t.Errorf("color on a buffer: %q", out)
	}
}

func TestRunIsRepeatable(t *testing.T) {
	// No state may leak between calls.
	for i := 0; i < 3; i++ {
		if code, _, _ := run(t, "-e", fx("fail.env.example"), "-f", fx("fail.env")); code != 1 {
			t.Fatalf("iteration %d: code = %d", i, code)
		}
		if code, _, _ := run(t, "-e", fx("ok.env.example"), "-f", fx("ok.env")); code != 0 {
			t.Fatalf("iteration %d: code = %d", i, code)
		}
	}
}

// TestNeverPrintsValues runs every fixture, format and flag combination, plus
// the error paths, and asserts that no value ever reaches stdout or stderr.
func TestNeverPrintsValues(t *testing.T) {
	entries, err := os.ReadDir(fx(""))
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := fx(e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), sentinel) {
			t.Errorf("fixture %s does not contain the sentinel; the test would be vacuous", e.Name())
		}
		files = append(files, p)
	}
	if len(files) < 8 {
		t.Fatalf("expected at least 8 fixtures, found %d", len(files))
	}

	check := func(args []string) {
		t.Helper()
		_, out, errOut := run(t, args...)
		if strings.Contains(out, sentinel) || strings.Contains(errOut, sentinel) {
			t.Errorf("value leaked with args %v\nstdout: %s\nstderr: %s", args, out, errOut)
		}
	}

	extras := [][]string{nil, {"--check-empty"}, {"--strict"}, {"--check-empty", "--strict", "--ignore", "LOCAL_*"}}
	for _, format := range []string{"text", "json", "github"} {
		for _, extra := range extras {
			for _, a := range files {
				for _, b := range files {
					args := append([]string{"--format", format, "-e", a, "-f", b}, extra...)
					check(args)
				}
			}
		}
	}

	ok := []string{"-e", fx("ok.env.example"), "-f", fx("ok.env")}
	for _, args := range [][]string{
		{"-e", fx("ok.env.example"), "-f", fx("missing.env")},
		{"-e", fx("ok.env.example"), "-f", fx("bad.env")},
		{"-e", fx("ok.env.example"), "-f", fx("bad_unclosed.env")},
		{"-e", fx("bad_unclosed.env"), "-f", fx("ok.env")},
		append([]string{"--format", "nope"}, ok...),
		append([]string{"--ignore", "["}, ok...),
		append([]string{"--bogus"}, ok...),
		append([]string{"-q"}, ok...),
		{"--version"},
	} {
		check(args)
	}
}
