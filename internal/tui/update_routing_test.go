package tui

import (
	"context"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestBackgroundColorMsgRoutingInCommitScreen(t *testing.T) {
	prevTheme := activeTheme
	t.Cleanup(func() { activeTheme = prevTheme })

	activeTheme = ThemeDark()

	m := newTestModel()
	_ = m.openCommitScreen()
	if m.view != CommitScreen {
		t.Fatalf("expected commit screen, got %v", m.view)
	}

	updated, cmd := sendMsg(t, m, tea.BackgroundColorMsg{
		Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	})
	if cmd != nil {
		t.Fatal("expected nil cmd for background color message")
	}
	if updated.view != CommitScreen {
		t.Fatalf("expected to remain in commit screen, got %v", updated.view)
	}

	wantTheme := ThemeLight()
	if got := activeTheme.ChromaStyleName; got != wantTheme.ChromaStyleName {
		t.Fatalf("expected active theme %q, got %q", wantTheme.ChromaStyleName, got)
	}

	promptFg := updated.filterInput.Styles().Focused.Prompt.GetForeground()
	wantPromptFg := wantTheme.PrimaryFg.GetForeground()
	if !testColorsEqual(promptFg, wantPromptFg) {
		t.Fatalf("expected filter prompt to restyle with light primary color")
	}
}

func testColorsEqual(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// --- 1. TestStatusLoadedMsgRouting ---

func TestStatusLoadedMsgRouting(t *testing.T) {
	testFiles := []chezmoi.FileStatus{
		{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		{Path: "/home/test/.vimrc", SourceStatus: 'A', DestStatus: ' '},
	}

	t.Run("fresh gen stores files and clears loading", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		gen := m.gen

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.ui.loading {
			t.Fatal("expected ui.loading to be false after fresh gen status load")
		}
		if len(updated.status.files) != len(testFiles) {
			t.Fatalf("expected %d status files, got %d", len(testFiles), len(updated.status.files))
		}
		if len(updated.status.filteredFiles) != len(testFiles) {
			t.Fatalf("expected %d filtered files, got %d", len(testFiles), len(updated.status.filteredFiles))
		}
		for i, f := range updated.status.files {
			if f.Path != testFiles[i].Path {
				t.Fatalf("file[%d] path = %q, want %q", i, f.Path, testFiles[i].Path)
			}
		}
	})

	t.Run("stale gen discards message", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		staleGen := m.gen + 1

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: staleGen}
		updated, _ := sendMsg(t, m, msg)

		if !updated.ui.loading {
			t.Fatal("expected ui.loading to remain true for stale gen")
		}
		if len(updated.status.files) != 0 {
			t.Fatalf("expected no status files for stale gen, got %d", len(updated.status.files))
		}
	})

	t.Run("error sets message and clears loading", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		gen := m.gen

		msg := chezmoiStatusLoadedMsg{err: errors.New("test error"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.ui.loading {
			t.Fatal("expected ui.loading to be false after error")
		}
		if !strings.HasPrefix(updated.ui.message, "Error:") {
			t.Fatalf("expected ui.message to start with 'Error:', got %q", updated.ui.message)
		}
		if len(updated.status.files) != 0 {
			t.Fatalf("expected no status files on error, got %d", len(updated.status.files))
		}
	})

	t.Run("panel load triggered on Status tab with wide terminal", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		m.activeTab = 0 // Status
		m.width = 120
		m.panel.manualOverride = true
		m.panel.visible = true
		// Pre-mark landing stats ready so the allLandingStatsLoaded path
		// does not fire the debounce cmd instead of the panel load cmd.
		m.landing.statsReady = true
		gen := m.gen

		// Pre-populate some state so that after buildChangesRows there is
		// a file row at a known cursor position for panelLoadForChanges.
		// The handler sets filteredFiles = testFiles and calls buildChangesRows,
		// which creates header rows and file rows. We need changesCursor to
		// point at a drift file row (index 2: [0]=incoming header, [1]=drift header, [2]=first drift file).
		m.status.changesCursor = 2

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: gen}
		_, cmd := sendMsg(t, m, msg)

		if cmd == nil {
			t.Fatal("expected non-nil cmd for panel load on Status tab with wide terminal")
		}
	})

	t.Run("no panel load on Files tab", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		m.activeTab = 1 // Files
		m.width = 120
		m.panel.manualOverride = true
		m.panel.visible = true
		m.landing.statsReady = true
		gen := m.gen

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: gen}
		_, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd when activeTab is Files, panel should not trigger for wrong tab")
		}
	})

	t.Run("builds changesRows after storing files", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		m.landing.statsReady = true
		gen := m.gen

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if len(updated.status.changesRows) == 0 {
			t.Fatal("expected changesRows to be populated after status load")
		}
		// Should have at least headers plus the file rows.
		foundFileRow := false
		for _, row := range updated.status.changesRows {
			if !row.isHeader && row.driftFile != nil {
				foundFileRow = true
				break
			}
		}
		if !foundFileRow {
			t.Fatal("expected at least one drift file row in changesRows")
		}
	})

	t.Run("landing stats debounce fires when all stats loaded", func(t *testing.T) {
		m := newTestModel()
		m.ui.loading = true
		m.landing.statsReady = false
		// Make sure managed and git are already done so allLandingStatsLoaded returns true.
		m.filesTab.views[managedViewManaged].loading = false
		m.filesTab.managedDeferred = false
		m.status.loadingGit = false
		m.status.gitDeferred = false
		m.status.statusDeferred = false
		gen := m.gen

		msg := chezmoiStatusLoadedMsg{files: testFiles, gen: gen}
		_, cmd := sendMsg(t, m, msg)

		// Should return a non-nil cmd (debounceLandingReadyCmd batch).
		if cmd == nil {
			t.Fatal("expected non-nil cmd for landing stats debounce")
		}
	})
}

