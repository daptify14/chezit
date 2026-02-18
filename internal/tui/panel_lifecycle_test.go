package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestCacheOperations(t *testing.T) {
	p := newFilePanel("")

	// Miss
	_, ok := p.cacheGet("/foo", panelModeDiff, changesSectionDrift)
	if ok {
		t.Error("expected cache miss on empty cache")
	}

	// Put and hit
	entry := panelCacheEntry{content: "hello", lines: []string{"hello"}}
	p.cachePut("/foo", panelModeDiff, changesSectionDrift, entry)

	got, ok := p.cacheGet("/foo", panelModeDiff, changesSectionDrift)
	if !ok {
		t.Error("expected cache hit after put")
	}
	if got.content != "hello" {
		t.Errorf("expected content 'hello', got %q", got.content)
	}

	// Different mode is a miss
	_, ok = p.cacheGet("/foo", panelModeContent, changesSectionDrift)
	if ok {
		t.Error("expected cache miss for different mode")
	}

	// Clear
	p.clearCache()
	_, ok = p.cacheGet("/foo", panelModeDiff, changesSectionDrift)
	if ok {
		t.Error("expected cache miss after clearCache")
	}
}

func TestCacheTrim(t *testing.T) {
	p := newFilePanel("")

	// Fill beyond max
	for i := range panelMaxCacheSize + 10 {
		p.cachePut("/path"+string(rune('0'+i%10)), panelContentMode(i%2), changesSectionDrift, panelCacheEntry{content: "x"})
	}

	// After trimming, cache should be well under max
	if len(p.cache) > panelMaxCacheSize {
		t.Errorf("cache should have been trimmed, got %d entries", len(p.cache))
	}
}

func TestResetForTab(t *testing.T) {
	p := newFilePanel("")
	p.contentMode = panelModeContent
	p.currentPath = "/some/file"
	p.cachePut("/some/file", panelModeContent, changesSectionDrift, panelCacheEntry{content: "x"})

	p.resetForTab("Status")
	if p.contentMode != panelModeDiff {
		t.Error("expected diff mode after reset for Status tab")
	}
	if p.currentPath != "" {
		t.Error("expected empty currentPath after reset")
	}
	if _, ok := p.cacheGet("/some/file", panelModeContent, changesSectionDrift); ok {
		t.Error("expected cache cleared after reset")
	}

	p.contentMode = panelModeDiff
	p.resetForTab("Files")
	if p.contentMode != panelModeContent {
		t.Error("expected content mode after reset for Files tab")
	}
}

func TestHandlePanelContentLoadedStaleGuard(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentPath = "/current/file"
	m.panel.contentMode = panelModeDiff

	// Simulate a stale response for a different path
	staleMsg := panelContentLoadedMsg{
		path:    "/old/file",
		mode:    panelModeDiff,
		section: changesSectionDrift,
		content: "stale diff content",
	}

	newM, cmd := m.handlePanelContentLoaded(staleMsg)
	if cmd != nil {
		t.Fatalf("expected nil cmd without pending load, got %#v", cmd)
	}

	// Should still be cached
	entry, ok := newM.panel.cacheGet("/old/file", panelModeDiff, changesSectionDrift)
	if !ok {
		t.Error("expected stale result to still be cached")
	}
	if entry.content != "stale diff content" {
		t.Errorf("unexpected cached content: %q", entry.content)
	}

	// Current state should match what we set, not the stale msg
	if newM.panel.currentPath != "/current/file" {
		t.Error("panel current path should not change on stale result")
	}
}

func TestHandlePanelContentLoadedFresh(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentPath = "/my/file"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = true

	freshMsg := panelContentLoadedMsg{
		path:    "/my/file",
		mode:    panelModeDiff,
		section: changesSectionDrift,
		content: "+added line\n-removed line",
	}

	newM, cmd := m.handlePanelContentLoaded(freshMsg)
	if cmd != nil {
		t.Fatalf("expected nil cmd for fresh load without pending target, got %#v", cmd)
	}

	if newM.panel.loading {
		t.Error("expected loading=false after content loaded")
	}

	entry, ok := newM.panel.cacheGet("/my/file", panelModeDiff, changesSectionDrift)
	if !ok {
		t.Error("expected content to be cached")
	}
	if len(entry.lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(entry.lines))
	}
}

