package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestStartFetchNoUpstreamManual(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncNoUpstream, 0, 0))

	next, cmd := m.startFetch(true)
	if cmd != nil {
		t.Fatal("expected no command when there is no upstream")
	}
	if next.ui.message != "no upstream configured" {
		t.Fatalf("message = %q, want %q", next.ui.message, "no upstream configured")
	}
	if next.status.fetchInProgress {
		t.Fatal("expected fetch not to start")
	}
}

func TestStartFetchNoUpstreamAutomaticIsSilent(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncNoUpstream, 0, 0))
	m.ui.message = "keep"

	next, cmd := m.startFetch(false)
	if cmd != nil || next.status.fetchInProgress {
		t.Fatal("expected automatic fetch to be skipped without upstream")
	}
	if next.ui.message != "keep" {
		t.Fatalf("automatic fetch must not touch the message line, got %q", next.ui.message)
	}
}

func TestStartFetchAlreadyInProgress(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))

	next, cmd := m.startFetch(true)
	if cmd != nil {
		t.Fatal("expected no command while a fetch is in flight")
	}
	if next.ui.message != "fetch already in progress" {
		t.Fatalf("message = %q", next.ui.message)
	}

	m.ui.message = "keep"
	next, cmd = m.startFetch(false)
	if cmd != nil || next.ui.message != "keep" {
		t.Fatal("expected automatic fetch to be skipped silently while in flight")
	}
}

func TestStartFetchCooldownOnlyForManual(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0))
	m.status.lastFetchTime = time.Now()

	next, cmd := m.startFetch(true)
	if cmd != nil || next.status.fetchInProgress {
		t.Fatal("expected manual fetch to respect the cooldown")
	}
	if !strings.Contains(next.ui.message, "fetch cooldown") {
		t.Fatalf("message = %q, want cooldown notice", next.ui.message)
	}

	next, cmd = m.startFetch(false)
	if cmd == nil || !next.status.fetchInProgress {
		t.Fatal("expected automatic fetch to ignore the cooldown")
	}
	if next.status.fetchManual {
		t.Fatal("automatic fetch must not be flagged manual")
	}
}

func TestStartFetchManualSetsFlagsAndMessage(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0))

	next, cmd := m.startFetch(true)
	if cmd == nil {
		t.Fatal("expected a fetch command")
	}
	if !next.status.fetchInProgress || !next.status.fetchManual {
		t.Fatalf("expected in-flight manual fetch, got inProgress=%t manual=%t", next.status.fetchInProgress, next.status.fetchManual)
	}
	if next.ui.message != "fetching..." {
		t.Fatalf("message = %q", next.ui.message)
	}
}

func TestStatusFetchKeyOnIncomingHeaderIsManual(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithDriftFiles(nil))
	m.status.changesCursor = 0 // Incoming header is always first

	updated, cmd := sendKey(t, m, runeKey("f"))
	if cmd == nil || !updated.status.fetchInProgress {
		t.Fatal("expected f on the Incoming header to start a fetch")
	}
	if !updated.status.fetchManual {
		t.Fatal("expected a key-initiated fetch to be manual")
	}
}

func TestHandleGitFetchDoneAutomaticFailureIsSilent(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	m.ui.message = "keep"

	err := fmt.Errorf("%w: could not resolve host", chezmoi.ErrFetchUnreachable)
	updated, cmd := sendMsg(t, m, chezmoiGitFetchDoneMsg{err: err, gen: m.gen})
	if cmd != nil {
		t.Fatal("expected no reload after a failed fetch")
	}
	if updated.status.fetchInProgress {
		t.Fatal("expected fetchInProgress cleared")
	}
	if updated.status.fetchOutcome != fetchFailed || updated.status.fetchReason != "unreachable" {
		t.Fatalf("outcome=%d reason=%q, want failed/unreachable", updated.status.fetchOutcome, updated.status.fetchReason)
	}
	if updated.ui.message != "keep" {
		t.Fatalf("automatic failure must not touch the message line, got %q", updated.ui.message)
	}
}

