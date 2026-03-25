package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// preRenderDiffContent renders all diff lines into a single styled string
// suitable for viewport.SetContent(). The viewport handles scroll/visibility.
// When pagerApplied is true, lines already contain ANSI colors from an external
// pager and are rendered as-is with only truncation/padding.
func preRenderDiffContent(lines []string, width int, pagerApplied bool) string {
	var b strings.Builder
	for i, line := range lines {
		b.WriteString("  ")
		if pagerApplied {
			b.WriteString(visualTruncate(line, width-2))
		} else {
			style := diffLineStyle(line)
			b.WriteString(style.Render(visualTruncate(line, width-2)))
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func diffLineStyle(line string) lipgloss.Style {
	switch {
	case strings.HasPrefix(line, "+++"):
		return activeTheme.BoldPrimary
	case strings.HasPrefix(line, "---"):
		return activeTheme.BoldPrimary
	case strings.HasPrefix(line, "+"):
		return activeTheme.SuccessFg
	case strings.HasPrefix(line, "-"):
		return activeTheme.DangerFg
	case strings.HasPrefix(line, "@@"):
		return activeTheme.AccentFg
	case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "):
		return activeTheme.DimText
	default:
		return activeTheme.Normal
	}
}

// diffDirectionHint returns a short label explaining what - and + mean
// in the diff for the given section. Returns "" for commit sections.
func diffDirectionHint(section changesSection) string {
	switch section {
	case changesSectionDrift:
		return "- on disk  + source"
	case changesSectionUnstaged:
		return "- committed  + working tree"
	case changesSectionStaged:
		return "- committed  + staged"
	default:
		return ""
	}
}

func diffSummary(lines []string) string {
	var added, removed int
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			// headers, skip
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	return fmt.Sprintf("+%d/-%d lines", added, removed)
}
