package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// scrollViewport handles shared viewport scroll keybindings.
// Returns true if the key was consumed by a scroll action.
func scrollViewport(vp *viewport.Model, msg tea.KeyPressMsg) bool {
	step := navigationStepForKey(msg)
	switch {
	case key.Matches(msg, ChezSharedKeys.Up):
		vp.ScrollUp(step)
	case key.Matches(msg, ChezSharedKeys.Down):
		vp.ScrollDown(step)
	case key.Matches(msg, ChezScrollKeys.HalfUp):
		vp.HalfPageUp()
	case key.Matches(msg, ChezScrollKeys.HalfDown):
		vp.HalfPageDown()
	case key.Matches(msg, ChezScrollKeys.PageUp):
		vp.PageUp()
	case key.Matches(msg, ChezScrollKeys.PageDown):
		vp.PageDown()
	case key.Matches(msg, ChezSharedKeys.Home):
		vp.GotoTop()
	case key.Matches(msg, ChezSharedKeys.End):
		vp.GotoBottom()
	default:
		return false
	}
	return true
}

// --- Diff view key handler ---

func (m Model) handleDiffKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Preview-apply mode: Esc cancels, Enter confirms and shells out
	if m.diff.previewApply {
		switch {
		case key.Matches(msg, ChezSharedKeys.Back):
			m.diff.previewApply = false
			m.view = StatusScreen
			m.diff.content = ""
			m.diff.lines = nil
			m.diff.resetViewport()
			return m, nil
		case key.Matches(msg, ChezCommandKeys.Run): // Enter
			m.diff.previewApply = false
			m.view = StatusScreen
			m.diff.content = ""
			m.diff.lines = nil
			m.diff.resetViewport()
			// Shell-out to apply with full TTY (sudo/scripts need it)
			cmd := m.service.ApplyAllCmd()
			wrapped := wrapWithPressEnter(cmd)
			return m, execCmdOrUnsupported(chezmoiActionApplyAll, wrapped, "chezmoi: apply not supported")
		}
		// Fall through to scroll keys below
	}

	switch {
	case key.Matches(msg, ChezSharedKeys.Back):
		m.view = StatusScreen
		m.diff.content = ""
		m.diff.lines = nil
		m.diff.resetViewport()
		m.actions.show = false
		return m, nil

	case key.Matches(msg, ChezDiffKeys.Edit):
		if !m.service.IsReadOnly() && m.diff.path != "" {
			return m, m.editSourceCmd(m.diff.path)
		}
	case key.Matches(msg, ChezDiffKeys.Actions):
		m.openDiffActionsMenu()
		return m, nil

	default:
		// Scroll — delegate to viewport via shared helper
		m = m.syncDiffViewportContent()
		if scrollViewport(&m.diff.viewport, msg) {
			return m, nil
		}
	}
	return m, nil
}

// syncDiffViewportContent ensures the diff viewport is ready and content is synced before scrolling.
func (m Model) syncDiffViewportContent() Model {
	if len(m.diff.lines) == 0 {
		return m
	}
	diffHeight := m.chezmoiDiffViewHeight()
	m.diff.ensureViewport(m.effectiveWidth(), diffHeight)
	content := preRenderDiffContent(m.diff.lines, m.effectiveWidth())
	currentOffset := m.diff.viewport.YOffset()
	m.diff.viewport.SetContent(content)
	m.diff.viewport.SetYOffset(currentOffset)
	return m
}
