package tui

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// logMsg logs a tea.Msg to the debug logger if one is configured.
// This is a no-op when debugLog is nil (the common case).
// Spinner ticks are silently dropped to reduce noise.
func (m Model) logMsg(msg tea.Msg) {
	if m.debugLog == nil {
		return
	}
	// Skip high-frequency spinner ticks (~8/sec, zero diagnostic value).
	if _, ok := msg.(spinner.TickMsg); ok {
		return
	}
	m.debugLog.Info("msg",
		"type", fmt.Sprintf("%T", msg),
		"detail", formatMsgDetail(msg),
	)
}

// formatMsgDetail extracts key fields from known message types for readable log output.
// Unknown types log their type name only — never %#v, which can leak secrets
// (e.g. tea.EnvMsg contains the full environment).
func formatMsgDetail(msg tea.Msg) string {
	switch msg := msg.(type) {
	// Bubbletea core messages
	case tea.KeyPressMsg:
		return msg.String()
	case tea.WindowSizeMsg:
		return fmt.Sprintf("%dx%d", msg.Width, msg.Height)
	case tea.MouseClickMsg:
		return fmt.Sprintf("x=%d y=%d", msg.X, msg.Y)
	case tea.MouseWheelMsg:
		return fmt.Sprintf("x=%d y=%d", msg.X, msg.Y)
	case tea.MouseReleaseMsg:
		return fmt.Sprintf("x=%d y=%d", msg.X, msg.Y)
	case tea.BackgroundColorMsg:
		return fmt.Sprintf("dark=%t", msg.IsDark())
	case tea.ColorProfileMsg:
		return fmt.Sprintf("profile=%d", msg.Profile)

	// Status tab
	case chezmoiStatusLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("files=%d", len(msg.files)))
	case chezmoiGitStatusLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("staged=%d unstaged=%d", len(msg.staged), len(msg.unstaged)))
	case chezmoiGitActionDoneMsg:
		return actionErr(msg.action, msg.err, msg.message)
	case chezmoiGitCommitsLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("unpushed=%d incoming=%d", len(msg.unpushed), len(msg.incoming)))
	case chezmoiGitFetchDoneMsg:
		return genErr(msg.gen, msg.err, "")
	case templatePathsLoadedMsg:
		return genErr(msg.gen, nil, fmt.Sprintf("paths=%d", len(msg.paths)))

	// Files tab
	case chezmoiManagedLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("files=%d", len(msg.files)))
	case chezmoiIgnoredLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("files=%d", len(msg.files)))
	case chezmoiUnmanagedLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("files=%d", len(msg.files)))
	case filesSearchDebouncedMsg:
		return fmt.Sprintf("request=%d", msg.requestID)
	case filesSearchCompletedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("query=%q results=%d", msg.query, len(msg.results)))

	// Info tab
	case infoContentLoadedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("view=%d len=%d", msg.view, len(msg.content)))

	// Cross-cutting
	case chezmoiDiffLoadedMsg:
		return pathErr(msg.path, msg.err)
	case chezmoiActionDoneMsg:
		return actionErr(msg.action, msg.err, msg.message)
	case chezmoiExecDoneMsg:
		return actionErr(msg.action, msg.err, "")
	case chezmoiForgetDoneMsg:
		return pathErr(msg.path, msg.err)
	case chezmoiAddDoneMsg:
		return pathErr(msg.path, msg.err)
	case chezmoiSourceContentMsg:
		return pathErr(msg.path, msg.err)
	case chezmoiCapturedOutputMsg:
		return actionErr(msg.action, msg.err, msg.label)
	case chezmoiArchiveDoneMsg:
		return pathErr(msg.path, msg.err)
	case sourceDirResolvedMsg:
		return pathErr(msg.path, msg.err)
	case panelContentLoadedMsg:
		return pathErr(msg.path, msg.err)
	case opaqueDirPopulatedMsg:
		return genErr(msg.gen, msg.err, fmt.Sprintf("path=%q children=%d", msg.relPath, len(msg.children)))

	// Simple signals
	case landingStatsReadyMsg:
		return ""
	case ExitMsg:
		return ""
	case RefreshMsg:
		return ""

	default:
		// Type name only — never %#v, which can leak secrets from
		// types like tea.EnvMsg that carry the full environment.
		return ""
	}
}

func genErr(gen uint64, err error, extra string) string {
	if err != nil {
		return fmt.Sprintf("gen=%d err=%q %s", gen, err.Error(), extra)
	}
	return fmt.Sprintf("gen=%d %s", gen, extra)
}

func pathErr(path string, err error) string {
	if err != nil {
		return fmt.Sprintf("path=%q err=%q", path, err.Error())
	}
	return fmt.Sprintf("path=%q", path)
}

func actionErr(action chezmoiAction, err error, label string) string {
	if err != nil {
		return fmt.Sprintf("action=%d label=%q err=%q", action, label, err.Error())
	}
	return fmt.Sprintf("action=%d label=%q", action, label)
}