// --- 2. TestManagedLoadedMsgRouting ---

func TestManagedLoadedMsgRouting(t *testing.T) {
	testFiles := []string{
		"/home/test/.bashrc",
		"/home/test/.config/nvim/init.lua",
	}

	t.Run("fresh gen stores files and resets cursor", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewManaged].loading = true
		m.filesTab.cursor = 5 // non-zero cursor to verify reset
		gen := m.gen

		msg := chezmoiManagedLoadedMsg{files: testFiles, gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewManaged].loading {
			t.Fatal("expected managed loading to be false after fresh gen")
		}
		if len(updated.filesTab.views[managedViewManaged].files) != len(testFiles) {
			t.Fatalf("expected %d managed files, got %d",
				len(testFiles), len(updated.filesTab.views[managedViewManaged].files))
		}
		if len(updated.filesTab.views[managedViewManaged].filteredFiles) != len(testFiles) {
			t.Fatalf("expected %d filtered managed files, got %d",
				len(testFiles), len(updated.filesTab.views[managedViewManaged].filteredFiles))
		}
		if updated.filesTab.cursor != 0 {
			t.Fatalf("expected cursor reset to 0, got %d", updated.filesTab.cursor)
		}
	})

	t.Run("error on Files tab sets message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewManaged].loading = true
		m.activeTab = 1 // Files tab
		gen := m.gen

		msg := chezmoiManagedLoadedMsg{err: errors.New("load failed"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewManaged].loading {
			t.Fatal("expected managed loading to be false after error")
		}
		if !strings.HasPrefix(updated.ui.message, "Error:") {
			t.Fatalf("expected ui.message to start with 'Error:', got %q", updated.ui.message)
		}
	})

	t.Run("panel load triggered on Files tab with wide terminal", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewManaged].loading = true
		m.activeTab = 1 // Files
		m.width = 120
		m.panel.manualOverride = true
		m.panel.visible = true
		m.landing.statsReady = true
		// Use flat view (not tree view) so selectedManagedPath picks the first file.
		m.filesTab.treeView = false
		m.filesTab.viewMode = managedViewManaged
		gen := m.gen

		msg := chezmoiManagedLoadedMsg{files: testFiles, gen: gen}
		_, cmd := sendMsg(t, m, msg)

		if cmd == nil {
			t.Fatal("expected non-nil cmd for panel load on Files tab with wide terminal")
		}
	})

	t.Run("no panel load on Status tab", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewManaged].loading = true
		m.activeTab = 0 // Status
		m.width = 120
		m.panel.manualOverride = true
		m.panel.visible = true
		m.landing.statsReady = true
		gen := m.gen

		msg := chezmoiManagedLoadedMsg{files: testFiles, gen: gen}
		_, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd when activeTab is Status, panel should not trigger for wrong tab")
		}
	})
}

