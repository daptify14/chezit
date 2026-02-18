package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// --- Commands tab key handler ---

func (m Model) handleCommandsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.IsRepeat && (key.Matches(msg, ChezCommandKeys.Run) || key.Matches(msg, ChezCommandKeys.DryRun)) {
		return m, nil
	}

	switch {
	case key.Matches(msg, ChezSharedKeys.Back):
		return m.escCmd()
	case key.Matches(msg, ChezSharedKeys.Up):
		m.cmds.cursor = moveCursorUp(m.cmds.cursor, navigationStepForKey(msg))
	case key.Matches(msg, ChezSharedKeys.Down):
		m.cmds.cursor = moveCursorDown(m.cmds.cursor, len(m.cmds.items), navigationStepForKey(msg))
	case key.Matches(msg, ChezSharedKeys.Home):
		m.cmds.cursor = 0
	case key.Matches(msg, ChezSharedKeys.End):
		if len(m.cmds.items) > 0 {
			m.cmds.cursor = len(m.cmds.items) - 1
		}
	case key.Matches(msg, ChezCommandKeys.Run):
		if m.cmds.cursor < len(m.cmds.items) && m.cmds.items[m.cmds.cursor].available {
			return m.executeChezmoiCommand(m.cmds.items[m.cmds.cursor].id)
		}
	case key.Matches(msg, ChezCommandKeys.DryRun):
		if m.cmds.cursor < len(m.cmds.items) && m.cmds.items[m.cmds.cursor].supportsDryRun {
			return m.executeDryRun(m.cmds.items[m.cmds.cursor].id)
		}
	}
	return m, nil
}
