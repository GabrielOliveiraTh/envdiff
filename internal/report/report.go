// Package report renders a diff.Result in several output formats.
package report

import (
	"fmt"
	"io"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
)

// Reporter writes a diff.Result to w.
type Reporter interface {
	Report(w io.Writer, r diff.Result) error
}

// Options configures reporters.
type Options struct{ Color bool }

// New returns the Reporter for format ("text", "json" or "github").
func New(format string, opts Options) (Reporter, error) {
	switch format {
	case "text":
		return textReporter{color: opts.Color}, nil
	case "json":
		return jsonReporter{}, nil
	case "github":
		return githubReporter{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (valid: text, json, github)", format)
	}
}
