package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestWindowSizeMsgResyncsPanelViewportWithCachedPath(t *testing.T) {
	path := "/tmp/example"

	m := NewModel(Options{Service: testService()})
	m.activeTab = 1 // Files
	m.filesTab.treeView = false
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.views[managedViewManaged].filteredFiles = []string{path}
	m.filesTab.cursor = 0

	m.panel = newFilePanel("show")
	m.panel.currentPath = path
	m.panel.contentMode = panelModeDiff
	m.panel.loading = false
	m.panel.cachePut(path, panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: "+line\n-line",
		lines:   []string{"+line", "-line"},
	})

	updatedAny, cmd := m.Update(tea.WindowSizeMsg{Width: 180, Height: 50})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if cmd != nil {
		t.Fatalf("expected nil cmd on resize with cached panel content")
	}
	if !updated.panel.viewportReady {
		t.Fatalf("expected panel viewport to be initialized on resize")
	}
	if updated.panel.viewport.Width() <= 0 || updated.panel.viewport.Height() <= 0 {
		t.Fatalf("expected positive viewport dimensions, got width=%d height=%d", updated.panel.viewport.Width(), updated.panel.viewport.Height())
	}
	if updated.panel.currentPath != path {
		t.Fatalf("expected current path %q, got %q", path, updated.panel.currentPath)
	}
}
