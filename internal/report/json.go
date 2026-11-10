package report

import (
	"encoding/json"
	"io"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
)

// jsonReporter serializes the Result as indented JSON.
type jsonReporter struct{}

// Report implements Reporter. The output ends with a newline.
func (jsonReporter) Report(w io.Writer, r diff.Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