// --- 3. TestGitStatusLoadedMsgRouting ---

func TestGitStatusLoadedMsgRouting(t *testing.T) {
	testStaged := []chezmoi.GitFile{{Path: "dot_bashrc"}}
	testUnstaged := []chezmoi.GitFile{{Path: "dot_vimrc"}}
	testInfo := chezmoi.GitInfo{Branch: "main"}

	t.Run("fresh gen stores git data and clears loadingGit", func(t *testing.T) {
		m := newTestModel()
		m.status.loadingGit = true
		gen := m.gen

		msg := chezmoiGitStatusLoadedMsg{
			staged:   testStaged,
			unstaged: testUnstaged,
			info:     testInfo,
			gen:      gen,
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.status.loadingGit {
			t.Fatal("expected loadingGit to be false after fresh gen")
		}
		if len(updated.status.gitStagedFiles) != len(testStaged) {
			t.Fatalf("expected %d staged files, got %d",
				len(testStaged), len(updated.status.gitStagedFiles))
		}
		if len(updated.status.gitUnstagedFiles) != len(testUnstaged) {
			t.Fatalf("expected %d unstaged files, got %d",
				len(testUnstaged), len(updated.status.gitUnstagedFiles))
		}
		if updated.status.gitInfo.Branch != testInfo.Branch {
			t.Fatalf("expected branch %q, got %q", testInfo.Branch, updated.status.gitInfo.Branch)
		}
	})

	t.Run("empty branch preserves previous git info", func(t *testing.T) {
		m := newTestModel()
		m.status.loadingGit = true
		m.status.gitInfo = chezmoi.GitInfo{Branch: "main", Ahead: 2, Behind: 1}
		gen := m.gen

		msg := chezmoiGitStatusLoadedMsg{
			staged:   testStaged,
			unstaged: testUnstaged,
			info:     chezmoi.GitInfo{},
			gen:      gen,
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.status.gitInfo.Branch != "main" {
			t.Fatalf("expected branch to remain %q, got %q", "main", updated.status.gitInfo.Branch)
		}
		if updated.status.gitInfo.Ahead != 2 || updated.status.gitInfo.Behind != 1 {
			t.Fatalf("expected ahead/behind to remain 2/1, got %d/%d", updated.status.gitInfo.Ahead, updated.status.gitInfo.Behind)
		}
		if len(updated.status.gitStagedFiles) != len(testStaged) {
			t.Fatalf("expected %d staged files, got %d", len(testStaged), len(updated.status.gitStagedFiles))
		}
		if len(updated.status.gitUnstagedFiles) != len(testUnstaged) {
			t.Fatalf("expected %d unstaged files, got %d", len(testUnstaged), len(updated.status.gitUnstagedFiles))
		}
	})

	t.Run("error on Status tab sets message", func(t *testing.T) {
		m := newTestModel()
		m.status.loadingGit = true
		m.activeTab = 0 // Status tab
		gen := m.gen

		msg := chezmoiGitStatusLoadedMsg{err: errors.New("git error"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.status.loadingGit {
			t.Fatal("expected loadingGit to be false after error")
		}
		if !strings.HasPrefix(updated.ui.message, "Error:") {
			t.Fatalf("expected ui.message to start with 'Error:', got %q", updated.ui.message)
		}
	})

	t.Run("error on non-Status tab does not set message", func(t *testing.T) {
		m := newTestModel()
		m.status.loadingGit = true
		m.activeTab = 1 // Files tab
		gen := m.gen

		msg := chezmoiGitStatusLoadedMsg{err: errors.New("git error"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.status.loadingGit {
			t.Fatal("expected loadingGit to be false after error")
		}
		if updated.ui.message != "" {
			t.Fatalf("expected empty ui.message on non-Status tab, got %q", updated.ui.message)
		}
	})
}

// --- 4. TestGitCommitsLoadedMsgRouting ---

func TestGitCommitsLoadedMsgRouting(t *testing.T) {
	testUnpushed := []chezmoi.GitCommit{
		{Hash: "abc1234", Message: "update dotfiles"},
	}
	testIncoming := []chezmoi.GitCommit{
		{Hash: "def5678", Message: "remote change"},
	}

	t.Run("fresh gen stores commits", func(t *testing.T) {
		m := newTestModel()
		gen := m.gen

		msg := chezmoiGitCommitsLoadedMsg{
			unpushed: testUnpushed,
			incoming: testIncoming,
			gen:      gen,
		}
		updated, _ := sendMsg(t, m, msg)

		if len(updated.status.unpushedCommits) != len(testUnpushed) {
			t.Fatalf("expected %d unpushed commits, got %d",
				len(testUnpushed), len(updated.status.unpushedCommits))
		}
		if len(updated.status.incomingCommits) != len(testIncoming) {
			t.Fatalf("expected %d incoming commits, got %d",
				len(testIncoming), len(updated.status.incomingCommits))
		}
		if updated.status.unpushedCommits[0].Hash != testUnpushed[0].Hash {
			t.Fatalf("unpushed commit hash = %q, want %q",
				updated.status.unpushedCommits[0].Hash, testUnpushed[0].Hash)
		}
		if updated.status.incomingCommits[0].Hash != testIncoming[0].Hash {
			t.Fatalf("incoming commit hash = %q, want %q",
				updated.status.incomingCommits[0].Hash, testIncoming[0].Hash)
		}
	})

	t.Run("error is non-fatal and returns nil cmd", func(t *testing.T) {
		m := newTestModel()
		gen := m.gen

		msg := chezmoiGitCommitsLoadedMsg{
			err: errors.New("git log failed"),
			gen: gen,
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for error in git commits loaded")
		}
		if len(updated.status.unpushedCommits) != 0 {
			t.Fatalf("expected no unpushed commits on error, got %d",
				len(updated.status.unpushedCommits))
		}
		if len(updated.status.incomingCommits) != 0 {
			t.Fatalf("expected no incoming commits on error, got %d",
				len(updated.status.incomingCommits))
		}
	})
}

// --- 5. TestFilesSearchDebouncedMsgRouting ---

func TestFilesSearchDebouncedMsgRouting(t *testing.T) {
	t.Run("stale request ID returns nil cmd", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.search.request = 5

		msg := filesSearchDebouncedMsg{requestID: 3} // stale
		_, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for stale request ID")
		}
	})

	t.Run("valid request but wrong tab resets search", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0 // Status, not Files
		m.filesTab.search.request = 7
		m.filesTab.search.searching = true

		msg := filesSearchDebouncedMsg{requestID: 7}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd when not on Files tab")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching to be false after reset")
		}
	})

	t.Run("valid request but empty query resets search", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 1 // Files
		m.filesTab.search.request = 10
		m.filesTab.search.searching = true
		m.filterInput.SetValue("") // empty query

		msg := filesSearchDebouncedMsg{requestID: 10}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd when query is empty")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching to be false after reset")
		}
	})
}

