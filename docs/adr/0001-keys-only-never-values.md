# ADR 0001: Compare keys only, never values

## Status

Accepted

## Context

`envdiff` runs in CI, in git hooks and in developer terminals. Output from those places ends up in build logs, shell history and hook output, often readable by many people. dotenv files hold secrets. A tool that prints or serializes a value, even in an error message or a debug line, leaks it.

## Decision

`envdiff` compares key names only. The parser stores for each entry just the key, its line and an `Empty bool`; the value is discarded while parsing. Reporters and error messages only have keys, paths and line numbers to work with. Tests enforce this by feeding files with a sentinel secret value and asserting it never appears in any output, any format, or any error.

## Consequences

- Leaking a value is structurally impossible: the data is not there to leak.
- The tool cannot compare values or detect that a value changed; that is explicitly out of scope.
- Emptiness is the only property of a value that is observable, via `--check-empty`.

## Alternatives considered

Storing values and promising not to print them (rejected: one future change away from a leak); masking values in output (rejected: still carries them through the program).
