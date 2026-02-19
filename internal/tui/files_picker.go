package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- View Picker ---

func (m *Model) openViewPicker() {
	m.overlays.viewPickerItems = nil

	m.overlays.viewPickerItems = append(m.overlays.viewPickerItems, viewPickerItem{
		mode:  managedViewManaged,
		label: "Managed",
		count: m.filesTab.views[managedViewManaged].tree.fileCount,
	})

	ignoredCount := -1
	if m.filesTab.views[managedViewIgnored].files != nil {
		ignoredCount = m.filesTab.views[managedViewIgnored].tree.fileCount
	}
	m.overlays.viewPickerItems = append(m.overlays.viewPickerItems, viewPickerItem{
		mode:  managedViewIgnored,
		label: "Ignored",
		count: ignoredCount,
	})

	unmanagedCount := -1
	if m.filesTab.views[managedViewUnmanaged].files != nil {
		unmanagedCount = m.filesTab.views[managedViewUnmanaged].tree.fileCount
	}
	m.overlays.viewPickerItems = append(m.overlays.viewPickerItems, viewPickerItem{
		mode:  managedViewUnmanaged,
		label: "Unmanaged",
		count: unmanagedCount,
	})

	allCount := -1
	if m.filesTab.views[managedViewManaged].files != nil {
		allCount = m.filesTab.views[managedViewManaged].tree.fileCount
		if m.filesTab.views[managedViewIgnored].files != nil {
			allCount += m.filesTab.views[managedViewIgnored].tree.fileCount
		}
		if m.filesTab.views[managedViewUnmanaged].files != nil {
			allCount += m.filesTab.views[managedViewUnmanaged].tree.fileCount
		}
	}
	m.overlays.viewPickerItems = append(m.overlays.viewPickerItems, viewPickerItem{
		mode:  managedViewAll,
		label: "All",
		count: allCount,
	})

	m.populateFilterCategories()
	m.overlays.viewPickerPendingMode = m.filesTab.viewMode

	// Pre-select cursor to current view mode.
	m.overlays.viewPickerCursor = 0
	for i, item := range m.overlays.viewPickerItems {
		if item.mode == m.overlays.viewPickerPendingMode {
			m.overlays.viewPickerCursor = i
			break
		}
	}
	m.overlays.showViewPicker = true
}

func (m *Model) toggleViewFilterSelection() {
	if m.overlays.viewPickerCursor < len(m.overlays.viewPickerItems) {
		m.overlays.viewPickerPendingMode = m.overlays.viewPickerItems[m.overlays.viewPickerCursor].mode
		return
	}
	filterIndex := m.overlays.viewPickerCursor - len(m.overlays.viewPickerItems)
	m.toggleFilterCategoryAt(filterIndex)
}

