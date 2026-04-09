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

  Sun    ░░░░░░░░░░░░░░░░░░░░    $0.00
  Mon    █░░░░░░░░░░░░░░░░░░░    $1.30  (  2 sessions,   1.1M tokens)
  Tue    ░░░░░░░░░░░░░░░░░░░░    $0.00
  Wed    █░░░░░░░░░░░░░░░░░░░    $3.40  (  3 sessions,   4.0M tokens)
  Thu    ████████████████████   $39.17  ( 44 sessions,  45.6M tokens)

  ╭────────────────────────────────────────────────╮
  │  Total:         $43.87 across 49 sessions      │
  │  Avg/session:    $0.90 (historical:    $3.13)  │
  │  Avg/day:        $8.77 (historical:   $40.33)  │
  ╰────────────────────────────────────────────────╯
```

Use `--days N` to look back further than the current week:

```
tokencoach stats --days 30
```

### `tokencoach tips`

AI-generated tips to reduce your costs. Analyzes your most expensive recent sessions and identifies patterns like retry loops, excessive exploration, or interrupted workflows.

```
$ tokencoach tips

  ⠼ Analyzing sessions......25.501s

                               7-Day Summary
  ╭──────────────────────────────────────────────────────────────────────────╮
  │  Spend: $46.57 (49 sessions)                                            │
  │  Avg:   $0.95/session (baseline: $3.14)                                 │
  │  Top 2 sessions = $21.72 (47% of total)                                 │
  ╰──────────────────────────────────────────────────────────────────────────╯

                    $16.53  Thu 9:14am  ~/Code/insights
  ╭──────────────────────────────────────────────────────────────────────────╮
  │  Building a CLI tool — session was interrupted mid-flow after extensive  │
  │  development work                                                       │
  │                                                                         │
  │  280 turns (5.5x your avg of 51), 705% context growth, $16.53 (5.3x    │
  │  your avg session cost of $3.14), 92 Bash + 56 Edit + 29 Read tool     │
  │  calls                                                                  │
  │                                                                         │
  │  Tip: When context grows past ~300%, start a fresh session with a       │
  │  focused prompt summarizing only what's needed next — carrying 705%     │
  │  accumulated context through 280 turns multiplied the cost              │
  │  dramatically.                                                          │
  ╰──────────────────────────────────────────────────────────────────────────╯

                      $2.80  Wed 1:14pm  ~/Code/perry
  ╭──────────────────────────────────────────────────────────────────────────╮
  │  Changing all TTLs to six months — a well-scoped task that nonetheless  │
  │  ran long                                                               │
  │                                                                         │
  │  10 errors (6x your avg of 1.68), 94 turns but only 12,949 output      │
  │  tokens (~138 tokens/turn vs your typical 300+), suggesting repeated    │
  │  failed attempts rather than productive output                          │
  │                                                                         │
  │  Tip: When error count spikes above 5, stop and diagnose the root      │
  │  cause manually before continuing — letting the agent retry 10 times    │
  │  compounds token cost without fixing the underlying issue.              │
  ╰──────────────────────────────────────────────────────────────────────────╯
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
