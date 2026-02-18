package tui

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// --- Files tab message handlers ---

func (m Model) handleManagedLoaded(msg chezmoiManagedLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.filesTab.views[managedViewManaged].loading = false
	if msg.err != nil {
		if m.activeTabName() == "Files" {
			m.ui.message = "Error: " + msg.err.Error()
		}
	} else {
		m.filesTab.views[managedViewManaged].files = msg.files
		m.filesTab.views[managedViewManaged].filteredFiles = msg.files
		m.filesTab.cursor = 0
		m.rebuildFileViewTree(managedViewManaged)
		m.rebuildDatasetAndAllView()
	}
	if m.allLandingStatsLoaded() && !m.landing.statsReady {
		return m, tea.Batch(nil, debounceLandingReadyCmd())
	}
	if m.panel.shouldShow(m.width) && m.activeTabName() == "Files" {
		var cmd tea.Cmd
		m, cmd = m.panelLoadForManaged()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleIgnoredLoaded(msg chezmoiIgnoredLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.filesTab.views[managedViewIgnored].loading = false
	if msg.err != nil {
		if m.activeTabName() == "Files" {
			m.ui.message = "Error loading ignored files: " + msg.err.Error()
		}
		return m, nil
	}
	m.filesTab.views[managedViewIgnored].files = msg.files
	m.filesTab.views[managedViewIgnored].filteredFiles = msg.files
	m.rebuildFileViewTree(managedViewIgnored)
	m.rebuildDatasetAndAllView()
	return m, nil
}

func (m Model) handleUnmanagedLoaded(msg chezmoiUnmanagedLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.filesTab.views[managedViewUnmanaged].loading = false
	if msg.err != nil {
		m.resetFilesSearch(true)
		if m.activeTabName() == "Files" {
			m.ui.message = "Error loading unmanaged files: " + msg.err.Error()
		}
		return m, nil
	}
	m.filesTab.views[managedViewUnmanaged].files = msg.files
	m.filesTab.views[managedViewUnmanaged].filteredFiles = msg.files
	m.rebuildFileViewTree(managedViewUnmanaged)
	// Invalidate any previous deep search results and stale in-flight searches.
	m.resetFilesSearch(true)
	m.rebuildDatasetAndAllView()
	if m.activeTabName() == "Files" && m.filterInput.Value() != "" {
		m.applyManagedFilter()
		if cmd := m.triggerFilesSearchIfNeeded(); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleSearchDebounced(msg filesSearchDebouncedMsg) (tea.Model, tea.Cmd) {
	if msg.requestID != m.filesTab.search.request {
		return m, nil
	}
	query := strings.TrimSpace(m.filterInput.Value())
	if m.activeTabName() != "Files" || query == "" {
		m.resetFilesSearch(false)
		return m, nil
	}
	if m.filesTab.viewMode != managedViewUnmanaged && m.filesTab.viewMode != managedViewAll {
		m.resetFilesSearch(false)
		return m, nil
	}

	searchRoot := normalizePath(m.targetPath)
	if searchRoot == "" {
		m.resetFilesSearch(false)
		return m, nil
	}
	searchRoots := []string{searchRoot}

	m.cancelFilesSearch()
	ctx, cancel := context.WithTimeout(context.Background(), filesSearchTimeout)
	m.filesTab.search.cancel = cancel
	return m, m.runFilesSearchCmd(ctx, msg.requestID, query, searchRoots)
}

func (m Model) handleSearchCompleted(msg filesSearchCompletedMsg) (tea.Model, tea.Cmd) {
	if msg.requestID != m.filesTab.search.request {
		return m, nil
	}
	if msg.gen != m.gen {
		return m, nil
	}
	m.filesTab.search.cancel = nil
	if msg.query != strings.TrimSpace(m.filterInput.Value()) {
		return m, nil
	}
	m.filesTab.search.query = msg.query
	m.filesTab.search.lastMetrics = msg.metrics
	if errors.Is(msg.err, context.Canceled) || errors.Is(msg.err, context.DeadlineExceeded) {
		m.filesTab.search.rawResults = msg.results
		m.filesTab.search.ready = len(msg.results) > 0
		m.filesTab.search.searching = false
		if m.activeTabName() == "Files" && m.filterInput.Value() != "" {
			m.applyManagedFilter()
		}
		return m, nil
	}
	if msg.err != nil {
		m.filesTab.search.rawResults = nil
		m.filesTab.search.searching = false
		m.filesTab.search.paused = false
		m.filesTab.search.ready = false
		if m.activeTabName() == "Files" {
			m.ui.message = "Search error: " + msg.err.Error()
		}
	} else {
		m.filesTab.search.rawResults = msg.results
		m.filesTab.search.searching = false
		m.filesTab.search.paused = false
		m.filesTab.search.ready = true
	}
	if m.activeTabName() == "Files" && m.filterInput.Value() != "" {
		m.applyManagedFilter()
	}
	return m, nil
}

func (m Model) handleOpaqueDirPopulated(msg opaqueDirPopulatedMsg) (tea.Model, tea.Cmd) {
	tree := m.filesTab.views[msg.viewMode].tree
	node := findTreeNodeByRelPath(tree, msg.relPath)
	if node == nil {
		return m, nil
	}
	if msg.requestID != node.loadingRequest {
		return m, nil
	}
	node.loading = false
	node.loadingRequest = 0
	if msg.gen != m.gen {
		return m, nil
	}
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	for _, child := range msg.children {
		child.parent = node
	}
	node.children = msg.children
	sortChildren(node)
	node.opaque = false
	node.expanded = true
	if m.activeTabName() == "Files" &&
		m.filesTab.viewMode == msg.viewMode &&
		strings.TrimSpace(m.filterInput.Value()) != "" {
		m.applyManagedFilter()
		return m, nil
	}
	m.reflattenTreeForView(msg.viewMode)
	return m, nil
}

// --- Files tab key handlers ---

func (m Model) handleFilesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if next, cmd, handled := m.handleFilesViewPickerOverlayKeys(msg); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleFilesFilterOverlayKeys(msg); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleFilesActionsMenuKeys(msg); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleFilesMainKeys(msg); handled {
		return next, cmd
	}

	if m.filesTab.treeView {
		return m.handleFilesTreeKeys(msg)
	}
	return m.handleFilesFlatKeys(msg)
}

func (m Model) handleFilesViewPickerOverlayKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.overlays.showViewPicker {
		return m, nil, false
	}

	switch {
	case key.Matches(msg, ChezViewPickerKeys.Dismiss):
		m.overlays.showViewPicker = false
		return m, nil, true
	case key.Matches(msg, ChezViewPickerKeys.Up):
		m.overlays.viewPickerCursor = m.nextViewFilterCursor(-1)
		return m, nil, true
	case key.Matches(msg, ChezViewPickerKeys.Down):
		m.overlays.viewPickerCursor = m.nextViewFilterCursor(1)
		return m, nil, true
	case key.Matches(msg, ChezFilterOverlayKeys.Toggle):
		m.toggleViewFilterSelection()
		return m, nil, true
	case key.Matches(msg, ChezSharedKeys.Tab1):
		if len(m.overlays.viewPickerItems) > 0 {
			m.overlays.viewPickerPendingMode = m.overlays.viewPickerItems[0].mode
			next, cmd := m.applyViewFilterOverlay()
			return next, cmd, true
		}
		return m, nil, true
	case key.Matches(msg, ChezSharedKeys.Tab2):
		if len(m.overlays.viewPickerItems) > 1 {
			m.overlays.viewPickerPendingMode = m.overlays.viewPickerItems[1].mode
			next, cmd := m.applyViewFilterOverlay()
			return next, cmd, true
		}
		return m, nil, true
	case key.Matches(msg, ChezSharedKeys.Tab3):
		if len(m.overlays.viewPickerItems) > 2 {
			m.overlays.viewPickerPendingMode = m.overlays.viewPickerItems[2].mode
			next, cmd := m.applyViewFilterOverlay()
			return next, cmd, true
		}
		return m, nil, true
	case key.Matches(msg, ChezSharedKeys.Tab4):
		if len(m.overlays.viewPickerItems) > 3 {
			m.overlays.viewPickerPendingMode = m.overlays.viewPickerItems[3].mode
			next, cmd := m.applyViewFilterOverlay()
			return next, cmd, true
		}
		return m, nil, true
	case key.Matches(msg, ChezManagedKeys.FilterOverlay):
		if firstFilterRow := m.viewFilterFirstFilterRow(); firstFilterRow >= 0 {
			m.overlays.viewPickerCursor = firstFilterRow
		}
		return m, nil, true
	case key.Matches(msg, ChezViewPickerKeys.Select):
		next, cmd := m.applyViewFilterOverlay()
		return next, cmd, true
	default:
		return m, nil, true
	}
}

func (m Model) handleFilesFilterOverlayKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.overlays.showFilterOverlay {
		return m, nil, false
	}

	switch {
	case key.Matches(msg, ChezFilterOverlayKeys.Dismiss):
		m.overlays.showFilterOverlay = false
		return m, nil, true
	case key.Matches(msg, ChezFilterOverlayKeys.Up):
		m.overlays.filterCursor = m.nextFilterCursor(-1)
		return m, nil, true
	case key.Matches(msg, ChezFilterOverlayKeys.Down):
		m.overlays.filterCursor = m.nextFilterCursor(1)
		return m, nil, true
	case key.Matches(msg, ChezFilterOverlayKeys.Toggle):
		m.toggleFilterCategory()
		return m, nil, true
	case key.Matches(msg, ChezFilterOverlayKeys.Apply):
		next, cmd := m.applyFilterOverlay()
		return next, cmd, true
	default:
		return m, nil, true
	}
}

func (m Model) handleFilesActionsMenuKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.actions.managedShow {
		return m, nil, false
	}

	switch {
	case key.Matches(msg, ChezActionMenuKeys.Close):
		m.actions.managedShow = false
		m.ui.message = ""
		return m, nil, true
	case key.Matches(msg, ChezActionMenuKeys.Up):
		m.actions.managedCursor = nextSelectableCursor(m.actions.managedItems, m.actions.managedCursor, -1)
		return m, nil, true
	case key.Matches(msg, ChezActionMenuKeys.Down):
		m.actions.managedCursor = nextSelectableCursor(m.actions.managedItems, m.actions.managedCursor, 1)
		return m, nil, true
	case key.Matches(msg, ChezActionMenuKeys.Select):
		if m.actions.managedCursor < len(m.actions.managedItems) {
			item := m.actions.managedItems[m.actions.managedCursor]
			if isChezmoiActionSelectable(item) {
				next, cmd := m.executeFilesAction(item.action)
				return next, cmd, true
			}
			if item.disabled {
				m.ui.message = actionUnavailableMessage(item.unavailableReason)
			}
		}
		return m, nil, true
	default:
		return m, nil, true
	}
}

