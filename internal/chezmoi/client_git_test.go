package chezmoi

import (
	"maps"
	"testing"
)

// gitScript builds a fake chezmoi body dispatching on the git subcommand
// ($1=git $2=-- $3=<subcommand> once the preamble strips --flags). Keys: head,
// upstream, remote, rev-list, fetch; missing keys default to a synced main.
func gitScript(overrides map[string]string) string {
	snippets := map[string]string{
		"head":     "echo main",
		"upstream": "echo origin/main",
		"remote":   "echo origin",
		"rev-list": "printf '0\\t0\\n'",
		"fetch":    "exit 0",
	}
	maps.Copy(snippets, overrides)
	return `
case "$1" in
  git) shift; [ "$1" = "--" ] && shift ;;
  *) exit 0 ;;
esac
case "$1" in
  rev-parse)
    case "$*" in
      *upstream*) ` + snippets["upstream"] + ` ;;
      *) ` + snippets["head"] + ` ;;
    esac ;;
  remote) ` + snippets["remote"] + ` ;;
  rev-list) ` + snippets["rev-list"] + ` ;;
  fetch) ` + snippets["fetch"] + ` ;;
esac
`
}

func newGitClient(t *testing.T, overrides map[string]string) *Client {
	t.Helper()
	return New(WithBinaryPath(writeFakeChezmoiBinary(t, gitScript(overrides))))
}

func TestGitBranchInfoSynced(t *testing.T) {
	t.Parallel()

	info, err := newGitClient(t, nil).GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	want := GitInfo{Branch: "main", Upstream: "origin/main", Remote: "origin", Sync: GitSyncSynced}
	if info != want {
		t.Fatalf("GitBranchInfo() = %+v, want %+v", info, want)
	}
}

func TestGitBranchInfoNoUpstream(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{
		"upstream": `echo "fatal: no upstream configured for branch 'main'" >&2; exit 128`,
	})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Sync != GitSyncNoUpstream {
		t.Fatalf("Sync = %s, want no-upstream", info.Sync)
	}
	if info.Upstream != "" || info.Ahead != 0 || info.Behind != 0 {
		t.Fatalf("expected empty upstream and zero counts, got %+v", info)
	}
}

func TestGitBranchInfoDetachedHead(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{
		"head":     "echo HEAD",
		"upstream": `echo "fatal: HEAD does not point to a branch" >&2; exit 128`,
	})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Branch != "HEAD" || info.Sync != GitSyncNoUpstream {
		t.Fatalf("expected detached HEAD with no upstream, got %+v", info)
	}
}

func TestGitBranchInfoUnrecognizedUpstreamFailureIsUnknown(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{
		"upstream": `echo "fatal: something else" >&2; exit 1`,
	})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Sync != GitSyncUnknown {
		t.Fatalf("Sync = %s, want unknown", info.Sync)
	}
}

func TestGitBranchInfoSyncStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		revList       string
		want          GitSyncState
		ahead, behind int
	}{
		{"0\\t0", GitSyncSynced, 0, 0},
		{"2\\t0", GitSyncBehind, 0, 2},
		{"0\\t3", GitSyncAhead, 3, 0},
		{"1\\t1", GitSyncDiverged, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.want.String(), func(t *testing.T) {
			t.Parallel()

			client := newGitClient(t, map[string]string{
				"rev-list": "printf '" + tc.revList + "\\n'",
			})
			info, err := client.GitBranchInfo()
			if err != nil {
				t.Fatalf("GitBranchInfo() error: %v", err)
			}
			if info.Sync != tc.want || info.Ahead != tc.ahead || info.Behind != tc.behind {
				t.Fatalf("got sync=%s ahead=%d behind=%d, want sync=%s ahead=%d behind=%d",
					info.Sync, info.Ahead, info.Behind, tc.want, tc.ahead, tc.behind)
			}
		})
	}
}

func TestGitBranchInfoRevListFailureIsUnknown(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{"rev-list": "exit 1"})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Sync != GitSyncUnknown || info.Ahead != 0 || info.Behind != 0 {
		t.Fatalf("expected unknown sync with zero counts, got %+v", info)
	}
	if info.Upstream != "origin/main" {
		t.Fatalf("Upstream = %q, want origin/main (upstream lookup succeeded)", info.Upstream)
	}
}

func TestGitBranchInfoGarbageCountIsUnknown(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{"rev-list": "echo x y"})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Sync != GitSyncUnknown {
		t.Fatalf("Sync = %s, want unknown", info.Sync)
	}
}

func TestGitBranchInfoRemoteTakesFirstLine(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{"remote": "printf 'origin\\nupstream\\n'"})
	info, err := client.GitBranchInfo()
	if err != nil {
		t.Fatalf("GitBranchInfo() error: %v", err)
	}
	if info.Remote != "origin" {
		t.Fatalf("Remote = %q, want origin", info.Remote)
	}
}

func TestGitBranchInfoBranchFailureIsError(t *testing.T) {
	t.Parallel()

	client := newGitClient(t, map[string]string{"head": `echo "fatal: not a git repository" >&2; exit 128`})
	if _, err := client.GitBranchInfo(); err == nil {
		t.Fatal("expected error when branch lookup fails")
	}
}

func TestClassifySync(t *testing.T) {
	t.Parallel()

	cases := []struct {
		ahead, behind int
		want          GitSyncState
	}{
		{0, 0, GitSyncSynced},
		{1, 0, GitSyncAhead},
		{0, 1, GitSyncBehind},
		{2, 3, GitSyncDiverged},
	}
	for _, tc := range cases {
		if got := classifySync(tc.ahead, tc.behind); got != tc.want {
			t.Errorf("classifySync(%d, %d) = %s, want %s", tc.ahead, tc.behind, got, tc.want)
		}
	}
}
