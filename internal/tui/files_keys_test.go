package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHandleManagedTreeKeysEditShortcutNoOp(t *testing.T) {
	home := t.TempDir()
	managedPath := filepath.Join(home, ".config", "chezit", "config.toml")
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(managedPath, []byte("x=1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.views[managedViewManaged].files = []string{managedPath}
	m.filesTab.views[managedViewManaged].filteredFiles = m.filesTab.views[managedViewManaged].files
	m.rebuildFileViewTree(managedViewManaged)

	rows := m.activeTreeRows()
	fileRow := -1
	for i, row := range rows {
		if !row.node.isDir && row.node.absPath == managedPath {
			fileRow = i
			break
		}
	}
	if fileRow == -1 {
		t.Fatal("expected managed file row in tree")
	}
	m.filesTab.cursor = fileRow

	_, cmd := m.handleFilesTreeKeys(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if cmd != nil {
		t.Fatal("expected no command for removed tree edit shortcut")
	}
}

func TestHandleManagedFlatKeysEditShortcutNoOp(t *testing.T) {
	home := t.TempDir()
	managedPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(managedPath, []byte("export ZDOTDIR=$HOME\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.treeView = false
	m.filesTab.views[managedViewManaged].filteredFiles = []string{managedPath}
	m.filesTab.cursor = 0

	_, cmd := m.handleFilesFlatKeys(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if cmd != nil {
		t.Fatal("expected no command for removed flat edit shortcut")
	}
}

func TestHandleManagedTreeKeysRepeatDownAccelerates(t *testing.T) {
	home := t.TempDir()
	files := []string{
		filepath.Join(home, ".config", "nvim", "init.lua"),
		filepath.Join(home, ".config", "git", "config"),
		filepath.Join(home, ".config", "starship.toml"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".gitignore"),
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.views[managedViewManaged].files = files
	m.filesTab.views[managedViewManaged].filteredFiles = files
	m.rebuildFileViewTree(managedViewManaged)
	m.filesTab.cursor = 0

	rows := m.activeTreeRows()
	if len(rows) < 4 {
		t.Fatalf("precondition: expected at least 4 tree rows, got %d", len(rows))
	}

	updatedAny, _ := m.handleFilesTreeKeys(tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model from handleFilesTreeKeys")
	}

	want := min(3, len(rows)-1)
	if updated.filesTab.cursor != want {
		t.Fatalf("expected repeat down to jump to %d in tree view, got %d", want, updated.filesTab.cursor)
	}
}

func TestHandleManagedFlatKeysRepeatDownAccelerates(t *testing.T) {
	home := t.TempDir()
	files := []string{
		filepath.Join(home, ".config", "nvim", "init.lua"),
		filepath.Join(home, ".config", "git", "config"),
		filepath.Join(home, ".config", "starship.toml"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.treeView = false
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.views[managedViewManaged].filteredFiles = files
	m.filesTab.cursor = 0

	updatedAny, _ := m.handleFilesFlatKeys(tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	want := min(3, len(files)-1)
	if updated.filesTab.cursor != want {
		t.Fatalf("expected repeat down to jump to %d in flat view, got %d", want, updated.filesTab.cursor)
	}
}

func TestHandleFilesTreeCollapsePreservesActiveSearchProjection(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "workspace")
	alphaDir := filepath.Join(workspaceDir, "alpha")
	otherDir := filepath.Join(workspaceDir, "other")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other: %v", err)
	}
	alphaFile := filepath.Join(alphaDir, "client.txt")
	if err := os.WriteFile(alphaFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	otherFile := filepath.Join(otherDir, "notes.txt")
	if err := os.WriteFile(otherFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.search.ready = true
	m.filesTab.search.query = "alpha"
	m.filesTab.search.rawResults = []string{alphaFile}
	m.filterInput.SetValue("alpha")
	m.applyManagedFilter()

	rows := m.activeTreeRows()
	collapseIdx := -1
	for i, row := range rows {
		if row.node.relPath == "workspace/alpha" && row.node.isDir {
			collapseIdx = i
			break
		}
	}
	if collapseIdx < 0 {
		t.Fatalf("expected workspace/alpha directory in filtered rows, got %#v", rows)
	}
	m.filesTab.cursor = collapseIdx

	updatedAny, _ := m.handleFilesTreeCollapse()
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.filterInput.Value() != "alpha" {
		t.Fatalf("expected filter input to remain alpha, got %q", updated.filterInput.Value())
	}
	hasAlphaFile := false
	for _, row := range updated.activeTreeRows() {
		if strings.HasPrefix(row.node.relPath, "workspace/other") {
			t.Fatalf("expected filtered projection to remain active, found unrelated row %q", row.node.relPath)
		}
		if row.node.relPath == "workspace/alpha/client.txt" {
			hasAlphaFile = true
		}
	}
	if hasAlphaFile {
		t.Fatal("expected alpha file to be hidden after collapsing workspace/alpha")
	}

	expandIdx := -1
	for i, row := range updated.activeTreeRows() {
		if row.node.relPath == "workspace/alpha" && row.node.isDir {
			expandIdx = i
			break
		}
	}
	if expandIdx < 0 {
		t.Fatal("expected workspace/alpha directory to remain visible after collapse")
	}
	updated.filesTab.cursor = expandIdx

	reexpandedAny, _ := updated.handleFilesTreeExpand()
	reexpanded, ok := reexpandedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	hasAlphaFile = false
	for _, row := range reexpanded.activeTreeRows() {
		if row.node.relPath == "workspace/alpha/client.txt" {
			hasAlphaFile = true
		}
		if strings.HasPrefix(row.node.relPath, "workspace/other") {
			t.Fatalf("expected filtered projection to remain active after re-expand, found unrelated row %q", row.node.relPath)
		}
	}
	if !hasAlphaFile {
		t.Fatal("expected alpha file to reappear after expanding workspace/alpha")
	}
}

func TestClearSearchKeyResetsFilteredTreeProjection(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "workspace")
	alphaDir := filepath.Join(workspaceDir, "alpha")
	otherDir := filepath.Join(workspaceDir, "other")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other: %v", err)
	}
	alphaFile := filepath.Join(alphaDir, "client.txt")
	if err := os.WriteFile(alphaFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	otherFile := filepath.Join(otherDir, "notes.txt")
	if err := os.WriteFile(otherFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.view = StatusScreen
	m.activeTab = 1 // Files
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.search.ready = true
	m.filesTab.search.query = "alpha"
	m.filesTab.search.rawResults = []string{alphaFile}
	m.filterInput.SetValue("alpha")
	m.applyManagedFilter()

	updated, _ := sendKey(t, m, runeKey("c"))
	if updated.filterInput.Value() != "" {
		t.Fatalf("expected filter to be cleared by c key, got %q", updated.filterInput.Value())
	}
	if updated.filesTab.search.ready || updated.filesTab.search.searching {
		t.Fatal("expected search state reset after clearing search")
	}

	hasFilteredLeaf := false
	hasWorkspaceRoot := false
	for _, row := range updated.activeTreeRows() {
		switch row.node.relPath {
		case "workspace/alpha/client.txt":
			hasFilteredLeaf = true
		case "workspace":
			hasWorkspaceRoot = true
		}
	}
	if hasFilteredLeaf {
		t.Fatal("expected filtered leaf rows to be cleared after c clear search")
	}
	if !hasWorkspaceRoot {
		t.Fatal("expected base workspace root row after c clear search")
	}
}

func TestEscToLandingThenFilesResetsFilteredTreeProjection(t *testing.T) {
	home := t.TempDir()
	workspaceDir := filepath.Join(home, "workspace")
	alphaDir := filepath.Join(workspaceDir, "alpha")
	otherDir := filepath.Join(workspaceDir, "other")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other: %v", err)
	}
	alphaFile := filepath.Join(alphaDir, "client.txt")
	if err := os.WriteFile(alphaFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	otherFile := filepath.Join(otherDir, "notes.txt")
	if err := os.WriteFile(otherFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home), EscBehavior: EscQuit})
	m.view = StatusScreen
	m.activeTab = 1 // Files
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspaceDir}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.search.ready = true
	m.filesTab.search.query = "alpha"
	m.filesTab.search.rawResults = []string{alphaFile}
	m.filterInput.SetValue("alpha")
	m.applyManagedFilter()

	landing, _ := sendKey(t, m, specialKey(tea.KeyEsc))
	if landing.view != LandingScreen {
		t.Fatalf("expected landing screen after Esc, got %v", landing.view)
	}

	reenteredAny, _ := landing.enterTabFromLanding(1)
	reentered, ok := reenteredAny.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", reenteredAny)
	}
	if reentered.filterInput.Value() != "" {
		t.Fatalf("expected filter cleared after re-entering Files tab, got %q", reentered.filterInput.Value())
	}

	hasFilteredLeaf := false
	hasWorkspaceRoot := false
	for _, row := range reentered.activeTreeRows() {
		switch row.node.relPath {
		case "workspace/alpha/client.txt":
			hasFilteredLeaf = true
		case "workspace":
			hasWorkspaceRoot = true
		}
	}
	if hasFilteredLeaf {
		t.Fatal("expected filtered leaf rows to be cleared after re-entering Files tab")
	}
	if !hasWorkspaceRoot {
		t.Fatal("expected base workspace root row after re-entering Files tab")
	}
}

func TestHandleManagedTreeExpandOpaqueQueuesPopulateCmd(t *testing.T) {
	rootAbs := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAbs, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(rootAbs)})
	m.activeTab = 1 // Files
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.views[managedViewManaged].tree = managedTree{
		roots: []*managedTreeNode{
			{
				name:    "opaque",
				relPath: "opaque",
				absPath: rootAbs,
				isDir:   true,
				opaque:  true,
				depth:   0,
			},
		},
	}
	m.reflattenActiveTree()
	m.filesTab.cursor = 0

	updatedAny, cmd := m.handleFilesTreeExpand()
	if cmd == nil {
		t.Fatal("expected async populate command for opaque directory")
	}

	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}
	node := updated.activeTreeRows()[0].node
	if !node.loading {
		t.Fatal("expected loading=true for in-flight populate")
	}
	if node.loadingRequest == 0 {
		t.Fatal("expected non-zero loading request id")
	}
	if node.expanded {
		t.Fatal("expected node to remain collapsed until async load finishes")
	}

	msg := cmd()
	populated, ok := msg.(opaqueDirPopulatedMsg)
	if !ok {
		t.Fatalf("expected opaqueDirPopulatedMsg, got %T", msg)
	}
	if populated.requestID != node.loadingRequest {
		t.Fatalf("expected request id %d, got %d", node.loadingRequest, populated.requestID)
	}
	if populated.relPath != "opaque" {
		t.Fatalf("expected relPath opaque, got %q", populated.relPath)
	}
}

func TestOpaqueDirPopulatedMsgAppliesChildren(t *testing.T) {
	rootAbs := t.TempDir()

	m := newOpaqueTreeModelForTest(rootAbs)
	node := m.activeTreeRows()[0].node
	node.loading = true
	node.loadingRequest = 1

	msg := opaqueDirPopulatedMsg{
		viewMode:  managedViewManaged,
		relPath:   "opaque",
		gen:       m.gen,
		requestID: 1,
		children: []*managedTreeNode{
			{
				name:    "a.txt",
				relPath: "opaque/a.txt",
				absPath: filepath.Join(rootAbs, "a.txt"),
				isDir:   false,
				depth:   1,
			},
		},
	}

	updatedAny, _ := m.Update(msg)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}
	got := findTreeNodeByRelPath(updated.filesTab.views[managedViewManaged].tree, "opaque")
	if got == nil {
		t.Fatal("expected opaque node")
	}
	if got.loading || got.loadingRequest != 0 {
		t.Fatalf("expected loading cleared, got loading=%v request=%d", got.loading, got.loadingRequest)
	}
	if got.opaque {
		t.Fatal("expected opaque=false after successful populate")
	}
	if !got.expanded {
		t.Fatal("expected node expanded after successful populate")
	}
	if len(got.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(got.children))
	}
	if got.children[0].parent != got {
		t.Fatal("expected child parent to be linked to populated node")
	}
}

func TestOpaqueDirPopulatedMsgOldRequestIgnored(t *testing.T) {
	rootAbs := t.TempDir()

	m := newOpaqueTreeModelForTest(rootAbs)
	node := m.activeTreeRows()[0].node
	node.loading = true
	node.loadingRequest = 2

	msg := opaqueDirPopulatedMsg{
		viewMode:  managedViewManaged,
		relPath:   "opaque",
		gen:       m.gen,
		requestID: 1, // stale/older request
		children: []*managedTreeNode{
			{name: "ignored", relPath: "opaque/ignored", absPath: filepath.Join(rootAbs, "ignored"), depth: 1},
		},
	}

	updatedAny, _ := m.Update(msg)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}
	got := findTreeNodeByRelPath(updated.filesTab.views[managedViewManaged].tree, "opaque")
	if got == nil {
		t.Fatal("expected opaque node")
	}
	if !got.loading || got.loadingRequest != 2 {
		t.Fatalf("expected loading to remain for active request, got loading=%v request=%d", got.loading, got.loadingRequest)
	}
	if len(got.children) != 0 {
		t.Fatalf("expected children unchanged, got %d", len(got.children))
	}
	if !got.opaque {
		t.Fatal("expected opaque to remain true")
	}
}