func (m Model) handleFilesMainKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, ChezSharedKeys.Back):
		next, cmd := m.escCmd()
		return next, cmd, true
	case key.Matches(msg, ChezManagedKeys.TreeToggle):
		m.filesTab.treeView = !m.filesTab.treeView
		m.filesTab.cursor = 0
		m.actions.managedShow = false
		return m, nil, true
	case key.Matches(msg, ChezManagedKeys.ViewPicker):
		m.openViewPicker()
		return m, nil, true
	case key.Matches(msg, ChezManagedKeys.FilterOverlay):
		m.openViewPicker()
		if firstFilterRow := m.viewFilterFirstFilterRow(); firstFilterRow >= 0 {
			m.overlays.viewPickerCursor = firstFilterRow
		}
		return m, nil, true
	case key.Matches(msg, ChezManagedKeys.ClearSearch):
		if m.filesTab.treeView && strings.TrimSpace(m.filterInput.Value()) != "" {
			m.filterInput.SetValue("")
			m.applyManagedFilter()
			m.resetFilesSearch(true)
			return m, nil, true
		}
		return m, nil, false
	case key.Matches(msg, ChezManagedKeys.Refresh):
		next, cmd := m.refreshFilesViewMode()
		return next, cmd, true
	default:
		return m, nil, false
	}
}