// --- 6. TestFilesSearchCompletedMsgRouting ---

func TestFilesSearchCompletedMsgRouting(t *testing.T) {
	t.Run("stale request ID discards message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.search.request = 10
		m.filesTab.search.searching = true

		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 5, // stale
			query:     "test",
			results:   []string{"/home/test/unmanaged.txt"},
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for stale request ID")
		}
		if !updated.filesTab.search.searching {
			t.Fatal("expected searching state unchanged for stale request")
		}
	})

	t.Run("context canceled preserves canonical partial raw results and projects unmanaged subset", func(t *testing.T) {
		home := t.TempDir()
		rootDir := filepath.Join(home, "root")
		if err := os.MkdirAll(rootDir, 0o755); err != nil {
			t.Fatalf("mkdir root: %v", err)
		}
		child := filepath.Join(rootDir, "child.txt")
		if err := os.WriteFile(child, []byte("x"), 0o644); err != nil {
			t.Fatalf("write child: %v", err)
		}
		outsideDir := filepath.Join(home, "other")
		if err := os.MkdirAll(outsideDir, 0o755); err != nil {
			t.Fatalf("mkdir other: %v", err)
		}
		outside := filepath.Join(outsideDir, "outside.txt")
		if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
			t.Fatalf("write outside: %v", err)
		}

		m := NewModel(Options{Service: testServiceWithTarget(home)})
		m.activeTab = 1 // Files
		m.targetPath = home
		m.filesTab.viewMode = managedViewUnmanaged
		m.filesTab.search.request = 15
		m.filesTab.search.searching = true
		m.filesTab.views[managedViewManaged].files = []string{}
		m.filesTab.views[managedViewIgnored].files = []string{}
		m.filesTab.views[managedViewUnmanaged].files = []string{rootDir}
		m.filesTab.dataset = rebuildDataset(&m.filesTab)
		m.filterInput.SetValue("child")

		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 15,
			query:     "child",
			results: []string{
				child,
				outside,
			},
			metrics: filesSearchMetrics{
				roots:      1,
				matches:    2,
				terminated: "canceled",
			},
			err: context.Canceled,
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for context.Canceled")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after context.Canceled")
		}
		if !updated.filesTab.search.ready {
			t.Fatal("expected ready=true when canceled search returns partial results")
		}
		if len(updated.filesTab.search.rawResults) != 2 {
			t.Fatalf("expected canonical raw partial results to be preserved, got %#v", updated.filesTab.search.rawResults)
		}
		projected := updated.filesSearchPathsForMode("child", managedViewUnmanaged)
		if len(projected) != 1 || projected[0] != child {
			t.Fatalf("expected projected unmanaged subset [%q], got %#v", child, projected)
		}
		if updated.filesTab.search.lastMetrics.terminated != "canceled" {
			t.Fatalf("expected canceled metrics termination, got %q", updated.filesTab.search.lastMetrics.terminated)
		}
	})

	t.Run("deadline exceeded keeps partial results without ui error", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 1 // Files
		m.filesTab.search.request = 16
		m.filesTab.search.searching = true
		m.filesTab.views[managedViewUnmanaged].files = []string{"/home/test/root"}
		m.filterInput.SetValue("test")

		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 16,
			query:     "test",
			results:   []string{"/home/test/root/deep/file.txt"},
			err:       context.DeadlineExceeded,
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for context.DeadlineExceeded")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after context.DeadlineExceeded")
		}
		if !updated.filesTab.search.ready {
			t.Fatal("expected ready=true when deadline-exceeded search returns partial results")
		}
		if strings.Contains(updated.ui.message, "Search error:") {
			t.Fatalf("expected no search error UI message, got %q", updated.ui.message)
		}
	})

	t.Run("query mismatch discards message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.search.request = 20
		m.filesTab.search.searching = true
		m.filterInput.SetValue("current-query")

		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 20,
			query:     "old-query", // does not match current filter input
			results:   []string{"/home/test/file.txt"},
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for query mismatch")
		}
		if updated.filesTab.search.cancel != nil {
			t.Fatal("expected search cancel func to be nil after completion")
		}
	})

	t.Run("success stores results and sets ready", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 1 // Files
		m.filesTab.search.request = 25
		m.filesTab.search.searching = true
		m.filesTab.views[managedViewUnmanaged].files = []string{"/home/test"}
		m.filterInput.SetValue("myquery")

		resultFiles := []string{"/home/test/found1.txt", "/home/test/found2.txt"}
		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 25,
			query:     "myquery",
			results:   resultFiles,
			metrics: filesSearchMetrics{
				roots:      1,
				matches:    len(resultFiles),
				terminated: "complete",
			},
		}
		updated, _ := sendMsg(t, m, msg)

		if !updated.filesTab.search.ready {
			t.Fatal("expected ready=true after successful search")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after successful search")
		}
		if len(updated.filesTab.search.rawResults) != len(resultFiles) {
			t.Fatalf("expected %d search results, got %d",
				len(resultFiles), len(updated.filesTab.search.rawResults))
		}
		if updated.filesTab.search.lastMetrics.terminated != "complete" {
			t.Fatalf("expected complete metrics termination, got %q", updated.filesTab.search.lastMetrics.terminated)
		}
	})

	t.Run("error clears ready and sets ui message", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 1 // Files
		m.filesTab.search.request = 30
		m.filesTab.search.searching = true
		m.filterInput.SetValue("errquery")

		msg := filesSearchCompletedMsg{
			gen:       m.gen,
			requestID: 30,
			query:     "errquery",
			err:       errors.New("search failed"),
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.search.ready {
			t.Fatal("expected ready=false after error")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after error")
		}
		if !strings.Contains(updated.ui.message, "Search error:") {
			t.Fatalf("expected ui.message to contain 'Search error:', got %q", updated.ui.message)
		}
	})

	t.Run("stale generation discards message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.search.request = 35
		m.filesTab.search.searching = true
		m.filterInput.SetValue("gen-query")
		m.nextGen() // msg generated before this refresh should be ignored

		msg := filesSearchCompletedMsg{
			gen:       m.gen - 1,
			requestID: 35,
			query:     "gen-query",
			results:   []string{"/home/test/old.txt"},
		}
		updated, cmd := sendMsg(t, m, msg)

		if cmd != nil {
			t.Fatal("expected nil cmd for stale generation")
		}
		if !updated.filesTab.search.searching {
			t.Fatal("expected searching state unchanged for stale generation")
		}
	})
}

