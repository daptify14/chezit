package tui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

var sectionHeaderBase = lipgloss.NewStyle().Bold(true)

// renderChangesTabContentWidth renders the changes tab list at a specific width.
func (m Model) renderChangesTabContentWidth(maxWidth int) string {
	var b strings.Builder

	b.WriteString(m.renderChezmoiSearchBoxWidth(maxWidth))
	b.WriteString("\n")

	if m.status.loadingGit {
		spinnerView := m.ui.loadingSpinner.View()
		fmt.Fprintf(&b, "  %s Loading git status...", spinnerView)
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderGitInfoHeader())
		b.WriteString("\n")
	}

	if len(m.status.changesRows) == 0 {
		b.WriteString(activeTheme.DimText.Render("  No changes"))
		return b.String()
	}

	listHeight := m.chezmoiChangesListHeight()
	start, end := visibleRange(len(m.status.changesRows), m.status.changesCursor, listHeight)
	visible := m.status.changesRows[start:end]
	rowMaxWidth := maxWidth - 2

	for i, row := range visible {
		idx := start + i
		isSelected := idx == m.status.changesCursor
		isRangeSelected := m.isStatusRowRangeSelected(idx) && !isSelected

		switch {
		case row.isHeader:
			b.WriteString(m.renderChangesSectionHeaderWidth(row.section, isSelected, maxWidth))
		case row.driftFile != nil:
			line := m.renderChangesDriftRow(*row.driftFile, isSelected, rowMaxWidth)
			if isRangeSelected {
				line = markStatusRangeRow(line)
			}
			b.WriteString(line)
		case row.gitFile != nil:
			staged := row.section == changesSectionStaged
			line := m.renderGitFileRow(*row.gitFile, isSelected, staged, rowMaxWidth)
			if isRangeSelected {
				line = markStatusRangeRow(line)
			}
			b.WriteString(line)
		case row.commit != nil:
			line := m.renderCommitRow(*row.commit, isSelected, rowMaxWidth)
			if isRangeSelected {
				line = markStatusRangeRow(line)
			}
			b.WriteString(line)
		}

		if i < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	if m.actions.show {
		b.WriteString("\n\n")
		b.WriteString(m.renderChezmoiActionsMenu())
	}

	return b.String()
}

// renderChangesSectionHeaderWidth renders a section header at a given width.
func (m Model) renderChangesSectionHeaderWidth(section changesSection, selected bool, maxWidth int) string {
	collapsed := m.status.sectionCollapsed[section]
	arrow := "▼"
	if collapsed {
		arrow = "▶"
	}

	var label string
	var count int
	var sectionColor color.Color

	switch section {
	case changesSectionDrift:
		label = "Local Drift"
		count = len(m.status.filteredFiles)
		sectionColor = activeTheme.Primary
	case changesSectionUnstaged:
		label = "Unstaged"
		count = len(m.status.gitUnstagedFiles)
		sectionColor = activeTheme.Warning
	case changesSectionStaged:
		label = "Staged"
		count = len(m.status.gitStagedFiles)
		sectionColor = activeTheme.Success
	case changesSectionUnpushed:
		label = "Unpushed Commits"
		count = len(m.status.unpushedCommits)
		sectionColor = activeTheme.Accent
	case changesSectionIncoming:
		label = "Incoming"
		count = len(m.status.incomingCommits)
		if m.status.fetchInProgress {
			label = "Incoming (fetching...)"
		} else {
			label += " " + m.incomingSectionActionHint()
		}
		sectionColor = activeTheme.Primary
	}

	header := fmt.Sprintf("  %s %s (%d)", arrow, label, count)
	if selected {
		return activeTheme.Selected.Width(maxWidth - 2).Render(header)
	}
	return sectionHeaderBase.Foreground(sectionColor).Render(header)
}

func (m Model) incomingSectionActionHint() string {
	// Promote pull only when there is something to pull and writes are allowed.
	if len(m.status.incomingCommits) > 0 && !m.service.IsReadOnly() {
		return "[p pull]"
	}
	return "[f fetch]"
}

const chezmoiColStatus = 4

const statusBlankSlot = '·'

func statusIndicatorRune(r rune) rune {
	if r == ' ' {
		return statusBlankSlot
	}
	return r
}

func driftIndicatorStyle(r rune) lipgloss.Style {
	if r == 'D' {
		return activeTheme.DangerFg
	}
	return activeTheme.PrimaryFg
}

func chezmoiStatusIndicator(src, dest rune, selected bool) string {
	left := string(statusIndicatorRune(src))
	right := string(statusIndicatorRune(dest))
	if selected {
		return left + right
	}
	return driftIndicatorStyle(src).Render(left) + driftIndicatorStyle(dest).Render(right)
}

func (m Model) renderChangesTabContent() string {
	return m.renderChangesTabContentWidth(m.effectiveWidth())
}

