package display

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Shared color palette
var (
	ColorBorder = lipgloss.Color("240")
	ColorAccent = lipgloss.Color("75")  // steel blue — headers, titles
	ColorDim    = lipgloss.Color("243") // gray — supporting detail
	ColorTip    = lipgloss.Color("114") // green — tips, positive callouts
	ColorBar    = lipgloss.Color("75")  // matches accent
	ColorBarBg  = lipgloss.Color("237") // dark gray — empty bar
)

func GetTermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// RoundedBox renders content inside a rounded border box.
func RoundedBox(content string, width int) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 2).
		MarginLeft(2).
		Width(width).
		Render(content)
}

func FormatCost(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

func FormatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func Bar(value, max float64, width int) string {
	if max == 0 {
		return lipgloss.NewStyle().Foreground(ColorBarBg).Render(strings.Repeat("░", width))
	}
	filled := int(float64(width) * value / max)
	if filled > width {
		filled = width
	}
	if value > 0 && filled == 0 {
		filled = 1
	}
	filledStr := lipgloss.NewStyle().Foreground(ColorBar).Render(strings.Repeat("█", filled))
	emptyStr := lipgloss.NewStyle().Foreground(ColorBarBg).Render(strings.Repeat("░", width-filled))
	return filledStr + emptyStr
}

// PlainBar returns an unstyled bar for use in tests.
func PlainBar(value, max float64, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(width) * value / max)
	if filled > width {
		filled = width
	}
	if value > 0 && filled == 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
