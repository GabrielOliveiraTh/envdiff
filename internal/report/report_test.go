package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
)

var update = flag.Bool("update", false, "update golden files")

type scenario struct {
	name string
	res  diff.Result
}

func base() diff.Result {
	return diff.Result{
		Example: ".env.example", Env: ".env", KeyCount: 12,
		Missing: []diff.Item{}, Extra: []diff.Item{}, Empty: []diff.Item{},
		Duplicates: []diff.Duplicate{},
	}
}

func scenarios() []scenario {
	clean := base()
	clean.OK = true

	failing := base()
	failing.Missing = []diff.Item{{Key: "DATABASE_URL", Line: 3}, {Key: "REDIS_URL", Line: 8}}
	failing.Extra = []diff.Item{{Key: "OLD_FLAG", Line: 7}}

	all := failing
	all.Empty = []diff.Item{{Key: "API_KEY", Line: 2}}
	all.Duplicates = []diff.Duplicate{{File: ".env", Key: "PORT", Lines: []int{3, 9}}}

	warn := base()
	warn.OK = true
	warn.Extra = []diff.Item{{Key: "OLD_FLAG", Line: 7}}
	warn.Duplicates = []diff.Duplicate{{File: ".env", Key: "PORT", Lines: []int{3, 9}}}

	strict := warn
	strict.OK = false
	strict.Strict = true

	return []scenario{
		{"clean", clean}, {"failing", failing}, {"everything", all},
		{"warnings", warn}, {"strict", strict},
	}
}

func render(t *testing.T, format string, color bool, r diff.Result) string {
	t.Helper()
	rep, err := New(format, Options{Color: color})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := rep.Report(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w := strings.ReplaceAll(string(want), "\r\n", "\n")
	if got != w {
		t.Errorf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, w)
	}
}

func TestGolden(t *testing.T) {
	for _, sc := range scenarios() {
		t.Run("text/"+sc.name, func(t *testing.T) {
			golden(t, "text_"+sc.name, render(t, "text", false, sc.res))
		})
		t.Run("text_color/"+sc.name, func(t *testing.T) {
			golden(t, "text_color_"+sc.name, render(t, "text", true, sc.res))
		})
		t.Run("json/"+sc.name, func(t *testing.T) {
			golden(t, "json_"+sc.name, render(t, "json", false, sc.res))
		})
		t.Run("github/"+sc.name, func(t *testing.T) {
			golden(t, "github_"+sc.name, render(t, "github", false, sc.res))
		})
	}
}

func TestNoANSIWithoutColor(t *testing.T) {
	for _, sc := range scenarios() {
		for _, f := range []string{"text", "json", "github"} {
			if strings.Contains(render(t, f, false, sc.res), "\x1b") {
				t.Errorf("%s/%s: unexpected escape code", f, sc.name)
			}
		}
	}
	// Color is ignored by json and github.
	for _, f := range []string{"json", "github"} {
		if strings.Contains(render(t, f, true, scenarios()[2].res), "\x1b") {
			t.Errorf("%s: escape code with Color=true", f)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	for _, sc := range scenarios() {
		t.Run(sc.name, func(t *testing.T) {
			var got diff.Result
			if err := jsonUnmarshal(render(t, "json", false, sc.res), &got); err != nil {
				t.Fatal(err)
			}
			want := sc.res
			want.Strict = false
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip mismatch:\ngot  %+v\nwant %+v", got, want)
			}
		})
	}
}

func TestJSONEmptyListsAndNoHTMLEscape(t *testing.T) {
	r := base()
	r.OK = true
	out := render(t, "json", false, r)
	if strings.Contains(out, "null") {
		t.Errorf("found null in %s", out)
	}
	if !strings.Contains(out, `"missing": []`) || !strings.HasSuffix(out, "}\n") {
		t.Errorf("unexpected output %s", out)
	}
	r.Env = "<a&b>"
	if out := render(t, "json", false, r); !strings.Contains(out, "<a&b>") {
		t.Errorf("HTML escaped: %s", out)
	}
}

func TestUnknownFormat(t *testing.T) {
	_, err := New("xml", Options{})
	if err == nil || err.Error() != `unknown format "xml" (valid: text, json, github)` {
		t.Fatalf("err = %v", err)
	}
}

type failWriter struct{}

var errWrite = errors.New("boom")

func (failWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestWriterErrorPropagation(t *testing.T) {
	for _, f := range []string{"text", "json", "github"} {
		for _, sc := range scenarios() {
			if f == "github" && sc.name == "clean" {
				continue // writes nothing
			}
			rep, _ := New(f, Options{})
			if err := rep.Report(failWriter{}, sc.res); !errors.Is(err, errWrite) {
				t.Errorf("%s/%s: err = %v", f, sc.name, err)
			}
		}
	}
}

func TestGithubNoFindings(t *testing.T) {
	if out := render(t, "github", false, scenarios()[0].res); out != "" {
		t.Errorf("output = %q", out)
	}
}

func TestGithubStrictUsesError(t *testing.T) {
	out := render(t, "github", false, scenarios()[4].res)
	if strings.Contains(out, "::warning") || !strings.Contains(out, "::error file=.env,line=7::Extra key") {
		t.Errorf("output = %q", out)
	}
}

func TestGithubEscaping(t *testing.T) {
	if got := escapeData("a%b\r\nc:d,e"); got != "a%25b%0D%0Ac:d,e" {
		t.Errorf("escapeData = %q", got)
	}
	if got := escapeProperty("a%b\r\nc:d,e"); got != "a%25b%0D%0Ac%3Ad%2Ce" {
		t.Errorf("escapeProperty = %q", got)
	}
	r := base()
	r.Example = "dir,x:y/.env.example"
	r.Missing = []diff.Item{{Key: "K", Line: 1}}
	if out := render(t, "github", false, r); !strings.HasPrefix(out, "::error file=dir%2Cx%3Ay/.env.example,line=1::") {
		t.Errorf("output = %q", out)
	}
}

func TestDuplicateWithoutLines(t *testing.T) {
	r := base()
	r.Duplicates = []diff.Duplicate{{File: ".env", Key: "K"}}
	if out := render(t, "github", false, r); !strings.Contains(out, "line=0::Duplicate key K (lines )") {
		t.Errorf("output = %q", out)
	}
}

func TestFailWithoutFindings(t *testing.T) {
	out := render(t, "text", false, base())
	if !strings.HasSuffix(out, "Result: FAIL\n") {
		t.Errorf("output = %q", out)
	}
}

func TestTextAlignment(t *testing.T) {
	r := base()
	r.Missing = []diff.Item{{Key: "A", Line: 1}, {Key: "LONG_KEY_NAME", Line: 22}}
	for _, color := range []bool{false, true} {
		out := render(t, "text", color, r)
		lines := strings.Split(out, "\n")
		var cols []int
		for _, l := range lines {
			if strings.HasPrefix(l, "  - ") {
				cols = append(cols, strings.Index(l, "("))
			}
		}
		if len(cols) != 2 || cols[0] != cols[1] || cols[0] != len("  - LONG_KEY_NAME  ") {
			t.Errorf("color=%v: columns = %v\n%s", color, cols, out)
		}
	}
}

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }
