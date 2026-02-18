package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func (m Model) renderGitFileRow(f chezmoi.GitFile, selected, staged bool, maxWidth int) string {
	cursor := "    "
	if selected {
		cursor = "  > "
	}

	icon := renderFileIcon(filepath.Base(f.Path), false, selected, m.iconMode)

	if selected {
		content := cursor + f.StatusCode + " " + icon + f.Path
		content = visualTruncate(content, maxWidth)
		return activeTheme.Selected.Width(maxWidth).Render(content)
	}

	statusStyle := activeTheme.WarningFg
	if staged {
		statusStyle = activeTheme.SuccessFg
	}
	if strings.ContainsRune(f.StatusCode, 'D') {
		statusStyle = activeTheme.DangerFg
	}
	statusStr := statusStyle.Render(f.StatusCode)
	content := cursor + statusStr + " " + icon + f.Path
	content = visualTruncate(content, maxWidth)
	return content
}

// renderCommitRow renders a single commit row (hash + message) for the unpushed/incoming sections.
func (m Model) renderCommitRow(c chezmoi.GitCommit, selected bool, maxWidth int) string {
	cursor := "    "
	if selected {
		cursor = "  > "
	}

	hashWidth := min(len(c.Hash), 8)
	msgWidth := max(maxWidth-len(cursor)-hashWidth-1, 10)
	msg := visualTruncate(c.Message, msgWidth)

	if selected {
		content := fmt.Sprintf("%s%s %s", cursor, c.Hash, msg)
		content = visualTruncate(content, maxWidth)
		return activeTheme.Selected.Width(maxWidth).Render(content)
	}

	hashStr := activeTheme.AccentFg.Render(c.Hash)
	content := cursor + hashStr + " " + msg
	content = visualTruncate(content, maxWidth)
	return content
}

func (m Model) renderGitInfoHeader() string {
	var parts []string

	if m.status.gitInfo.Branch != "" {
		parts = append(parts, activeTheme.BoldPrimary.Render(m.status.gitInfo.Branch))
	}

	if m.status.gitInfo.Ahead > 0 {
		parts = append(parts, activeTheme.SuccessFg.Render(fmt.Sprintf("↑%d", m.status.gitInfo.Ahead)))
	}
	if m.status.gitInfo.Behind > 0 {
		parts = append(parts, activeTheme.WarningFg.Render(fmt.Sprintf("↓%d", m.status.gitInfo.Behind)))
	}

	return "  " + strings.Join(parts, " · ")
}
