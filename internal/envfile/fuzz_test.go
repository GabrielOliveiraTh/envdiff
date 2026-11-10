package envfile

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

var fuzzKeyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)

// FuzzParse checks that Parse never panics and keeps its invariants.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"", "\n", "A=b", "A=\n", "\xef\xbb\xbfA=1", "A=1\r\nB=2\r\n",
		"export A=1", "export", "export ", "export=1", "export =1", "exported=1",
		"\t A \t=\t b", "A=b #c", "A=b#c", "A=   # c", "=", "=x", "A", "1A=x", "A-B=x",
		"A='x'", "A=\"x\"", "A=''", "A=\"\"", "A='l1\nl2'", "A=\"l1\nl2\"",
		`A="a\nb\r\t\"\\"`, `A="a\qb"`, `A="x\`, `A="x\"`, "A='x", "A=\"x",
		"A=\"x\" y", "A='x'#c", "A=\"x\" # c", "A=${B}", "A=1\nA=2\nA=",
		"A=\xff\xfe", "# only comment", "A=\"\\\n\"", "\r", "A=\r", "'", "\"",
		"export  \t=", "S3CR3T_SENTINEL", "A=\"S3CR3T_SENTINEL\" S3CR3T_SENTINEL",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		file, err := Parse("f.env", strings.NewReader(in))
		if file.Duplicates == nil || file.Entries == nil {
			t.Fatal("nil Entries or Duplicates")
		}
		seen := map[string]bool{}
		prev := 0
		for _, e := range file.Entries {
			if !fuzzKeyRE.MatchString(e.Key) {
				t.Fatalf("bad key %q", e.Key)
			}
			if e.Line < 1 || seen[e.Key] {
				t.Fatalf("bad entry %+v", e)
			}
			if e.Line <= prev {
				t.Fatalf("entries not in file order: %+v", e)
			}
			prev = e.Line
			seen[e.Key] = true
		}
		for k, ls := range file.Duplicates {
			if len(ls) < 2 || !seen[k] {
				t.Fatalf("bad duplicates for %q: %v", k, ls)
			}
		}
		if err != nil {
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("unexpected non-ParseError: %v", err)
			}
			if pe.Line < 1 {
				t.Fatalf("bad line %d", pe.Line)
			}
			// Messages are fixed strings; they must never carry input text.
			switch pe.Msg {
			case "expected KEY=VALUE", "invalid key", "unclosed quote", "unexpected text after closing quote":
			default:
				t.Fatalf("suspicious message %q", pe.Msg)
			}
		}
	})
}
