# envdiff

Checks that your `.env` has the same keys as `.env.example`, and fails when it doesn't.

[![CI](https://github.com/GabrielOliveiraTh/envdiff/actions/workflows/ci.yml/badge.svg)](https://github.com/GabrielOliveiraTh/envdiff/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/GabrielOliveiraTh/envdiff)](https://github.com/GabrielOliveiraTh/envdiff/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/GabrielOliveiraTh/envdiff)](https://goreportcard.com/report/github.com/GabrielOliveiraTh/envdiff)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Someone adds a variable to `.env.example`, and your `.env` (or the one in CI or staging) never gets it.
The app boots fine and then breaks at runtime, far from the cause.
`envdiff` catches that early and returns an exit code you can use in CI or a git hook.

It only looks at key names. Values are never printed, logged or kept in the results, so it's safe to run where logs are public.

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

That run exits with 1. When everything matches you get one line and exit 0:

```text
✓ ok.env matches .env.example (6 keys)
```

## Install

```sh
go install github.com/GabrielOliveiraTh/envdiff/cmd/envdiff@latest
```

Or grab a binary for Linux, macOS or Windows from the [Releases](https://github.com/GabrielOliveiraTh/envdiff/releases) page.

## Flags

| Flag | Default | What it does |
|------|---------|--------------|
| `-e`, `--example <path>` | `.env.example` | Reference file |
| `-f`, `--env <path>` | `.env` | File to check |
| `--format text\|json\|github` | `text` | Output format |
| `--ignore <pattern>` | | Skip keys matching a glob (`'LOCAL_*'`). Repeatable |
| `--check-empty` | off | Also flag keys that exist in both files but are empty in `--env` |
| `--strict` | off | Make extra and duplicate keys fail too |
| `-q`, `--quiet` | off | No output, just the exit code |
| `--no-color` | off | Turn colors off |
| `--version` | | Print version, commit and build date |

`--ignore` uses Go's `path.Match` syntax and applies to every kind of finding.

Colors only show up with the text format, when stdout is a terminal, `NO_COLOR` isn't set and you didn't pass `--no-color`. Very old Windows consoles may print raw escape codes, so use `--no-color` there.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Nothing that fails the run |
| `1` | Differences found |
| `2` | Bad usage, unreadable file, or syntax error in a dotenv file |

Missing keys always fail. Empty keys fail (only checked with `--check-empty`). Extra and duplicate keys are warnings, and only fail with `--strict`.
The report goes to stdout. Errors go to stderr, prefixed with `envdiff: `.

## Output formats

**text** is the default, shown above. If there are only warnings, the last line is `Result: OK with warnings (...)`.

**json** is a stable schema, so you can build on it:

```json
{
  "example": ".env.example",
  "env": ".env",
  "missing": [{ "key": "DATABASE_URL", "line": 4 }],
  "extra": [{ "key": "OLD_FLAG", "line": 5 }],
  "empty": [{ "key": "API_KEY", "line": 3 }],
  "duplicates": [{ "file": ".env", "key": "PORT", "lines": [2, 6] }],
  "key_count": 6,
  "ok": false
}
```

(The real output is indented one field per line.) `line` in `missing` points into the reference file; in `extra` and `empty` it points into the env file. `key_count` is the number of reference keys after `--ignore`. Lists are always arrays, never `null`, and sorted by key. `ok` is true when the exit code would be 0.

**github** prints one [workflow command](https://docs.github.com/actions/reference/workflow-commands-for-github-actions) per finding, so they show up as annotations on the files:

```text
::error file=.env.example,line=4::Missing key DATABASE_URL in .env
::error file=.env.example,line=6::Missing key REDIS_URL in .env
::error file=.env,line=3::Empty value for key API_KEY in .env
::warning file=.env,line=5::Extra key OLD_FLAG not present in .env.example
::warning file=.env,line=2::Duplicate key PORT (lines 2, 6)
```

With `--strict`, the warnings become errors.

## Recipes

GitHub Actions (install the binary first, e.g. with `go install`):

```yaml
- name: Check env keys
  run: envdiff --example .env.example --env .env.ci --format github
```

Git pre-commit hook (`.git/hooks/pre-commit`, make it executable):

```sh
#!/bin/sh
envdiff --quiet || { envdiff; exit 1; }
```

Makefile:

```make
env-check:
	envdiff --check-empty
```

## What the parser accepts

It's deliberately strict: a line it can't understand is an error (exit 2), not something to skip silently. See [ADR 0002](docs/adr/0002-strict-dotenv-parsing.md).

- `KEY=value`, with an optional `export ` in front. Keys match `[A-Za-z_][A-Za-z0-9_.]*`.
- Blank lines and `#` comment lines are ignored. Unquoted values can have an inline ` # comment`.
- Single quotes are literal. Double quotes understand `\n`, `\r`, `\t`, `\"` and `\\`. Both can span several lines.
- After a closing quote, only spaces or a comment may follow. An unclosed quote is an error that points at the line where it started.
- `KEY=`, `KEY=""`, `KEY=''` and `KEY= # note` all count as empty.
- A repeated key isn't an error: the last one wins, and it's reported as a duplicate.
- No interpolation: `${VAR}` is just text.
- UTF-8, with or without BOM; `\n` and `\r\n` line endings.
- Error messages never include values. The parser never panics (there's a fuzz test for that).

## Design decisions

- [ADR 0001: keys only, never values](docs/adr/0001-keys-only-never-values.md)
- [ADR 0002: strict dotenv parsing](docs/adr/0002-strict-dotenv-parsing.md)

Not planned: comparing values, `${VAR}` interpolation, fixing or syncing files, more than two files, a config file.

## Contributing

```sh
make test    # go test -race ./...
make lint    # golangci-lint
make cover   # coverage
make fuzz    # fuzz the parser for 30s
```

Reporter golden files are regenerated with `go test ./internal/report/ -update`.

The packages depend on each other in one direction: `cli` → `report` → `diff` → `envfile`.

## License

MIT, see [LICENSE](LICENSE).

## Open decisions

Places where I had to pick something:

- Duplicates are reported for both files. `empty` only applies to keys present in both.
- The `Result:` labels aren't pluralized (`2 missing, 1 extra, 3 duplicate`).
- `NO_COLOR` only counts when it's set to a non-empty value.
- Unknown escapes in double quotes (like `\q`) are kept as-is instead of failing. A lone `\r` doesn't end a line.
- The parser stops at the first error.
- I couldn't check the GitHub Action versions when writing the workflows (no network), so please verify them: `actions/checkout@v5`, `actions/setup-go@v6`, `actions/upload-artifact@v4`, `golangci/golangci-lint-action@v8`, `goreleaser/goreleaser-action@v6`.
