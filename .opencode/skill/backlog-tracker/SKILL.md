---
name: backlog-tracker
description: |-
  Update `docs/backlog/tracker.md` when a backlog story in `docs/backlog/` is completed, blocked, or moved into progress. Use proactively when finishing a story implementation, verifying a backlog item, or changing story status so the project tracker stays current.

  Examples:
  - user: "Mark P1-003 complete" → update the tracker row for `P1-003-structured-results.md` with `done`, date, and note
  - user: "Set P2-001 in progress" → update the tracker row for `P2-001-rules-package.md` to `in_progress`
  - user: "P3-002 is blocked on analyzer design" → update the tracker row with `blocked` and the reason
---
# Backlog Tracker

Use this skill when updating `docs/backlog/tracker.md` for backlog stories.

## Goal

Keep backlog story status accurate and easy to review.

## Files

- tracker file: `docs/backlog/tracker.md`
- story files: `docs/backlog/P*-*.md`

## Workflow

1. Identify the story file that changed status.
2. Read `docs/backlog/tracker.md` and locate the matching story row.
3. Update only the relevant row.
4. Keep status values limited to:
   - `todo`
   - `in_progress`
   - `done`
   - `blocked`
5. When marking a story `done`:
   - set `Completed On` to the current date in `YYYY-MM-DD` format
   - add a short note describing what was delivered
6. When marking a story `blocked`:
   - leave `Completed On` blank
   - add a concise blocker note
7. When marking a story `in_progress` or `todo`:
   - leave `Completed On` blank unless the story is still done
   - clear or adjust notes so they match the current state

## Constraints

- update only the relevant tracker row unless the user asks for broader cleanup
- do not rename stories in the tracker unless the backlog story file changed
- do not mark a story `done` unless implementation and verification are complete
- preserve markdown table formatting

## Output Expectations

After updating the tracker, report:

1. which story row was updated
2. the new status
3. whether a completion date or blocker note was added
