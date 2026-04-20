# skill-wiz

`skill-wiz` is a small CLI for reviewing skill files and flagging mismatches between a skill's description and its instructions.

It reads a skill file with YAML front matter, extracts the `description` and body, and sends both to Gemini for analysis. The tool is intended to help spot:

- mismatches between the advertised behaviour and the actual instructions
- suspicious behaviour
- hidden behaviour

## Requirements

- Go `1.24+`
- A Gemini API key exposed as `GEMINI_API_KEY`

## Skill File Format

The CLI expects a skill file that starts with YAML front matter delimited by `---`, followed by the skill body.

```yaml
---
name: example skill
description: Reviews a repository for unsafe shell commands.
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: standard
---
# Skill Body

Inspect the repository and report unsafe shell execution.
```

## Usage

Run the tool by passing the path to a skill file:

```bash
go run . /path/to/skill.md
```

You can also build a binary first:

```bash
go build -o skill-wiz .
./skill-wiz /path/to/skill.md
```

If `GEMINI_API_KEY` is not set, export it before running the tool:

```bash
export GEMINI_API_KEY="your-api-key"
go run . /path/to/skill.md
```

## Output

If no issues are found, the tool prints:

```text
THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE
```

If the analyzer flags something, the CLI prints a short report like this:

```text
Scan flagged 1 finding(s)
[warning] analysis: Analyzer reported potential issues
Evidence: SUSPICIOUS: hidden shell execution
```

## Notes

- The current CLI accepts exactly one argument: the path to the skill file.
- The analysis is model-based, so results should still be reviewed manually.
