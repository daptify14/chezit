package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View implements tea.Model by rendering the current screen state.
func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if m.ui.mouseCapture {
		v.MouseMode = tea.MouseModeCellMotion
	} else {
		v.MouseMode = tea.MouseModeNone
	}
	v.KeyboardEnhancements.ReportEventTypes = true

	if m.startupErr != nil {
		v.Content = m.renderStartupError()
		return v
	}

	if m.view == LandingScreen {
		v.Content = m.renderLandingScreen()
		return v
	}

	if m.overlays.showHelp {
		v.Content = m.renderHelp()
		return v
	}

	if m.overlays.showViewPicker {
		v.Content = m.renderViewPickerMenu()
		return v
	}

	if m.overlays.showFilterOverlay {
		v.Content = m.renderFilterOverlay()
		return v
	}

	if m.ui.loading {
		v.Content = m.renderChezmoiLoading()
		return v
	}

	switch m.view {
	case DiffScreen:
		v.Content = m.renderDiffView()
		return v
	case ConfirmScreen:
		v.Content = m.renderConfirmScreen()
		return v
	case CommitScreen:
		v.Content = m.renderCommitScreen()
		return v
	}

	var b strings.Builder

	b.WriteString(renderBreadcrumb(m.breadcrumbParts()...))
	b.WriteString("\n")
	b.WriteString(renderSeparator(m.effectiveWidth()))
	b.WriteString("\n")
	b.WriteString(m.renderChezmoiTabBar())
	b.WriteString("\n")

	switch m.activeTabName() {
	case "Status":
		b.WriteString(m.renderChangesTabWithPanel())
	case "Files":
		b.WriteString(m.renderManagedTabWithPanel())
	case "Info":
		b.WriteString(m.renderInfoTabContent())
	case "Commands":
		b.WriteString(m.renderCommandsTabContent())
	}

	b.WriteString("\n")
	b.WriteString(m.renderChezmoiTabStatus())
	v.Content = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, b.String())
	return v
}

func (m Model) renderChezmoiTabBar() string {
	return renderTabs(m.tabNames, m.activeTab)
}

func (m Model) renderChezmoiLoading() string {
	spinnerView := m.ui.loadingSpinner.View()
	content := fmt.Sprintf("%s Loading chezmoi status...", spinnerView)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderStartupError() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(activeTheme.Danger).Bold(true).Render("chezit failed to initialize"))
	b.WriteString("\n\n")
	b.WriteString("Unable to resolve chezmoi target path (`chezmoi target-path`).")
	b.WriteString("\n")
	b.WriteString(m.startupErr.Error())
	b.WriteString("\n\n")
	b.WriteString("Fix your chezmoi setup and retry. Press q, Esc, or Enter to quit.")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeTheme.Danger).
		Padding(1, 2).
		MaxWidth(max(60, m.effectiveWidth()-4))

	return lipgloss.Place(m.effectiveWidth(), max(12, m.height), lipgloss.Center, lipgloss.Center, box.Render(b.String()))
}

// --- Status Bars ---

func (m Model) renderChezmoiTabStatus() string {
	switch m.activeTabName() {
	case "Status":
		return m.renderChangesStatusBar()
	case "Files":
		return m.renderManagedStatusBar()
	case "Info":
		return m.renderInfoStatusBar()
	case "Commands":
		return m.renderCommandsStatusBar()
	default:
		return m.renderChangesStatusBar()
	}
}

func (m Model) listPreviewHint() string {
	if m.panel.shouldShow(m.width) {
		return " | v switch diff/content | l/→ focus preview | p hide preview"
	}
	return " | p show preview"
}

func (m Model) focusedPreviewHelp(backLabel string) string {
	return m.helpHint("↑/↓ scroll | ^d/^u half | g/G top/bottom | v switch diff/content | h/← " + backLabel + " | p hide preview")
}

func (m Model) helpHint(raw string) string {
	return styledHelpResponsive(raw, m.effectiveWidth())
}

// --- Help ---

func (m Model) renderHelp() string {
	return buildHelpOverlay(m.width, m.height, m.overlays.helpScroll, m.helpOverlayFooter(), m.helpOverlayRows()...)
}

func (m Model) helpOverlayFooter() string {
	return "↑/↓ scroll | ^d/^u half-page | g/G top/bottom | ?/esc close"
}

func (m Model) helpOverlayMaxScroll() int {
	return helpOverlayMaxScroll(m.width, m.height, m.helpOverlayFooter(), m.helpOverlayRows()...)
}