func TestOpaqueDirPopulatedMsgStaleGenClearsMatchingLoading(t *testing.T) {
	rootAbs := t.TempDir()

	m := newOpaqueTreeModelForTest(rootAbs)
	node := m.activeTreeRows()[0].node
	node.loading = true
	node.loadingRequest = 5

	msg := opaqueDirPopulatedMsg{
		viewMode:  managedViewManaged,
		relPath:   "opaque",
		gen:       m.gen + 1, // stale generation
		requestID: 5,
		children: []*managedTreeNode{
			{name: "ignored", relPath: "opaque/ignored", absPath: filepath.Join(rootAbs, "ignored"), depth: 1},
		},
	}

	updatedAny, _ := m.Update(msg)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}
	got := findTreeNodeByRelPath(updated.filesTab.views[managedViewManaged].tree, "opaque")
	if got == nil {
		t.Fatal("expected opaque node")
	}
	if got.loading || got.loadingRequest != 0 {
		t.Fatalf("expected matching loading cleared, got loading=%v request=%d", got.loading, got.loadingRequest)
	}
	if len(got.children) != 0 {
		t.Fatalf("expected children unchanged on stale generation, got %d", len(got.children))
	}
	if !got.opaque {
		t.Fatal("expected opaque to remain true on stale generation")
	}
	if got.expanded {
		t.Fatal("expected node to remain collapsed on stale generation")
	}
}