// --- 7. TestInfoContentLoadedMsgRouting ---

func TestInfoContentLoadedMsgRouting(t *testing.T) {
	t.Run("fresh gen stores content and marks loaded", func(t *testing.T) {
		m := newTestModel()
		m.info.views[infoViewDoctor].loading = true
		gen := m.gen

		content := "ok doctor output\nline two"
		msg := infoContentLoadedMsg{
			view:    infoViewDoctor,
			content: content,
			gen:     gen,
		}
		updated, _ := sendMsg(t, m, msg)

		view := updated.info.views[infoViewDoctor]
		if !view.loaded {
			t.Fatal("expected info view loaded=true after fresh gen")
		}
		if view.loading {
			t.Fatal("expected info view loading=false after fresh gen")
		}
		if view.content != content {
			t.Fatalf("expected content %q, got %q", content, view.content)
		}
		if len(view.lines) == 0 {
			t.Fatal("expected lines to be populated")
		}
		if len(view.lines) != 2 {
			t.Fatalf("expected 2 lines for doctor output, got %d", len(view.lines))
		}
	})

	t.Run("stale gen discards message", func(t *testing.T) {
		m := newTestModel()
		m.info.views[infoViewConfig].loading = true
		staleGen := m.gen + 1

		msg := infoContentLoadedMsg{
			view:    infoViewConfig,
			content: "some config",
			gen:     staleGen,
		}
		updated, _ := sendMsg(t, m, msg)

		view := updated.info.views[infoViewConfig]
		if view.loaded {
			t.Fatal("expected info view loaded=false for stale gen")
		}
		if view.content != "" {
			t.Fatalf("expected empty content for stale gen, got %q", view.content)
		}
	})

	t.Run("error stores error in lines and marks loaded", func(t *testing.T) {
		m := newTestModel()
		m.info.views[infoViewFull].loading = true
		gen := m.gen

		msg := infoContentLoadedMsg{
			view: infoViewFull,
			err:  errors.New("config dump failed"),
			gen:  gen,
		}
		updated, _ := sendMsg(t, m, msg)

		view := updated.info.views[infoViewFull]
		if !view.loaded {
			t.Fatal("expected info view loaded=true even on error")
		}
		if view.loading {
			t.Fatal("expected info view loading=false after error")
		}
		if len(view.lines) == 0 {
			t.Fatal("expected error lines to be populated")
		}
		if !strings.HasPrefix(view.lines[0], "Error:") {
			t.Fatalf("expected first line to start with 'Error:', got %q", view.lines[0])
		}
	})

	t.Run("config view applies syntax highlighting", func(t *testing.T) {
		m := newTestModel()
		m.info.views[infoViewConfig].loading = true
		gen := m.gen

		content := "[core]\n  editor = vim"
		msg := infoContentLoadedMsg{
			view:    infoViewConfig,
			content: content,
			gen:     gen,
		}
		updated, _ := sendMsg(t, m, msg)

		view := updated.info.views[infoViewConfig]
		if !view.loaded {
			t.Fatal("expected info view loaded=true")
		}
		if view.content != content {
			t.Fatalf("expected raw content preserved, got %q", view.content)
		}
		if len(view.lines) == 0 {
			t.Fatal("expected lines populated after highlighting")
		}
	})

	t.Run("data view with json format", func(t *testing.T) {
		m := newTestModel()
		m.info.format = "json"
		m.info.views[infoViewData].loading = true
		gen := m.gen

		content := `{"key": "value"}`
		msg := infoContentLoadedMsg{
			view:    infoViewData,
			content: content,
			gen:     gen,
		}
		updated, _ := sendMsg(t, m, msg)

		view := updated.info.views[infoViewData]
		if !view.loaded {
			t.Fatal("expected info view loaded=true")
		}
		if view.content != content {
			t.Fatalf("expected raw content preserved, got %q", view.content)
		}
	})
}