func (m Model) refreshFilesViewMode() (tea.Model, tea.Cmd) {
	m.ui.message = ""
	m.nextGen()
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		m.filesTab.views[managedViewIgnored].loading = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadIgnoredCmd())
	case managedViewUnmanaged:
		m.filesTab.views[managedViewUnmanaged].loading = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadUnmanagedCmd())
	case managedViewAll:
		m.filesTab.views[managedViewManaged].loading = true
		m.filesTab.views[managedViewIgnored].loading = true
		m.filesTab.views[managedViewUnmanaged].loading = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadManagedCmd(), m.loadIgnoredCmd(), m.loadUnmanagedCmd())
	default:
		m.filesTab.views[managedViewManaged].loading = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadManagedCmd())
	}
}

func (m Model) handleFilesTreeKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.activeTreeRows()
	switch {
	case key.Matches(msg, ChezSharedKeys.Up):
		next := moveCursorUp(m.filesTab.cursor, navigationStepForKey(msg))
		if next != m.filesTab.cursor {
			m.filesTab.cursor = next
			if m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForManaged()
				return m, panelCmd
			}
		}
	case key.Matches(msg, ChezSharedKeys.Down):
		next := moveCursorDown(m.filesTab.cursor, len(rows), navigationStepForKey(msg))
		if next != m.filesTab.cursor {
			m.filesTab.cursor = next
			if m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForManaged()
				return m, panelCmd
			}
		}
	case key.Matches(msg, ChezManagedKeys.Expand):
		return m.handleFilesTreeExpand()
	case key.Matches(msg, ChezManagedKeys.Collapse):
		return m.handleFilesTreeCollapse()
	case key.Matches(msg, ChezManagedKeys.Actions):
		m.openFilesActiveMenu()
	}
	return m, nil
}

