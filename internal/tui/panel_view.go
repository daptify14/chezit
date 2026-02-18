package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

const panelLineNumberMinWidth = 4

// renderChangesTabWithPanel wraps the Status tab content with an optional side panel.
func (m Model) renderChangesTabWithPanel() string {
	if !m.panel.shouldShow(m.width) {
		return m.renderChangesTabContent()
	}

	panelW := panelWidthFor(m.width)
	listW := m.width - panelW - 1
	listContent := m.renderChangesTabContentWidth(listW)
	panelContent := m.renderFilePanel(panelW)

	return lipgloss.JoinHorizontal(lipgloss.Top, listContent, panelContent)
}

// renderManagedTabWithPanel wraps the Files tab content with an optional side panel.
func (m Model) renderManagedTabWithPanel() string {
	if !m.panel.shouldShow(m.width) {
		return m.renderManagedTabContent()
	}

	panelW := panelWidthFor(m.width)
	listW := m.width - panelW - 1
	listContent := m.renderManagedTabContentWidth(listW)
	panelContent := m.renderFilePanel(panelW)

	return lipgloss.JoinHorizontal(lipgloss.Top, listContent, panelContent)
}

// --- Panel rendering ---

// renderFilePanel renders the right-side preview panel.
func (m Model) renderFilePanel(width int) string {
	contentWidth := max(width-4, 20) // account for border + padding

	var b strings.Builder

	// Title bar
	b.WriteString(m.renderPanelTitleBar(contentWidth))
	b.WriteString("\n")

	// Content area (read-only; viewport is synchronized in Update handlers).
	if m.panel.viewportReady {
		b.WriteString(m.panel.viewport.View())
	} else {
		b.WriteString(m.panelViewportContentForWidth(contentWidth))
	}

	// Wrap in panel style with focus indicator
	panelStyle := activeTheme.Panel.Width(width)
	if m.panel.focusZone == panelFocusPanel {
		panelStyle = panelStyle.BorderForeground(activeTheme.Primary)
	}

	return panelStyle.Render(b.String())
}

// renderPanelTitleBar renders the panel title with file name and mode badge.
func (m Model) renderPanelTitleBar(width int) string {
	if m.panel.currentPath == "" {
		return activeTheme.BoldPrimary.Render("Preview")
	}

	name := filepath.Base(m.panel.currentPath)

	badgeText := " [file]"
	switch m.panel.contentMode {
	case panelModeDiff:
		if m.panel.currentSection == changesSectionUnpushed || m.panel.currentSection == changesSectionIncoming {
			badgeText = " [commit]"
		} else {
			badgeText = " [diff]"
		}
	case panelModeContent:
		badgeText = panelContentModeBadge(m.panel.currentPath)
	}
	badge := activeTheme.DimText.Render(badgeText)

	titleWidth := max(width-ansi.StringWidth(badgeText), 1)
	title := activeTheme.BoldPrimary.Render(visualTruncate(name, titleWidth)) + badge

	// Add diff summary, direction hint, and side qualifier if in diff mode
	if m.panel.contentMode == panelModeDiff {
		if entry, ok := m.panel.cacheGet(m.panel.currentPath, panelModeDiff, m.panel.currentSection); ok && entry.err == nil {
			summary := diffSummary(entry.lines)
			summaryStr := activeTheme.DimText.Render("  " + summary)
			title += summaryStr
		}
		var detailParts []string
		if hint := diffDirectionHint(m.panel.currentSection); hint != "" {
			detailParts = append(detailParts, hint)
		}
		if side := m.panelDriftSideLabel(); side != "" {
			detailParts = append(detailParts, side)
		}
		if len(detailParts) > 0 {
			title += "\n" + activeTheme.DimText.Render("  "+strings.Join(detailParts, " · "))
		}
	}

	return title
}