// --- 8. TestLandingStatsReadyMsg ---

func TestLandingStatsReadyMsg(t *testing.T) {
	t.Run("sets statsReady to true", func(t *testing.T) {
		m := newTestModel()
		m.landing.statsReady = false

		msg := landingStatsReadyMsg{}
		updated, cmd := sendMsg(t, m, msg)

		if !updated.landing.statsReady {
			t.Fatal("expected landing.statsReady to be true after landingStatsReadyMsg")
		}
		if cmd != nil {
			t.Fatal("expected nil cmd from landingStatsReadyMsg")
		}
	})
}

// --- 9. TestDiffLoadedMsgRouting ---

func TestDiffLoadedMsgRouting(t *testing.T) {
	t.Run("success sets DiffScreen and stores content", func(t *testing.T) {
		m := newTestModel()
		m.view = StatusScreen
		m.ui.busyAction = true

		diffContent := "--- a/file\n+++ b/file\n@@ -1 +1 @@\n-old\n+new"
		msg := chezmoiDiffLoadedMsg{
			path: "/home/test/.bashrc",
			diff: diffContent,
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.view != DiffScreen {
			t.Fatalf("expected view=DiffScreen, got %d", updated.view)
		}
		if updated.ui.busyAction {
			t.Fatal("expected busyAction to be false after diff loaded")
		}
		if updated.diff.content != diffContent {
			t.Fatalf("expected diff content %q, got %q", diffContent, updated.diff.content)
		}
		if updated.diff.path != "/home/test/.bashrc" {
			t.Fatalf("expected diff path %q, got %q", "/home/test/.bashrc", updated.diff.path)
		}
		if len(updated.diff.lines) == 0 {
			t.Fatal("expected diff lines to be populated")
		}
		expectedLines := strings.Split(diffContent, "\n")
		if len(updated.diff.lines) != len(expectedLines) {
			t.Fatalf("expected %d diff lines, got %d", len(expectedLines), len(updated.diff.lines))
		}
		if updated.actions.show {
			t.Fatal("expected actions.show to be false after diff loaded")
		}
	})

	t.Run("error sets message and stays in current view", func(t *testing.T) {
		m := newTestModel()
		m.view = StatusScreen
		m.ui.busyAction = true

		msg := chezmoiDiffLoadedMsg{
			path: "/home/test/.bashrc",
			err:  errors.New("diff command failed"),
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.view != StatusScreen {
			t.Fatalf("expected view to remain StatusScreen on error, got %d", updated.view)
		}
		if updated.ui.busyAction {
			t.Fatal("expected busyAction to be false after error")
		}
		if !strings.Contains(updated.ui.message, "Error loading diff:") {
			t.Fatalf("expected ui.message to contain 'Error loading diff:', got %q", updated.ui.message)
		}
	})

	t.Run("empty diff still transitions to DiffScreen", func(t *testing.T) {
		m := newTestModel()
		m.view = StatusScreen
		m.ui.busyAction = true

		msg := chezmoiDiffLoadedMsg{
			path: "/home/test/.config",
			diff: "",
		}
		updated, _ := sendMsg(t, m, msg)

		if updated.view != DiffScreen {
			t.Fatalf("expected view=DiffScreen even for empty diff, got %d", updated.view)
		}
		if updated.diff.path != "/home/test/.config" {
			t.Fatalf("expected diff path stored, got %q", updated.diff.path)
		}
	})
}

// --- Additional edge case tests ---

func TestIgnoredLoadedMsgRouting(t *testing.T) {
	testFiles := []string{".git", ".DS_Store"}

	t.Run("fresh gen stores ignored files", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewIgnored].loading = true
		gen := m.gen

		msg := chezmoiIgnoredLoadedMsg{files: testFiles, gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewIgnored].loading {
			t.Fatal("expected ignored loading to be false after fresh gen")
		}
		if len(updated.filesTab.views[managedViewIgnored].files) != len(testFiles) {
			t.Fatalf("expected %d ignored files, got %d",
				len(testFiles), len(updated.filesTab.views[managedViewIgnored].files))
		}
	})

	t.Run("error on Files tab sets message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewIgnored].loading = true
		m.activeTab = 1 // Files
		gen := m.gen

		msg := chezmoiIgnoredLoadedMsg{err: errors.New("permission denied"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewIgnored].loading {
			t.Fatal("expected ignored loading to be false after error")
		}
		if !strings.Contains(updated.ui.message, "Error loading ignored files:") {
			t.Fatalf("expected error message about ignored files, got %q", updated.ui.message)
		}
	})
}

