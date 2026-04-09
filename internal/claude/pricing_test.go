package claude

import (
	"math"
	"testing"
)

func TestModelFamily(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-6-20260401", "opus-new"},
		{"claude-opus-4-5-20250514", "opus-new"},
		{"claude-opus-4-20250514", "opus-legacy"},
		{"claude-opus-4-1-20250514", "opus-legacy"},
		{"claude-sonnet-4-20250514", "sonnet"},
		{"claude-sonnet-4-6-20260401", "sonnet"},
		{"claude-haiku-4-5-20251001", "haiku-new"},
		{"claude-haiku-3-5-20241022", "haiku-legacy"},
		{"CLAUDE-OPUS-4-6", "opus-new"},
		{"some-unknown-model", "sonnet"}, // default
		{"", "sonnet"},                   // empty defaults to sonnet
	}
	for _, tt := range tests {
		got := modelFamily(tt.model)
		if got != tt.want {
			t.Errorf("modelFamily(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestComputeCost(t *testing.T) {
	tests := []struct {
		name  string
		model string
		usage Usage
		want  float64
	}{
		{
			name:  "zero usage",
			model: "sonnet",
			usage: Usage{},
			want:  0,
		},
		{
			name:  "sonnet input only",
			model: "claude-sonnet-4-20250514",
			usage: Usage{InputTokens: 1_000_000},
			want:  3.00,
		},
		{
			name:  "sonnet output only",
			model: "claude-sonnet-4-20250514",
			usage: Usage{OutputTokens: 1_000_000},
			want:  15.00,
		},
		{
			name:  "opus 4 input only",
			model: "claude-opus-4-20250514",
			usage: Usage{InputTokens: 1_000_000},
			want:  15.00,
		},
		{
			name:  "opus 4 output only",
			model: "claude-opus-4-20250514",
			usage: Usage{OutputTokens: 1_000_000},
			want:  75.00,
		},
		{
			name:  "opus 4.6 input only",
			model: "claude-opus-4-6-20260401",
			usage: Usage{InputTokens: 1_000_000},
			want:  5.00,
		},
		{
			name:  "opus 4.6 output only",
			model: "claude-opus-4-6-20260401",
			usage: Usage{OutputTokens: 1_000_000},
			want:  25.00,
		},
		{
			name:  "opus 4.5 input only",
			model: "claude-opus-4-5-20250514",
			usage: Usage{InputTokens: 1_000_000},
			want:  5.00,
		},
		{
			name:  "haiku 3.5 input only",
			model: "claude-haiku-3-5-20241022",
			usage: Usage{InputTokens: 1_000_000},
			want:  0.80,
		},
		{
			name:  "haiku 4.5 input only",
			model: "claude-haiku-4-5-20251001",
			usage: Usage{InputTokens: 1_000_000},
			want:  1.00,
		},
		{
			name:  "haiku 4.5 output only",
			model: "claude-haiku-4-5-20251001",
			usage: Usage{OutputTokens: 1_000_000},
			want:  5.00,
		},
		{
			name:  "cache read",
			model: "claude-sonnet-4-20250514",
			usage: Usage{CacheReadInputTokens: 1_000_000},
			want:  0.30,
		},
		{
			name:  "cache creation",
			model: "claude-sonnet-4-20250514",
			usage: Usage{CacheCreationInputTokens: 1_000_000},
			want:  3.75,
		},
		{
			name:  "mixed usage sonnet",
			model: "claude-sonnet-4-20250514",
			usage: Usage{
				InputTokens:              500_000,
				OutputTokens:             100_000,
				CacheReadInputTokens:     200_000,
				CacheCreationInputTokens: 50_000,
			},
			// 0.5M * 3 + 0.1M * 15 + 0.2M * 0.3 + 0.05M * 3.75
			// = 1.50 + 1.50 + 0.06 + 0.1875 = 3.2475
			want: 3.2475,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCost(tt.model, tt.usage)
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("ComputeCost(%q, %+v) = %v, want %v", tt.model, tt.usage, got, tt.want)
			}
		})
	}
}