// TestPanelEndToEndUpdateViewCycle simulates the full Update→View cycle
// to verify content appears in the rendered output.
func TestPanelEndToEndUpdateViewCycle(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 140
	m.height = 40

	// Simulate status loaded with some drift files
	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/user/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		{Path: "/home/user/.vimrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()

	// Step 1: Verify panel is visible at this width
	if !m.panel.shouldShow(m.width) {
		t.Fatal("panel should be visible at width 140")
	}

	// Step 2: Simulate cursor on first drift file row (after Incoming header + Drift header)
	m.status.changesCursor = 2

	// Step 3: Call panelLoadForChanges to trigger async load
	m2, cmd := m.panelLoadForChanges()

	t.Logf("After panelLoadForChanges:")
	t.Logf("  currentPath=%q", m2.panel.currentPath)
	t.Logf("  loading=%v", m2.panel.loading)
	t.Logf("  contentMode=%v", m2.panel.contentMode)

	if m2.panel.currentPath == "" {
		t.Fatal("panelLoadForChanges did not set currentPath")
	}
	if !m2.panel.loading {
		t.Fatal("panelLoadForChanges did not set loading=true")
	}
	if cmd == nil {
		t.Fatal("panelLoadForChanges returned nil cmd")
	}

	// Step 4: Execute the async command to get the message
	msg := cmd()
	loaded, ok := msg.(panelContentLoadedMsg)
	if !ok {
		t.Fatalf("expected panelContentLoadedMsg, got %T", msg)
	}

	t.Logf("Async result: path=%q mode=%v content=%q err=%v",
		loaded.path, loaded.mode, loaded.content, loaded.err)

	// Step 5: Handle the loaded message
	m3, nextCmd := m2.handlePanelContentLoaded(loaded)
	if nextCmd != nil {
		t.Fatalf("did not expect follow-up cmd without pending target, got %#v", nextCmd)
	}

	t.Logf("After handlePanelContentLoaded:")
	t.Logf("  loading=%v", m3.panel.loading)
	t.Logf("  currentPath=%q", m3.panel.currentPath)

	if m3.panel.loading {
		t.Error("expected loading=false after content loaded")
	}

	// Step 6: Check cache
	entry, cached := m3.panel.cacheGet(m3.panel.currentPath, m3.panel.contentMode, m3.panel.currentSection)
	t.Logf("  cache hit=%v content=%q err=%v lines=%d",
		cached, entry.content, entry.err, len(entry.lines))

	if !cached {
		t.Fatal("expected cache to contain loaded content")
	}

	// Step 7: Render the panel and check output
	panelW := panelWidthFor(m3.width)
	output := m3.renderFilePanel(panelW)

	t.Logf("Panel output length: %d", len(output))
	if len(output) < 10 {
		t.Logf("Panel output:\n%s", output)
	}

	// Step 8: Render the full tab with panel
	fullOutput := m3.renderChangesTabWithPanel()
	t.Logf("Full tab output length: %d", len(fullOutput))

	// The output should contain something meaningful from the panel
	// (even if the diff is empty, we should see the title bar with filename)
	if !containsAny(output, ".bashrc") {
		t.Errorf("panel output does not contain filename .bashrc")
	}
}

func TestCacheKeepsSectionsIsolated(t *testing.T) {
	p := newFilePanel("")
	p.cachePut("/home/user/.bashrc", panelModeDiff, changesSectionUnstaged, panelCacheEntry{
		content: "unstaged-diff",
		lines:   []string{"unstaged-diff"},
	})
	p.cachePut("/home/user/.bashrc", panelModeDiff, changesSectionStaged, panelCacheEntry{
		content: "staged-diff",
		lines:   []string{"staged-diff"},
	})

	unstaged, ok := p.cacheGet("/home/user/.bashrc", panelModeDiff, changesSectionUnstaged)
	if !ok {
		t.Fatal("expected unstaged entry")
	}
	staged, ok := p.cacheGet("/home/user/.bashrc", panelModeDiff, changesSectionStaged)
	if !ok {
		t.Fatal("expected staged entry")
	}
	if unstaged.content == staged.content {
		t.Fatalf("expected section-aware cache isolation, got same content %q", unstaged.content)
	}
}

func TestHandlePanelContentLoadedRunsQueuedLatestLoad(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.loading = true
	m.panel.currentPath = "/current"
	m.panel.currentSection = changesSectionDrift
	m.panel.contentMode = panelModeDiff
	m.panel.pendingLoad = true
	m.panel.pendingPath = "/next"
	m.panel.pendingMode = panelModeContent
	m.panel.pendingSection = changesSectionUnstaged

	stale := panelContentLoadedMsg{
		path:    "/current",
		mode:    panelModeDiff,
		section: changesSectionDrift,
		content: "old",
	}

	nextModel, cmd := m.handlePanelContentLoaded(stale)
	if cmd == nil {
		t.Fatal("expected queued follow-up panel load cmd")
	}
	if !nextModel.panel.loading {
		t.Fatal("expected panel loading=true while queued target loads")
	}
	if nextModel.panel.currentPath != "/next" {
		t.Fatalf("expected currentPath=/next, got %q", nextModel.panel.currentPath)
	}
	if nextModel.panel.currentSection != changesSectionUnstaged {
		t.Fatalf("expected currentSection=%v, got %v", changesSectionUnstaged, nextModel.panel.currentSection)
	}
	if nextModel.panel.contentMode != panelModeContent {
		t.Fatalf("expected content mode queued as file mode, got %v", nextModel.panel.contentMode)
	}
}

// TestPanelRenderWithEmptyDiff verifies the panel shows meaningful output
// even when the manager Diff() returns empty string.
func TestPanelRenderWithEmptyDiff(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 140
	m.height = 40

	m.panel.currentPath = "/home/user/.bashrc"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = false

	// Cache empty diff (what the real manager might return for no changes)
	m.panel.cachePut("/home/user/.bashrc", panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: "",
		lines:   []string{""},
		err:     nil,
	})

	panelW := panelWidthFor(m.width)
	output := m.renderFilePanel(panelW)

	// Even with empty diff, title should show
	if !containsAny(output, ".bashrc") {
		t.Errorf("empty diff panel should still show filename in title")
	}

	t.Logf("Empty diff panel output length: %d chars", len(output))
	t.Logf("Output preview:\n%s", output[:min(len(output), 200)])
}

