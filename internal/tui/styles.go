package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const appName = "chezit"

var breadcrumbPadStyle = lipgloss.NewStyle().Padding(0, 1)

func renderBreadcrumb(segments ...string) string {
	var parts []string
	chevron := activeTheme.Branch.Render(" > ")
	style := activeTheme.HintText.Bold(true)
	for _, seg := range segments {
		parts = append(parts, style.Render(seg))
	}
	content := strings.Join(parts, chevron)
	return breadcrumbPadStyle.Render(content)
}

func renderSeparator(width int) string {
	return activeTheme.Branch.Render(strings.Repeat("─", width))
}
