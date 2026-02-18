package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// TestRenderFilePanelWithCachedContent tests the full rendering path
// from cached content through viewport to final output.
func TestRenderFilePanelWithCachedContent(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40

	// Simulate: panel is visible, content is cached for a file
	m.panel.currentPath = "/my/file.txt"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = false

	diffContent := "+added line\n-removed line\n context line"
	lines := []string{"+added line", "-removed line", " context line"}
	m.panel.cachePut("/my/file.txt", panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: diffContent,
		lines:   lines,
	})

	// Render the panel
	panelW := panelWidthFor(m.width) // 48
	output := m.renderFilePanel(panelW)

	if output == "" {
		t.Fatal("renderFilePanel returned empty string")
	}

	// Check that the output contains some of the diff content
	if !containsAny(output, "added line", "removed line", "context line") {
		t.Errorf("renderFilePanel output does not contain expected diff content.\nOutput length: %d\nOutput:\n%s", len(output), output)
	}

	// Check that title bar contains the filename
	if !containsAny(output, "file.txt") {
		t.Errorf("renderFilePanel output does not contain filename.\nOutput:\n%s", output)
	}
}

// TestRenderFilePanelContentMode tests rendering in file content mode.
func TestRenderFilePanelContentMode(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40

	m.panel.currentPath = "/my/config.yaml"
	m.panel.contentMode = panelModeContent
	m.panel.loading = false

	fileContent := "key: value\nother: data\nthird: line"
	lines := []string{"key: value", "other: data", "third: line"}
	m.panel.cachePut("/my/config.yaml", panelModeContent, changesSectionDrift, panelCacheEntry{
		content: fileContent,
		lines:   lines,
	})

	panelW := panelWidthFor(m.width)
	output := m.renderFilePanel(panelW)

	if output == "" {
		t.Fatal("renderFilePanel returned empty string")
	}

	stripped := ansi.Strip(output)
	if !containsAny(stripped, "key: value", "other: data") {
		t.Errorf("renderFilePanel output does not contain file content.\nOutput length: %d\nOutput:\n%s", len(output), output)
	}
}

func TestRenderPanelFileContentUsesStableLineNumberColumn(t *testing.T) {
	short := renderPanelFileContent([]string{"alpha"}, 80, "test.txt")
	longLines := make([]string, 120)
	for i := range longLines {
		longLines[i] = "alpha"
	}
	long := renderPanelFileContent(longLines, 80, "test.txt")

	shortLine := strings.Split(short, "\n")[0]
	longLine := strings.Split(long, "\n")[0]

	shortIdx := strings.Index(shortLine, "alpha")
	longIdx := strings.Index(longLine, "alpha")
	if shortIdx < 0 || longIdx < 0 {
		t.Fatalf("failed to locate content column: shortIdx=%d longIdx=%d", shortIdx, longIdx)
	}
	if shortIdx != longIdx {
		t.Fatalf("line-number column should stay aligned across files: short=%d long=%d", shortIdx, longIdx)
	}
}

// TestRenderFilePanelLoadingState tests the loading indicator.
func TestRenderFilePanelLoadingState(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40

	m.panel.currentPath = "/some/file"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = true

	output := m.renderFilePanel(48)

	if !containsAny(output, "Loading") {
		t.Errorf("expected Loading indicator when panel.loading=true.\nOutput:\n%s", output)
	}
}

// TestRenderFilePanelEmptyPath tests the placeholder state.
func TestRenderFilePanelEmptyPath(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40

	m.panel.currentPath = ""
	m.panel.loading = false

	output := m.renderFilePanel(48)

	if !containsAny(output, "Select an item") {
		t.Errorf("expected placeholder text when currentPath is empty.\nOutput:\n%s", output)
	}
}

// TestRenderFilePanelNoCacheShowsLoading tests the fallback loading state
// when currentPath is set but cache has no entry.
func TestRenderFilePanelNoCacheShowsLoading(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40

	m.panel.currentPath = "/uncached/file"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = false // not loading, but not cached either

	output := m.renderFilePanel(48)

	if !containsAny(output, "Loading") {
		t.Errorf("expected Loading indicator when cache miss.\nOutput:\n%s", output)
	}
}

