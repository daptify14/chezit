package tui

import tea "charm.land/bubbletea/v2"

func isStatusShiftUp(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyUp && msg.Mod&tea.ModShift != 0
}

func isStatusShiftDown(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyDown && msg.Mod&tea.ModShift != 0
}

func (m *Model) clearStatusSelection() {
	m.status.selectionActive = false
	m.status.selectionAnchor = m.status.changesCursor
}

func (m *Model) beginStatusSelectionIfNeeded() {
	if m.status.selectionActive {
		return
	}
	m.status.selectionActive = true
	m.status.selectionAnchor = m.status.changesCursor
}

func clampStatusRowIndex(idx, total int) int {
	if total <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= total {
		return total - 1
	}
	return idx
}

func (m Model) statusSelectionBounds() (start, end int, ok bool) {
	if !m.status.selectionActive || len(m.status.changesRows) == 0 {
		return 0, 0, false
	}
	anchor := clampStatusRowIndex(m.status.selectionAnchor, len(m.status.changesRows))
	cursor := clampStatusRowIndex(m.status.changesCursor, len(m.status.changesRows))
	if anchor <= cursor {
		return anchor, cursor, true
	}
	return cursor, anchor, true
}

func (m Model) statusSelectionSection() (changesSection, bool) {
	if !m.status.selectionActive || len(m.status.changesRows) == 0 {
		return 0, false
	}
	anchor := clampStatusRowIndex(m.status.selectionAnchor, len(m.status.changesRows))
	return m.status.changesRows[anchor].section, true
}

func (m Model) statusSelectionSectionBounds() (start, end int, ok bool) {
	section, ok := m.statusSelectionSection()
	if !ok {
		return 0, 0, false
	}
	anchor := clampStatusRowIndex(m.status.selectionAnchor, len(m.status.changesRows))
	start, end = anchor, anchor
	for start > 0 && m.status.changesRows[start-1].section == section {
		start--
	}
	for end+1 < len(m.status.changesRows) && m.status.changesRows[end+1].section == section {
		end++
	}
	return start, end, true
}

func (m Model) clampStatusSelectionCursor(next int) int {
	next = clampStatusRowIndex(next, len(m.status.changesRows))
	start, end, ok := m.statusSelectionSectionBounds()
	if !ok {
		return next
	}
	if next < start {
		return start
	}
	if next > end {
		return end
	}
	return next
}

func (m Model) isStatusRowRangeSelected(idx int) bool {
	start, end, ok := m.statusSelectionBounds()
	if !ok {
		return false
	}
	section, sectionOK := m.statusSelectionSection()
	if !sectionOK {
		return false
	}
	if idx < 0 || idx >= len(m.status.changesRows) {
		return false
	}
	if m.status.changesRows[idx].section != section {
		return false
	}
	return idx >= start && idx <= end
}

func (m Model) selectedStatusActionableRows() []changesRow {
	start, end, ok := m.statusSelectionBounds()
	if !ok {
		return nil
	}
	section, sectionOK := m.statusSelectionSection()
	if !sectionOK {
		return nil
	}
	rows := make([]changesRow, 0, end-start+1)
	for idx := start; idx <= end; idx++ {
		row := m.status.changesRows[idx]
		if row.isHeader || row.section != section {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func dedupePaths(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func (m Model) selectedStageTargets() (driftPaths, unstagedPaths []string) {
	for _, row := range m.selectedStatusActionableRows() {
		switch row.section {
		case changesSectionDrift:
			if row.driftFile != nil {
				driftPaths = append(driftPaths, row.driftFile.Path)
			}
		case changesSectionUnstaged:
			if row.gitFile != nil {
				unstagedPaths = append(unstagedPaths, row.gitFile.Path)
			}
		}
	}
	return dedupePaths(driftPaths), dedupePaths(unstagedPaths)
}

func (m Model) selectedReAddTargets() []string {
	var paths []string
	for _, row := range m.selectedStatusActionableRows() {
		if row.section != changesSectionDrift || row.driftFile == nil {
			continue
		}
		f := row.driftFile
		if !m.canReAddDriftFile(*f) {
			continue
		}
		paths = append(paths, f.Path)
	}
	return dedupePaths(paths)
}

func (m Model) selectedUnstageTargets() []string {
	var paths []string
	for _, row := range m.selectedStatusActionableRows() {
		if row.section == changesSectionStaged && row.gitFile != nil {
			paths = append(paths, row.gitFile.Path)
		}
	}
	return dedupePaths(paths)
}

func (m Model) selectedDiscardTargets() []string {
	var paths []string
	for _, row := range m.selectedStatusActionableRows() {
		if row.section == changesSectionUnstaged && row.gitFile != nil {
			if row.gitFile.StatusCode != "U" {
				paths = append(paths, row.gitFile.Path)
			}
		}
	}
	return dedupePaths(paths)
}

func (m Model) selectedActionableCount() int {
	return len(m.selectedStatusActionableRows())
}