func TestUnmanagedLoadedMsgRouting(t *testing.T) {
	testFiles := []string{"random.txt", "notes.md"}

	t.Run("fresh gen stores unmanaged files", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewUnmanaged].loading = true
		gen := m.gen

		msg := chezmoiUnmanagedLoadedMsg{files: testFiles, gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewUnmanaged].loading {
			t.Fatal("expected unmanaged loading to be false after fresh gen")
		}
		if len(updated.filesTab.views[managedViewUnmanaged].files) != len(testFiles) {
			t.Fatalf("expected %d unmanaged files, got %d",
				len(testFiles), len(updated.filesTab.views[managedViewUnmanaged].files))
		}
	})

	t.Run("error resets search and sets ui message", func(t *testing.T) {
		m := newTestModel()
		m.filesTab.views[managedViewUnmanaged].loading = true
		m.activeTab = 1 // Files
		gen := m.gen

		msg := chezmoiUnmanagedLoadedMsg{err: errors.New("scan failed"), gen: gen}
		updated, _ := sendMsg(t, m, msg)

		if updated.filesTab.views[managedViewUnmanaged].loading {
			t.Fatal("expected unmanaged loading to be false after error")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after error")
		}
		if updated.filesTab.search.ready {
			t.Fatal("expected ready=false after error")
		}
		if !strings.Contains(updated.ui.message, "Error loading unmanaged files:") {
			t.Fatalf("expected error message, got %q", updated.ui.message)
		}
	})
}

