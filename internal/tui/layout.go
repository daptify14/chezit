package tui

// --- Layout Calculations ---

// Layout constants shared between rendering and mouse-hit-testing.
const (
	// statusFilesHeaderLines is the number of header rows above the list
	// in the Status and Files tabs: breadcrumb + separator + tab bar +
	// search box + git header + summary line.
	statusFilesHeaderLines = 7

	// statusFilesFooterLines is the number of footer rows below the list:
	// status bar + help line.
	statusFilesFooterLines = 2
)

// clampListHeight ensures a computed list height is at least 1.
// Use only after the m.height == 0 (uninitialized) guard.
func clampListHeight(height int) int {
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) chezmoiChangesListHeight() int {
	if m.height == 0 {
		return 0
	}
	actionsLines := 0
	if m.actions.show {
		actionsLines = len(m.actions.items) + 3
	}
	return clampListHeight(m.height - statusFilesHeaderLines - statusFilesFooterLines - actionsLines)
}

func (m Model) chezmoiManagedListHeight() int {
	if m.height == 0 {
		return 0
	}
	actionsLines := 0
	if m.actions.managedShow {
		actionsLines = len(m.actions.managedItems) + 3
		// Account for description help line
		if m.actions.managedCursor >= 0 && m.actions.managedCursor < len(m.actions.managedItems) &&
			m.actions.managedItems[m.actions.managedCursor].description != "" {
			actionsLines++
		}
	}
	return clampListHeight(m.height - statusFilesHeaderLines - statusFilesFooterLines - actionsLines)
}

func (m Model) infoViewHeight() int {
	if m.height == 0 {
		return 0
	}
	headerLines := 3 // breadcrumb + separator + tab bar
	subViewBar := 2  // sub-view selector bar + blank line
	return clampListHeight(m.height - headerLines - subViewBar - statusFilesFooterLines)
}

func (m Model) chezmoiCommandsListHeight() int {
	if m.height == 0 {
		return 0
	}
	headerLines := 5 // breadcrumb + separator + tab bar + padding
	return clampListHeight(m.height - headerLines - statusFilesFooterLines)
}

func (m Model) chezmoiDiffViewHeight() int {
	if m.height == 0 {
		return 0
	}
	headerLines := 2
	actionsLines := 0
	if m.actions.show {
		actionsLines = len(m.actions.items) + 3
	}
	return clampListHeight(m.height - headerLines - statusFilesFooterLines - actionsLines)
}

func (m Model) effectiveWidth() int {
	if m.width == 0 {
		return 80
	}
	return m.width
}
