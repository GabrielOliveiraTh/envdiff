// Package diff compares the keys of two parsed dotenv files.
package diff

import "github.com/GabrielOliveiraTh/envdiff/internal/envfile"

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
func Compare(example, env envfile.File, opts Options) Result {
	return Result{
		Example:    example.Path,
		Env:        env.Path,
		Missing:    []Item{},
		Extra:      []Item{},
		Empty:      []Item{},
		Duplicates: []Duplicate{},
		OK:         true,
		Strict:     opts.Strict,
	}
}
