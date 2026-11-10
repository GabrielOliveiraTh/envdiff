package diff

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/GabrielOliveiraTh/envdiff/internal/envfile"
)

type kv struct {
	k     string
	empty bool
}

func file(path string, entries ...kv) envfile.File {
	f := envfile.File{Path: path, Duplicates: map[string][]int{}}
	for i, e := range entries {
		f.Entries = append(f.Entries, envfile.Entry{Key: e.k, Line: i + 1, Empty: e.empty})
	}
	return f
}

func keys(ks ...string) []kv {
	var out []kv
	for _, k := range ks {
		out = append(out, kv{k: k})
	}
	return out
}

func items(r []Item) []string {
	out := []string{}
	for _, i := range r {
		out = append(out, i.Key)
	}
	return out
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name    string
		ex, env envfile.File
		opts    Options
		missing []string
		extra   []string
		empty   []string
		dups    int
		count   int
		ok      bool
	}{
		{"identical", file("a", keys("A", "B")...), file("b", keys("B", "A")...), Options{}, nil, nil, nil, 0, 2, true},
		{"only missing", file("a", keys("A", "B", "C")...), file("b", keys("A")...), Options{}, []string{"B", "C"}, nil, nil, 0, 3, false},
		{"only extra", file("a", keys("A")...), file("b", keys("A", "Z", "X")...), Options{}, nil, []string{"X", "Z"}, nil, 0, 1, true},
		{"only extra strict", file("a", keys("A")...), file("b", keys("A", "X")...), Options{Strict: true}, nil, []string{"X"}, nil, 0, 1, false},
		{"both", file("a", keys("A", "B")...), file("b", keys("B", "C")...), Options{}, []string{"A"}, []string{"C"}, nil, 0, 2, false},
		{"ignore glob all categories", file("a", keys("A", "LOCAL_X", "LOCAL_Y")...), file("b", keys("A", "LOCAL_Z")...), Options{Ignore: []string{"LOCAL_*"}}, nil, nil, nil, 0, 1, true},
		{"empty off", file("a", keys("A")...), file("b", kv{"A", true}), Options{}, nil, nil, nil, 0, 1, true},
		{"empty on", file("a", keys("A", "B")...), file("b", kv{"A", true}, kv{"B", false}), Options{CheckEmpty: true}, nil, nil, []string{"A"}, 0, 2, false},
		{"missing not empty", file("a", keys("A")...), file("b"), Options{CheckEmpty: true}, []string{"A"}, nil, nil, 0, 1, false},
		{"empty ignored", file("a", keys("A")...), file("b", kv{"A", true}), Options{CheckEmpty: true, Ignore: []string{"A"}}, nil, nil, nil, 0, 0, true},
		{"empty reference", file("a"), file("b", keys("A")...), Options{}, nil, []string{"A"}, nil, 0, 0, true},
		{"empty env", file("a", keys("A")...), file("b"), Options{}, []string{"A"}, nil, nil, 0, 1, false},
		{"both empty", file("a"), file("b"), Options{}, nil, nil, nil, 0, 0, true},
		{"malformed glob", file("a", keys("A")...), file("b", keys("A")...), Options{Ignore: []string{"[", "B"}}, nil, nil, nil, 0, 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Compare(tc.ex, tc.env, tc.opts)
			for _, c := range []struct {
				n    string
				got  []Item
				want []string
			}{{"missing", r.Missing, tc.missing}, {"extra", r.Extra, tc.extra}, {"empty", r.Empty, tc.empty}} {
				w := c.want
				if w == nil {
					w = []string{}
				}
				if got := items(c.got); !reflect.DeepEqual(got, w) {
					t.Errorf("%s = %v, want %v", c.n, got, w)
				}
			}
			if len(r.Duplicates) != tc.dups {
				t.Errorf("dups = %d, want %d", len(r.Duplicates), tc.dups)
			}
			if r.KeyCount != tc.count {
				t.Errorf("KeyCount = %d, want %d", r.KeyCount, tc.count)
			}
			if r.OK != tc.ok {
				t.Errorf("OK = %v, want %v", r.OK, tc.ok)
			}
			if r.Strict != tc.opts.Strict || r.Example != tc.ex.Path || r.Env != tc.env.Path {
				t.Errorf("metadata not copied: %+v", r)
			}
		})
	}
}

