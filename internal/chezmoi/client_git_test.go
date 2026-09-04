package chezmoi

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func newGitClient(t *testing.T, overrides map[string]string, opts ...Option) *Client {
	t.Helper()
	binary := writeFakeChezmoiBinary(t, gitScript(overrides))
	return New(append([]Option{WithBinaryPath(binary)}, opts...)...)
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

// --- GitFetch ---

func fetchFailing(stderr string, code int) map[string]string {
	return map[string]string{"fetch": "printf '%s\\n' " + shellQuote(stderr) + " >&2; exit " + strconv.Itoa(code)}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func TestGitFetchSuccess(t *testing.T) {
	t.Parallel()

	if err := newGitClient(t, nil).GitFetch(); err != nil {
		t.Fatalf("GitFetch() error: %v", err)
	}
}

func TestGitFetchClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stderr string
		want   error
	}{
		{
			name:   "dns failure",
			stderr: "fatal: unable to access 'https://github.com/o/r/': Could not resolve host: github.com",
			want:   ErrFetchUnreachable,
		},
		{
			name:   "gitea connection refused",
			stderr: "fatal: unable to access 'https://git.example.lan/o/r/': Failed to connect to git.example.lan port 443: Connection refused",
			want:   ErrFetchUnreachable,
		},
		{
			name:   "ssh auth failure wins over unreachable",
			stderr: "git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.",
			want:   ErrFetchAuth,
		},
		{
			name:   "https prompt disabled",
			stderr: "fatal: could not read Username for 'https://github.com': terminal prompts disabled",
			want:   ErrFetchAuth,
		},
		{
			name:   "http 403",
			stderr: "fatal: unable to access 'https://github.com/o/r/': The requested URL returned error: 403",
			want:   ErrFetchAuth,
		},
		{
			name:   "host key verification",
			stderr: "Host key verification failed.\nfatal: Could not read from remote repository.",
			want:   ErrFetchAuth,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := newGitClient(t, fetchFailing(tc.stderr, 128)).GitFetch()
			if !errors.Is(err, tc.want) {
				t.Fatalf("GitFetch() = %v, want errors.Is %v", err, tc.want)
			}
			firstLine, _, _ := strings.Cut(tc.stderr, "\n")
			if !strings.Contains(err.Error(), firstLine) {
				t.Fatalf("error %q should contain git output %q", err, firstLine)
			}
		})
	}
}

func TestGitFetchUnknownFailureIsPlainError(t *testing.T) {
	t.Parallel()

	err := newGitClient(t, fetchFailing("error: something odd", 1)).GitFetch()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, sentinel := range []error{ErrFetchTimeout, ErrFetchAuth, ErrFetchUnreachable} {
		if errors.Is(err, sentinel) {
			t.Fatalf("unexpected classification %v for %v", sentinel, err)
		}
	}
	if !strings.Contains(err.Error(), "something odd") {
		t.Fatalf("error %q should contain git output", err)
	}
}

func TestGitFetchTimeout(t *testing.T) {
	t.Parallel()

	// sleep runs as a child of the fake binary's shell. Without a process
	// group kill it would keep the output pipe open for its full duration.
	client := newGitClient(t, map[string]string{"fetch": "sleep 2"}, WithFetchTimeout(50*time.Millisecond))

	start := time.Now()
	err := client.GitFetch()
	elapsed := time.Since(start)

	if !errors.Is(err, ErrFetchTimeout) {
		t.Fatalf("GitFetch() = %v, want ErrFetchTimeout", err)
	}
	if !strings.Contains(err.Error(), "50ms") {
		t.Fatalf("error %q should name the timeout", err)
	}
	if elapsed >= time.Second {
		t.Fatalf("GitFetch took %s, expected the process group kill to release the pipe promptly", elapsed)
	}
}

func TestGitFetchArgsAndEnv(t *testing.T) {
	// Not parallel: t.Setenv seeds inherited askpass helpers that the fetch
	// must override.
	t.Setenv("GIT_ASKPASS", "/usr/bin/yes")
	t.Setenv("SSH_ASKPASS_REQUIRE", "force")

	capture := filepath.Join(t.TempDir(), "capture")
	client := newGitClient(t, map[string]string{
		"fetch": `printf '%s\n' "$*" "${GIT_TERMINAL_PROMPT:-unset}" "${GIT_ASKPASS:-unset}" "${SSH_ASKPASS_REQUIRE:-unset}" > ` + shellQuote(capture),
	})
	if err := client.GitFetch(); err != nil {
		t.Fatalf("GitFetch() error: %v", err)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"fetch --quiet", "0", "/bin/false", "never"}
	if len(got) != len(want) {
		t.Fatalf("capture = %q, want %d lines", data, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewSeedsFetchTimeout(t *testing.T) {
	t.Parallel()

	if got := New().FetchTimeout; got != DefaultFetchTimeout {
		t.Fatalf("FetchTimeout = %s, want %s", got, DefaultFetchTimeout)
	}
	if got := New(WithFetchTimeout(0)).FetchTimeout; got != DefaultFetchTimeout {
		t.Fatalf("FetchTimeout after WithFetchTimeout(0) = %s, want default", got)
	}
	if got := New(WithFetchTimeout(3 * time.Second)).FetchTimeout; got != 3*time.Second {
		t.Fatalf("FetchTimeout = %s, want 3s", got)
	}
}

func TestRunTimeoutWrapsDeadlineExceeded(t *testing.T) {
	t.Parallel()

	binary := writeFakeChezmoiBinary(t, "exec sleep 2")
	client := New(WithBinaryPath(binary), WithTimeout(50*time.Millisecond))

	_, err := client.run("status")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("run() = %v, want errors.Is context.DeadlineExceeded", err)
	}
}
