// Package cli wires flags, files and reporters into the envdiff command.
package cli

import "io"

// Run executes envdiff with args and returns the process exit code.
// It never calls os.Exit.
func Run(args []string, stdout, stderr io.Writer) int {
	return 0
}
