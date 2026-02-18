package tui

import tea "charm.land/bubbletea/v2"

const repeatNavigationStep = 3

func navigationStepForKey(msg tea.KeyPressMsg) int {
	if msg.IsRepeat {
		return repeatNavigationStep
	}
	return 1
}

func moveCursorUp(cursor, step int) int {
	return max(0, cursor-step)
}

func moveCursorDown(cursor, total, step int) int {
	if total <= 0 {
		return 0
	}
	return min(total-1, cursor+step)
}
