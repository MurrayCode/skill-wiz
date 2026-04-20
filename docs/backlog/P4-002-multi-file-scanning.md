# P4-002 Multi-File Scanning

## Story

As a CLI user, I want to scan multiple files or directories so that `skill-wiz` can be used across a whole repository.

## Scope

- scan multiple explicit file paths
- optionally scan directories for matching skill files
- return per-file results

## Proposed Changes

- extend input handling beyond a single positional path
- add file discovery rules for supported skill files
- aggregate results without losing file-level context

## Acceptance Criteria

- CLI can scan more than one file in a single run
- directory scanning returns per-file findings
- invalid files do not prevent reporting on valid files where reasonable

## Dependencies

- depends on `P4-001`
