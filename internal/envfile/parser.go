// Package envfile parses dotenv files, keeping only key names and line
// numbers. Values are never stored.
package envfile

import (
	"fmt"
	"io"
)

// Entry is a single key found in a dotenv file.
type Entry struct {
	Key   string
	Line  int  // 1-based line where the key starts
	Empty bool // value is empty after parsing; the value itself is NOT stored
}

// File is the parsed form of a dotenv file.
type File struct {
	Path       string
	Entries    []Entry          // file order; for duplicate keys the last occurrence wins
	Duplicates map[string][]int // key -> every line where it appears (repeated keys only)
}

// ParseError describes a syntax error. Msg never contains variable values.
type ParseError struct {
	Path string
	Line int
	Msg  string
}

// Error implements the error interface as "<path>:<line>: <msg>".
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Msg)
}

// Parse reads a dotenv file from r. The path is used only for reporting.
func Parse(path string, r io.Reader) (File, error) {
	return File{Path: path, Duplicates: map[string][]int{}}, nil
}
