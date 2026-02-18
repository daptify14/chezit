package tui

// WARNING: github.com/charmbracelet/x/exp/golden is a pre-v1 experimental
// package. Its API may change in future releases. Pin the module version in
// go.mod and review changelogs before upgrading.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// ── Help Overlay ────────────────────────────────────────────────────

func TestGoldenHelpOverlay(t *testing.T) {
	tabs := []struct {
		name string
		tab  int
	}{
		{"status", 0},
		{"files", 1},
		{"info", 2},
		{"commands", 3},
	}

	for _, tc := range tabs {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(WithTab(tc.tab))
			m.overlays.showHelp = true
			output := stripForGolden(m.renderHelp())
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// ── Status Tab ──────────────────────────────────────────────────────

func TestGoldenStatusTab(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m := newTestModel()
		m.landing.statsReady = true
		m.status.loadingGit = false
		output := stripForGolden(m.renderChangesTabContent())
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("two_drift_files", func(t *testing.T) {
		m := newTestModel(
			WithDriftFiles([]chezmoi.FileStatus{
				{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: 'M'},
				{Path: "/home/test/.config/nvim/init.lua", SourceStatus: 'A', DestStatus: ' '},
			}),
		)
		m.status.loadingGit = false
		output := stripForGolden(m.renderChangesTabContent())
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("drift_with_template", func(t *testing.T) {
		m := newTestModel(
			WithDriftFiles([]chezmoi.FileStatus{
				{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
				{Path: "/home/test/.config/starship.toml", SourceStatus: 'M', DestStatus: ' ', IsTemplate: true},
			}),
		)
		m.status.loadingGit = false
		output := stripForGolden(m.renderChangesTabContent())
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Files Tab ───────────────────────────────────────────────────────

func TestGoldenFilesTab(t *testing.T) {
	managedFiles := []string{
		"/home/test/.bashrc",
		"/home/test/.config/nvim/init.lua",
		"/home/test/.config/nvim/lua/plugins.lua",
	}

	t.Run("tree_view", func(t *testing.T) {
		m := newTestModel(
			WithTab(1),
			WithManagedFiles(managedFiles),
		)
		m.filesTab.treeView = true
		m.rebuildFileViewTree(managedViewManaged)
		output := stripForGolden(m.renderManagedTabContentWidth(90))
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("flat_view", func(t *testing.T) {
		m := newTestModel(
			WithTab(1),
			WithManagedFiles(managedFiles),
		)
		m.filesTab.treeView = false
		output := stripForGolden(m.renderManagedTabContentWidth(90))
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Info Tab ────────────────────────────────────────────────────────

func TestGoldenInfoTab(t *testing.T) {
	t.Run("config_loaded", func(t *testing.T) {
		content := "[core]\n  editor = vim\n[data]\n  name = test\n"
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
		m := newTestModel(
			WithInfoContent(infoViewConfig, content, len(lines)),
		)
		// Overwrite the auto-generated lines with the actual content lines
		// so that the render uses real data rather than placeholder text.
		m.info.views[infoViewConfig].lines = lines
		output := stripForGolden(m.renderInfoTabContent())
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Diff View ───────────────────────────────────────────────────────

func TestGoldenDiffView(t *testing.T) {
	t.Run("short_diff", func(t *testing.T) {
		diffContent := "--- a/.bashrc\n+++ b/.bashrc\n@@ -1,3 +1,4 @@\n export EDITOR=vim\n+export PAGER=less\n alias ll='ls -la'\n"
		diffLines := strings.Split(strings.TrimRight(diffContent, "\n"), "\n")
		m := newTestModel(
			WithDiffContent("/home/test/.bashrc", diffContent, len(diffLines)),
		)
		// Replace auto-generated lines with actual diff lines.
		m.diff.lines = diffLines
		output := stripForGolden(m.renderDiffView())
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Landing View ────────────────────────────────────────────────────

func TestGoldenLandingScreen(t *testing.T) {
	t.Run("stats_ready", func(t *testing.T) {
		m := newTestModel(WithView(LandingScreen))
		m.view = LandingScreen
		m.landing.statsReady = true
		output := stripForGolden(m.renderLandingScreen())
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Panel ───────────────────────────────────────────────────────────

func TestGoldenPanel(t *testing.T) {
	panelWidth := panelWidthFor(140)

	t.Run("diff_mode", func(t *testing.T) {
		m := newTestModel(
			WithPanelVisible(),
			WithSize(140, 40),
		)
		m.panel.currentPath = "/home/test/.bashrc"
		m.panel.contentMode = panelModeDiff
		m.panel.cachePut("/home/test/.bashrc", panelModeDiff, changesSectionDrift, panelCacheEntry{
			content: "+added line\n-removed line",
			lines:   []string{"+added line", "-removed line"},
		})
		output := stripForGolden(m.renderFilePanel(panelWidth))
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("content_mode", func(t *testing.T) {
		m := newTestModel(
			WithPanelVisible(),
			WithSize(140, 40),
		)
		m.panel.currentPath = "/home/test/.bashrc"
		m.panel.contentMode = panelModeContent
		m.panel.cachePut("/home/test/.bashrc", panelModeContent, changesSectionDrift, panelCacheEntry{
			content: "export EDITOR=vim\nalias ll='ls -la'",
			lines:   []string{"export EDITOR=vim", "alias ll='ls -la'"},
		})
		output := stripForGolden(m.renderFilePanel(panelWidth))
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("loading", func(t *testing.T) {
		m := newTestModel(
			WithPanelVisible(),
			WithSize(140, 40),
		)
		m.panel.currentPath = "/home/test/.bashrc"
		m.panel.contentMode = panelModeDiff
		// No cache entry -- triggers loading state.
		output := stripForGolden(m.renderFilePanel(panelWidth))
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("empty_path", func(t *testing.T) {
		m := newTestModel(
			WithPanelVisible(),
			WithSize(140, 40),
		)
		m.panel.currentPath = ""
		output := stripForGolden(m.renderFilePanel(panelWidth))
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Actions Menu ────────────────────────────────────────────────────

func TestGoldenActionsMenu(t *testing.T) {
	t.Run("status_tab", func(t *testing.T) {
		m := newTestModel(
			WithDriftFiles([]chezmoi.FileStatus{
				{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: 'M'},
				{Path: "/home/test/.zshrc", SourceStatus: 'A', DestStatus: ' '},
			}),
		)
		m.ui.loading = false
		// Position cursor on the first file row (skip headers).
		m.status.changesCursor = findFirstFileRow(t, m)
		m.openStatusActionsMenu()
		output := stripForGolden(m.View().Content)
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Filter Overlay ──────────────────────────────────────────────────

func TestGoldenFilterOverlay(t *testing.T) {
	t.Run("active_with_query", func(t *testing.T) {
		m := newTestModel(
			WithTab(1),
			WithManagedFiles([]string{
				"/home/test/.bashrc",
				"/home/test/.config/nvim/init.lua",
				"/home/test/.config/nvim/lua/plugins.lua",
			}),
		)
		m.ui.loading = false
		m.filterInput.SetValue("nvim")
		m.filterInput.Focus()
		// Use full View() to capture the search box with active query.
		output := stripForGolden(m.View().Content)
		golden.RequireEqual(t, []byte(output))
	})
}

// ── Error States ────────────────────────────────────────────────────

func TestGoldenErrorStates(t *testing.T) {
	t.Run("status_load_error", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = false
		m.ui.message = "chezmoi status: exit status 1"
		output := stripForGolden(m.View().Content)
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("diff_load_error", func(t *testing.T) {
		m := newTestModel(WithView(DiffScreen))
		m.ui.loading = false
		m.diff.path = "/home/test/.bashrc"
		m.ui.message = "chezmoi diff: file not found"
		output := stripForGolden(m.View().Content)
		golden.RequireEqual(t, []byte(output))
	})
}