func TestOpaqueDirPopulatedMsgPreservesFilteredProjection(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace")
	alphaFile := filepath.Join(workspace, "alpha.txt")
	betaFile := filepath.Join(workspace, "beta.txt")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(alphaFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := os.WriteFile(betaFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write beta: %v", err)
	}

	m := NewModel(Options{Service: testServiceWithTarget(home)})
	m.activeTab = 1 // Files
	m.targetPath = home
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewUnmanaged
	m.filesTab.views[managedViewManaged].files = []string{}
	m.filesTab.views[managedViewIgnored].files = []string{}
	m.filesTab.views[managedViewUnmanaged].files = []string{workspace}
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	m.rebuildFileViewTree(managedViewUnmanaged)
	m.filesTab.search.ready = true
	m.filesTab.search.query = "alpha"
	m.filesTab.search.rawResults = []string{alphaFile}
	m.filterInput.SetValue("alpha")
	m.applyManagedFilter()

	node := findTreeNodeByRelPath(m.filesTab.views[managedViewUnmanaged].tree, "workspace")
	if node == nil {
		t.Fatal("expected workspace node")
	}
	node.loading = true
	node.loadingRequest = 1

	msg := opaqueDirPopulatedMsg{
		viewMode:  managedViewUnmanaged,
		relPath:   "workspace",
		gen:       m.gen,
		requestID: 1,
		children: []*managedTreeNode{
			{
				name:    "alpha.txt",
				relPath: "workspace/alpha.txt",
				absPath: alphaFile,
				isDir:   false,
				depth:   1,
			},
			{
				name:    "beta.txt",
				relPath: "workspace/beta.txt",
				absPath: betaFile,
				isDir:   false,
				depth:   1,
			},
		},
	}

	updatedAny, _ := m.Update(msg)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	hasAlpha := false
	for _, row := range updated.activeTreeRows() {
		switch row.node.relPath {
		case "workspace/alpha.txt":
			hasAlpha = true
		case "workspace/beta.txt":
			t.Fatal("expected filtered projection to exclude non-matching beta.txt")
		}
	}
	if !hasAlpha {
		t.Fatal("expected filtered projection to keep alpha.txt")
	}
}

func newOpaqueTreeModelForTest(rootAbs string) Model {
	m := NewModel(Options{Service: testServiceWithTarget(rootAbs)})
	m.activeTab = 1 // Files
	m.filesTab.treeView = true
	m.filesTab.viewMode = managedViewManaged
	m.filesTab.views[managedViewManaged].tree = managedTree{
		roots: []*managedTreeNode{
			{
				name:    "opaque",
				relPath: "opaque",
				absPath: rootAbs,
				isDir:   true,
				opaque:  true,
				depth:   0,
			},
		},
	}
	m.reflattenActiveTree()
	m.filesTab.cursor = 0
	return m
}
