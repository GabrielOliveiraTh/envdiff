# envdiff

Compare the keys of `.env.example` and `.env` and fail CI when they drift. Never reads or prints values.

[![CI](https://github.com/GabrielOliveiraTh/envdiff/actions/workflows/ci.yml/badge.svg)](https://github.com/GabrielOliveiraTh/envdiff/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/GabrielOliveiraTh/envdiff)](https://github.com/GabrielOliveiraTh/envdiff/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/GabrielOliveiraTh/envdiff)](https://goreportcard.com/report/github.com/GabrielOliveiraTh/envdiff)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## The problem

Someone adds a variable to `.env.example` and your `.env` (or the CI/staging one) never gets it.
The app starts, then fails at runtime far from the cause.
`envdiff` catches the drift up front, with an exit code for CI and git hooks, and it never touches your secrets: only key names and line numbers are ever read into the report.

## Usage

```console
$ envdiff --check-empty
```

```text
Comparing .env against .env.example

✗ Missing in .env (2):
  - DATABASE_URL  (.env.example:4)
  - REDIS_URL     (.env.example:6)

✗ Empty in .env (1):
  - API_KEY  (.env:3)

! Extra in .env (1):
  - OLD_FLAG  (.env:5)

! Duplicate keys (1):
  - PORT  (.env: lines 2, 6)

Result: FAIL (2 missing, 1 empty, 1 extra, 1 duplicate)
```

The process exits with status 1. A clean run prints a single line and exits 0 (here `envdiff -f ok.env`, where `ok.env` has the same keys):

```text
✓ ok.env matches .env.example (6 keys)
```

## Installation

```sh
go install github.com/GabrielOliveiraTh/envdiff/cmd/envdiff@latest
```

Or download a prebuilt binary for Linux, macOS or Windows from the
[GitHub Releases](https://github.com/GabrielOliveiraTh/envdiff/releases) page.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-e`, `--example <path>` | `.env.example` | Reference file. |
| `-f`, `--env <path>` | `.env` | File to check against the reference. |
| `--format text\|json\|github` | `text` | Output format. |
| `--ignore <pattern>` | none | `path.Match` glob on key names, e.g. `'LOCAL_*'`. Repeatable. Applies to every category (missing, extra, empty, duplicates) and to the key count. |
| `--check-empty` | off | Also report keys present in both files whose value in `--env` is empty. |
| `--strict` | off | Extra and duplicate findings also fail the run. |
| `-q`, `--quiet` | off | Print nothing to stdout; only the exit code. |
| `--no-color` | off | Disable colors. |
| `--version` | | Print version, commit and build date. |

Colors are used only with `--format text`, only when stdout is a terminal, `NO_COLOR` is unset or empty and `--no-color` was not passed.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | No failing findings. |
| `1` | Differences found. |
| `2` | Usage error, file read error or dotenv syntax error. |

What fails the run: missing keys always; empty keys (only with `--check-empty`); extra keys and duplicates are warnings and fail only with `--strict`.
Errors go to stderr prefixed with `envdiff: `; the report goes to stdout.

## Output formats

### text

The default, shown above. With only warnings the result line reads `Result: OK with warnings (1 extra, 1 duplicate)`.

### json

The JSON document is a public contract.

```json
{
  "example": ".env.example",
  "env": ".env",
  "missing": [
    {
      "key": "DATABASE_URL",
      "line": 4
    },
    {
      "key": "REDIS_URL",
      "line": 6
    }
  ],
  "extra": [
    {
      "key": "OLD_FLAG",
      "line": 5
    }
  ],
  "empty": [
    {
      "key": "API_KEY",
      "line": 3
    }
  ],
  "duplicates": [
    {
      "file": ".env",
      "key": "PORT",
      "lines": [
        2,
        6
      ]
    }
  ],
  "key_count": 6,
  "ok": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `example` | string | Path of the reference file. |
| `env` | string | Path of the checked file. |
| `missing` | array of `{key, line}` | In the reference, absent from env. `line` is in the reference file. |
| `extra` | array of `{key, line}` | In env, absent from the reference. `line` is in env. |
| `empty` | array of `{key, line}` | Present in both, empty in env (only with `--check-empty`). `line` is in env. |
| `duplicates` | array of `{file, key, lines}` | Keys defined more than once in a file. `lines` lists every line, ascending. Covers both files. |
| `key_count` | integer | Number of reference keys after `--ignore`. |
| `ok` | boolean | `true` when the exit code would be 0. |

Lists are always arrays, never `null`. Findings are sorted by key. The `--strict` setting is not serialized; it is reflected only in `ok`.

### github

Emits one [workflow command](https://docs.github.com/actions/reference/workflow-commands-for-github-actions) per finding, so they show as annotations on the file and line. Nothing is printed when there are no findings.

```text
::error file=.env.example,line=4::Missing key DATABASE_URL in .env
::error file=.env.example,line=6::Missing key REDIS_URL in .env
::error file=.env,line=3::Empty value for key API_KEY in .env
::warning file=.env,line=5::Extra key OLD_FLAG not present in .env.example
::warning file=.env,line=2::Duplicate key PORT (lines 2, 6)
```

With `--strict`, extra and duplicate findings are emitted as `::error` instead of `::warning`.

## Recipes

### GitHub Actions

```yaml
- name: Check env keys
  run: envdiff --example .env.example --env .env.ci --format github
```

Install the binary first, for example with `go install github.com/GabrielOliveiraTh/envdiff/cmd/envdiff@latest`.

### Git pre-commit hook

`.git/hooks/pre-commit` (make it executable):

```sh
#!/bin/sh
envdiff --quiet || { envdiff; exit 1; }
```

Or with [pre-commit](https://pre-commit.com):

```yaml
repos:
  - repo: local
    hooks:
      - id: envdiff
        name: envdiff
        entry: envdiff
        language: system
        pass_filenames: false
```

### Makefile

```make
.PHONY: env-check
env-check:
	envdiff --check-empty
```

## Supported parsing rules

The parser is strict (see [ADR 0002](docs/adr/0002-strict-dotenv-parsing.md)). It extracts keys and an "empty" flag; values are never stored.

1. Input is UTF-8; a leading BOM is stripped; `\n` and `\r\n` line endings are accepted.
2. Blank lines and lines starting with `#` are ignored.
3. An optional `export` prefix (followed by spaces or tabs) is dropped. `export=1` is a key named `export`.
4. Keys must match `^[A-Za-z_][A-Za-z0-9_.]*$`; anything else is a syntax error.
5. Whitespace around `=` is tolerated. A non-comment line without `=` is a syntax error.
6. Unquoted values run to the end of the line; an inline comment starts at ` #` (a `#` preceded by whitespace).
7. Single-quoted values are literal and may span multiple lines.
8. Double-quoted values support the escapes `\n`, `\r`, `\t`, `\"` and `\\`, and may span multiple lines. Unknown escapes (such as `\q`) are kept literally and are not an error.
9. After a closing quote only spaces and a `#` comment may follow.
10. An unclosed quote is an error reported at the line where the quote started.
11. Empty values: `KEY=`, `KEY=""`, `KEY=''` and `KEY= # comment` are all empty.
12. A repeated key is reported as a duplicate; the last occurrence wins.
13. No interpolation: `${VAR}` and `$VAR` are plain text.
14. The parser stops at the first error.
15. It never panics, including on invalid UTF-8 (covered by a fuzz test).
16. Error messages contain only `path:line: message`, never a value.

## Design decisions

- [ADR 0001: Compare keys only, never values](docs/adr/0001-keys-only-never-values.md)
- [ADR 0002: Strict dotenv parsing](docs/adr/0002-strict-dotenv-parsing.md)

Out of scope: value comparison, interpolation, fixing or syncing files, more than two files, a config file of its own.

## Contributing

```sh
make test    # go test -race ./...
make lint    # golangci-lint run
make fuzz    # fuzz the parser for 30s
make cover   # coverage report
```

Golden files for the reporters are regenerated with:

```sh
go test ./internal/report/ -update
```

Package layout; dependencies point in one direction:

```text
cmd/envdiff -> internal/cli -> internal/report -> internal/diff -> internal/envfile
```

- `envfile`: strict parser; keeps keys, lines and an empty flag.
- `diff`: pure comparison producing the `Result` (also the JSON schema).
- `report`: text, json and github renderers.
- `cli`: flags, file loading, exit codes.

## License

MIT. See [LICENSE](LICENSE).

## Open decisions

Choices made where the specification was ambiguous:

- Duplicate keys are reported for both files (reference and env), each labelled with its file.
- `empty` is evaluated only for keys present in both files; a missing key is never also reported as empty.
- The `Result:` line labels are not pluralized (`2 missing, 1 extra, 1 duplicate`).
- `NO_COLOR`: colors are disabled when the variable is set to a non-empty value (per no-color.org; a defined-but-empty value is ignored); `--no-color` does the same.
- Unknown escape sequences in double quotes are kept literally instead of raising an error.
- The parser stops at the first syntax error rather than collecting all of them.
- GitHub Action versions in the workflows could not be verified at authoring time because the network was blocked: `actions/checkout@v5`, `actions/setup-go@v6`, `actions/upload-artifact@v4`, `golangci/golangci-lint-action@v8`, `goreleaser/goreleaser-action@v6`.
