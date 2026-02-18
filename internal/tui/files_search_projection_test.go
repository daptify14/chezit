package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectFilteredTreeRowsIncludesAncestors(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	rows := projectFilteredTreeRows([]string{notes}, home)
	if len(rows) < 2 {
		t.Fatalf("expected ancestor + leaf rows, got %d", len(rows))
	}

	hasParent := false
	hasLeaf := false
	for _, row := range rows {
		if row.node.relPath == "Documents/workspace" && row.node.isDir {
			hasParent = true
		}
		if row.node.relPath == "Documents/workspace/notes.txt" && !row.node.isDir {
			hasLeaf = true
		}
	}
	if !hasParent || !hasLeaf {
		t.Fatalf("expected projected rows to include parent and leaf, got %#v", rows)
	}
}

func TestApplyManagedFilterUsesUnmanagedSearchResults(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.treeView = true
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.search.rawResults = []string{notes}
	m.filesTab.search.query = "notes.txt"
	m.filesTab.search.ready = true
	m.filterInput.SetValue("notes.txt")

	m.applyManagedFilter()

	if len(m.filesTab.views[managedViewUnmanaged].filteredFiles) == 0 {
		t.Fatalf("expected unmanaged filtered results, got none")
	}
	if m.filesTab.views[managedViewUnmanaged].filteredFiles[0] != notes {
		t.Fatalf("expected first filtered result %q, got %q", notes, m.filesTab.views[managedViewUnmanaged].filteredFiles[0])
	}

	hasLeaf := false
	for _, row := range m.filesTab.views[managedViewUnmanaged].treeRows {
		if row.node.relPath == "Documents/workspace/notes.txt" {
			hasLeaf = true
			break
		}
	}
	if !hasLeaf {
		t.Fatalf("expected projected tree rows to include deep file, got %#v", m.filesTab.views[managedViewUnmanaged].treeRows)
	}
}

func TestFilesSearchPathsForModeProjectsUnmanagedSubsetFromRawResults(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	managed := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(managed, []byte("export ZDOTDIR=$HOME\n"), 0o644); err != nil {
		t.Fatalf("write managed: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.views[managedViewManaged].files = []string{managed}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.filesTab.search.ready = true
	m.filesTab.search.query = "notes"
	// Simulate accidentally mixed-in results from an overly broad walk.
	m.filesTab.search.rawResults = []string{managed, notes}

	got := m.filesSearchPathsForMode("notes", managedViewUnmanaged)
	if len(got) != 1 || got[0] != notes {
		t.Fatalf("expected only unmanaged descendant %q, got %#v", notes, got)
	}
}

func TestFilesSearchPathsForModeAllExcludesUnknownPaths(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	managed := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(managed, []byte("export ZDOTDIR=$HOME\n"), 0o644); err != nil {
		t.Fatalf("write managed: %v", err)
	}
	ignored := filepath.Join(home, ".cache", "ignored.log")
	if err := os.MkdirAll(filepath.Dir(ignored), 0o755); err != nil {
		t.Fatalf("mkdir ignored dir: %v", err)
	}
	if err := os.WriteFile(ignored, []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}
	unknown := filepath.Join(home, "other", "outside.txt")
	if err := os.MkdirAll(filepath.Dir(unknown), 0o755); err != nil {
		t.Fatalf("mkdir unknown dir: %v", err)
	}
	if err := os.WriteFile(unknown, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write unknown: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.views[managedViewManaged].files = []string{managed}
	m.filesTab.views[managedViewIgnored].files = []string{ignored}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.filesTab.search.query = "notes"
	m.filesTab.search.ready = true
	m.filesTab.search.rawResults = []string{managed, ignored, notes, unknown}

	got := m.filesSearchPathsForMode("notes", managedViewAll)
	want := []string{managed, ignored, notes}
	if len(got) != len(want) {
		t.Fatalf("expected %d projected all-view paths, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected projected[%d]=%q, got %q", i, want[i], got[i])
		}
	}
}

func TestFilesSearchPathsForModeWaitsForReadyResults(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.filesTab.search.query = "notes"
	m.filesTab.search.ready = false
	m.filesTab.search.rawResults = []string{notes}

	got := m.filesSearchPathsForMode("notes", managedViewUnmanaged)
	if len(got) != 0 {
		t.Fatalf("expected no provisional search paths before ready, got %#v", got)
	}
}

func TestFuzzyMatchFilesRequiresContiguousSubstring(t *testing.T) {
	home := t.TempDir()
	homePrefix := home + string(filepath.Separator)

	contiguousPath := filepath.Join(home, ".scripts", "darwin", "sync-token-servers.sh")
	fuzzyOnlyPath := filepath.Join(home, ".scripts", "darwin", "sync-t-o-k-e-n-servers.sh")

	filtered, _ := fuzzyMatchFiles("token", []string{fuzzyOnlyPath, contiguousPath}, homePrefix)
	if len(filtered) != 1 {
		t.Fatalf("expected exactly 1 contiguous match, got %#v", filtered)
	}
	if filtered[0] != contiguousPath {
		t.Fatalf("expected %q, got %q", contiguousPath, filtered[0])
	}
}

func TestOpenActiveActionsMenuAllViewTreatsDeepUnmanagedAsUnmanaged(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.viewMode = managedViewAll
	m.filesTab.treeView = false
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewAll)
	m.filesTab.views[managedViewUnmanaged].filteredFiles = []string{notes}
	m.filesTab.cursor = 0

	m.openFilesActiveMenu()

	if len(m.actions.managedItems) == 0 {
		t.Fatal("expected unmanaged actions to be opened")
	}
	if !strings.HasPrefix(m.actions.managedItems[0].label, "Add") {
		t.Fatalf("expected unmanaged Add action first, got %q", m.actions.managedItems[0].label)
	}
}

func TestOpenUnmanagedActionsMenuFileIncludesOpenInEditor(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.treeView = false
	m.filesTab.views[managedViewUnmanaged].filteredFiles = []string{notes}
	m.filesTab.cursor = 0

	m.openFilesUnmanagedMenu()

	hasOpenInEditor := false
	for _, item := range m.actions.managedItems {
		if item.action == chezmoiActionEditTarget {
			hasOpenInEditor = true
			break
		}
	}
	if !hasOpenInEditor {
		t.Fatal("expected unmanaged file actions to include Open in Editor")
	}
}

func TestOpenUnmanagedActionsMenuDirectoryOmitsOpenInEditor(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "Documents", "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	notes := filepath.Join(workspaceDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.treeView = true
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.views[managedViewUnmanaged].filteredFiles = m.filesTab.views[managedViewUnmanaged].files
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.cursor = 0

	m.openFilesUnmanagedMenu()

	for _, item := range m.actions.managedItems {
		if item.action == chezmoiActionEditTarget {
			t.Fatal("expected unmanaged directory actions to omit Open in Editor")
		}
	}
}
