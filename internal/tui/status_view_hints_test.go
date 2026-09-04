package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestIncomingSectionActionHint(t *testing.T) {
	t.Run("shows fetch when incoming is empty", func(t *testing.T) {
		m := newTestModel()
		m.status.incomingCommits = nil

		if got := m.incomingSectionActionHint(); got != "[f fetch]" {
			t.Fatalf("expected [f fetch], got %q", got)
		}
		if got := m.incomingRowActionHint(); got != "f fetch" {
			t.Fatalf("expected f fetch, got %q", got)
		}
	})

	t.Run("shows pull when incoming exists in write mode", func(t *testing.T) {
		m := newTestModel()
		m.status.incomingCommits = []chezmoi.GitCommit{{Hash: "abc123", Message: "incoming commit"}}

		if got := m.incomingSectionActionHint(); got != "[p pull]" {
			t.Fatalf("expected [p pull], got %q", got)
		}
		if got := m.incomingRowActionHint(); got != "p pull" {
			t.Fatalf("expected p pull, got %q", got)
		}
	})

	t.Run("keeps fetch in read-only even when incoming exists", func(t *testing.T) {
		m := newTestModel(WithReadOnly())
		m.status.incomingCommits = []chezmoi.GitCommit{{Hash: "abc123", Message: "incoming commit"}}

		if got := m.incomingSectionActionHint(); got != "[f fetch]" {
			t.Fatalf("expected [f fetch], got %q", got)
		}
		if got := m.incomingRowActionHint(); got != "f fetch" {
			t.Fatalf("expected f fetch, got %q", got)
		}
	})
}

func TestDriftRowHelpHintShowsReAddOnlyWhenAvailable(t *testing.T) {
	t.Run("shows re-add for re-addable drift", func(t *testing.T) {
		m := newTestModel(
			WithDriftFiles([]chezmoi.FileStatus{
				{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			}),
		)
		m.view = StatusScreen
		m.activeTab = 0
		m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionDrift)

		rendered := ansi.Strip(m.renderChangesStatusBar())
		if !strings.Contains(rendered, "s re-add") {
			t.Fatalf("expected drift help hint to include %q, got %q", "s re-add", rendered)
		}
	})

	t.Run("hides re-add for non-readdable drift", func(t *testing.T) {
		m := newTestModel(
			WithDriftFiles([]chezmoi.FileStatus{
				{Path: "/home/test/.chezmoiscripts/run_once_install.sh", SourceStatus: 'R', DestStatus: ' '},
			}),
		)
		m.view = StatusScreen
		m.activeTab = 0
		m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionDrift)

		rendered := ansi.Strip(m.renderChangesStatusBar())
		if strings.Contains(rendered, "s re-add") {
			t.Fatalf("expected drift help hint to hide %q, got %q", "s re-add", rendered)
		}
		if !strings.Contains(rendered, "a actions") {
			t.Fatalf("expected drift help hint to keep actions hint, got %q", rendered)
		}
	})
}

func TestTabHintsSayEscBack(t *testing.T) {
	for tab := range 4 {
		m := newTestModel(WithView(StatusScreen), WithTab(tab), WithSize(120, 40))
		m.ui.loading = false
		m.status.loadingGit = false
		m.filesTab.views[managedViewManaged].loading = false
		out := stripForGolden(m.View().Content)
		if strings.Contains(out, "esc quit") {
			t.Fatalf("tab %d still says esc quit:\n%s", tab, out)
		}
		if !strings.Contains(out, "esc back") {
			t.Fatalf("tab %d does not say esc back:\n%s", tab, out)
		}
	}
}
