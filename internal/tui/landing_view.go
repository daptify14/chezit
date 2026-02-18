package tui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Logo art: unicode block letters spelling CHEZIT.
const chezitLogo = "" +
	"  ██████ ██   ██ ███████ ██████ ██ ████████\n" +
	" ██      ██   ██ ██         ███ ██    ██\n" +
	" ██      ███████ █████     ███  ██    ██\n" +
	" ██      ██   ██ ██       ███   ██    ██\n" +
	"  ██████ ██   ██ ███████ ██████ ██    ██"

const chezitTagline = "chezmoi TUI manager"

var landingPadStyle = lipgloss.NewStyle().Padding(0, 1)

// landingItem represents one selectable row on the landing page.
type landingItem struct {
	label       string // tab name: "Status", "Files", etc.
	description string // short description shown on landing page
	tab         int    // index into tabNames
}

// renderLandingScreen composes the full welcome banner landing page.
func (m Model) renderLandingScreen() string {
	var sections []string

	sections = append(sections,
		renderChezitLogo(),
		renderLandingTagline(),
		"", // spacer
		m.renderSummaryBox(),
		"", // spacer
		m.renderLandingList(),
		"", // spacer
		renderLandingHelpBar(),
	)

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			activeTheme.CenteredBox.Render(content))
	}
	return content
}

