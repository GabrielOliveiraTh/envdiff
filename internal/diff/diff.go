// Package diff compares the keys of two parsed dotenv files.
package diff

import (
	"path"
	"sort"

	"github.com/GabrielOliveiraTh/envdiff/internal/envfile"
)

// Item is a key together with the line where it appears.
type Item struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
}

// Duplicate is a key defined more than once in a single file.
type Duplicate struct {
	File  string `json:"file"`
	Key   string `json:"key"`
	Lines []int  `json:"lines"`
}

// Options tunes Compare.
type Options struct {
	Ignore     []string // already-validated path.Match globs
	CheckEmpty bool
	Strict     bool
}

// Result is the outcome of a comparison. It is also the public JSON schema.
type Result struct {
	Example    string      `json:"example"`
	Env        string      `json:"env"`
	Missing    []Item      `json:"missing"` // in reference, absent from env; Line is in the reference
	Extra      []Item      `json:"extra"`   // in env, absent from reference; Line is in env
	Empty      []Item      `json:"empty"`   // Line is in env
	Duplicates []Duplicate `json:"duplicates"`
	KeyCount   int         `json:"key_count"` // reference keys after ignore
	OK         bool        `json:"ok"`
	Strict     bool        `json:"-"` // used by reporters; not serialized
}

// Compare diffs the keys of example (reference) against env.
//
// Ignore globs apply to every category and to KeyCount. A key that is missing
// from env is never also reported as empty. Missing and empty keys always make
// the result fail; extra keys and duplicates only fail when opts.Strict is set.
// The inputs are never modified and no values are ever handled.
func Compare(example, env envfile.File, opts Options) Result {
	ignored := func(key string) bool {
		for _, g := range opts.Ignore {
			if ok, err := path.Match(g, key); err == nil && ok {
				return true
			}
		}
		return false
	}

	envIdx := make(map[string]envfile.Entry, len(env.Entries))
	for _, e := range env.Entries {
		if !ignored(e.Key) {
			envIdx[e.Key] = e
		}
	}

	res := Result{
		Example:    example.Path,
		Env:        env.Path,
		Missing:    []Item{},
		Extra:      []Item{},
		Empty:      []Item{},
		Duplicates: []Duplicate{},
		Strict:     opts.Strict,
	}

	refKeys := make(map[string]struct{}, len(example.Entries))
	for _, e := range example.Entries {
		if ignored(e.Key) {
			continue
		}
		refKeys[e.Key] = struct{}{}
		res.KeyCount++
		got, ok := envIdx[e.Key]
		switch {
		case !ok:
			res.Missing = append(res.Missing, Item{Key: e.Key, Line: e.Line})
		case opts.CheckEmpty && got.Empty:
			res.Empty = append(res.Empty, Item{Key: got.Key, Line: got.Line})
		}
	}
	for _, e := range envIdx {
		if _, ok := refKeys[e.Key]; !ok {
			res.Extra = append(res.Extra, Item{Key: e.Key, Line: e.Line})
		}
	}

	sortItems(res.Missing)
	sortItems(res.Extra)
	sortItems(res.Empty)

	res.Duplicates = append(res.Duplicates, duplicates(example, ignored)...)
	res.Duplicates = append(res.Duplicates, duplicates(env, ignored)...)

	fail := len(res.Missing) > 0 || len(res.Empty) > 0
	if opts.Strict && (len(res.Extra) > 0 || len(res.Duplicates) > 0) {
		fail = true
	}
	res.OK = !fail
	return res
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
}

// duplicates lists the repeated, non-ignored keys of f sorted by key.
func duplicates(f envfile.File, ignored func(string) bool) []Duplicate {
	var out []Duplicate
	for key, lines := range f.Duplicates {
		if ignored(key) {
			continue
		}
		ls := append([]int{}, lines...)
		sort.Ints(ls)
		out = append(out, Duplicate{File: f.Path, Key: key, Lines: ls})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