func (m Model) renderChangesDriftRow(f chezmoi.FileStatus, isSelected bool, maxWidth int) string {
	indicator := chezmoiStatusIndicator(f.SourceStatus, f.DestStatus, isSelected)
	icon := renderFileIcon(filepath.Base(f.Path), false, isSelected, m.iconMode)
	displayPath := shortenPath(f.Path, m.targetPath)

	tmplTag := ""
	if f.IsTemplate {
		tmplTag = " (tmpl)"
	}

	pathWidth := max(maxWidth-chezmoiColStatus-len(tmplTag)-6, 20)
	displayPath = visualTruncate(icon+displayPath, pathWidth)

	cursor := "    "
	if isSelected {
		cursor = "  > "
	}

	var suffix string
	if f.IsTemplate && !isSelected {
		suffix = " " + activeTheme.AccentFg.Render("(tmpl)")
	} else if f.IsTemplate {
		suffix = " (tmpl)"
	}

	line := fmt.Sprintf("%s%-*s  %s%s",
		cursor,
		chezmoiColStatus, indicator,
		displayPath,
		suffix)

	if isSelected {
		return activeTheme.Selected.Width(maxWidth).Render(line)
	}
	return activeTheme.Normal.Render(line)
}

func markStatusRangeRow(line string) string {
	if before, after, ok := strings.Cut(line, "    "); ok {
		return before + "  * " + after
	}
	return line
}

func (m Model) renderChangesStatusBar() string {
	var statusParts []string

	if m.status.gitInfo.Branch != "" {
		statusParts = append(statusParts, m.status.gitInfo.Branch)
	}
	if m.status.gitInfo.Ahead > 0 || m.status.gitInfo.Behind > 0 {
		statusParts = append(statusParts, fmt.Sprintf("↑%d ↓%d", m.status.gitInfo.Ahead, m.status.gitInfo.Behind))
	}

	statusParts = append(statusParts,
		fmt.Sprintf("%d drift", len(m.status.filteredFiles)),
		fmt.Sprintf("%d unstaged", len(m.status.gitUnstagedFiles)),
		fmt.Sprintf("%d staged", len(m.status.gitStagedFiles)),
	)
	if len(m.status.unpushedCommits) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d unpushed", len(m.status.unpushedCommits)))
	}
	if len(m.status.incomingCommits) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d incoming", len(m.status.incomingCommits)))
	}

	total := len(m.status.changesRows)
	if total > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d/%d", m.status.changesCursor+1, total))
	}
	if m.status.selectionActive {
		selectedCount := m.selectedActionableCount()
		if selectedCount > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d selected", selectedCount))
		}
	}
	row := m.currentChangesRow()
	if !row.isHeader && row.section == changesSectionDrift && row.driftFile != nil {
		if subtype := row.driftFile.SideLabel(); subtype != "" {
			statusParts = append(statusParts, subtype)
		}
	}

	status := " " + strings.Join(statusParts, " | ") + " "
	if m.ui.message != "" {
		status = " " + m.ui.message + " "
	}
	if m.ui.busyAction {
		status = " " + m.ui.loadingSpinner.View() + " Working... "
	}

	statusBar := activeTheme.StatusBar.Width(m.effectiveWidth()).Render(status)

	var help string
	switch {
	case m.actions.show:
		help = m.helpHint("↑/↓ navigate | enter select | esc back")
	case m.panel.shouldShow(m.width) && m.panel.focusZone == panelFocusPanel:
		help = m.focusedPreviewHelp("back to list")
	default:
		panelHint := m.listPreviewHint()
		if m.status.selectionActive {
			help = m.helpHint("S-↑/S-↓ range | s stage | u unstage | x discard | a actions | ↑/↓ clear range" + panelHint + " | esc quit")
			return statusBar + "\n" + help
		}
		row := m.currentChangesRow()
		switch {
		case row.isHeader:
			help = m.helpHint("↑/↓ nav | enter toggle | c commit | P push | r refresh | / filter" + panelHint + " | esc quit")
		case row.section == changesSectionDrift:
			if row.driftFile != nil && m.canReAddDriftFile(*row.driftFile) {
				help = m.helpHint("↑/↓ nav | enter diff | s re-add | a actions | c commit | r refresh" + panelHint + " | esc quit")
			} else {
				help = m.helpHint("↑/↓ nav | enter diff | a actions | c commit | r refresh" + panelHint + " | esc quit")
			}
		case row.section == changesSectionUnstaged:
			help = m.helpHint("↑/↓ nav | enter diff | s stage | x discard | S all | c commit | r refresh" + panelHint + " | esc quit")
		case row.section == changesSectionStaged:
			help = m.helpHint("↑/↓ nav | enter diff | u unstage | U all | c commit | P push" + panelHint + " | esc quit")
		case row.section == changesSectionUnpushed:
			help = m.helpHint("↑/↓ nav | enter show | x undo commit | P push | r refresh" + panelHint + " | esc quit")
		case row.section == changesSectionIncoming:
			help = m.helpHint("↑/↓ nav | enter show | " + m.incomingRowActionHint() + " | r refresh" + panelHint + " | esc quit")
		default:
			help = m.helpHint("↑/↓ nav | enter diff | r refresh | / filter | tab switch | ? keys" + panelHint + " | esc quit")
		}
	}
	return statusBar + "\n" + help
}

func (m Model) incomingRowActionHint() string {
	if len(m.status.incomingCommits) > 0 && !m.service.IsReadOnly() {
		return "p pull"
	}
	return "f fetch"
}
