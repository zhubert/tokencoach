package claude

import "strings"

type ModelPricing struct {
	InputPerMillion       float64
	OutputPerMillion      float64
	CacheReadPerMillion   float64
	CacheCreatePerMillion float64
}

var pricing = map[string]ModelPricing{
	// Opus 4.5+ ($5/$25)
	"opus-new": {
		InputPerMillion:       5.00,
		OutputPerMillion:      25.00,
		CacheReadPerMillion:   0.50,
		CacheCreatePerMillion: 6.25,
	},
	// Opus 4, 4.1 ($15/$75)
	"opus-legacy": {
		InputPerMillion:       15.00,
		OutputPerMillion:      75.00,
		CacheReadPerMillion:   1.50,
		CacheCreatePerMillion: 18.75,
	},
	"sonnet": {
		InputPerMillion:       3.00,
		OutputPerMillion:      15.00,
		CacheReadPerMillion:   0.30,
		CacheCreatePerMillion: 3.75,
	},
	// Haiku 4.5+ ($1/$5)
	"haiku-new": {
		InputPerMillion:       1.00,
		OutputPerMillion:      5.00,
		CacheReadPerMillion:   0.10,
		CacheCreatePerMillion: 1.25,
	},
	// Haiku 3.5 and earlier ($0.80/$4)
	"haiku-legacy": {
		InputPerMillion:       0.80,
		OutputPerMillion:      4.00,
		CacheReadPerMillion:   0.08,
		CacheCreatePerMillion: 1.00,
	},
}

func modelFamily(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		if strings.Contains(m, "opus-4-5") || strings.Contains(m, "opus-4-6") {
			return "opus-new"
		}
		return "opus-legacy"
	case strings.Contains(m, "haiku"):
		if strings.Contains(m, "haiku-4") {
			return "haiku-new"
		}
		return "haiku-legacy"
	default:
		return "sonnet"
	}
}

func ComputeCost(model string, usage Usage) float64 {
	p := pricing[modelFamily(model)]
	cost := float64(usage.InputTokens) / 1_000_000 * p.InputPerMillion
	cost += float64(usage.OutputTokens) / 1_000_000 * p.OutputPerMillion
	cost += float64(usage.CacheReadInputTokens) / 1_000_000 * p.CacheReadPerMillion
	cost += float64(usage.CacheCreationInputTokens) / 1_000_000 * p.CacheCreatePerMillion
	return cost
}