func TestHandleGitFetchDoneManualFailureReports(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	m.status.fetchManual = true

	err := fmt.Errorf("%w: permission denied", chezmoi.ErrFetchAuth)
	updated, _ := sendMsg(t, m, chezmoiGitFetchDoneMsg{err: err, gen: m.gen})
	if updated.ui.message != "fetch failed: auth failed" {
		t.Fatalf("message = %q", updated.ui.message)
	}
	if updated.status.fetchManual {
		t.Fatal("expected fetchManual reset after completion")
	}
}

func TestHandleGitFetchDoneSuccessReloads(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchFailed, "unreachable", true))
	m.ui.message = "keep"

	updated, cmd := sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if cmd == nil {
		t.Fatal("expected git status and commits reload after a successful fetch")
	}
	if updated.status.fetchOutcome != fetchOK || updated.status.fetchReason != "" {
		t.Fatalf("outcome=%d reason=%q, want ok/empty", updated.status.fetchOutcome, updated.status.fetchReason)
	}
	if time.Since(updated.status.lastFetchTime) > time.Minute {
		t.Fatalf("lastFetchTime not updated: %v", updated.status.lastFetchTime)
	}
	if updated.ui.message != "keep" {
		t.Fatalf("automatic success must not touch the message line, got %q", updated.ui.message)
	}

	m.status.fetchManual = true
	updated, _ = sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if updated.ui.message != "fetch complete" {
		t.Fatalf("manual success message = %q", updated.ui.message)
	}
}

func TestHandleGitFetchDoneIgnoresGen(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	staleGen := m.gen
	m.gen += 3 // user pressed r while the fetch was in flight

	updated, _ := sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: staleGen})
	if updated.status.fetchInProgress {
		t.Fatal("a fetch result must clear fetchInProgress regardless of gen")
	}
	if updated.status.fetchOutcome != fetchOK {
		t.Fatalf("outcome = %d, want fetchOK", updated.status.fetchOutcome)
	}
}

