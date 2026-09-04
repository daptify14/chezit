package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestEnterTabFromLandingResetsPanelModeForFiles(t *testing.T) {
	m := NewModel(Options{
		Service:     testService(),
		EscBehavior: EscQuit,
	})
	if len(m.tabNames) < 2 {
		t.Fatalf("expected at least Status and Files tabs, got %v", m.tabNames)
	}
	if m.tabNames[1] != "Files" {
		t.Fatalf("expected Files tab at index 1, got %q", m.tabNames[1])
	}
	if m.panel.contentMode != panelModeDiff {
		t.Fatalf("expected initial panel mode diff on landing (Status default), got %v", m.panel.contentMode)
	}

	updatedModel, _ := m.enterTabFromLanding(1)
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}

	if updated.view != StatusScreen {
		t.Fatalf("expected status view after landing selection, got %v", updated.view)
	}
	if updated.activeTabName() != "Files" {
		t.Fatalf("expected active tab Files, got %q", updated.activeTabName())
	}
	if updated.panel.contentMode != panelModeContent {
		t.Fatalf("expected panel mode content for Files tab, got %v", updated.panel.contentMode)
	}
}

func TestLandingUpstreamRowStates(t *testing.T) {
	cases := []struct {
		name  string
		opts  []TestModelOption
		want  string
		green bool
	}{
		{"in flight", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true)}, "checking…", false},
		{"in flight after earlier success", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOK, "", true)}, "checking…", false},
		{"fetched", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOK, "", false)}, "fetched 2m ago", true},
		{"unreachable", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "unreachable", false)}, "unreachable", false},
		{"auth failed", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "auth failed", false)}, "auth failed", false},
		{"timed out", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "timed out", false)}, "timed out", false},
		{"generic failure", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "fetch failed", false)}, "fetch failed", false},
		{"auto fetch off", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOff, "", false)}, "not fetched · f", true},
		{"deferred not started", []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", false)}, "not fetched", false},
		{"no upstream ignores fetch failure", []TestModelOption{WithGitSync(chezmoi.GitSyncNoUpstream, 0, 0), WithFetchState(fetchFailed, "fetch failed", false)}, "none", true},
		{"comparison failed after fetch", []TestModelOption{WithGitSync(chezmoi.GitSyncUnknown, 0, 0), WithFetchState(fetchOK, "", false)}, "sync?", false},
		{"behind after fetch", []TestModelOption{WithGitSync(chezmoi.GitSyncBehind, 0, 2), WithFetchState(fetchOK, "", false)}, "fetched 2m ago", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(tc.opts...)
			m.landing.statsReady = true

			out := ansi.Strip(m.renderSummaryBox())
			if !strings.Contains(out, "upstream") || !strings.Contains(out, tc.want) {
				t.Fatalf("summary box missing upstream row %q:\n%s", tc.want, out)
			}
			if got := m.isAllInSync(); got != tc.green {
				t.Fatalf("isAllInSync() = %t, want %t", got, tc.green)
			}
			if got := strings.Contains(out, "all in sync"); got != tc.green {
				t.Fatalf("box shows 'all in sync' = %t, want %t:\n%s", got, tc.green, out)
			}
		})
	}
}

func TestLandingNotInSyncWithPendingWorkEvenWhenFetched(t *testing.T) {
	m := newTestModel(
		WithGitSync(chezmoi.GitSyncSynced, 0, 0),
		WithFetchState(fetchOK, "", false),
		WithDriftFiles([]chezmoi.FileStatus{{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '}}),
	)
	if m.isAllInSync() {
		t.Fatal("drift must keep the box out of the in-sync state")
	}
}

func TestSummaryBoxAlwaysFourRows(t *testing.T) {
	cases := []struct {
		name  string
		ready bool
		opts  []TestModelOption
	}{
		{"loading", false, nil},
		{"in sync", true, []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOK, "", false)}},
		{"changes", true, []TestModelOption{WithGitSync(chezmoi.GitSyncBehind, 0, 2), WithFetchState(fetchOK, "", false)}},
		{"not fetched", true, []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(tc.opts...)
			m.landing.statsReady = tc.ready

			lines := strings.Split(strings.TrimRight(ansi.Strip(m.renderSummaryBox()), "\n"), "\n")
			// 4 content rows + top and bottom border.
			if len(lines) != 6 {
				t.Fatalf("summary box has %d lines, want 6:\n%s", len(lines), strings.Join(lines, "\n"))
			}
		})
	}
}