func TestLines(t *testing.T) {
	ex := file("a", keys("A")...)
	ex.Entries = []envfile.Entry{{Key: "A", Line: 7}, {Key: "B", Line: 3}}
	env := file("b", keys("A", "B")...)
	env.Entries[0].Line = 10
	r := Compare(ex, file("b", keys("Z")...), Options{})
	if r.Missing[0] != (Item{"A", 7}) || r.Missing[1] != (Item{"B", 3}) {
		t.Errorf("missing lines: %+v", r.Missing)
	}
	if r.Extra[0] != (Item{"Z", 1}) {
		t.Errorf("extra lines: %+v", r.Extra)
	}
	r = Compare(ex, env, Options{CheckEmpty: true})
	if len(r.Empty) != 0 {
		t.Errorf("unexpected empty: %+v", r.Empty)
	}
	env.Entries[0].Empty = true
	r = Compare(ex, env, Options{CheckEmpty: true})
	if len(r.Empty) != 1 || r.Empty[0] != (Item{"A", 10}) {
		t.Errorf("empty line should come from env: %+v", r.Empty)
	}
}

func TestDuplicates(t *testing.T) {
	ex := file("ex", keys("A", "B")...)
	ex.Duplicates = map[string][]int{"B": {2, 1}, "A": {1, 4}}
	env := file("env", keys("A", "B", "LOCAL_D")...)
	env.Duplicates = map[string][]int{"Z": {5, 6}, "C": {2, 3}, "LOCAL_D": {1, 2}}
	want := []Duplicate{
		{"ex", "A", []int{1, 4}},
		{"ex", "B", []int{1, 2}},
		{"env", "C", []int{2, 3}},
		{"env", "Z", []int{5, 6}},
	}
	opts := Options{Ignore: []string{"LOCAL_*"}}

	r := Compare(ex, env, opts)
	if !reflect.DeepEqual(r.Duplicates, want) {
		t.Errorf("duplicates = %+v, want %+v", r.Duplicates, want)
	}
	if !r.OK {
		t.Error("duplicates must not fail without strict")
	}
	if !reflect.DeepEqual(ex.Duplicates["B"], []int{2, 1}) {
		t.Error("input duplicate lines mutated")
	}

	opts.Strict = true
	if Compare(ex, env, opts).OK {
		t.Error("duplicates must fail with strict")
	}
}

func TestDeterministicOrdering(t *testing.T) {
	a := Compare(file("a", keys("C", "A", "B")...), file("b", keys("Z", "Y", "X")...), Options{})
	b := Compare(file("a", keys("B", "C", "A")...), file("b", keys("X", "Z", "Y")...), Options{})
	if !reflect.DeepEqual(items(a.Missing), items(b.Missing)) || !reflect.DeepEqual(items(a.Extra), items(b.Extra)) {
		t.Error("ordering depends on input order")
	}
	if got := items(a.Missing); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("missing not sorted: %v", got)
	}
	if got := items(a.Extra); !reflect.DeepEqual(got, []string{"X", "Y", "Z"}) {
		t.Errorf("extra not sorted: %v", got)
	}
}

func TestNonNilLists(t *testing.T) {
	r := Compare(file("a"), file("b"), Options{})
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "null") {
		t.Errorf("json contains null: %s", s)
	}
	for _, k := range []string{"missing", "extra", "empty", "duplicates"} {
		if !strings.Contains(s, `"`+k+`":[]`) {
			t.Errorf("%s not an empty array: %s", k, s)
		}
	}
	if strings.Contains(s, "strict") {
		t.Errorf("strict must not be serialized: %s", s)
	}
}

func TestInputsNotMutated(t *testing.T) {
	ex := file("a", keys("C", "A", "B")...)
	env := file("b", keys("Z", "Y")...)
	ex.Duplicates["A"] = []int{3, 1}
	exCopy := file("a", keys("C", "A", "B")...)
	exCopy.Duplicates["A"] = []int{3, 1}
	envCopy := file("b", keys("Z", "Y")...)
	Compare(ex, env, Options{CheckEmpty: true, Ignore: []string{"B"}})
	if !reflect.DeepEqual(ex, exCopy) || !reflect.DeepEqual(env, envCopy) {
		t.Error("inputs were mutated")
	}
}

func TestMalformedGlobNeverPanics(t *testing.T) {
	r := Compare(file("a", keys("A")...), file("b"), Options{Ignore: []string{"[a-", "\\"}})
	if len(r.Missing) != 1 {
		t.Errorf("malformed glob must not match: %+v", r.Missing)
	}
}