func TestPanelModeKeysFromListToggleView(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.width = 140
	m.height = 40
	m.activeTab = 0 // Status
	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/user/.bashrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 2 // first drift file row (after Incoming header + Drift header)
	m.panel.focusZone = panelFocusList
	m.panel.contentMode = panelModeDiff

	updatedModel, cmd := m.handleKeyMsg(tea.KeyPressMsg{Code: 'v', Text: "v"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if updated.panel.contentMode != panelModeContent {
		t.Fatalf("expected v from list focus to toggle into file mode")
	}
	if cmd == nil {
		t.Fatal("expected panel reload command after toggling panel mode")
	}
}

func TestPanelModeKeysFromListToggleBackToDiff(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.width = 140
	m.height = 40
	m.activeTab = 0 // Status
	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/user/.bashrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 2
	m.panel.focusZone = panelFocusList
	m.panel.contentMode = panelModeContent

	updatedModel, cmd := m.handleKeyMsg(tea.KeyPressMsg{Code: 'v', Text: "v"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if updated.panel.contentMode != panelModeDiff {
		t.Fatalf("expected v from content to toggle back to diff, got %d", updated.panel.contentMode)
	}
	if cmd == nil {
		t.Fatal("expected panel reload command after toggling panel mode")
	}
}

func TestPanelFocusedArrowScrollChangesViewportOffset(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.width = 140
	m.height = 40
	m.activeTab = 0 // Status
	m.panel.focusZone = panelFocusPanel
	m.panel.currentPath = "/home/user/.bashrc"
	m.panel.contentMode = panelModeContent
	m.panel.loading = false

	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "line"
	}
	m.panel.cachePut(m.panel.currentPath, panelModeContent, changesSectionDrift, panelCacheEntry{
		content: strings.Join(lines, "\n"),
		lines:   lines,
	})
	m = m.syncPanelViewportContent()

	if got := m.panel.viewport.YOffset(); got != 0 {
		t.Fatalf("expected initial offset 0, got %d", got)
	}

	updatedModel, _ := m.handleKeyMsg(tea.KeyPressMsg{Code: tea.KeyDown})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if got := updated.panel.viewport.YOffset(); got <= 0 {
		t.Fatalf("expected panel to scroll with down arrow, got offset=%d", got)
	}
}

func TestPanelFocusedRepeatArrowScrollAccelerates(t *testing.T) {
	buildModel := func() Model {
		m := NewModel(Options{Service: testService()})
		m.view = StatusScreen
		m.width = 140
		m.height = 40
		m.activeTab = 0 // Status
		m.panel.focusZone = panelFocusPanel
		m.panel.currentPath = "/home/user/.bashrc"
		m.panel.contentMode = panelModeContent
		m.panel.loading = false

		lines := make([]string, 200)
		for i := range lines {
			lines[i] = "line"
		}
		m.panel.cachePut(m.panel.currentPath, panelModeContent, changesSectionDrift, panelCacheEntry{
			content: strings.Join(lines, "\n"),
			lines:   lines,
		})
		return m.syncPanelViewportContent()
	}

	singleModel := buildModel()
	singleAny, _ := singleModel.handleKeyMsg(tea.KeyPressMsg{Code: tea.KeyDown})
	single, ok := singleAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	repeatModel := buildModel()
	repeatAny, _ := repeatModel.handleKeyMsg(tea.KeyPressMsg{Code: tea.KeyDown, IsRepeat: true})
	repeat, ok := repeatAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if repeat.panel.viewport.YOffset() <= single.panel.viewport.YOffset() {
		t.Fatalf(
			"expected repeat down in panel to scroll farther than single key press: repeat=%d single=%d",
			repeat.panel.viewport.YOffset(),
			single.panel.viewport.YOffset(),
		)
	}
}

func TestPanelFocusedMouseWheelScrollChangesViewportOffset(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.width = 140
	m.height = 40
	m.activeTab = 0 // Status
	m.panel.focusZone = panelFocusPanel
	m.panel.currentPath = "/home/user/.bashrc"
	m.panel.contentMode = panelModeContent
	m.panel.loading = false

	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "line"
	}
	m.panel.cachePut(m.panel.currentPath, panelModeContent, changesSectionDrift, panelCacheEntry{
		content: strings.Join(lines, "\n"),
		lines:   lines,
	})
	m = m.syncPanelViewportContent()

	if got := m.panel.viewport.YOffset(); got != 0 {
		t.Fatalf("expected initial offset 0, got %d", got)
	}

	panelX := m.width - panelWidthFor(m.width) + 1
	mouseMsg := tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		X:      panelX,
		Y:      10,
	}
	updatedModel, _ := m.handleMouseWheel(mouseMsg)
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if got := updated.panel.viewport.YOffset(); got <= 0 {
		t.Fatalf("expected panel to scroll with mouse wheel, got offset=%d", got)
	}
}

// NOTE: TestPanelContentModeUnstagedReadsFromSourceDir and related tests
// (WithoutHomePrefixLayout, WithHomeSuffixLayout, AbsoluteHomePathMapsToSource)
// were removed during the app-to-chezmoi flattening. They relied on a mock
// Backend to inject sourceDir/catTargetErr. The underlying panelReadSourceFile
// and resolvePanelSourcePath logic is still tested via the pure-function tests
// below (TestPanelRelativeHomePathFromAbsolute*).

func TestPanelRelativeHomePathFromAbsoluteAllowsDotDotPrefixPathSegments(t *testing.T) {
	targetPath := filepath.FromSlash("/home/test")
	absPath := filepath.FromSlash("/home/test/..config/nvim/init.lua")

	rel, ok := panelRelativeHomePathFromAbsolute(absPath, targetPath)
	if !ok {
		t.Fatal("expected path under target to map successfully")
	}
	want := filepath.FromSlash("..config/nvim/init.lua")
	if rel != want {
		t.Fatalf("expected rel=%q, got %q", want, rel)
	}
}

func TestPanelRelativeHomePathFromAbsoluteRejectsPathsOutsideTarget(t *testing.T) {
	targetPath := filepath.FromSlash("/home/test")
	absPath := filepath.FromSlash("/home/other/.config/nvim/init.lua")

	if rel, ok := panelRelativeHomePathFromAbsolute(absPath, targetPath); ok {
		t.Fatalf("expected outside path to be rejected, got rel=%q", rel)
	}
}

// NOTE: TestPanelContentModeUnstagedTemplatePathReadsFromSource,
// TestPanelContentModeUnstagedDirectoryShowsFriendlyMessage,
// TestPanelContentModeUnstagedBinaryShowsFriendlyMessage, and
// TestPanelContentModeFilesTabNotManagedFallsBackToLocalFile were removed
// during the app-to-chezmoi flattening. They relied on mock Backend fields
// (sourceDir, catTargetErr, catTargetCalls) that no longer exist.

// TestPanelLoadForChangesHeaderClearsViewport verifies that moving the cursor
// to a section header clears the panel viewport content (not stale diff).
func TestPanelLoadForChangesHeaderClearsViewport(t *testing.T) {
	m := newTestModel(
		WithSize(140, 40),
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		}),
		WithPanelVisible(),
	)

	// Pre-load panel with a cached diff so the viewport has content.
	m.panel.currentPath = "/home/test/.bashrc"
	m.panel.currentSection = changesSectionDrift
	m.panel.cachePut("/home/test/.bashrc", panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: "diff --git a/.bashrc",
		lines:   []string{"diff --git a/.bashrc"},
	})
	m = m.syncPanelViewportContent()

	// Move cursor to the first header row (Incoming header at index 0).
	m.status.changesCursor = 0

	// Call panelLoadForChanges which should clear the panel.
	m2, cmd := m.panelLoadForChanges()

	if m2.panel.currentPath != "" {
		t.Errorf("expected currentPath to be empty after header, got %q", m2.panel.currentPath)
	}
	if cmd != nil {
		t.Error("expected nil cmd for header row, got non-nil")
	}

	// Verify the viewport now shows the placeholder, not stale diff.
	output := m2.renderFilePanel(48)
	if !containsAny(output, "Select an item") {
		t.Errorf("expected placeholder text after moving to header, got:\n%s", output)
	}
	if containsAny(output, "bashrc", "diff") {
		t.Errorf("expected stale diff content to be cleared, got:\n%s", output)
	}
}

