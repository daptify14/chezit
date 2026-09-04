package tui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
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
		m.renderLandingMessage(), // spacer, or the message line when set
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

// renderLandingMessage fills the spacer above the help bar with the message
// line when one is set, so feedback for f appears without shifting the layout.
func (m Model) renderLandingMessage() string {
	if m.ui.message == "" {
		return ""
	}
	return activeTheme.HintText.Render(m.ui.message)
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

// isAllInSync requires nothing pending and a trustworthy comparison: a
// successful fetch this session, auto_fetch off, or no upstream to compare.
func (m Model) isAllInSync() bool {
	if !m.hasNothingPending() {
		return false
	}
	switch m.status.gitInfo.Sync {
	case chezmoi.GitSyncSynced:
		if m.status.fetchInProgress || m.status.gitReadPending {
			return false
		}
		return m.status.fetchOutcome == fetchOK || m.status.fetchOutcome == fetchOff
	case chezmoi.GitSyncNoUpstream:
		return true // nothing to compare against
	default:
		return false
	}
}

// hasNothingPending reports no drift, staged/unstaged files, or ahead/behind commits.
func (m Model) hasNothingPending() bool {
	return len(m.status.filteredFiles) == 0 &&
		len(m.status.gitStagedFiles) == 0 &&
		len(m.status.gitUnstagedFiles) == 0 &&
		m.status.gitInfo.Ahead == 0 &&
		m.status.gitInfo.Behind == 0
}

// Summary box column layout. The right column must fit "not fetched · f".
const (
	statsLeftWidth  = 14
	statsRightWidth = 16
	statsGap        = "    "
	statsRowWidth   = statsLeftWidth + len(statsGap) + statsRightWidth
)

// renderSummaryBox renders a bordered box with git and chezmoi summary data.
// All states output exactly 4 rows to prevent layout shifts during async
// loading; row 4 is always the upstream freshness row.
func (m Model) renderSummaryBox() string {
	t := &activeTheme
	isLoading := !m.landing.statsReady

	var rows []string
	var borderColor color.Color

	switch {
	case isLoading:
		// Spinner + label on row 1, three empty rows to hold height
		loadingMsg := m.ui.loadingSpinner.View() + " " +
			t.HintText.Render("checking…")
		rows = append(rows,
			lipgloss.NewStyle().Width(statsRowWidth).Render(loadingMsg),
			"",
			"",
			"",
		)
		borderColor = t.Dim
	case m.isAllInSync():
		// Row 1: branch + managed count
		managed := formatCountWithLabel(t, len(m.filesTab.views[managedViewManaged].files), "managed")
		rows = append(rows,
			formatStatsRow(t.Normal.Render(m.landingBranch()), managed),
			"", // spacer
			lipgloss.NewStyle().Width(statsRowWidth).Align(lipgloss.Center).Render(
				t.SuccessFg.Render("all in sync"),
			),
			m.renderUpstreamRow(t),
		)
		borderColor = t.Success
	default:
		// Full stats view
		aheadBehind := formatAheadBehind(t, m.status.gitInfo.Ahead, m.status.gitInfo.Behind)
		rows = append(rows, formatStatsRow(t.Normal.Render(m.landingBranch()), aheadBehind))

		left2 := formatCountWithLabel(t, len(m.status.filteredFiles), "changed")
		right2 := formatCountWithLabel(t, len(m.status.gitStagedFiles), "staged")
		rows = append(rows, formatStatsRow(left2, right2))

		left3 := formatCountWithLabel(t, len(m.filesTab.views[managedViewManaged].files), "managed")
		right3 := formatCountWithLabel(t, len(m.status.gitUnstagedFiles), "unstaged")
		rows = append(rows, formatStatsRow(left3, right3), m.renderUpstreamRow(t))

		// Warn only for real pending work; a failed or pending fetch alone
		// stays neutral so a flaky network does not paint the box orange.
		borderColor = t.Dim
		if !m.hasNothingPending() {
			borderColor = t.Warning
		}
	}

	content := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 4).
		Align(lipgloss.Center).
		Render(content)
}

// landingBranch returns the branch name or a placeholder when unknown.
func (m Model) landingBranch() string {
	if m.status.gitInfo.Branch == "" {
		return "—"
	}
	return m.status.gitInfo.Branch
}

// renderUpstreamRow renders the summary box's fourth row.
func (m Model) renderUpstreamRow(t *Theme) string {
	return formatStatsRow(t.Normal.Render("upstream"), m.formatUpstreamFreshness(t))
}

// formatUpstreamFreshness renders the right column of the upstream row.
func (m Model) formatUpstreamFreshness(t *Theme) string {
	switch {
	case m.status.gitInfo.Sync == chezmoi.GitSyncNoUpstream:
		return t.HintText.Render("none")
	case m.status.fetchInProgress:
		return m.ui.loadingSpinner.View() + " " + t.HintText.Render("checking…")
	}
	return m.upstreamOutcomeToken(t, false)
}

// formatStatsRow creates a properly aligned row with left and right columns.
func formatStatsRow(left, right string) string {
	// Pad left column to fixed width
	leftPadded := lipgloss.NewStyle().Width(statsLeftWidth).Align(lipgloss.Left).Render(left)
	// Pad right column to fixed width
	rightPadded := lipgloss.NewStyle().Width(statsRightWidth).Align(lipgloss.Left).Render(right)
	return leftPadded + statsGap + rightPadded
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
	case key.Matches(msg, ChezChangesKeys.Fetch):
		next, cmd := m.startFetch(true)
		return next, cmd
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
		{"f", "fetch"},
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
