package display

import "testing"

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost float64
		want string
	}{
		{0.0, "$0.0000"},
		{0.005, "$0.0050"},
		{0.01, "$0.01"},
		{1.50, "$1.50"},
		{10.0, "$10.00"},
		{100.456, "$100.46"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.cost)
		if got != tt.want {
			t.Errorf("FormatCost(%v) = %q, want %q", tt.cost, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{50000, "50.0k"},
		{999999, "1000.0k"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestBar(t *testing.T) {
	tests := []struct {
		value float64
		max   float64
		width int
		want  string
	}{
		{0, 0, 10, "░░░░░░░░░░"},
		{0, 10, 10, "░░░░░░░░░░"},
		{10, 10, 10, "██████████"},
		{5, 10, 10, "█████░░░░░"},
		{1, 10, 10, "█░░░░░░░░░"},
		{0.01, 10, 10, "█░░░░░░░░░"}, // tiny positive still gets 1 filled
	}
	for _, tt := range tests {
		got := Bar(tt.value, tt.max, tt.width)
		if got != tt.want {
			t.Errorf("Bar(%v, %v, %d) = %q, want %q", tt.value, tt.max, tt.width, got, tt.want)
		}
	}
}
