---
name: git-branch-commit-push
description: |-
  Create branches, write Conventional Commits, and push work in this repository. Use whenever a change is ready to be branched, committed, or pushed, or when a commit message needs to be written or corrected.

  Examples:
  - user: "Commit this" → branch if needed, stage the change, write a Conventional Commit, stop before pushing
  - user: "Start work on P4-003" → create a branch named for the story before editing
  - user: "Push this branch" → push the current branch and set upstream on first push
  - user: "Fix that commit message" → amend to a valid Conventional Commit subject
---
# Git Branch, Commit, and Push

Use this skill when creating branches, committing changes, or pushing them.

## Goal

Keep history readable and attributable to the repository owner alone, using
Conventional Commits on short-lived topic branches.

## Branching

1. Never commit directly to `main`. If the current branch is `main`, create a branch first.
2. Name branches `<type>/<short-kebab-description>`, using the same type vocabulary as commits.
   When the work maps to a backlog story, include the story id: `feat/p4-003-exit-codes`.
3. Branch from an up-to-date `main` unless the work builds on another branch.
4. Keep one logical change per branch.

## Committing

1. Review what is staged and unstaged before committing; stage only files relevant to the change.
2. Write the subject as a Conventional Commit:

   ```
   <type>(<scope>): <summary>
   ```

   - types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `build`, `ci`, `perf`, `style`, `revert`
   - scope is optional; use the backlog story id or package name when it helps (`feat(p4-001):`, `fix(scanner):`)
   - summary is imperative, lower case, no trailing full stop, ideally under 72 characters
3. Add a body only when the reason for the change is not obvious from the diff. Wrap at ~72 columns.
4. Use `BREAKING CHANGE:` in the footer for incompatible changes.
5. Verify the change before committing — at minimum `go build ./...` and the relevant `go test` scope.

## Attribution — hard rules

These apply to commit messages, commit bodies, branch names, PR titles, and PR bodies:

- **Never add `Co-Authored-By` trailers.** No co-authors, ever.
- **Never include Claude session links, session ids, or `Claude-Session:` trailers.**
- **Never add generated-by or tool-attribution lines** (for example "Generated with Claude Code").

The commit is authored by the repository owner. Nothing in the message should reference the
assistant, the session, or the tooling that produced the change.

## Pushing

1. Push only when the user asks.
2. On the first push of a branch, set upstream: `git push -u origin <branch>`.
3. Never force-push a shared branch. If a force push is genuinely needed, use
   `--force-with-lease` and confirm with the user first.
4. Do not open a pull request unless asked. If asked, the PR title follows the same
   Conventional Commit format and the body carries no attribution lines.

## Constraints

- do not amend or rebase commits that are already pushed unless the user asks
- do not stage unrelated files, build output, or anything matching `.gitignore`
- do not commit secrets; `GEMINI_API_KEY` and `.env` never enter a commit
- do not use interactive git flags — they are unavailable in this environment

## Output Expectations

After committing or pushing, report:

1. the branch name
2. the commit subject
3. whether the branch was pushed, and whether upstream was set
