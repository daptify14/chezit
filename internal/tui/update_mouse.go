package tui

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// --- Mouse click handler ---

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.actions.show || m.actions.managedShow || m.overlays.showHelp || m.filterInput.Focused() {
		return m, nil
	}

	tab := m.activeTabName()
	if m.view == StatusScreen && (tab == "Status" || tab == "Files") && m.panel.shouldShow(m.width) {
		listWidth := m.width - panelWidthFor(m.width) - 1
		inPanel := msg.X >= listWidth

		if msg.Button == tea.MouseLeft {
			if inPanel {
				m.panel.focusZone = panelFocusPanel
				m = m.syncPanelViewportContent()
				return m, nil
			}
			m.panel.focusZone = panelFocusList
		}
	}

	if m.view != StatusScreen || m.activeTab != 0 {
		if m.view == StatusScreen && m.activeTab == 1 {
			return m.handleFilesMouseClick(msg)
		}
		return m, nil
	}

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	listHeight := m.chezmoiChangesListHeight()
	headerOffset := statusFilesHeaderLines

	row := msg.Y - headerOffset
	if row < 0 || row >= listHeight {
		return m, nil
	}
	start, _ := visibleRange(len(m.status.changesRows), m.status.changesCursor, listHeight)
	idx := start + row
	if idx >= 0 && idx < len(m.status.changesRows) {
		m.clearStatusSelection()
		if m.status.changesCursor == idx {
			r := m.status.changesRows[idx]
			switch {
			case r.isHeader:
				m.status.sectionCollapsed[r.section] = !m.status.sectionCollapsed[r.section]
				m.buildChangesRows()
			case r.section == changesSectionDrift && r.driftFile != nil:
				m.diff.sourceSection = changesSectionDrift
				m.ui.busyAction = true
				return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadDiffCmd(r.driftFile.Path))
			case r.gitFile != nil:
				m.diff.sourceSection = r.section
				m.ui.busyAction = true
				staged := r.section == changesSectionStaged
				return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadGitDiffCmd(r.gitFile.Path, staged))
			}
		}
		m.status.changesCursor = idx
		if m.panel.shouldShow(m.width) {
			var panelCmd tea.Cmd
			m, panelCmd = m.panelLoadForChanges()
			return m, panelCmd
		}
	}

	return m, nil
}

func (m Model) handleFilesMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	listHeight := m.chezmoiManagedListHeight()
	headerOffset := statusFilesHeaderLines
	row := msg.Y - headerOffset
	if row < 0 || row >= listHeight {
		return m, nil
	}

	var total int
	if m.filesTab.treeView {
		total = len(m.activeTreeRows())
	} else {
		total = len(m.activeFlatFiles())
	}
	if total == 0 {
		return m, nil
	}

	start, _ := visibleRange(total, m.filesTab.cursor, listHeight)
	idx := start + row
	if idx < 0 || idx >= total {
		return m, nil
	}

	isSelected := idx == m.filesTab.cursor
	m.filesTab.cursor = idx

	if isSelected {
		if m.filesTab.treeView {
			return m.handleFilesTreeExpand()
		}
		m.openFilesActiveMenu()
		return m, nil
	}

	if m.panel.shouldShow(m.width) {
		var cmd tea.Cmd
		m, cmd = m.panelLoadForManaged()
		return m, cmd
	}
	return m, nil
}

// --- Mouse wheel handler ---

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.actions.show || m.actions.managedShow || m.overlays.showHelp || m.filterInput.Focused() {
		return m, nil
	}

	if m.view == DiffScreen {
		m = m.syncDiffViewportContent()
		if scrollViewportByMouse(&m.diff.viewport, msg.Button, 3) {
			return m, nil
		}
	}

	if m.view == StatusScreen && m.activeTabName() == "Info" {
		m = m.syncInfoViewportContent()
		if scrollViewportByMouse(&m.info.views[m.info.activeView].viewport, msg.Button, 3) {
			return m, nil
		}
	}

	tab := m.activeTabName()
	if m.view == StatusScreen && (tab == "Status" || tab == "Files") && m.panel.shouldShow(m.width) {
		listWidth := m.width - panelWidthFor(m.width) - 1
		inPanel := msg.X >= listWidth
		if inPanel || m.panel.focusZone == panelFocusPanel {
			m.panel.focusZone = panelFocusPanel
			m = m.syncPanelViewportContent()
			if scrollViewportByMouse(&m.panel.viewport, msg.Button, 3) {
				return m, nil
			}
		}
	}

	if m.view != StatusScreen || m.activeTab != 0 {
		if m.view == StatusScreen && m.activeTab == 1 {
			before := m.filesTab.cursor
			m.scrollFilesCursor(msg.Button, 3)
			if m.filesTab.cursor != before && m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForManaged()
				return m, panelCmd
			}
			return m, nil
		}
		return m, nil
	}

	m.scrollStatusCursor(msg.Button, 3)
	return m, nil
}

func scrollViewportByMouse(vp *viewport.Model, btn tea.MouseButton, amount int) bool {
	switch btn {
	case tea.MouseWheelUp:
		vp.ScrollUp(amount)
		return true
	case tea.MouseWheelDown:
		vp.ScrollDown(amount)
		return true
	}
	return false
}

func (m *Model) scrollStatusCursor(btn tea.MouseButton, amount int) {
	total := len(m.status.changesRows)
	switch btn {
	case tea.MouseWheelUp:
		m.status.changesCursor = max(0, m.status.changesCursor-amount)
	case tea.MouseWheelDown:
		if total > 0 {
			m.status.changesCursor = min(total-1, m.status.changesCursor+amount)
		}
	}
}

func (m *Model) scrollFilesCursor(btn tea.MouseButton, amount int) {
	total := len(m.activeFlatFiles())
	if m.filesTab.treeView {
		total = len(m.activeTreeRows())
	}
	switch btn {
	case tea.MouseWheelUp:
		m.filesTab.cursor = max(0, m.filesTab.cursor-amount)
	case tea.MouseWheelDown:
		if total > 0 {
			m.filesTab.cursor = min(total-1, m.filesTab.cursor+amount)
		}
	}
}