// renderChezitLogo renders the logo with ch-EZ-it coloring:
// CH and IT in Peach, EZ in Green ("easy").
func renderChezitLogo() string {
	t := &activeTheme
	chStyle := t.BoldWarning
	ezStyle := t.BoldSuccess
	itStyle := chStyle

	lines := strings.Split(chezitLogo, "\n")
	var b strings.Builder
	for i, line := range lines {
		runes := []rune(line)
		n := len(runes)

		// Split at rune boundaries: CH=[0,16) EZ=[16,31) IT=[31,)
		chEnd := min(16, n)
		ezEnd := min(31, n)

		b.WriteString(chStyle.Render(string(runes[:chEnd])))
		if chEnd < n {
			b.WriteString(ezStyle.Render(string(runes[chEnd:ezEnd])))
		}
		if ezEnd < n {
			b.WriteString(itStyle.Render(string(runes[ezEnd:])))
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return landingPadStyle.Render(b.String())
}

// renderLandingTagline renders the subtitle below the logo.
func renderLandingTagline() string {
	return landingPadStyle.Foreground(activeTheme.SubtleText).Render(chezitTagline)
}

// isAllInSync returns true when there are no pending changes across chezmoi and git.
func (m Model) isAllInSync() bool {
	return len(m.status.filteredFiles) == 0 &&
		len(m.status.gitStagedFiles) == 0 &&
		len(m.status.gitUnstagedFiles) == 0 &&
		m.status.gitInfo.Ahead == 0 &&
		m.status.gitInfo.Behind == 0
}

// renderSummaryBox renders a bordered box with git and chezmoi summary data.
// All three states (loading, in-sync, has-changes) output exactly 3 rows
// to prevent layout shifts during async data loading.
func (m Model) renderSummaryBox() string {
	t := &activeTheme
	isLoading := !m.landing.statsReady

	// Match the total content width from formatStatsRow so spinner and
	// centered text align with the column-based rows.
	const totalRowWidth = 14 + 4 + 12 // leftWidth + len(gap) + rightWidth

	var rows []string
	var borderColor color.Color

	switch {
	case isLoading:
		// Spinner + label on row 1, two empty rows to hold height
		loadingMsg := m.ui.loadingSpinner.View() + " " +
			t.HintText.Render("checking…")
		rows = append(rows,
			lipgloss.NewStyle().Width(totalRowWidth).Render(loadingMsg),
			"",
			"",
		)
		borderColor = t.Dim
	case m.isAllInSync():
		// Row 1: branch + managed count
		branch := m.status.gitInfo.Branch
		if branch == "" {
			branch = "—"
		}
		managed := formatCountWithLabel(t, len(m.filesTab.views[managedViewManaged].files), "managed")
		rows = append(rows,
			formatStatsRow(t.Normal.Render(branch), managed),
			"", // spacer
			lipgloss.NewStyle().Width(totalRowWidth).Align(lipgloss.Center).Render(
				t.SuccessFg.Render("all in sync"),
			),
		)
		borderColor = t.Success
	default:
		// Full 3-row stats view
		branch := m.status.gitInfo.Branch
		if branch == "" {
			branch = "—"
		}
		aheadBehind := formatAheadBehind(t, m.status.gitInfo.Ahead, m.status.gitInfo.Behind)
		rows = append(rows, formatStatsRow(t.Normal.Render(branch), aheadBehind))

		left2 := formatCountWithLabel(t, len(m.status.filteredFiles), "changed")
		right2 := formatCountWithLabel(t, len(m.status.gitStagedFiles), "staged")
		rows = append(rows, formatStatsRow(left2, right2))

		left3 := formatCountWithLabel(t, len(m.filesTab.views[managedViewManaged].files), "managed")
		right3 := formatCountWithLabel(t, len(m.status.gitUnstagedFiles), "unstaged")
		rows = append(rows, formatStatsRow(left3, right3))
		borderColor = t.Warning
	}

	content := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 4).
		Align(lipgloss.Center).
		Render(content)
}

// formatStatsRow creates a properly aligned row with left and right columns.
func formatStatsRow(left, right string) string {
	const (
		leftWidth  = 14
		rightWidth = 12
		gap        = "    "
	)
	// Pad left column to fixed width
	leftPadded := lipgloss.NewStyle().Width(leftWidth).Align(lipgloss.Left).Render(left)
	// Pad right column to fixed width
	rightPadded := lipgloss.NewStyle().Width(rightWidth).Align(lipgloss.Left).Render(right)
	return leftPadded + gap + rightPadded
}

// formatAheadBehind renders ahead/behind indicators.
func formatAheadBehind(t *Theme, ahead, behind int) string {
	var parts []string
	if ahead > 0 {
		parts = append(parts, t.SuccessFg.Render(fmt.Sprintf("↑%d", ahead)))
	} else {
		parts = append(parts, t.HintText.Render("↑0"))
	}
	if behind > 0 {
		parts = append(parts, t.WarningFg.Render(fmt.Sprintf("↓%d", behind)))
	} else {
		parts = append(parts, t.HintText.Render("↓0"))
	}
	return strings.Join(parts, " ")
}

// formatCountWithLabel renders a count with its label, colored based on value.
func formatCountWithLabel(t *Theme, count int, label string) string {
	style := t.HintText
	if count > 0 {
		if label == "changed" || label == "unstaged" {
			style = t.WarningFg
		} else {
			style = t.Normal
		}
	}
	return style.Render(fmt.Sprintf("%d %s", count, label))
}

// renderLandingList renders the selectable tab list with aligned columns.
func (m Model) renderLandingList() string {
	t := &activeTheme
	items := m.landingItems()

	// Fixed column widths for clean table alignment
	const numWidth = 2
	const cursorLabelWidth = 12 // includes cursor (2 chars) + label
	const gap = "    "

	var rows []string
	for i, item := range items {
		cursor := "  "
		labelStyle := t.Normal
		if i == m.landing.cursor {
			cursor = "> "
			labelStyle = t.Selected
		}

		num := t.BoldAccent.
			Width(numWidth).Align(lipgloss.Right).Render(strconv.Itoa(i + 1))
		label := labelStyle.Width(cursorLabelWidth).Align(lipgloss.Left).Render(cursor + item.label)
		desc := t.HintText.Render(item.description)

		row := num + " " + label + gap + desc
		rows = append(rows, row)
	}

	// Pad all rows to uniform width so JoinVertical centering
	// shifts every line by the same amount, preserving column alignment.
	maxW := 0
	for _, row := range rows {
		if w := lipgloss.Width(row); w > maxW {
			maxW = w
		}
	}
	for i, row := range rows {
		rows[i] = lipgloss.NewStyle().Width(maxW).Align(lipgloss.Left).Render(row)
	}

	return strings.Join(rows, "\n")
}

// landingItems returns the list of selectable items for the landing page,
// built from the available tab names.
func (m Model) landingItems() []landingItem {
	items := make([]landingItem, len(m.tabNames))
	for i, name := range m.tabNames {
		items[i] = landingItem{
			label:       name,
			description: landingItemDescription(name),
			tab:         i,
		}
	}
	return items
}

// landingItemDescription returns a short description for a landing page item.
func landingItemDescription(label string) string {
	switch label {
	case "Status":
		return "View changes and manage git staging"
	case "Files":
		return "Browse and edit managed dotfiles"
	case "Info":
		return "View configuration, data, and diagnostics"
	case "Commands":
		return "Run common chezmoi operations"
	}
	return ""
}

// handleLandingKeys processes key events on the landing page.
func (m Model) handleLandingKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	itemCount := len(m.tabNames)

	switch {
	case key.Matches(msg, ChezSharedKeys.Up):
		m.landing.cursor = (m.landing.cursor - 1 + itemCount) % itemCount
		return m, nil
	case key.Matches(msg, ChezSharedKeys.Down):
		m.landing.cursor = (m.landing.cursor + 1) % itemCount
		return m, nil
	case key.Matches(msg, ChezSharedKeys.Enter):
		return m.enterTabFromLanding(m.landing.cursor)
	case key.Matches(msg, ChezSharedKeys.Tab1):
		if itemCount > 0 {
			return m.enterTabFromLanding(0)
		}
	case key.Matches(msg, ChezSharedKeys.Tab2):
		if itemCount > 1 {
			return m.enterTabFromLanding(1)
		}
	case key.Matches(msg, ChezSharedKeys.Tab3):
		if itemCount > 2 {
			return m.enterTabFromLanding(2)
		}
	case key.Matches(msg, ChezSharedKeys.Tab4):
		if itemCount > 3 {
			return m.enterTabFromLanding(3)
		}
	case key.Matches(msg, ChezSharedKeys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

// enterTabFromLanding transitions from the landing page to a specific tab.
func (m Model) enterTabFromLanding(tabIndex int) (tea.Model, tea.Cmd) {
	if tabIndex >= 0 && tabIndex < len(m.tabNames) {
		m.view = StatusScreen
		cmd := m.switchTab(tabIndex)
		return m, cmd
	}
	return m, nil
}

// renderLandingHelpBar renders navigation hints for the landing page.
func renderLandingHelpBar() string {
	t := &activeTheme
	sep := t.HintText.Render("  ")

	hints := []struct{ key, action string }{
		{"↑/↓", "navigate"},
		{"1-4", "jump"},
		{"enter", "open"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range hints {
		k := t.BoldPrimary.Render(h.key)
		a := t.HintText.Render(h.action)
		parts = append(parts, k+" "+a)
	}
	return strings.Join(parts, sep)
}
