// Package cli wires flags, files and reporters into the envdiff command.
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/GabrielOliveiraTh/envdiff/internal/diff"
	"github.com/GabrielOliveiraTh/envdiff/internal/envfile"
	"github.com/GabrielOliveiraTh/envdiff/internal/report"
)

// Build information. These are package-level variables (not constants)
// because the linker can only overwrite variables: the Makefile and goreleaser
// set them with
//
//	-X github.com/GabrielOliveiraTh/envdiff/internal/cli.version=...
//
// They are never modified at run time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Exit codes: part of the public contract.
const (
	exitOK      = 0 // nothing that causes failure
	exitDiff    = 1 // differences found
	exitUsage   = 2 // usage, file read or dotenv syntax error
	errorPrefix = "envdiff: "
)

type options struct {
	example    string
	env        string
	format     string
	ignore     []string
	checkEmpty bool
	strict     bool
	quiet      bool
	noColor    bool
	showVer    bool
}

// Run executes envdiff with args and returns the process exit code.
// It never calls os.Exit and keeps no state between calls.
func Run(args []string, stdout, stderr io.Writer) int {
	var (
		opts options
		code = exitOK
	)

	cmd := &cobra.Command{
		Use:           "envdiff",
		Short:         "Compare the keys of two dotenv files",
		Long:          "envdiff checks that a .env file defines every key listed in a reference file (such as .env.example). Only key names are compared; values are never read into the report.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			var err error
			code, err = execute(opts, stdout)
			return err
		},
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	f := cmd.Flags()
	f.StringVarP(&opts.example, "example", "e", ".env.example", "reference file")
	f.StringVarP(&opts.env, "env", "f", ".env", "file to verify")
	f.StringVar(&opts.format, "format", "text", "output format: text, json or github")
	f.StringArrayVar(&opts.ignore, "ignore", nil, "glob (path.Match syntax) of key names to ignore; repeatable")
	f.BoolVar(&opts.checkEmpty, "check-empty", false, "report keys whose value in the env file is empty")
	f.BoolVar(&opts.strict, "strict", false, "make extra and duplicate keys fail too")
	f.BoolVarP(&opts.quiet, "quiet", "q", false, "print nothing to stdout, only set the exit code")
	f.BoolVar(&opts.noColor, "no-color", false, "disable colors")
	f.BoolVar(&opts.showVer, "version", false, "print version, commit and build date")

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(stderr, errorPrefix+err.Error())
		return exitUsage
	}
	return code
}

// execute does the actual work and returns the exit code. A non-nil error
// always means exit code 2 and is printed by Run.
func execute(o options, stdout io.Writer) (int, error) {
	if o.showVer {
		_, err := fmt.Fprintln(stdout, versionString())
		return exitOK, err
	}

	rep, err := report.New(o.format, report.Options{Color: useColor(o, stdout)})
	if err != nil {
		return exitUsage, err
	}
	for _, g := range o.ignore {
		if err := validateGlob(g); err != nil {
			return exitUsage, err
		}
	}

	example, err := load(o.example)
	if err != nil {
		return exitUsage, err
	}
	env, err := load(o.env)
	if err != nil {
		return exitUsage, err
	}

	res := diff.Compare(example, env, diff.Options{
		Ignore:     o.ignore,
		CheckEmpty: o.checkEmpty,
		Strict:     o.strict,
	})

	if !o.quiet {
		if err := rep.Report(stdout, res); err != nil {
			return exitUsage, fmt.Errorf("writing report: %w", err)
		}
	}
	if !res.OK {
		return exitDiff, nil
	}
	return exitOK, nil
}

// load opens and parses one dotenv file. Errors only mention the path, the
// line number and fixed text, never file content.
func load(p string) (envfile.File, error) {
	fh, err := os.Open(p)
	if err != nil {
		return envfile.File{}, readError(p, err)
	}
	defer func() { _ = fh.Close() }() // read-only handle: a close error is not actionable

	f, err := envfile.Parse(p, fh)
	if err != nil {
		var pe *envfile.ParseError
		if errors.As(err, &pe) {
			return envfile.File{}, pe
		}
		return envfile.File{}, readError(p, err)
	}
	return f, nil
}

func readError(p string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err // avoid repeating the path
	}
	return fmt.Errorf("cannot read %s: %w", p, err)
}

// validateGlob rejects malformed path.Match patterns. path.Match checks the
// whole pattern for syntax errors only as far as it needs to decide a match,
// so a few probes are used to make detection robust.
func validateGlob(g string) error {
	for _, probe := range []string{"", "a", "A_B1", "-", "\x00"} {
		if _, err := path.Match(g, probe); err != nil {
			return fmt.Errorf("invalid --ignore pattern %q: %w", g, err)
		}
	}
	return nil
}

// useColor reports whether the text report should be colored: format text,
// stdout is a character device (terminal), --no-color absent and NO_COLOR
// unset. Following no-color.org, NO_COLOR disables colors when present and
// non-empty; a defined-but-empty NO_COLOR is ignored.
func useColor(o options, stdout io.Writer) bool {
	if o.format != "text" || o.noColor || o.quiet {
		return false
	}
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return false
	}
	s, ok := stdout.(interface{ Stat() (os.FileInfo, error) })
	if !ok {
		return false
	}
	fi, err := s.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// versionString formats the build information. When the binary was not built
// with ldflags (version "dev"), the module version embedded by "go install"
// is used if available.
func versionString() string {
	return formatVersion(buildInfo())
}

func buildInfo() (v, c, d string) {
	v, c, d = version, commit, date
	if v == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		}
	}
	return v, c, d
}

func formatVersion(v, c, d string) string {
	return fmt.Sprintf("envdiff version %s (commit %s, built %s)", v, c, d)
}
