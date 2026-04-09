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
- `make release` — goreleaser release

## Architecture

**CLI layer** (`cmd/`): Cobra commands. `root.go` registers subcommands; `stats.go` and `tips.go` implement the two commands.

**Session parsing** (`internal/claude/sessions.go`): Reads JSONL files from `~/.claude/projects/*/`. Each file is one session. Parses assistant entries for token usage, tool counts, and context growth. Parses user entries for summaries, errors, and interruptions. `AllSessions()` is the main entry point — it scans all project dirs, parses each session file, and returns them sorted by start time.

**Pricing** (`internal/claude/pricing.go`): Maps model families (opus/sonnet/haiku) to per-million-token rates. `modelFamily()` extracts the family from a full model ID string. `ComputeCost()` computes dollar cost from model + usage.

**Display** (`internal/display/display.go`): Formatting helpers for costs, token counts, and bar charts.

**Tips command** (`cmd/tips.go`): Builds a JSON summary of recent sessions, then shells out to `claude -p` to get AI-generated cost-saving advice. The `--model` flag controls which model analyzes the data.

## Testing

Tests use table-driven patterns and testdata fixtures (`internal/claude/testdata/*.jsonl`). Session parsing tests cover: basic parsing, errors/interruptions, empty files, string content, tool-heavy sessions, and nonexistent files.
