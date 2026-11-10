package envfile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParseValid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entry
		dups map[string][]int
	}{
		{"empty input", "", []Entry{}, nil},
		{"simple", "A=b", []Entry{{"A", 1, false}}, nil},
		{"BOM", "\xef\xbb\xbfA=b\n", []Entry{{"A", 1, false}}, nil},
		{"CRLF", "A=b\r\nC=d\r\n", []Entry{{"A", 1, false}, {"C", 2, false}}, nil},
		{"CRLF empty and quoted", "A=\r\nB=\"x\"\r\nC='y' # c\r\n", []Entry{{"A", 1, true}, {"B", 2, false}, {"C", 3, false}}, nil},
		{"blank and comments", "\n  \n# c\n   # d\n\t#e\nA=1\n", []Entry{{"A", 6, false}}, nil},
		{"export prefix", "export A=1\nexport \t B=2", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"export key no space", "export=1\nexported=2\n", []Entry{{"export", 1, false}, {"exported", 2, false}}, nil},
		{"export key empty value", "export  =", []Entry{{"export", 1, true}}, nil},
		{"export key with space before equals", "export =1", []Entry{{"export", 1, false}}, nil},
		{"tabs and spaces", "\t A \t=\t b\n", []Entry{{"A", 1, false}}, nil},
		{"key charset", "_a.b1=x\nA.B=y", []Entry{{"_a.b1", 1, false}, {"A.B", 2, false}}, nil},
		{"inline comment", "A=b #c\nB=b\t#c", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"hash without space is value", "A=b#c\nB=#c", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"comment only value is empty", "A=   # comment\nB= #x", []Entry{{"A", 1, true}, {"B", 2, true}}, nil},
		{"empty variants", "A=\nB=\"\"\nC=''\nD=  \nE=\"\" # c", []Entry{{"A", 1, true}, {"B", 2, true}, {"C", 3, true}, {"D", 4, true}, {"E", 5, true}}, nil},
		{"not empty quoted space", "A=\" \"\nB=' '", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"single quoted literal", `A='a\'`, []Entry{{"A", 1, false}}, nil},
		{"single quote hash and equals", "A='x # y = z'", []Entry{{"A", 1, false}}, nil},
		{"single multiline", "A='l1\nl2\n'\nB=1", []Entry{{"A", 1, false}, {"B", 4, false}}, nil},
		{"double multiline", "A=\"l1\nl2\"\nB=1", []Entry{{"A", 1, false}, {"B", 3, false}}, nil},
		{"double escapes", `A="a\nb\r\t\"\\"` + "\nB=1", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"unknown escape kept", `A="a\qb"`, []Entry{{"A", 1, false}}, nil},
		{"escaped newline in double", "A=\"a\\\nb\"\nB=1", []Entry{{"A", 1, false}, {"B", 3, false}}, nil},
		{"double quote inside single", `A='say "hi"'`, []Entry{{"A", 1, false}}, nil},
		{"after quote comment", "A=\"x\"   # c\nB='y'\t#c", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"interpolation is plain", "A=${B}\nB=\"${A}\"", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"equals in value", "A=b=c", []Entry{{"A", 1, false}}, nil},
		{"invalid utf8 value", "A=\xff\xfe\nB=\"\xc3\"", []Entry{{"A", 1, false}, {"B", 2, false}}, nil},
		{"no trailing newline", "A=1\nB=", []Entry{{"A", 1, false}, {"B", 2, true}}, nil},
		{
			"duplicates last wins",
			"A=1\nB=2\nA=\nC=3\nA=4\n",
			[]Entry{{"B", 2, false}, {"C", 4, false}, {"A", 5, false}},
			map[string][]int{"A": {1, 3, 5}},
		},
		{
			"duplicates last empty",
			"A=1\nA=''",
			[]Entry{{"A", 2, true}},
			map[string][]int{"A": {1, 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse("p.env", strings.NewReader(tt.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f.Path != "p.env" {
				t.Errorf("Path = %q", f.Path)
			}
			if f.Entries == nil || f.Duplicates == nil {
				t.Fatalf("Entries/Duplicates must be non-nil")
			}
			if !reflect.DeepEqual(f.Entries, tt.want) {
				t.Errorf("Entries = %+v, want %+v", f.Entries, tt.want)
			}
			wantDups := tt.dups
			if wantDups == nil {
				wantDups = map[string][]int{}
			}
			if !reflect.DeepEqual(f.Duplicates, wantDups) {
				t.Errorf("Duplicates = %v, want %v", f.Duplicates, wantDups)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		line int
		msg  string
	}{
		{"missing equals", "A=1\nJUSTAKEY\n", 2, "expected KEY=VALUE"},
		{"export without equals", "export FOO", 1, "expected KEY=VALUE"},
		{"bare export", "export", 1, "expected KEY=VALUE"},
		{"lone equals", "=", 1, "invalid key"},
		{"empty key", "=value", 1, "invalid key"},
		{"digit start", "1A=x", 1, "invalid key"},
		{"dash in key", "A-B=x", 1, "invalid key"},
		{"space in key", "A B=x", 1, "invalid key"},
		{"quoted key", "\"A\"=x", 1, "invalid key"},
		{"text after double quote", "A=\"x\" y", 1, "unexpected text after closing quote"},
		{"text after single quote", "A='x'y", 1, "unexpected text after closing quote"},
		{"comment glued to quote", "A=\"x\"#c", 1, "unexpected text after closing quote"},
		{"text after multiline quote", "\nA='x\ny' z\n", 2, "unexpected text after closing quote"},
		{"unclosed single", "A=1\nB='x\nC=2\n", 2, "unclosed quote"},
		{"unclosed double", "\n\nA=\"x\nC=2\n", 3, "unclosed quote"},
		{"escaped closing quote unclosed", `A="x\"`, 1, "unclosed quote"},
		{"trailing backslash unclosed", `A="x\`, 1, "unclosed quote"},
		{"error after crlf", "A=1\r\nB\r\n", 2, "expected KEY=VALUE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse("x.env", strings.NewReader(tt.in))
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("err = %v, want *ParseError", err)
			}
			if pe.Line != tt.line || pe.Msg != tt.msg || pe.Path != "x.env" {
				t.Errorf("got %+v, want line %d msg %q", *pe, tt.line, tt.msg)
			}
			if f.Entries == nil || f.Duplicates == nil {
				t.Error("File maps/slices must be non-nil on error")
			}
		})
	}
}

func TestParseErrorFormat(t *testing.T) {
	e := &ParseError{Path: "a/.env", Line: 7, Msg: "invalid key"}
	if got, want := e.Error(), "a/.env:7: invalid key"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	_, err := Parse(".env", strings.NewReader("\nbad line"))
	if err == nil || err.Error() != ".env:2: expected KEY=VALUE" {
		t.Errorf("err = %v", err)
	}
}

func TestParseReadError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Parse("r.env", iotest.ErrReader(boom))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapped boom", err)
	}
	var pe *ParseError
	if errors.As(err, &pe) {
		t.Error("read error must not be a ParseError")
	}
}

func TestParseLongLine(t *testing.T) {
	in := "A=" + strings.Repeat("x", 5<<20) + "\nB=1\n"
	f, err := Parse("big.env", strings.NewReader(in))
	if err != nil || len(f.Entries) != 2 {
		t.Fatalf("entries=%d err=%v", len(f.Entries), err)
	}
}

func TestErrorsNeverLeakValues(t *testing.T) {
	const s = "S3CR3T_SENTINEL"
	inputs := []string{
		"A B=" + s,
		"1A=" + s,
		"A=\"" + s + "\" " + s,
		"A='" + s + "'" + s,
		"A=\"" + s,
		"A='" + s + "\nmore",
		"=" + s,
		s,
		"export " + s,
		"A-B=" + s,
		"A=\"" + s + "\\\"",
	}
	for _, in := range inputs {
		_, err := Parse("f.env", strings.NewReader(in))
		if err == nil {
			t.Errorf("%q: expected error", in)
			continue
		}
		if strings.Contains(err.Error(), s) {
			t.Errorf("%q: error leaks value: %v", in, err)
		}
	}
}
