package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

// renderGitInfoHeader renders branch, ahead/behind, and a freshness token; a
// branch with no upstream shows "no upstream" instead of arrows and freshness.
func (m Model) renderGitInfoHeader() string {
	t := &activeTheme
	info := m.status.gitInfo
	if info.Branch == "" {
		return "  "
	}

	parts := []string{t.BoldPrimary.Render(info.Branch)}
	if info.Sync == chezmoi.GitSyncNoUpstream {
		parts = append(parts, t.HintText.Render("no upstream"))
		return "  " + strings.Join(parts, " · ")
	}

	if info.Ahead > 0 {
		parts = append(parts, t.SuccessFg.Render(fmt.Sprintf("↑%d", info.Ahead)))
	}
	if info.Behind > 0 {
		parts = append(parts, t.WarningFg.Render(fmt.Sprintf("↓%d", info.Behind)))
	}
	parts = append(parts, m.upstreamHeaderToken(t))
	return "  " + strings.Join(parts, " · ")
}

// upstreamHeaderToken renders the freshness token for the Status header.
func (m Model) upstreamHeaderToken(t *Theme) string {
	if m.status.fetchInProgress {
		return t.HintText.Render("fetching…")
	}
	return m.upstreamOutcomeToken(t, true)
}

// upstreamOutcomeToken renders the completed-fetch token shared by the landing
// box and the Status header, which differ only in phrasing (header selects it).
func (m Model) upstreamOutcomeToken(t *Theme, header bool) string {
	switch m.status.fetchOutcome {
	case fetchOK:
		if m.status.gitInfo.Sync == chezmoi.GitSyncUnknown {
			return t.HintText.Render("sync?")
		}
		return t.HintText.Render("fetched " + humanAge(time.Since(m.status.lastFetchTime)))
	case fetchFailed:
		reason := m.status.fetchReason
		if header && reason != fetchReasonGeneric {
			reason = "upstream " + reason
		}
		return t.WarningFg.Render(reason)
	case fetchOff:
		if header {
			return t.HintText.Render("not fetched")
		}
		return t.HintText.Render("not fetched · f")
	default:
		return t.HintText.Render("not fetched")
	}
}