// TestPanelLoadForManagedDirectoryClearsViewport verifies that selecting a
// directory node in the Files tab tree view clears the panel viewport.
func TestPanelLoadForManagedDirectoryClearsViewport(t *testing.T) {
	m := newTestModel(
		WithTab(1), // Files tab
		WithSize(140, 40),
		WithManagedFiles([]string{
			"/home/test/.config/nvim/init.lua",
			"/home/test/.bashrc",
		}),
		WithPanelVisible(),
	)

	// Enable tree view and build tree.
	m.filesTab.treeView = true
	m.filesTab.views[managedViewManaged].tree = buildManagedTree(
		m.filesTab.views[managedViewManaged].filteredFiles, "/home/test",
	)
	m.filesTab.views[managedViewManaged].treeRows = flattenManagedTree(
		m.filesTab.views[managedViewManaged].tree,
	)

	// Pre-load panel with cached content.
	m.panel.currentPath = "/home/test/.bashrc"
	m.panel.cachePut("/home/test/.bashrc", panelModeContent, changesSectionDrift, panelCacheEntry{
		content: "# .bashrc",
		lines:   []string{"# .bashrc"},
	})
	m = m.syncPanelViewportContent()

	// Move cursor to a directory node (index 0 is typically .config/ dir).
	m.filesTab.cursor = 0

	m2, cmd := m.panelLoadForManaged()

	if m2.panel.currentPath != "" {
		t.Errorf("expected currentPath to be empty for directory, got %q", m2.panel.currentPath)
	}
	if cmd != nil {
		t.Error("expected nil cmd for directory node, got non-nil")
	}

	output := m2.renderFilePanel(48)
	if !containsAny(output, "Select an item") {
		t.Errorf("expected placeholder text after selecting directory, got:\n%s", output)
	}
}
