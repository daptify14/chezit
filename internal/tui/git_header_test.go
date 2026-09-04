package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestGitInfoHeaderTokens(t *testing.T) {
	cases := []struct {
		name   string
		opts   []TestModelOption
		want   []string
		reject []string
	}{
		{
			name:   "no upstream suppresses arrows and freshness",
			opts:   []TestModelOption{WithGitSync(chezmoi.GitSyncNoUpstream, 0, 0), WithFetchState(fetchFailed, "fetch failed", false)},
			want:   []string{"main", "no upstream"},
			reject: []string{"↑", "↓", "fetch failed", "not fetched"},
		},
		{
			name: "fetch in flight",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true)},
			want: []string{"main", "fetching…"},
		},
		{
			name: "fetched",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOK, "", false)},
			want: []string{"main · fetched 2m ago"},
		},
		{
			name: "unreachable",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "unreachable", false)},
			want: []string{"upstream unreachable"},
		},
		{
			name: "auth failed",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "auth failed", false)},
			want: []string{"upstream auth failed"},
		},
		{
			name: "timed out",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "timed out", false)},
			want: []string{"upstream timed out"},
		},
		{
			name:   "generic failure has no upstream prefix",
			opts:   []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "fetch failed", false)},
			want:   []string{"· fetch failed"},
			reject: []string{"upstream fetch failed"},
		},
		{
			name: "comparison failed after fetch",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncUnknown, 0, 0), WithFetchState(fetchOK, "", false)},
			want: []string{"sync?"},
		},
		{
			name:   "auto fetch off",
			opts:   []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchOff, "", false)},
			want:   []string{"not fetched"},
			reject: []string{"· f"},
		},
		{
			name: "deferred not started",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncSynced, 0, 0)},
			want: []string{"not fetched"},
		},
		{
			name:   "behind shows only the down arrow",
			opts:   []TestModelOption{WithGitSync(chezmoi.GitSyncBehind, 0, 2), WithFetchState(fetchOK, "", false)},
			want:   []string{"↓2", "fetched 2m ago"},
			reject: []string{"↑"},
		},
		{
			name: "diverged shows both arrows",
			opts: []TestModelOption{WithGitSync(chezmoi.GitSyncDiverged, 1, 2), WithFetchState(fetchOK, "", false)},
			want: []string{"↑1", "↓2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(tc.opts...)
			out := ansi.Strip(m.renderGitInfoHeader())
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Fatalf("header %q missing %q", out, w)
				}
			}
			for _, r := range tc.reject {
				if strings.Contains(out, r) {
					t.Fatalf("header %q must not contain %q", out, r)
				}
			}
		})
	}
}

func TestGitInfoHeaderEmptyWithoutBranch(t *testing.T) {
	m := newTestModel(WithFetchState(fetchFailed, "unreachable", false))
	if got := ansi.Strip(m.renderGitInfoHeader()); strings.TrimSpace(got) != "" {
		t.Fatalf("header without branch should be blank, got %q", got)
	}
}
