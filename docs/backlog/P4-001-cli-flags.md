# P4-001 CLI Flags

## Story

As a CLI user, I want configurable flags so that I can control output format and analysis settings without editing code.

## Scope

- `--json`
- `--model`
- `--timeout`

## Proposed Changes

- introduce flag parsing in the CLI entrypoint
- pass runtime configuration into the scanner and analyzer layers
- keep sensible defaults for simple usage

## Acceptance Criteria

- CLI accepts and applies the supported flags
- invalid flag values return a clear error
- default behaviour remains simple for single-file scanning

## Dependencies

- depends on structured results and analyzer refactoring