// --- Generation counter guard comprehensive test ---

func TestGenerationGuardConsistency(t *testing.T) {
	m := newTestModel()
	currentGen := m.gen
	staleGen := currentGen + 99

	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{
			name: "stale chezmoiStatusLoadedMsg",
			msg: chezmoiStatusLoadedMsg{
				files: []chezmoi.FileStatus{{Path: "test"}},
				gen:   staleGen,
			},
		},
		{
			name: "stale chezmoiManagedLoadedMsg",
			msg: chezmoiManagedLoadedMsg{
				files: []string{"test"},
				gen:   staleGen,
			},
		},
		{
			name: "stale chezmoiGitStatusLoadedMsg",
			msg: chezmoiGitStatusLoadedMsg{
				staged: []chezmoi.GitFile{{Path: "test"}},
				gen:    staleGen,
			},
		},
		{
			name: "stale chezmoiGitCommitsLoadedMsg",
			msg: chezmoiGitCommitsLoadedMsg{
				unpushed: []chezmoi.GitCommit{{Hash: "abc"}},
				gen:      staleGen,
			},
		},
		{
			name: "stale infoContentLoadedMsg",
			msg: infoContentLoadedMsg{
				view:    infoViewDoctor,
				content: "test",
				gen:     staleGen,
			},
		},
		{
			name: "stale chezmoiIgnoredLoadedMsg",
			msg: chezmoiIgnoredLoadedMsg{
				files: []string{"test"},
				gen:   staleGen,
			},
		},
		{
			name: "stale chezmoiUnmanagedLoadedMsg",
			msg: chezmoiUnmanagedLoadedMsg{
				files: []string{"test"},
				gen:   staleGen,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, cmd := sendMsg(t, m, tc.msg)
			if cmd != nil {
				t.Fatalf("expected nil cmd for stale gen message %s", tc.name)
			}
		})
	}
}

// --- Startup error guard ---

func TestStartupErrorGuardsAllMessages(t *testing.T) {
	m := newTestModel()
	m.startupErr = errors.New("chezmoi not found")

	msg := chezmoiStatusLoadedMsg{
		files: []chezmoi.FileStatus{{Path: "test"}},
		gen:   m.gen,
	}
	_, cmd := sendMsg(t, m, msg)

	if cmd != nil {
		t.Fatal("expected nil cmd when startupErr is set for non-key message")
	}
}
