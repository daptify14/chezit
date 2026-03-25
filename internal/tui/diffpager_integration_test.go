package tui

import (
	"strings"
	"testing"
)

// --- Full-screen diff state tests ---

func TestDiffLoaded_WithPager_StoresRawAndRenderedLines(t *testing.T) {
	m := newTestModel()
	m.diffPagerCmd = "delta" // not actually invoked in this test

	raw := "+added\n-removed\n context"
	rendered := "\x1b[32m+added\x1b[0m\n\x1b[31m-removed\x1b[0m\n context"

	msg := chezmoiDiffLoadedMsg{
		path:         "test.txt",
		diff:         raw,
		renderedDiff: rendered,
		pagerApplied: true,
	}

	m, _ = sendMsg(t, m, msg)

	if m.view != DiffScreen {
		t.Fatalf("view = %d, want DiffScreen", m.view)
	}
	if !m.diff.pagerApplied {
		t.Fatal("pagerApplied should be true")
	}
	// rawLines should be from raw diff
	wantRaw := strings.Split(raw, "\n")
	if len(m.diff.rawLines) != len(wantRaw) {
		t.Fatalf("rawLines len = %d, want %d", len(m.diff.rawLines), len(wantRaw))
	}
	// lines should be from rendered diff
	wantRendered := strings.Split(rendered, "\n")
	if len(m.diff.lines) != len(wantRendered) {
		t.Fatalf("lines len = %d, want %d", len(m.diff.lines), len(wantRendered))
	}
	if m.diff.lines[0] == m.diff.rawLines[0] {
		t.Fatal("lines[0] should differ from rawLines[0] when pager is applied")
	}
}

func TestDiffLoaded_WithoutPager_LinesEqualRawLines(t *testing.T) {
	m := newTestModel()

	raw := "+added\n-removed\n context"
	msg := chezmoiDiffLoadedMsg{
		path:         "test.txt",
		diff:         raw,
		pagerApplied: false,
	}

	m, _ = sendMsg(t, m, msg)

	if m.diff.pagerApplied {
		t.Fatal("pagerApplied should be false")
	}
	// lines and rawLines should be the same slice
	if len(m.diff.lines) != len(m.diff.rawLines) {
		t.Fatalf("lines len = %d, rawLines len = %d, should match", len(m.diff.lines), len(m.diff.rawLines))
	}
}

func TestDiffSummary_UsesRawLines_WhenPagerActive(t *testing.T) {
	m := newTestModel()
	m.diff.rawLines = []string{"+added", "-removed", " context"}
	m.diff.lines = []string{"\x1b[32madded\x1b[0m", "\x1b[31mremoved\x1b[0m", " context"}
	m.diff.pagerApplied = true

	summary := diffSummary(m.diffRawLines())

	if !strings.Contains(summary, "+1") || !strings.Contains(summary, "-1") {
		t.Fatalf("summary = %q, expected +1/-1", summary)
	}
}

func TestDiffSummary_UsesLines_WhenNoPager(t *testing.T) {
	m := newTestModel()
	m.diff.rawLines = nil
	m.diff.lines = []string{"+added", "-removed", " context"}
	m.diff.pagerApplied = false

	summary := diffSummary(m.diffRawLines())

	if !strings.Contains(summary, "+1") || !strings.Contains(summary, "-1") {
		t.Fatalf("summary = %q, expected +1/-1", summary)
	}
}

// --- Panel cache tests ---

func TestPanelCache_StoresRawAndRenderedLines(t *testing.T) {
	raw := "+added\n-removed"
	rendered := "\x1b[32m+added\x1b[0m\n\x1b[31m-removed\x1b[0m"

	msg := panelContentLoadedMsg{
		path:         "test.txt",
		mode:         panelModeDiff,
		section:      changesSectionDrift,
		content:      raw,
		rendered:     rendered,
		pagerApplied: true,
	}

	m := newTestModel()
	m.diffPagerCmd = "delta"
	m.panel.currentPath = "test.txt"
	m.panel.currentSection = changesSectionDrift
	m.panel.contentMode = panelModeDiff

	m, _ = sendMsg(t, m, msg)

	entry, ok := m.panel.cacheGet("test.txt", panelModeDiff, changesSectionDrift)
	if !ok {
		t.Fatal("expected cache entry")
	}
	if !entry.pagerApplied {
		t.Fatal("pagerApplied should be true in cache entry")
	}
	if entry.rawLines == nil {
		t.Fatal("rawLines should not be nil for diff mode")
	}

	// rawLines should be from raw content
	wantRaw := strings.Split(raw, "\n")
	if len(entry.rawLines) != len(wantRaw) {
		t.Fatalf("rawLines len = %d, want %d", len(entry.rawLines), len(wantRaw))
	}

	// lines should be from rendered content
	wantRendered := strings.Split(rendered, "\n")
	if len(entry.lines) != len(wantRendered) {
		t.Fatalf("lines len = %d, want %d", len(entry.lines), len(wantRendered))
	}
}

