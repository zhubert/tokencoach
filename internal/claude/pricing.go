package claude

import "strings"

type ModelPricing struct {
	InputPerMillion       float64
	OutputPerMillion      float64
	CacheReadPerMillion   float64
	CacheCreatePerMillion float64
}

var pricing = map[string]ModelPricing{
	"opus": {
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
	"haiku": {
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
		return "opus"
	case strings.Contains(m, "haiku"):
		return "haiku"
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