func panelContentModeBadge(path string) string {
	lexer := detectLexer(path)
	if lexer == nil {
		return " [file]"
	}

	lexerName := normalizeLexerName(lexer.Config().Name)
	if lexerName == "" {
		return " [file]"
	}

	return fmt.Sprintf(" [%s file]", lexerName)
}

func normalizeLexerName(name string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(name), " "))
	if normalized == "" {
		return ""
	}
	return normalized
}

// renderPanelViewportContent renders the content for the panel viewport.
func (m Model) renderPanelViewportContent(lines []string, width int) string {
	if m.panel.contentMode == panelModeDiff {
		return m.renderPanelDiff(lines, width)
	}
	return renderPanelFileContent(lines, width, m.panel.currentPath)
}

// renderPanelDiff renders diff lines with syntax coloring for the panel viewport.
func (m Model) renderPanelDiff(lines []string, width int) string {
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return activeTheme.DimText.Render("  " + m.emptyPanelDiffMessage())
	}

	var b strings.Builder
	for i, line := range lines {
		style := diffLineStyle(line)
		rendered := style.Render(visualTruncate(line, width-2))
		b.WriteString("  ")
		b.WriteString(rendered)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) emptyPanelDiffMessage() string {
	switch m.panel.currentSection {
	case changesSectionUnstaged:
		if m.panel.currentPath != "" {
			if file, ok := m.lookupPanelGitFile(changesSectionUnstaged, m.panel.currentPath); ok && panelIsUntrackedGitStatus(file.StatusCode) {
				return "Untracked file (no unstaged diff; use [file] view)"
			}
		}
		return "No unstaged changes"
	case changesSectionStaged:
		return "No staged changes"
	case changesSectionUnpushed, changesSectionIncoming:
		return "No commit diff available"
	default:
		return "No changes (file matches source state)"
	}
}

func (m Model) lookupPanelGitFile(section changesSection, path string) (chezmoi.GitFile, bool) {
	var files []chezmoi.GitFile
	switch section {
	case changesSectionUnstaged:
		files = m.status.gitUnstagedFiles
	case changesSectionStaged:
		files = m.status.gitStagedFiles
	default:
		return chezmoi.GitFile{}, false
	}
	for _, file := range files {
		if file.Path == path {
			return file, true
		}
	}
	return chezmoi.GitFile{}, false
}

// panelDriftSideLabel returns the SideLabel for the current panel file
// when it is a drift entry, or "" otherwise.
func (m Model) panelDriftSideLabel() string {
	return m.driftSideLabel(m.panel.currentSection, m.panel.currentPath)
}

// driftSideLabel returns the SideLabel for a drift file, or "" if not applicable.
func (m Model) driftSideLabel(section changesSection, path string) string {
	if section != changesSectionDrift || path == "" {
		return ""
	}
	for i := range m.status.files {
		if m.status.files[i].Path == path {
			return m.status.files[i].SideLabel()
		}
	}
	return ""
}

func panelIsUntrackedGitStatus(code string) bool {
	status := strings.ToUpper(strings.TrimSpace(code))
	return status == "U" || status == "?"
}

// renderPanelFileContent renders file content with line numbers and syntax highlighting.
func renderPanelFileContent(lines []string, width int, filename string) string {
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return activeTheme.DimText.Render("  Empty file")
	}

	// Apply syntax highlighting to the full content, then split back into lines.
	source := strings.Join(lines, "\n")
	highlighted := highlightCode(source, filename)
	hlLines := strings.Split(highlighted, "\n")

	lineNumWidth := max(panelLineNumberMinWidth, len(strconv.Itoa(len(hlLines))))
	contentWidth := max(width-lineNumWidth-4, 10) // padding + gutter

	var b strings.Builder
	for i, line := range hlLines {
		num := fmt.Sprintf("%*d", lineNumWidth, i+1)
		numStr := activeTheme.DimText.Render(num)
		content := visualTruncate(line, contentWidth)

		b.WriteString(" ")
		b.WriteString(numStr)
		b.WriteString("  ")
		b.WriteString(content)
		if i < len(hlLines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