func (m Model) applyViewFilterOverlay() (tea.Model, tea.Cmd) {
	m.overlays.showViewPicker = false

	newFilter := m.entryFilterFromCategories()
	filterChanged := !entryFilterEqual(m.filesTab.entryFilter, newFilter)
	viewChanged := m.overlays.viewPickerPendingMode != m.filesTab.viewMode

	if !viewChanged && !filterChanged {
		return m, nil
	}

	needManagedLoad := false
	needIgnoredLoad := false
	needUnmanagedLoad := false

	if viewChanged {
		m.filesTab.viewMode = m.overlays.viewPickerPendingMode
		m.filesTab.cursor = 0
		m.actions.managedShow = false
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.resetFilesSearch(true)
		m.ui.message = ""

		switch m.filesTab.viewMode {
		case managedViewIgnored:
			needIgnoredLoad = m.filesTab.views[managedViewIgnored].files == nil
		case managedViewUnmanaged:
			needUnmanagedLoad = m.filesTab.views[managedViewUnmanaged].files == nil
		case managedViewAll:
			needIgnoredLoad = m.filesTab.views[managedViewIgnored].files == nil
			needUnmanagedLoad = m.filesTab.views[managedViewUnmanaged].files == nil
		}
	}

	if filterChanged {
		m.filesTab.entryFilter = newFilter
		needManagedLoad = true

		if m.filesTab.views[managedViewIgnored].files != nil || m.filesTab.viewMode == managedViewIgnored || m.filesTab.viewMode == managedViewAll {
			needIgnoredLoad = true
		}
		if m.filesTab.views[managedViewUnmanaged].files != nil || m.filesTab.viewMode == managedViewUnmanaged || m.filesTab.viewMode == managedViewAll {
			needUnmanagedLoad = true
		}
	}

	if needManagedLoad || needIgnoredLoad || needUnmanagedLoad {
		m.nextGen()
	}

	var cmds []tea.Cmd
	if needManagedLoad {
		m.filesTab.views[managedViewManaged].loading = true
		cmds = append(cmds, m.loadManagedCmd())
	}
	if needIgnoredLoad {
		m.filesTab.views[managedViewIgnored].loading = true
		cmds = append(cmds, m.loadIgnoredCmd())
	}
	if needUnmanagedLoad {
		m.filesTab.views[managedViewUnmanaged].loading = true
		cmds = append(cmds, m.loadUnmanagedCmd())
	}

	if len(cmds) > 0 {
		cmds = append([]tea.Cmd{m.ui.loadingSpinner.Tick}, cmds...)
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// --- Filter Overlay ---

func (m *Model) populateFilterCategories() {
	allTypes := chezmoi.AllEntryTypes()
	m.overlays.filterCategories = make([]filterCategory, 0, len(allTypes)+1)

	// When Include is set, only those types are enabled.
	// When Include is empty (no filter), all types are enabled.
	activeIncludes := make(map[chezmoi.EntryType]bool, len(m.filesTab.entryFilter.Include))
	for _, inc := range m.filesTab.entryFilter.Include {
		activeIncludes[inc] = true
	}
	hasFilter := len(activeIncludes) > 0

	m.overlays.filterCategories = append(m.overlays.filterCategories, filterCategory{
		entryType: "",
		label:     "Reset type filters",
		enabled:   true,
	})

	for _, et := range allTypes {
		enabled := !hasFilter || activeIncludes[et]
		m.overlays.filterCategories = append(m.overlays.filterCategories, filterCategory{
			entryType: et,
			label:     string(et),
			enabled:   enabled,
		})
	}
}

func (m *Model) toggleFilterCategory() {
	m.toggleFilterCategoryAt(m.overlays.filterCursor)
}

func (m *Model) toggleFilterCategoryAt(idx int) {
	if idx < 0 || idx >= len(m.overlays.filterCategories) {
		return
	}
	cat := &m.overlays.filterCategories[idx]
	if cat.entryType == "" {
		// "Reset all" sentinel: enable everything
		for i := range m.overlays.filterCategories {
			m.overlays.filterCategories[i].enabled = true
		}
		return
	}
	cat.enabled = !cat.enabled
}

func (m Model) applyFilterOverlay() (tea.Model, tea.Cmd) {
	m.overlays.showFilterOverlay = false

	newFilter := m.entryFilterFromCategories()

	// Only reload if filter actually changed
	if entryFilterEqual(m.filesTab.entryFilter, newFilter) {
		return m, nil
	}

	m.filesTab.entryFilter = newFilter
	m.nextGen()

	// Force reload all currently-loaded data sources
	m.filesTab.views[managedViewManaged].loading = true
	cmds := []tea.Cmd{m.ui.loadingSpinner.Tick, m.loadManagedCmd()}
	if m.filesTab.views[managedViewIgnored].files != nil {
		m.filesTab.views[managedViewIgnored].loading = true
		cmds = append(cmds, m.loadIgnoredCmd())
	}
	if m.filesTab.views[managedViewUnmanaged].files != nil {
		m.filesTab.views[managedViewUnmanaged].loading = true
		cmds = append(cmds, m.loadUnmanagedCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m Model) entryFilterFromCategories() chezmoi.EntryFilter {
	var includes []chezmoi.EntryType
	allEnabled := true
	for _, cat := range m.overlays.filterCategories {
		if cat.entryType == "" {
			continue
		}
		if cat.enabled {
			includes = append(includes, cat.entryType)
		} else {
			allEnabled = false
		}
	}
	if allEnabled {
		return chezmoi.EntryFilter{}
	}
	return chezmoi.EntryFilter{Include: includes}
}

func entryFilterEqual(a, b chezmoi.EntryFilter) bool {
	if len(a.Include) != len(b.Include) || len(a.Exclude) != len(b.Exclude) {
		return false
	}
	for i := range a.Include {
		if a.Include[i] != b.Include[i] {
			return false
		}
	}
	for i := range a.Exclude {
		if a.Exclude[i] != b.Exclude[i] {
			return false
		}
	}
	return true
}

func (m Model) viewFilterRowCount() int {
	return len(m.overlays.viewPickerItems) + len(m.overlays.filterCategories)
}

func (m Model) viewFilterFirstFilterRow() int {
	if len(m.overlays.filterCategories) == 0 {
		return -1
	}
	return len(m.overlays.viewPickerItems)
}

func (m Model) nextViewFilterCursor(delta int) int {
	n := m.viewFilterRowCount()
	if n == 0 {
		return 0
	}
	idx := max(m.overlays.viewPickerCursor+delta, 0)
	if idx >= n {
		idx = n - 1
	}
	return idx
}

func (m Model) nextFilterCursor(delta int) int {
	n := len(m.overlays.filterCategories)
	if n == 0 {
		return 0
	}
	idx := max(m.overlays.filterCursor+delta, 0)
	if idx >= n {
		idx = n - 1
	}
	return idx
}