func (m Model) handleFilesFlatKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	files := m.activeFlatFiles()
	switch {
	case key.Matches(msg, ChezSharedKeys.Up):
		next := moveCursorUp(m.filesTab.cursor, navigationStepForKey(msg))
		if next != m.filesTab.cursor {
			m.filesTab.cursor = next
			if m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForManaged()
				return m, panelCmd
			}
		}
	case key.Matches(msg, ChezSharedKeys.Down):
		next := moveCursorDown(m.filesTab.cursor, len(files), navigationStepForKey(msg))
		if next != m.filesTab.cursor {
			m.filesTab.cursor = next
			if m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForManaged()
				return m, panelCmd
			}
		}
	case key.Matches(msg, ChezManagedFlatKeys.Actions):
		m.openFilesActiveMenu()
	}
	return m, nil
}

func (m Model) handleFilesTreeExpand() (tea.Model, tea.Cmd) {
	rows := m.activeTreeRows()
	if m.filesTab.cursor < 0 || m.filesTab.cursor >= len(rows) {
		return m, nil
	}
	row := rows[m.filesTab.cursor]
	if row.node.isDir {
		if row.node.opaque && len(row.node.children) == 0 {
			if row.node.loading {
				return m, nil
			}
			requestID := m.nextOpaquePopulateRequestID()
			row.node.loading = true
			row.node.loadingRequest = requestID
			return m, m.populateOpaqueDirCmd(
				m.filesTab.viewMode,
				row.node.relPath,
				row.node.absPath,
				row.node.depth,
				requestID,
			)
		}
		row.node.expanded = !row.node.expanded
		m.reflattenActiveTree()
	} else {
		m.openFilesActiveMenu()
	}
	return m, nil
}

func (m Model) handleFilesTreeCollapse() (tea.Model, tea.Cmd) {
	rows := m.activeTreeRows()
	if m.filesTab.cursor < 0 || m.filesTab.cursor >= len(rows) {
		return m, nil
	}
	row := rows[m.filesTab.cursor]
	if row.node.isDir && row.node.expanded {
		row.node.expanded = false
		m.reflattenActiveTree()
	} else {
		parentIdx := findParentRow(rows, m.filesTab.cursor)
		if parentIdx >= 0 {
			m.filesTab.cursor = parentIdx
		}
	}
	return m, nil
}
