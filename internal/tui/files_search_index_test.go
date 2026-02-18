package tui

import (
	"testing"
)

func TestTriggerFilesSearchIfNeededQueuesDebounce(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Files"}
	m.activeTab = 0
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = []string{home}
	m.filesTab.search.paused = true
	m.filterInput.SetValue("settings")

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd == nil {
		t.Fatal("expected debounce command")
	}
	if !m.filesTab.search.searching {
		t.Fatal("expected searching=true")
	}
	if m.filesTab.search.paused {
		t.Fatal("expected paused=false when search is retriggered")
	}

	msg := cmd()
	debounced, ok := msg.(filesSearchDebouncedMsg)
	if !ok {
		t.Fatalf("expected filesSearchDebouncedMsg, got %T", msg)
	}
	if debounced.requestID != m.filesTab.search.request {
		t.Fatalf("expected request id %d, got %d", m.filesTab.search.request, debounced.requestID)
	}
}

func TestTriggerFilesSearchIfNeededCancelsInFlightSearch(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Files"}
	m.activeTab = 0
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = []string{home}
	m.filterInput.SetValue("settings")

	canceled := false
	m.filesTab.search.cancel = func() {
		canceled = true
	}
	beforeRequest := m.filesTab.search.request

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd == nil {
		t.Fatal("expected debounce command")
	}
	if !canceled {
		t.Fatal("expected in-flight search to be canceled")
	}
	if m.filesTab.search.request != beforeRequest+1 {
		t.Fatalf("expected request id to increment from %d to %d, got %d", beforeRequest, beforeRequest+1, m.filesTab.search.request)
	}
}

func TestTriggerFilesSearchIfNeededSkipsNonFilesTab(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Status", "Files"}
	m.activeTab = 0 // Status tab
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = []string{home}
	m.filterInput.SetValue("settings")

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd != nil {
		t.Fatal("expected no search command when not on Files tab")
	}
}

func TestTriggerFilesSearchIfNeededSkipsEmptyQuery(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Files"}
	m.activeTab = 0
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = []string{home}
	m.filterInput.SetValue("")

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd != nil {
		t.Fatal("expected no search command for empty query")
	}
}

func TestTriggerFilesSearchIfNeededUsesTargetPathWithoutUnmanagedRoots(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Files"}
	m.activeTab = 0
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = nil
	m.filterInput.SetValue("settings")

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd == nil {
		t.Fatal("expected search command when targetPath is available even without unmanaged roots")
	}
}

func TestTriggerFilesSearchIfNeededSkipsReadyResults(t *testing.T) {
	home := t.TempDir()
	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.tabNames = []string{"Files"}
	m.activeTab = 0
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewUnmanaged].files = []string{home}
	m.filterInput.SetValue("settings")
	m.filesTab.search.ready = true
	m.filesTab.search.query = "settings"

	cmd := m.triggerFilesSearchIfNeeded()
	if cmd != nil {
		t.Fatal("expected no search command when results already ready for same query")
	}
}

func TestClassifyPathMarksUnmanagedDescendant(t *testing.T) {
	m := Model{
		filesTab: filesTab{
			dataset: filesDataset{
				classMap:          map[string]fileClass{"/home/test/Documents/workspace": fileClassUnmanaged},
				unmanagedDirRoots: []string{"/home/test/Documents/workspace"},
			},
		},
	}
	if got := m.classifyPath("/home/test/Documents/workspace/notes.txt"); got != pathClassUnmanaged {
		t.Fatalf("expected unmanaged class, got %v", got)
	}
	if got := m.classifyPath("/home/test/.zshrc"); got != pathClassManaged {
		t.Fatalf("expected managed class, got %v", got)
	}
}
