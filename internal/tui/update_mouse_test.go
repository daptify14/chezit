package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func setupFilesMouseModel(t *testing.T, tree bool) Model {
	t.Helper()

	home := t.TempDir()
	files := []string{
		filepath.Join(home, ".config", "nvim", "init.lua"),
		filepath.Join(home, ".config", "nvim", "lua", "plugins.lua"),
		filepath.Join(home, ".config", "git", "config"),
		filepath.Join(home, ".local", "bin", "chezit-helper"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".gitconfig"),
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.view = StatusScreen
	m.activeTab = 1 // Files
	m.width = 100   // keep panel hidden so list mouse interactions are deterministic
	m.height = 40
	m.targetPath = home
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.treeView = tree
	m.filesTab.views[managedViewManaged].files = files
	m.filesTab.views[managedViewManaged].filteredFiles = files
	m.rebuildFileViewTree(managedViewManaged)
	for _, root := range m.filesTab.views[managedViewManaged].tree.roots {
		expandAll(root)
	}
	m.filesTab.views[managedViewManaged].treeRows = flattenManagedTree(m.filesTab.views[managedViewManaged].tree)
	return m
}

func asModel(t *testing.T, got tea.Model) Model {
	t.Helper()
	typed, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", got)
	}
	return typed
}

func TestMouseClickFilesTreeSelectsRow(t *testing.T) {
	m := setupFilesMouseModel(t, true)
	rows := m.activeTreeRows()
	if len(rows) < 2 {
		t.Fatalf("precondition: expected at least 2 tree rows, got %d", len(rows))
	}

	msg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      2,
		Y:      8, // header(7) + row(1)
	}
	updated := asModel(t, func() tea.Model {
		next, _ := m.handleMouseClick(msg)
		return next
	}())

	if updated.filesTab.cursor != 1 {
		t.Fatalf("expected files cursor at 1 after click, got %d", updated.filesTab.cursor)
	}
}

func TestMouseClickFilesTreeOnSelectedDirectoryTogglesCollapse(t *testing.T) {
	m := setupFilesMouseModel(t, true)
	rows := m.activeTreeRows()

	dirIdx := -1
	var dirRel string
	for i, row := range rows {
		if row.node.isDir && row.node.expanded && len(row.node.children) > 0 {
			dirIdx = i
			dirRel = row.node.relPath
			break
		}
	}
	if dirIdx < 0 {
		t.Fatal("precondition: expected expanded directory row with children")
	}
	m.filesTab.cursor = dirIdx
	beforeCount := len(rows)

	msg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      2,
		Y:      7 + dirIdx,
	}
	updated := asModel(t, func() tea.Model {
		next, _ := m.handleMouseClick(msg)
		return next
	}())

	afterRows := updated.activeTreeRows()
	if len(afterRows) >= beforeCount {
		t.Fatalf("expected collapsed tree to reduce visible rows: before=%d after=%d", beforeCount, len(afterRows))
	}

	node := findTreeNodeByRelPath(updated.activeTree(), dirRel)
	if node == nil {
		t.Fatalf("expected directory %q to remain in tree", dirRel)
	}
	if node.expanded {
		t.Fatalf("expected directory %q to be collapsed after click", dirRel)
	}
}

func TestMouseClickFilesFlatOnSelectedRowOpensActions(t *testing.T) {
	m := setupFilesMouseModel(t, false)
	m.filesTab.cursor = 0

	msg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      2,
		Y:      7, // header(7) + row(0)
	}
	updated := asModel(t, func() tea.Model {
		next, _ := m.handleMouseClick(msg)
		return next
	}())

	if !updated.actions.managedShow {
		t.Fatal("expected clicking selected flat row to open files actions menu")
	}
}

func TestMouseWheelFilesTreeMovesCursor(t *testing.T) {
	m := setupFilesMouseModel(t, true)
	if len(m.activeTreeRows()) < 4 {
		t.Fatalf("precondition: expected at least 4 tree rows, got %d", len(m.activeTreeRows()))
	}

	downMsg := tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		X:      2,
		Y:      10,
	}
	down := asModel(t, func() tea.Model {
		next, _ := m.handleMouseWheel(downMsg)
		return next
	}())

	if down.filesTab.cursor <= 0 {
		t.Fatalf("expected cursor to move down after mouse wheel, got %d", down.filesTab.cursor)
	}

	upMsg := tea.MouseWheelMsg{
		Button: tea.MouseWheelUp,
		X:      2,
		Y:      10,
	}
	up := asModel(t, func() tea.Model {
		next, _ := down.handleMouseWheel(upMsg)
		return next
	}())

	if up.filesTab.cursor >= down.filesTab.cursor {
		t.Fatalf("expected cursor to move up after wheel-up: before=%d after=%d", down.filesTab.cursor, up.filesTab.cursor)
	}
}

func TestMouseWheelIgnoredWhileManagedActionsOpen(t *testing.T) {
	m := setupFilesMouseModel(t, true)
	m.filesTab.cursor = 1
	m.actions.managedShow = true

	msg := tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		X:      2,
		Y:      10,
	}
	updated := asModel(t, func() tea.Model {
		next, _ := m.handleMouseWheel(msg)
		return next
	}())

	if updated.filesTab.cursor != 1 {
		t.Fatalf("expected cursor unchanged while managed actions menu is open, got %d", updated.filesTab.cursor)
	}
}

func TestMouseClickStatusMovesCursorAndLoadsPanel(t *testing.T) {
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			{Path: "/home/test/.zshrc", SourceStatus: 'A', DestStatus: ' '},
		}),
		WithPanelVisible(),
		WithSize(120, 40),
	)
	m.view = StatusScreen
	m.activeTab = 0

	first := findFirstSectionFileRow(t, m, changesSectionDrift)
	second := first + 1
	if second >= len(m.status.changesRows) {
		t.Fatalf("precondition: expected second drift row, rows=%d", len(m.status.changesRows))
	}
	m.status.changesCursor = first

	msg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      2,
		Y:      statusFilesHeaderLines + second,
	}

	next, cmd := m.handleMouseClick(msg)
	updated := asModel(t, next)

	if updated.status.changesCursor != second {
		t.Fatalf("expected cursor to move to row %d, got %d", second, updated.status.changesCursor)
	}
	if cmd == nil {
		t.Fatal("expected non-nil panel load command after clicking a different status row")
	}
	if updated.panel.currentPath != "/home/test/.zshrc" {
		t.Fatalf("expected panel path to update to clicked row, got %q", updated.panel.currentPath)
	}
}
