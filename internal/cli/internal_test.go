package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	got := formatVersion("1.2.3", "abc123", "2026-01-02")
	want := "envdiff version 1.2.3 (commit abc123, built 2026-01-02)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildInfoUsesInjectedValues(t *testing.T) {
	oldV, oldC, oldD := version, commit, date
	defer func() { version, commit, date = oldV, oldC, oldD }()
	version, commit, date = "9.9.9", "deadbeef", "today"
	v, c, d := buildInfo()
	if v != "9.9.9" || c != "deadbeef" || d != "today" {
		t.Errorf("got %q %q %q", v, c, d)
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"--version"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if want := "envdiff version 9.9.9 (commit deadbeef, built today)\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestValidateGlob(t *testing.T) {
	for _, g := range []string{"[", "[a-", `\`, "a[", "[]"} {
		if validateGlob(g) == nil {
			t.Errorf("%q accepted", g)
		}
	}
	for _, g := range []string{"", "*", "LOCAL_*", "A?", "[A-Z]*", `\*`} {
		if err := validateGlob(g); err != nil {
			t.Errorf("%q rejected: %v", g, err)
		}
	}
}

type statWriter struct {
	bytes.Buffer
	mode os.FileMode
	err  error
}

type fakeInfo struct {
	os.FileInfo
	mode os.FileMode
}

func (f fakeInfo) Mode() os.FileMode { return f.mode }

func (s *statWriter) Stat() (os.FileInfo, error) {
	return fakeInfo{mode: s.mode}, s.err
}

func TestUseColor(t *testing.T) {
	term := &statWriter{mode: os.ModeDevice | os.ModeCharDevice}
	file := &statWriter{}
	base := options{format: "text"}

	t.Setenv("NO_COLOR", "")
	if !useColor(base, term) {
		t.Error("terminal should get color")
	}
	if useColor(base, file) {
		t.Error("regular file should not get color")
	}
	if useColor(base, &bytes.Buffer{}) {
		t.Error("buffer should not get color")
	}
	if useColor(options{format: "json"}, term) {
		t.Error("json should not get color")
	}
	if useColor(options{format: "text", noColor: true}, term) {
		t.Error("--no-color should disable color")
	}
	if useColor(options{format: "text", quiet: true}, term) {
		t.Error("--quiet should disable color")
	}
	if useColor(base, &statWriter{mode: os.ModeCharDevice, err: os.ErrInvalid}) {
		t.Error("stat error should disable color")
	}
	t.Setenv("NO_COLOR", "1")
	if useColor(base, term) {
		t.Error("NO_COLOR should disable color")
	}
}

func TestReadErrorWithoutPathError(t *testing.T) {
	if err := readError("x", os.ErrInvalid); err == nil || err.Error() != "cannot read x: invalid argument" {
		t.Errorf("got %v", err)
	}
}

func TestLoadUnreadable(t *testing.T) {
	if _, err := load(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected error")
	}
}
