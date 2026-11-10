# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-10

### Added

- Key comparison of a reference dotenv file (`.env.example`) against `.env`, reporting missing, extra, empty and duplicate keys.
- Strict dotenv parser: BOM, CRLF, `export` prefix, single and double quotes (multi-line), inline comments, with errors reported as `path:line`.
- Flags `--example`, `--env`, `--format`, `--ignore` (repeatable glob), `--check-empty`, `--strict`, `--quiet`, `--no-color` and `--version`.
- Output formats: `text` (colored on terminals, honoring `NO_COLOR`), `json` (stable schema) and `github` (workflow commands).
- Exit codes for CI and git hooks: 0 ok, 1 differences, 2 usage, read or syntax error.
- Guarantee that variable values are never printed, logged or serialized.
- CI and release workflows, golangci-lint, GoReleaser and Dependabot configuration.
- Golden tests for reporters and a fuzz test for the parser.

[Unreleased]: https://github.com/GabrielOliveiraTh/envdiff/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/GabrielOliveiraTh/envdiff/releases/tag/v0.1.0