func TestRenderChangesTabContentWidthRespectsMaxWidth(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 140
	m.height = 40

	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/user/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		{Path: "/home/user/.vimrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 1

	const listWidth = 80
	output := m.renderChangesTabContentWidth(listWidth)
	assertRenderedLinesFitWidth(t, output, listWidth)
}

func TestRenderManagedTabContentWidthRespectsMaxWidth(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 140
	m.height = 40

	m.filesTab.views[managedViewManaged].files = []string{
		"/home/user/.bashrc",
		"/home/user/.config/nvim/init.lua",
	}
	m.filesTab.views[managedViewManaged].filteredFiles = m.filesTab.views[managedViewManaged].files
	m.filesTab.treeView = false
	m.filesTab.cursor = 0

	const listWidth = 80
	output := m.renderManagedTabContentWidth(listWidth)
	assertRenderedLinesFitWidth(t, output, listWidth)
}

func TestRenderChangesTabWithPanelFitsTerminalWidth(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 140
	m.height = 40

	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/user/.bashrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 1

	m.panel.currentPath = "/home/user/.bashrc"
	m.panel.contentMode = panelModeDiff
	m.panel.loading = false
	m.panel.cachePut("/home/user/.bashrc", panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: "",
		lines:   []string{""},
		err:     nil,
	})

	if !m.panel.shouldShow(m.width) {
		t.Fatalf("expected panel visible at width %d", m.width)
	}

	output := m.renderChangesTabWithPanel()
	assertRenderedLinesFitWidth(t, output, m.width)
	if !containsAny(output, "[diff]", "No changes (file matches source state)") {
		t.Fatalf("expected panel markers in output, got:\n%s", output)
	}
}

func TestRenderPanelTitleBarUsesDetectedFileTypeBadge(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentPath = "/tmp/config.yaml"
	m.panel.contentMode = panelModeContent

	got := ansi.Strip(m.renderPanelTitleBar(80))
	if !containsAny(got, "[yaml file]") {
		t.Fatalf("expected yaml file badge, got: %q", got)
	}
}

func TestRenderPanelTitleBarFallsBackToGenericFileBadge(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentPath = "/tmp/unknown.chezitunknown"
	m.panel.contentMode = panelModeContent

	got := ansi.Strip(m.renderPanelTitleBar(80))
	if !containsAny(got, "[file]") {
		t.Fatalf("expected generic file badge, got: %q", got)
	}
}

func TestRenderPanelDiffEmptyUntrackedShowsRelevantMessage(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentSection = changesSectionUnstaged
	m.panel.currentPath = "home/private_dot_config/tmux/tmux.conf"
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "home/private_dot_config/tmux/tmux.conf", StatusCode: "U"},
	}

	got := m.renderPanelDiff([]string{""}, 80)
	if !containsAny(got, "Untracked file") {
		t.Fatalf("expected untracked-specific empty-diff message, got:\n%s", got)
	}
}

func TestRenderFilePanelFriendlyErrorOmitsErrorPrefix(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.height = 40
	m.panel.currentPath = "/tmp/example"
	m.panel.contentMode = panelModeContent
	m.panel.loading = false
	m.panel.cachePut("/tmp/example", panelModeContent, changesSectionDrift, panelCacheEntry{
		err: newPanelPreviewError("Directory selected; preview skipped"),
	})

	out := m.renderFilePanel(panelWidthFor(m.width))
	if strings.Contains(out, "Error:") {
		t.Fatalf("friendly preview message should not contain Error: prefix:\n%s", out)
	}
	if !containsAny(out, "Directory selected; preview skipped") {
		t.Fatalf("expected friendly preview message, got:\n%s", out)
	}
}

func TestPanelErrorTextWrappedUserMessageOmitsPrefix(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", newPanelPreviewError("Directory selected; preview skipped"))

	got := panelErrorText(err)
	if got != "Directory selected; preview skipped" {
		t.Fatalf("expected user-facing message from wrapped error, got %q", got)
	}
}

func TestPanelErrorTextRegularErrorIncludesPrefix(t *testing.T) {
	err := errors.New("boom")

	got := panelErrorText(err)
	if got != "Error: boom" {
		t.Fatalf("expected default prefixed message, got %q", got)
	}
}

func TestRenderPanelTitleBarIncludesDriftSubtypeLabel(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.panel.currentPath = "/home/test/.bashrc"
	m.panel.currentSection = changesSectionDrift
	m.panel.contentMode = panelModeDiff
	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
	}
	m.panel.cachePut("/home/test/.bashrc", panelModeDiff, changesSectionDrift, panelCacheEntry{
		content: "--- a/.bashrc\n+++ b/.bashrc\n+alias gs='git status'",
		lines:   []string{"--- a/.bashrc", "+++ b/.bashrc", "+alias gs='git status'"},
	})

	got := ansi.Strip(m.renderPanelTitleBar(100))
	if !containsAny(got, "pending apply") {
		t.Fatalf("expected panel title details to include drift subtype, got: %q", got)
	}
}

func TestRenderChezmoiDiffStatusIncludesDriftSubtypeLabel(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.diff.sourceSection = changesSectionDrift
	m.diff.path = "/home/test/.bashrc"
	m.diff.lines = []string{"+alias gs='git status'"}
	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: 'M'},
	}

	got := ansi.Strip(m.renderChezmoiDiffStatus())
	if !containsAny(got, "diverged") {
		t.Fatalf("expected diff status line to include drift subtype, got: %q", got)
	}
}
