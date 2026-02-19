package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

var warningDialogBase = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1, 2).
	Width(60)

// --- Diff View ---

func (m Model) renderDiffView() string {
	var b strings.Builder

	fileName := filepath.Base(m.diff.path)
	b.WriteString(renderBreadcrumb(append(m.breadcrumbParts(), fileName)...))
	b.WriteString("\n")
	b.WriteString(renderSeparator(m.effectiveWidth()))
	b.WriteString("\n")

	var detailParts []string
	if hint := diffDirectionHint(m.diff.sourceSection); hint != "" {
		detailParts = append(detailParts, hint)
	}
	if side := m.driftSideLabel(m.diff.sourceSection, m.diff.path); side != "" {
		detailParts = append(detailParts, side)
	}
	if len(detailParts) > 0 {
		b.WriteString(activeTheme.DimText.Render("  " + strings.Join(detailParts, " · ")))
		b.WriteString("\n")
	}

	if len(m.diff.lines) == 0 {
		b.WriteString(activeTheme.DimText.Render("  No diff content"))
	} else {
		diffHeight := m.chezmoiDiffViewHeight()
		m.diff.ensureViewport(m.effectiveWidth(), diffHeight)

		content := preRenderDiffContent(m.diff.lines, m.effectiveWidth())
		offset := m.diff.viewport.YOffset()
		m.diff.viewport.SetContent(content)
		m.diff.viewport.SetYOffset(offset)

		b.WriteString(m.diff.viewport.View())
	}

	if m.actions.show {
		b.WriteString("\n\n")
		b.WriteString(m.renderChezmoiActionsMenu())
	}
	b.WriteString("\n")
	b.WriteString(m.renderChezmoiDiffStatus())
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, b.String())
}

// --- Confirm View ---

func (m Model) renderConfirmScreen() string {
	label := m.overlays.confirmLabel
	if label == "" {
		label = "this action"
	}

	content := fmt.Sprintf(
		"\n  Are you sure you want to %s?\n\n  Press y to confirm, n or Esc to cancel.\n",
		label,
	)

	box := warningDialogBase.BorderForeground(activeTheme.Warning)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m Model) renderChezmoiDiffStatus() string {
	summary := diffSummary(m.diff.lines)
	if hint := diffDirectionHint(m.diff.sourceSection); hint != "" {
		summary = summary + " | " + hint
	}
	if side := m.driftSideLabel(m.diff.sourceSection, m.diff.path); side != "" {
		summary = summary + " | " + side
	}
	scrollInfo := ""
	if len(m.diff.lines) > 0 {
		scrollInfo = fmt.Sprintf(" | line %d/%d", m.diff.viewport.YOffset()+1, len(m.diff.lines))
	}
	status := fmt.Sprintf(" %s%s ", summary, scrollInfo)
	if m.diff.previewApply {
		status = " Preview: chezmoi apply" + scrollInfo + " "
	}
	if m.ui.message != "" {
		status = " " + m.ui.message + " "
	}
	statusBar := activeTheme.StatusBar.Width(m.effectiveWidth()).Render(status)

	var help string
	switch {
	case m.diff.previewApply:
		help = m.helpHint("↑/↓ scroll | ^d/^u half-page | enter apply | esc cancel")
	case m.actions.show:
		help = m.helpHint("↑/↓ navigate | enter select | esc back")
	default:
		help = m.helpHint("↑/↓ scroll | ^d/^u half-page | g top | G bottom | e edit | a actions | esc back")
	}
	return statusBar + "\n" + help
}

// --- View Picker Menu ---

func (m Model) renderViewPickerMenu() string {
	var b strings.Builder
	b.WriteString("\n  Files View & Filter\n")
	b.WriteString("  " + strings.Repeat("─", 44) + "\n")
	b.WriteString(activeTheme.DimText.Render("  Views"))
	b.WriteString("\n")

	row := 0
	for i, item := range m.overlays.viewPickerItems {
		selector := "( )"
		if item.mode == m.overlays.viewPickerPendingMode {
			selector = "(●)"
		}

		countLabel := "…"
		if item.count >= 0 {
			countLabel = strconv.Itoa(item.count)
		}

		line := fmt.Sprintf("  %s %d. %-10s %s", selector, i+1, item.label, countLabel)
		if row == m.overlays.viewPickerCursor {
			b.WriteString(activeTheme.Selected.Render("> " + strings.TrimSpace(line)))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
		row++
	}

	b.WriteString("\n")
	b.WriteString(activeTheme.DimText.Render("  Type Filters"))
	b.WriteString("\n")
	if m.overlays.viewPickerPendingMode == managedViewIgnored {
		b.WriteString(activeTheme.DimText.Render("  filters not supported for ignored"))
		b.WriteString("\n")
	}

	for _, cat := range m.overlays.filterCategories {
		var line string
		if cat.entryType == "" {
			line = "  ↺ " + cat.label
		} else {
			check := "[ ]"
			if cat.enabled {
				check = "[x]"
			}
			line = fmt.Sprintf("  %s %s", check, cat.label)
		}

		switch {
		case row == m.overlays.viewPickerCursor:
			b.WriteString(activeTheme.Selected.Render("> " + strings.TrimSpace(line)))
		case !cat.enabled && cat.entryType != "":
			b.WriteString(activeTheme.DimText.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
		row++
	}

	b.WriteString("\n")
	b.WriteString(activeTheme.HintText.Render("↑/↓ navigate | space toggle/select | 1-4 quick view | enter apply | esc back"))

	box := activeTheme.Filter

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(b.String()))
}

func managedViewModeLabel(mode managedViewMode) string {
	switch mode {
	case managedViewIgnored:
		return "Ignored"
	case managedViewUnmanaged:
		return "Unmanaged"
	case managedViewAll:
		return "All"
	default:
		return "Managed"
	}
}

// --- Filter Overlay ---

func (m Model) renderFilterOverlay() string {
	var b strings.Builder
	b.WriteString("\n  Entry Type Filter\n")
	b.WriteString("  " + strings.Repeat("─", 30) + "\n")
	if m.filesTab.viewMode == managedViewIgnored {
		b.WriteString(activeTheme.DimText.Render("  filters not supported for ignored"))
		b.WriteString("\n")
	}

	for i, cat := range m.overlays.filterCategories {
		isSelected := i == m.overlays.filterCursor

		var check string
		switch {
		case cat.entryType == "":
			// "Reset all" sentinel
			check = "  "
		case cat.enabled:
			check = "[x]"
		default:
			check = "[ ]"
		}

		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%s %s", cursor, check, cat.label)

		if cat.entryType == "" && i > 0 {
			b.WriteString("  " + strings.Repeat("─", 30) + "\n")
		}

		switch {
		case isSelected:
			b.WriteString(activeTheme.Selected.Render(line))
		case !cat.enabled && cat.entryType != "":
			b.WriteString(activeTheme.DimText.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n  space toggle | enter apply | esc back\n")

	box := activeTheme.Filter

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(b.String()))
}

// --- Search Box ---

// renderChezmoiSearchBoxWidth renders the search box constrained to a caller-provided width.
func (m Model) renderChezmoiSearchBoxWidth(totalWidth int) string {
	boxWidth := max(totalWidth-4, 40)

	var content string
	switch {
	case m.filterInput.Focused():
		content = m.filterInput.View()
	case m.filterInput.Value() != "":
		content = activeTheme.DimText.Render("Filter: ") + m.filterInput.Value()
	default:
		content = activeTheme.DimText.Render("/ to search files...")
	}

	box := activeTheme.Filter.Width(boxWidth)

	return box.Render(content)
}
