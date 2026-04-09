# tokencoach

Cost analytics and AI-powered coaching for your Claude Code sessions.

tokencoach reads your local Claude Code session data (`~/.claude/projects/`) and gives you visibility into how much you're spending, broken down by day, session, and model.

## Install

```
brew install zhubert/tap/tokencoach
```

Or build from source:

```
go build -o tokencoach .
```

## Commands

### `tokencoach stats`

Daily cost breakdown with historical comparison. Shows a bar chart of spending per day with session counts, token usage, and averages compared to your historical baseline.

```
$ tokencoach stats

  Sun  ░░░░░░░░░░░░░░░░░░░░    $0.00
  Mon  ████████████░░░░░░░░  $12.34  ( 8 sessions, 450.2k tokens)
  Tue  ████████████████████  $18.50  (12 sessions, 820.1k tokens)
  Wed  ██████░░░░░░░░░░░░░░   $6.20  ( 4 sessions, 210.0k tokens)

  Total:       $37.04 across 24 sessions
  Avg/session: $1.54  (historical: $1.32)
  Avg/day:     $9.26  (historical: $8.15)
```

Use `--days N` to look back further than the current week:

```
tokencoach stats --days 30
```

### `tokencoach tips`

AI-generated tips to reduce your costs. Analyzes your most expensive recent sessions and identifies patterns like retry loops, excessive exploration, or interrupted workflows.

```
$ tokencoach tips

  Analyzed sessions in 4.2s

  ╭─ 7-Day Summary ──────────────────────────────────╮
  │  Spend: $67.75 (18 sessions)                      │
  │  Avg:   $3.76/session (baseline: $9.64)           │
  ╰───────────────────────────────────────────────────╯

  ┌ $49.59 · Thu 9:14am · ~/Code/tools
  │ Building a complementary CLI tool concept
  │ 280 turns · 705% context growth · 92 Bash, 56 Edit, 29 Read calls
  └ Tip: Synthesize findings after 3-4 exploratory rounds instead of cycling.

  ┌ $8.40 · Wed 1:14pm · ~/Code/myapp
  │ Changing all TTLs to six months
  │ 94 turns · 10 errors (5.7× avg) · only 12,949 output tokens
  └ Tip: Split large refactors into smaller, independently testable chunks.
```

Flags:
- `--days N` — number of days to analyze (default: 7)
- `--top N` — number of top sessions to analyze (default: 10)
- `--model MODEL` — model for analysis: haiku, sonnet, opus (default: sonnet)

Requires the [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) to be installed.

## How It Works

tokencoach parses the JSONL session logs that Claude Code writes to `~/.claude/projects/`. For each session it extracts token usage (input, output, cache read, cache creation), tool usage, errors, interruptions, and context growth. Costs are computed using per-model pricing for the Opus, Sonnet, and Haiku model families.

## License

MIT
