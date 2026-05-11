# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

tokencoach is a Go CLI that analyzes Claude Code session data (`~/.claude/projects/`) to show cost breakdowns and AI-generated usage tips. It reads JSONL session logs, computes token costs by model, and displays daily stats with historical comparisons.

## Commands

- `make build` — build the binary
- `make test` — run all tests (`go test ./...`)
- `go test ./internal/claude/` — run tests for a single package
- `go test ./internal/claude/ -run TestParseSession_Basic` — run a single test
- `make clean` — remove binary and dist/
- `scripts/release.sh` — goreleaser release (the `make release` target no longer exists)

### Global flags (persistent, on all subcommands)

- `--debug` — enable debug logging to stderr
- `--config-dir` — override the Claude config directory (also settable via `CLAUDE_CONFIG_DIR` env var)

### `stats` flags

- `--days N` — show the last N days (default: current week starting Sunday)

### `tips` flags

- `--days N` — number of days to analyze (default: 7)
- `--top N` — number of top-cost sessions to send to the AI (default: 10)
- `--model NAME` — Claude model to use for analysis: haiku, sonnet, or opus (default: sonnet)

## Architecture

**CLI layer** (`cmd/`): Cobra commands. `root.go` registers subcommands and the two persistent flags (`--debug`, `--config-dir`); `stats.go` and `tips.go` implement the two commands.

**Session parsing** (`internal/claude/sessions.go`): Reads JSONL files from the Claude config directory (default `~/.claude/projects/*/`, overridable via `--config-dir` / `CLAUDE_CONFIG_DIR` / `ConfigDirOverride`). Each file is one session. Parses assistant entries for token usage, tool counts, and context growth. Parses user entries for summaries, errors, and interruptions. `AllSessions()` is the main entry point — it scans all project dirs, parses each session file, and returns them sorted by start time. `SessionsSince(since time.Time)` is a convenience wrapper that filters to sessions starting at or after the given time.

**Pricing** (`internal/claude/pricing.go`): Maps five model families to per-million-token rates. `modelFamily()` extracts the family from a full model ID string (case-insensitive). `ComputeCost()` computes dollar cost from model + usage across all four token types (input, output, cache read, cache creation).

| Family | Models | Input | Output |
|---|---|---|---|
| `opus-new` | Opus 4.5+ | $5 | $25 |
| `opus-legacy` | Opus 4, 4.1 | $15 | $75 |
| `sonnet` | all Sonnet (default for unknown) | $3 | $15 |
| `haiku-new` | Haiku 4.x | $1 | $5 |
| `haiku-legacy` | Haiku 3.5 and earlier | $0.80 | $4 |

**Display** (`internal/display/display.go`): Formatting helpers for costs, token counts, and bar charts. Uses a Tokyo Night colour palette:

- `ColorAccent` — blue, used for headers, titles, costs
- `ColorDim` — foreground, used for body text and metrics
- `ColorTip` — purple, used for tips and positive callouts
- `ColorWarn` — orange, used for quoted problems and attention
- `ColorBar` / `ColorBarBg` — blue fill / dark background for bar charts

Key helpers: `RoundedBox(content, width)` renders content in a rounded lipgloss border box; `PlainBar(value, max, width)` is an unstyled bar for use in tests; `GetTermWidth()` returns the terminal width (falls back to 80).

**Tips command** (`cmd/tips.go`): Two-pass AI analysis with a spinner while waiting for the Claude CLI (`claude -p`).

- **Pass 1** — builds a JSON summary of the top-N sessions plus a historical baseline, sends it to Claude, and receives 2–3 `sessionTip` objects identifying costly or problematic sessions with actionable tips.
- **Pass 2** — launches one goroutine per tip in parallel; each goroutine sends the session's user prompts to Claude and asks it to identify the most problematic prompt and suggest a rewrite (`promptMatch`). Results are collected from a channel and displayed alongside the tips.

## Testing

Tests use table-driven patterns and testdata fixtures (`internal/claude/testdata/*.jsonl`).

**`internal/claude/sessions_test.go`**: covers `TestParseSession_Basic`, `TestParseSession_ErrorsAndInterruptions`, `TestParseSession_Empty`, `TestParseSession_StringContent`, `TestParseSession_ToolsHeavy`, `TestParseSession_NonexistentFile`, plus helper tests `TestProjectDirToName` and `TestTruncate`.

**`internal/claude/pricing_test.go`**: covers `TestModelFamily` (all five families, case-insensitive matching, unknown model defaults to sonnet) and `TestComputeCost` (per-token-type costs for every model family plus mixed usage).

**`internal/display/display_test.go`**: covers `TestFormatCost`, `TestFormatTokens`, and `TestBar` (via `PlainBar`, the unstyled variant).
