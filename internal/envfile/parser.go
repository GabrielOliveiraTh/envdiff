// Package envfile parses dotenv files, keeping only key names and line
// numbers. Values are never stored.
package envfile

import (
	"bytes"
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
//
// Behavior notes:
//   - A leading UTF-8 BOM is stripped; "\n" and "\r\n" line endings are accepted.
//   - An optional "export" prefix followed by spaces or tabs is dropped. A key
//     literally named "export" (as in "export=1" or "export =1") is still a key.
//   - Whitespace between the key and "=" is tolerated and dropped.
//   - Quoted values may span lines. Inside double quotes a backslash escapes
//     the next byte (so \" does not close the value); since values are never
//     stored, unknown escapes such as \q are simply kept literally and are not
//     an error. Single quotes have no escapes.
//   - After a closing quote only whitespace and a "#" comment (separated from
//     the quote by whitespace) may follow.
//   - A repeated key yields one Entry carrying the LAST occurrence's Line and
//     Empty, ordered by the position of that last occurrence. Every line of a
//     repeated key is listed (ascending) in Duplicates.
//   - Invalid UTF-8 is processed bytewise and never panics.
//
// Syntax errors are returned as *ParseError and never contain value content.
// A failure of r itself is returned as a plain wrapped error.
func Parse(path string, r io.Reader) (File, error) {
	f := File{Path: path, Entries: []Entry{}, Duplicates: map[string][]int{}}
	data, err := io.ReadAll(r)
	if err != nil {
		return f, fmt.Errorf("reading %s: %w", path, err)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var all []Entry
	lines := map[string][]int{}
	pos, lineNo := 0, 1
	for pos < len(data) {
		e, next, nl, perr := parseStatement(data, pos, lineNo)
		if perr != nil {
			perr.Path = path
			return File{Path: path, Entries: []Entry{}, Duplicates: map[string][]int{}}, perr
		}
		if e.Key != "" {
			all = append(all, e)
			lines[e.Key] = append(lines[e.Key], e.Line)
		}
		pos, lineNo = next, lineNo+nl
	}

	for i, e := range all {
		if ls := lines[e.Key]; len(ls) > 1 {
			f.Duplicates[e.Key] = ls
			if ls[len(ls)-1] != all[i].Line {
				continue // not the last occurrence
			}
		}
		f.Entries = append(f.Entries, e)
	}
	return f, nil
}

func isBlank(c byte) bool { return c == ' ' || c == '\t' }

func isKeyStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func validKey(k string) bool {
	if k == "" || !isKeyStart(k[0]) {
		return false
	}
	for i := 1; i < len(k); i++ {
		c := k[i]
		if !isKeyStart(c) && (c < '0' || c > '9') && c != '.' {
			return false
		}
	}
	return true
}

// trimBlank trims spaces, tabs and carriage returns from both ends.
func trimBlank(b []byte) []byte {
	for len(b) > 0 && (isBlank(b[0]) || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (isBlank(b[len(b)-1]) || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// parseStatement parses the statement starting at data[pos]. It returns the
// entry (zero Key for blank/comment lines), the offset of the next statement,
// and how many newlines were consumed. Error messages never include values.
func parseStatement(data []byte, pos, lineNo int) (Entry, int, int, *ParseError) {
	end := bytes.IndexByte(data[pos:], '\n')
	if end < 0 {
		end = len(data)
	} else {
		end += pos
	}
	next := end + 1
	if end == len(data) {
		next = len(data)
	}
	nl := 0
	if end < len(data) {
		nl = 1
	}
	fail := func(msg string) (Entry, int, int, *ParseError) {
		return Entry{}, 0, 0, &ParseError{Line: lineNo, Msg: msg}
	}

	ls, le := pos, end
	for ls < le && (isBlank(data[ls]) || data[ls] == '\r') {
		ls++
	}
	for le > ls && (isBlank(data[le-1]) || data[le-1] == '\r') {
		le--
	}
	if ls == le || data[ls] == '#' {
		return Entry{}, next, nl, nil
	}
	if le-ls > 6 && string(data[ls:ls+6]) == "export" && isBlank(data[ls+6]) {
		j := ls + 7
		for j < le && isBlank(data[j]) {
			j++
		}
		if j < le && data[j] != '=' {
			ls = j
		}
	}
	eq := bytes.IndexByte(data[ls:le], '=')
	if eq < 0 {
		return fail("expected KEY=VALUE")
	}
	key := string(trimBlank(data[ls : ls+eq]))
	if !validKey(key) {
		return fail("invalid key")
	}
	vs := ls + eq
	vs++
	for vs < end && isBlank(data[vs]) {
		vs++
	}
	e := Entry{Key: key, Line: lineNo}

	if vs < end && (data[vs] == '\'' || data[vs] == '"') {
		q := data[vs]
		i, extra := vs+1, 0
		closed := false
		for i < len(data) {
			c := data[i]
			switch {
			case c == '\n':
				extra++
			case q == '"' && c == '\\' && i+1 < len(data):
				i++
				if data[i] == '\n' {
					extra++
				}
			case c == q:
				closed = true
			}
			if closed {
				break
			}
			i++
		}
		if !closed {
			return fail("unclosed quote")
		}
		e.Empty = i == vs+1
		le := bytes.IndexByte(data[i:], '\n')
		if le < 0 {
			le = len(data)
		} else {
			le += i
		}
		tail := data[i+1 : le]
		trimmed := trimBlank(tail)
		if len(trimmed) > 0 && (trimmed[0] != '#' || len(tail) == 0 || !isBlank(tail[0])) {
			return fail("unexpected text after closing quote")
		}
		nl = extra
		next = le
		if le < len(data) {
			nl++
			next = le + 1
		}
		return e, next, nl, nil
	}

	val := data[vs:end]
	for i := 0; i < len(val); i++ {
		if val[i] == '#' && (i > 0 && isBlank(val[i-1]) || i == 0 && vs > 0 && isBlank(data[vs-1])) {
			val = val[:i]
			break
		}
	}
	e.Empty = len(trimBlank(val)) == 0
	return e, next, nl, nil
}
