# ADR 0002: Strict dotenv parsing

## Status

Accepted

## Context

dotenv has no formal grammar, and many loaders silently skip lines they do not understand. For `envdiff`, a skipped malformed line can hide a key that is actually missing, which is exactly the failure the tool exists to catch.

## Decision

The parser is strict. A line that is not blank, a comment or a valid `KEY=value` statement (no `=`, invalid key, unclosed quote, trailing junk after a closing quote) is a syntax error. The first error stops parsing. It maps to exit code 2, distinct from exit code 1 (differences found), so CI can tell "the files differ" from "the files could not be checked". Error messages contain only `path:line: message` and never any value.

## Consequences

- Legacy files with junk lines must be fixed before they can be checked.
- No finding can be silently lost to a parsing quirk.
- Only the first error is reported per run.
- Unknown escape sequences in double quotes are kept literally, since values are never stored and such sequences cannot affect the result.

## Alternatives considered

Tolerant parsing that skips bad lines with a warning (rejected: warnings are ignored and the key still looks absent or present by accident); collecting all errors (deferred: more complexity for little gain).