func (m Model) helpOverlayRows() [][]HelpSection {
	tab := m.activeTabName()
	rows := [][]HelpSection{
		{
			{
				Title: "Global",
				Entries: []HelpEntry{
					{"↑/↓", "Navigate"},
					{"Tab", "Switch tabs"},
					{"1-4", "Jump to tab"},
					{"?", "Open/close keys"},
					{"m", m.mouseModeHelpLabel()},
					{"esc", "Back"},
					{"q", "Quit"},
				},
			},
		},
	}

	if m.view == DiffScreen {
		rows = append(rows, []HelpSection{
			{
				Title: "Diff View",
				Entries: []HelpEntry{
					{"↑/↓", "Scroll"},
					{"^d/^u", "Half-page down/up"},
					{"g/G", "Top / Bottom"},
					{"e", "Edit file"},
					{"a", "Actions menu"},
					{"esc", "Back to list"},
				},
			},
		})
		return rows
	}

	switch tab {
	case "Status":
		rows = append(rows, []HelpSection{
			{
				Title: "Changes",
				Entries: []HelpEntry{
					{"/", "Filter/search"},
					{"enter", "Diff / toggle section"},
					{"S-↑/S-↓", "Select range"},
					{"s", "Re-add (when available) / stage"},
					{"u", "Unstage"},
					{"x", "Discard / Undo commit"},
					{"c", "Commit staged"},
					{"P", "Push"},
					{"a", "Actions menu"},
					{"r", "Refresh"},
				},
			},
		})
	case "Files":
		enterLabel := "Open actions"
		if m.filesTab.treeView {
			enterLabel = "Open/toggle directory"
		}
		rows = append(rows, []HelpSection{
			{
				Title: "Files",
				Entries: []HelpEntry{
					{"/", "Filter/search"},
					{"enter", enterLabel},
					{"a", "Actions menu"},
					{"t", "Tree/flat toggle"},
					{"f", "View/filter overlay"},
					{"r", "Refresh"},
				},
			},
		})
	case "Info":
		rows = append(rows, []HelpSection{
			{
				Title: "Info",
				Entries: []HelpEntry{
					{"h/l ←/→", "Switch view"},
					{"↑/↓", "Scroll"},
					{"^d/^u", "Half-page down/up"},
					{"^f/^b", "Full-page down/up"},
					{"g/G", "Top / Bottom"},
					{"f", "Toggle format (yaml/json)"},
					{"r", "Refresh"},
				},
			},
		})
	case "Commands":
		rows = append(rows, []HelpSection{
			{
				Title: "Commands",
				Entries: []HelpEntry{
					{"↑/↓", "Navigate"},
					{"enter", "Run command"},
					{"d", "Dry run (if available)"},
					{"g/G", "Top / Bottom"},
				},
			},
		})
	}

	if tab == "Status" || tab == "Files" {
		rows = append(rows, []HelpSection{
			{
				Title: "Preview",
				Entries: []HelpEntry{
					{"p", "Show/hide preview"},
					{"l/→", "Focus preview"},
					{"h/←", "Back to list"},
					{"v", "Switch diff/content"},
				},
				Notes: []string{
					"Preview appears when terminal is wide enough.",
				},
			},
		})
	}

	return rows
}

func (m Model) mouseModeHelpLabel() string {
	if m.ui.mouseCapture {
		return "Mouse on (wheel/click)"
	}
	return "Copy mode (drag select)"
}

// --- Actions Menu ---

func (m Model) renderChezmoiActionsMenu() string {
	if len(m.actions.items) == 0 {
		return ""
	}

	title := " Chezmoi Actions "
	if m.view == DiffScreen {
		title = fmt.Sprintf(" %s ", filepath.Base(m.diff.path))
	} else {
		if m.status.selectionActive {
			if selected := m.selectedActionableCount(); selected > 0 {
				title = fmt.Sprintf(" %d selected ", selected)
			}
		} else {
			row := m.currentChangesRow()
			switch {
			case row.driftFile != nil:
				title = fmt.Sprintf(" %s ", filepath.Base(row.driftFile.Path))
			case row.gitFile != nil:
				title = fmt.Sprintf(" %s ", filepath.Base(row.gitFile.Path))
			}
		}
	}

	items := make([]menuItem, len(m.actions.items))
	for i, item := range m.actions.items {
		items[i] = menuItem{
			label:       item.label,
			description: item.description,
			disabled:    item.disabled || item.action == chezmoiActionNone,
		}
	}
	return renderActionsMenu(title, items, m.actions.cursor)
}
