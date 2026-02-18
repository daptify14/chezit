package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// --- Cross-cutting message handlers ---
//
// These handlers respond to async messages that touch multiple tab states
// (UI, overlays, panel, diff, files) and don't belong to any single tab.

func (m Model) handleDiffLoaded(msg chezmoiDiffLoadedMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error loading diff: " + msg.err.Error()
		return m, nil
	}
	m.view = DiffScreen
	m.diff.content = msg.diff
	m.diff.path = msg.path
	m.diff.lines = strings.Split(msg.diff, "\n")
	m.diff.resetViewport()
	m.actions.show = false
	return m, nil
}

func (m Model) handleActionDone(msg chezmoiActionDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	m.ui.message = msg.message
	m.panel.clearCache()
	return m, tea.Batch(m.postActionReloadCmds(), sendRefreshMsg())
}

func (m Model) handleForgetDone(msg chezmoiForgetDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	m.ui.message = "forgot " + msg.path
	m.filesTab.views[managedViewManaged].loading = true
	m.panel.clearCache()
	m.nextGen()
	reloadCmds := []tea.Cmd{m.ui.loadingSpinner.Tick, m.loadManagedCmd()}
	reloadCmds = append(reloadCmds, m.reloadStatusAndGitCmds()...)
	reloadCmds = append(reloadCmds, sendRefreshMsg())
	return m, tea.Batch(reloadCmds...)
}

func (m Model) handleAddDone(msg chezmoiAddDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + mapAddError(msg.err)
		return m, nil
	}
	m.ui.message = "Added " + filepath.Base(msg.path)
	m.nextGen()
	reloadCmds := []tea.Cmd{m.loadManagedCmd()}
	if m.filesTab.views[managedViewUnmanaged].files != nil {
		m.filesTab.views[managedViewUnmanaged].loading = true
		reloadCmds = append(reloadCmds, m.loadUnmanagedCmd())
	}
	reloadCmds = append(reloadCmds, m.reloadStatusAndGitCmds()...)
	return m, tea.Batch(append(reloadCmds, sendRefreshMsg())...)
}

func (m Model) handleSourceContent(msg chezmoiSourceContentMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.diff.previewApply = false
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	if m.diff.previewApply && strings.TrimSpace(msg.content) == "" {
		m.diff.previewApply = false
		m.ui.message = "Nothing to apply — destination matches source"
		return m, nil
	}
	m.view = DiffScreen
	m.diff.content = msg.content
	m.diff.path = msg.path
	m.diff.lines = strings.Split(msg.content, "\n")
	m.diff.resetViewport()
	m.actions.show = false
	return m, nil
}

func (m Model) handleCapturedOutput(msg chezmoiCapturedOutputMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	m.diff.previewApply = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		m.panel.clearCache()
		cmd := m.postActionReloadCmds()
		return m, cmd
	}
	if strings.TrimSpace(msg.output) == "" {
		m.ui.message = msg.label + ": no changes"
		m.panel.clearCache()
		reloadCmd := m.postActionReloadCmds()
		return m, tea.Batch(reloadCmd, sendRefreshMsg())
	}
	m.view = DiffScreen
	m.diff.content = msg.output
	m.diff.path = msg.label
	m.diff.lines = strings.Split(msg.output, "\n")
	m.diff.resetViewport()
	m.actions.show = false
	m.panel.clearCache()
	return m, tea.Batch(m.postActionReloadCmds(), sendRefreshMsg())
}

func (m Model) handleArchiveDone(msg chezmoiArchiveDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Archive failed: " + msg.err.Error()
		return m, nil
	}
	sizeStr := ""
	if msg.size >= 0 {
		sizeStr = fmt.Sprintf(" (%s)", humanSize(msg.size))
	}
	m.ui.message = fmt.Sprintf("Archive created: %s%s", msg.path, sizeStr)
	return m, nil
}

func (m Model) handleSourceDirResolved(msg sourceDirResolvedMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	switch msg.action {
	case chezmoiActionEditIgnoreFile:
		cmd := m.editorCmd(msg.path)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return chezmoiExecDoneMsg{action: chezmoiActionEditIgnoreFile, err: err}
		})
	default:
		return m, nil
	}
}

func (m Model) handleExecDone(msg chezmoiExecDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		if msg.action == chezmoiActionEditTarget {
			return m, nil
		}
		m.panel.clearCache()
		cmd := m.postActionReloadCmds()
		return m, cmd
	}
	reload := true
	switch msg.action {
	case chezmoiActionApplyFile:
		m.ui.message = "applied file"
	case chezmoiActionApplyAll:
		m.ui.message = "applied all files"
	case chezmoiActionUpdate:
		m.ui.message = "update complete"
	case chezmoiActionEditSource:
		m.ui.message = "edit complete"
	case chezmoiActionEditIgnoreFile:
		m.ui.message = "edit complete"
	case chezmoiActionEditTarget:
		m.ui.message = "editor closed"
		reload = false
	}
	if m.view == DiffScreen {
		m.view = StatusScreen
		m.diff.content = ""
		m.diff.lines = nil
		m.diff.resetViewport()
	}
	if !reload {
		return m, nil
	}
	m.panel.clearCache()
	return m, tea.Batch(m.postActionReloadCmds(), sendRefreshMsg())
}

// --- Cross-cutting reload helpers ---

func (m *Model) postActionReloadCmds() tea.Cmd {
	m.nextGen()
	cmds := []tea.Cmd{m.loadManagedCmd()}
	cmds = append(cmds, m.reloadStatusAndGitCmds()...)
	return tea.Batch(cmds...)
}

// mapAddError converts chezmoi add errors into concise user-friendly messages.
// Known encryption-related patterns are mapped to actionable hints.
func mapAddError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// Known encryption misconfiguration patterns
	encryptionPatterns := []string{
		"no value for",      // missing age recipient/identity
		"encryption",        // generic encryption failure
		"gpg failed",        // GPG key issue
		"age: no identity",  // age missing identity
		"age: no recipient", // age missing recipients
		"no secret key",     // GPG secret key missing
		"recipient",         // missing recipient config
	}
	for _, pattern := range encryptionPatterns {
		if strings.Contains(strings.ToLower(msg), pattern) {
			return "Encryption not configured. Run 'chezmoi doctor' to diagnose."
		}
	}

	// Fallback: trim and return the original error
	return msg
}
