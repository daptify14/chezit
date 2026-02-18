package tui

import (
	"fmt"
	"strings"
)

// --- Tab 2: Info (sub-views: Config, Full, Data, Doctor) ---

func (m Model) renderInfoTabContent() string {
	var b strings.Builder

	// Sub-view selector bar
	b.WriteString(m.renderInfoSubViewBar())
	b.WriteString("\n")

	view := &m.info.views[m.info.activeView]

	if view.loading {
		spinnerView := m.ui.loadingSpinner.View()
		fmt.Fprintf(&b, "  %s Loading...", spinnerView)
		return b.String()
	}

	if len(view.lines) == 0 {
		b.WriteString(activeTheme.DimText.Render("  No data"))
		return b.String()
	}

	listHeight := m.infoViewHeight()
	maxWidth := m.effectiveWidth() - 4
	view.ensureViewport(m.effectiveWidth(), listHeight)

	content := m.preRenderInfoContent(maxWidth)
	offset := view.viewport.YOffset()
	view.viewport.SetContent(content)
	view.viewport.SetYOffset(offset)

	b.WriteString(view.viewport.View())
	return b.String()
}

// preRenderInfoContent renders all info lines into a single styled string
// suitable for viewport.SetContent(). The viewport handles scroll/visibility.
func (m Model) preRenderInfoContent(maxWidth int) string {
	view := m.info.views[m.info.activeView]
	var b strings.Builder
	for i, line := range view.lines {
		b.WriteString(m.renderInfoLine(line, maxWidth))
		if i < len(view.lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) renderInfoSubViewBar() string {
	var parts []string
	for i, name := range m.info.viewNames {
		if i == m.info.activeView {
			styled := activeTheme.ActiveTab.Render("[" + name + "]")
			parts = append(parts, styled)
		} else {
			parts = append(parts, activeTheme.DimText.Render(" "+name+" "))
		}
	}
	arrows := activeTheme.DimText.Render("◀ ")
	arrowsR := activeTheme.DimText.Render(" ▶")
	return "  " + arrows + strings.Join(parts, activeTheme.DimText.Render("·")) + arrowsR
}

func (m Model) renderInfoLine(line string, maxWidth int) string {
	// Doctor sub-view: apply row coloring based on result prefix
	if m.info.activeView == infoViewDoctor {
		// chezmoi doctor output: "RESULT   CHECK   MESSAGE"
		// Result is the first non-space field: ok, info, warning, error
		prefix := strings.TrimSpace(line)
		field, _, _ := strings.Cut(prefix, " ")
		field = strings.ToLower(field)

		truncated := visualTruncate(line, maxWidth)
		switch field {
		case "ok":
			return "  " + activeTheme.SuccessFg.Render(truncated)
		case "info":
			return "  " + activeTheme.PrimaryFg.Render(truncated)
		case "warning":
			return "  " + activeTheme.WarningFg.Render(truncated)
		case "error":
			return "  " + activeTheme.DangerFg.Render(truncated)
		case "result":
			// Header row — render bold
			return "  " + activeTheme.BoldOnly.Render(truncated)
		}
		return "  " + truncated
	}

	// For highlighted content (Config, Full, Data), lines are pre-highlighted with ANSI codes
	return "  " + line
}

func (m Model) renderInfoStatusBar() string {
	view := m.info.views[m.info.activeView]
	viewName := m.info.viewNames[m.info.activeView]

	status := " " + viewName + " "
	if len(view.lines) > 0 {
		status = fmt.Sprintf(" %s | line %d/%d ", viewName, view.viewport.YOffset()+1, len(view.lines))
	}
	// Show format indicator for Full and Data
	if m.info.activeView == infoViewFull || m.info.activeView == infoViewData {
		status = fmt.Sprintf(" %s (%s) | line %d/%d ", viewName, m.info.format, view.viewport.YOffset()+1, max(len(view.lines), 1))
	}
	if m.ui.message != "" {
		status = " " + m.ui.message + " "
	}
	statusBar := activeTheme.StatusBar.Width(m.effectiveWidth()).Render(status)

	helpText := "h/l switch | ↑/↓ scroll | ^d/^u half-page"
	if m.info.activeView == infoViewFull || m.info.activeView == infoViewData {
		helpText += " | f format"
	}
	helpText += " | r refresh | tab switch | ? keys | esc quit"
	help := m.helpHint(helpText)
	return statusBar + "\n" + help
}