func TestFetchReasonFor(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fmt.Errorf("%w after 10s", chezmoi.ErrFetchTimeout), "timed out"},
		{fmt.Errorf("%w: x", chezmoi.ErrFetchAuth), "auth failed"},
		{fmt.Errorf("%w: x", chezmoi.ErrFetchUnreachable), "unreachable"},
		{errors.New("something odd"), "fetch failed"},
	}
	for _, tc := range cases {
		if got := fetchReasonFor(tc.err); got != tc.want {
			t.Errorf("fetchReasonFor(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestFetchAgeTickArmedOnceAndReleasedOnFailure(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))

	updated, _ := sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if !updated.status.fetchAgeTicking {
		t.Fatal("expected a successful fetch to arm the age tick")
	}

	again, cmd := sendMsg(t, updated, fetchAgeTickMsg{})
	if cmd == nil || !again.status.fetchAgeTicking {
		t.Fatal("expected the tick to re-arm while the last fetch succeeded")
	}

	again.status.fetchOutcome = fetchFailed
	released, cmd := sendMsg(t, again, fetchAgeTickMsg{})
	if cmd != nil || released.status.fetchAgeTicking {
		t.Fatal("expected the tick to stop once there is no age to show")
	}
}

func TestFetchLifecycleReachesHandlersWhileCommitFormOpen(t *testing.T) {
	m := newTestModel(WithView(CommitScreen), WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))

	updated, _ := sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if updated.status.fetchInProgress {
		t.Fatal("fetch completion must clear fetchInProgress even while the commit form is open")
	}
	if updated.view != CommitScreen {
		t.Fatalf("view = %v, want CommitScreen untouched", updated.view)
	}

	ticked, cmd := sendMsg(t, updated, fetchAgeTickMsg{})
	if cmd == nil || !ticked.status.fetchAgeTicking {
		t.Fatal("age tick must re-arm while the commit form is open")
	}

	info := chezmoi.GitInfo{Branch: "main", Upstream: "origin/main", Behind: 1, Sync: chezmoi.GitSyncBehind}
	loaded, _ := sendMsg(t, ticked, chezmoiGitStatusLoadedMsg{gen: ticked.gen, seq: ticked.status.gitReadSeq, info: info})
	if loaded.status.gitInfo.Behind != 1 {
		t.Fatal("git status results must be applied while the commit form is open")
	}
}

func TestHandleGitFetchDoneRetiresPreFetchReads(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	m.status.fetchAgeTicking = true // keep the minute tick out of the executed batch
	before := m.status.gitReadSeq

	updated, cmd := sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if updated.status.gitReadSeq != before+1 || !updated.status.gitReadPending {
		t.Fatalf("seq=%d pending=%t, want seq %d and pending", updated.status.gitReadSeq, updated.status.gitReadPending, before+1)
	}
	for _, msg := range collectInitAndBatchMsgs(t, cmd) {
		switch msg := msg.(type) {
		case chezmoiGitStatusLoadedMsg:
			if msg.seq != updated.status.gitReadSeq {
				t.Fatalf("post-fetch status read seq = %d, want %d", msg.seq, updated.status.gitReadSeq)
			}
		case chezmoiGitCommitsLoadedMsg:
			if msg.seq != updated.status.gitReadSeq {
				t.Fatalf("post-fetch commits read seq = %d, want %d", msg.seq, updated.status.gitReadSeq)
			}
		}
	}
}

func TestStaleGitReadsDroppedAfterFetch(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	m, _ = sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	fresh := chezmoi.GitInfo{Branch: "main", Upstream: "origin/main", Behind: 1, Sync: chezmoi.GitSyncBehind}
	stale := chezmoi.GitInfo{Branch: "main", Upstream: "origin/main", Sync: chezmoi.GitSyncSynced}

	m, _ = sendMsg(t, m, chezmoiGitStatusLoadedMsg{gen: m.gen, seq: m.status.gitReadSeq, info: fresh})
	if m.status.gitInfo.Behind != 1 || m.status.gitReadPending {
		t.Fatalf("fresh result not applied: %+v pending=%t", m.status.gitInfo, m.status.gitReadPending)
	}

	// The pre-fetch read finishes late and must not win.
	m, _ = sendMsg(t, m, chezmoiGitStatusLoadedMsg{gen: m.gen, seq: m.status.gitReadSeq - 1, info: stale})
	if m.status.gitInfo.Sync != chezmoi.GitSyncBehind || m.status.gitInfo.Behind != 1 {
		t.Fatalf("stale pre-fetch result overwrote the fresh comparison: %+v", m.status.gitInfo)
	}

	m.status.incomingCommits = []chezmoi.GitCommit{{Hash: "abc1234", Message: "upstream"}}
	m, _ = sendMsg(t, m, chezmoiGitCommitsLoadedMsg{gen: m.gen, seq: m.status.gitReadSeq - 1})
	if len(m.status.incomingCommits) != 1 {
		t.Fatal("stale pre-fetch commits result overwrote the fresh list")
	}
}

func TestLandingNotGreenUntilPostFetchComparison(t *testing.T) {
	m := newTestModel(WithGitSync(chezmoi.GitSyncSynced, 0, 0), WithFetchState(fetchNotStarted, "", true))
	m, _ = sendMsg(t, m, chezmoiGitFetchDoneMsg{gen: m.gen})
	if m.isAllInSync() {
		t.Fatal("old snapshot must not read as in sync before the post-fetch comparison lands")
	}

	synced := chezmoi.GitInfo{Branch: "main", Upstream: "origin/main", Sync: chezmoi.GitSyncSynced}
	m, _ = sendMsg(t, m, chezmoiGitStatusLoadedMsg{gen: m.gen, seq: m.status.gitReadSeq, info: synced})
	if !m.isAllInSync() {
		t.Fatal("expected in sync once the post-fetch comparison confirms it")
	}
}