func TestPanelCache_NonDiff_NoRawLines(t *testing.T) {
	msg := panelContentLoadedMsg{
		path:    "test.txt",
		mode:    panelModeContent,
		section: changesSectionDrift,
		content: "file content here",
	}

	m := newTestModel()
	m.panel.currentPath = "test.txt"
	m.panel.currentSection = changesSectionDrift
	m.panel.contentMode = panelModeContent

	m, _ = sendMsg(t, m, msg)

	entry, ok := m.panel.cacheGet("test.txt", panelModeContent, changesSectionDrift)
	if !ok {
		t.Fatal("expected cache entry")
	}
	if entry.rawLines != nil {
		t.Fatal("rawLines should be nil for non-diff content")
	}
	if entry.pagerApplied {
		t.Fatal("pagerApplied should be false for non-diff content")
	}
}

// --- Reset/cleanup tests ---

func TestDiffClear_ResetsAllPagerFields(t *testing.T) {
	var d diffViewState
	d.content = "diff content"
	d.lines = []string{"+added"}
	d.rawLines = []string{"+added"}
	d.pagerApplied = true
	d.viewportReady = true

	d.clear()

	if d.content != "" {
		t.Fatal("content should be empty after clear")
	}
	if d.lines != nil {
		t.Fatal("lines should be nil after clear")
	}
	if d.rawLines != nil {
		t.Fatal("rawLines should be nil after clear")
	}
	if d.pagerApplied {
		t.Fatal("pagerApplied should be false after clear")
	}
	if d.viewportReady {
		t.Fatal("viewportReady should be false after clear")
	}
}

// --- Builtin fallback tests ---

func TestDiffLoaded_PagerFailed_FallsBackToBuiltin(t *testing.T) {
	m := newTestModel()
	m.diffPagerCmd = "delta" // configured but pager "failed"

	raw := "+added\n-removed"
	msg := chezmoiDiffLoadedMsg{
		path:         "test.txt",
		diff:         raw,
		pagerApplied: false, // pager invocation failed
	}

	m, _ = sendMsg(t, m, msg)

	if m.diff.pagerApplied {
		t.Fatal("pagerApplied should be false when pager fails")
	}
	// lines should equal rawLines (builtin fallback)
	if len(m.diff.lines) != len(m.diff.rawLines) {
		t.Fatal("lines and rawLines should match on fallback")
	}
}

// --- Source content (apply preview) tests ---

func TestSourceContent_WithPager_StoresPagerOutput(t *testing.T) {
	m := newTestModel()
	m.diffPagerCmd = "delta"
	m.diff.previewApply = true

	raw := "+added\n-removed"
	rendered := "\x1b[32m+added\x1b[0m\n\x1b[31m-removed\x1b[0m"

	msg := chezmoiSourceContentMsg{
		path:         "Preview: chezmoi apply",
		content:      raw,
		renderedDiff: rendered,
		pagerApplied: true,
	}

	m, _ = sendMsg(t, m, msg)

	if !m.diff.pagerApplied {
		t.Fatal("pagerApplied should be true")
	}
	if len(m.diff.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.diff.rawLines))
	}
}

func TestSourceContent_NonDiff_NoPager(t *testing.T) {
	m := newTestModel()
	m.diffPagerCmd = "delta"

	msg := chezmoiSourceContentMsg{
		path:    "chezmoi doctor",
		content: "doctor output",
	}

	m, _ = sendMsg(t, m, msg)

	if m.diff.pagerApplied {
		t.Fatal("pagerApplied should be false for non-diff content")
	}
}
