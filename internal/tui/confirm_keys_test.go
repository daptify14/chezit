package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- Helpers ---

// newConfirmModel returns a Model in ConfirmScreen with the given action/label.
func newConfirmModel(action chezmoiAction, label string) Model {
	m := NewModel(Options{Service: testService()})
	m.view = ConfirmScreen
	m.width = 120
	m.height = 40
	m.overlays.confirmAction = action
	m.overlays.confirmLabel = label
	return m
}

// --- Cancel flow ---

func TestConfirmCancelViaN(t *testing.T) {
	m := newConfirmModel(chezmoiActionGitStageAll, "stage all 3 unstaged files")

	updated, _ := sendKey(t, m, runeKey("n"))

	if updated.view != StatusScreen {
		t.Fatalf("expected view=StatusScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionNone {
		t.Fatalf("expected confirmAction=None, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel != "" {
		t.Fatalf("expected confirmLabel cleared, got %q", updated.overlays.confirmLabel)
	}
}

func TestConfirmCancelViaEsc(t *testing.T) {
	m := newConfirmModel(chezmoiActionPush, "push committed changes to remote")

	updated, _ := sendKey(t, m, specialKey(tea.KeyEscape))

	if updated.view != StatusScreen {
		t.Fatalf("expected view=StatusScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionNone {
		t.Fatalf("expected confirmAction=None, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel != "" {
		t.Fatalf("expected confirmLabel cleared, got %q", updated.overlays.confirmLabel)
	}
}

func TestConfirmIgnoresUnrelatedKeys(t *testing.T) {
	m := newConfirmModel(chezmoiActionGitStageAll, "stage all 3 unstaged files")

	updated, _ := sendKey(t, m, runeKey("j"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionGitStageAll {
		t.Fatalf("expected confirmAction unchanged, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel != "stage all 3 unstaged files" {
		t.Fatalf("expected confirmLabel unchanged, got %q", updated.overlays.confirmLabel)
	}
}

// --- Confirm dispatches action ---

func TestConfirmDispatchesAction(t *testing.T) {
	tests := []struct {
		name         string
		action       chezmoiAction
		label        string
		path         string
		paths        []string
		wantBusy     bool
		needFilePath bool // set a currentFilePath for applyFile
	}{
		{
			name:     "stage_all",
			action:   chezmoiActionGitStageAll,
			label:    "stage all 3 unstaged files",
			wantBusy: true,
		},
		{
			name:     "unstage_all",
			action:   chezmoiActionGitUnstageAll,
			label:    "unstage all 2 staged files",
			wantBusy: true,
		},
		{
			name:     "push",
			action:   chezmoiActionPush,
			label:    "push committed changes to remote",
			wantBusy: true,
		},
		{
			name:     "pull",
			action:   chezmoiActionPull,
			label:    "pull changes from remote",
			wantBusy: true,
		},
		{
			name:     "discard",
			action:   chezmoiActionGitDiscard,
			label:    "discard changes to .bashrc",
			path:     "/home/test/.bashrc",
			wantBusy: true,
		},
		{
			name:     "discard_selected",
			action:   chezmoiActionGitDiscardSelected,
			label:    "discard changes in 2 selected files",
			paths:    []string{"/home/test/.bashrc", "/home/test/.zshrc"},
			wantBusy: true,
		},
		{
			name:     "undo_commit",
			action:   chezmoiActionGitUndoCommit,
			label:    "undo last commit (changes return to staged)",
			wantBusy: true,
		},
		{
			name:     "apply_all",
			action:   chezmoiActionApplyAll,
			label:    "apply all changes to destination",
			wantBusy: false, // uses tea.ExecProcess
		},
		{
			name:     "update",
			action:   chezmoiActionUpdate,
			label:    "update (pull from remote and apply)",
			wantBusy: false, // uses tea.ExecProcess
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newConfirmModel(tc.action, tc.label)
			m.overlays.confirmPath = tc.path
			m.overlays.confirmPaths = tc.paths

			updatedAny, cmd := m.Update(runeKey("y"))
			updated, ok := updatedAny.(Model)
			if !ok {
				t.Fatal("Update did not return Model")
			}

			if updated.view != StatusScreen {
				t.Fatalf("expected view=StatusScreen, got %d", updated.view)
			}
			if updated.overlays.confirmAction != chezmoiActionNone {
				t.Fatalf("expected confirmAction=None, got %d", updated.overlays.confirmAction)
			}
			if updated.overlays.confirmLabel != "" {
				t.Fatalf("expected confirmLabel cleared, got %q", updated.overlays.confirmLabel)
			}
			if cmd == nil {
				t.Fatal("expected non-nil cmd after confirm")
			}
			if tc.wantBusy && !updated.ui.busyAction {
				t.Fatal("expected busyAction=true")
			}
		})
	}
}

// --- Confirm clears path fields ---

func TestConfirmClearsSinglePath(t *testing.T) {
	m := newConfirmModel(chezmoiActionGitDiscard, "discard changes to .bashrc")
	m.overlays.confirmPath = "/home/test/.bashrc"

	updated, _ := sendKey(t, m, runeKey("y"))

	if updated.overlays.confirmPath != "" {
		t.Fatalf("expected confirmPath cleared, got %q", updated.overlays.confirmPath)
	}
}

func TestConfirmClearsMultiPaths(t *testing.T) {
	m := newConfirmModel(chezmoiActionGitDiscardSelected, "discard changes in 2 selected files")
	m.overlays.confirmPaths = []string{"/home/test/.bashrc", "/home/test/.zshrc"}

	updated, _ := sendKey(t, m, runeKey("y"))

	if updated.overlays.confirmPaths != nil {
		t.Fatalf("expected confirmPaths nil, got %v", updated.overlays.confirmPaths)
	}
}

func TestConfirmCancelClearsPaths(t *testing.T) {
	m := newConfirmModel(chezmoiActionGitDiscard, "discard changes to .bashrc")
	m.overlays.confirmPath = "/home/test/.bashrc"
	m.overlays.confirmPaths = []string{"/home/test/.zshrc"}

	updated, _ := sendKey(t, m, runeKey("n"))

	if updated.overlays.confirmPath != "" {
		t.Fatalf("expected confirmPath cleared on cancel, got %q", updated.overlays.confirmPath)
	}
	if updated.overlays.confirmPaths != nil {
		t.Fatalf("expected confirmPaths nil on cancel, got %v", updated.overlays.confirmPaths)
	}
}

// --- Key triggers open confirm ---

func TestStageAllKeyOpensConfirm(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
		{Path: "/home/test/.vimrc", StatusCode: "M"},
	}
	m.buildChangesRows()

	updated, _ := sendKey(t, m, runeKey("S"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionGitStageAll {
		t.Fatalf("expected confirmAction=GitStageAll, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel == "" {
		t.Fatal("expected confirmLabel to be set")
	}
}

func TestUnstageAllKeyOpensConfirm(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false
	m.status.gitStagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "A"},
	}
	m.buildChangesRows()

	updated, _ := sendKey(t, m, runeKey("U"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionGitUnstageAll {
		t.Fatalf("expected confirmAction=GitUnstageAll, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel == "" {
		t.Fatal("expected confirmLabel to be set")
	}
}

func TestPushKeyOpensConfirm(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false

	updated, _ := sendKey(t, m, runeKey("P"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionPush {
		t.Fatalf("expected confirmAction=Push, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmLabel != "push committed changes to remote" {
		t.Fatalf("expected push label, got %q", updated.overlays.confirmLabel)
	}
}

func TestDiscardKeyOnUnstagedOpensConfirm(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.bashrc", StatusCode: "M"},
	}
	m.buildChangesRows()

	// Position cursor on the unstaged file row.
	row := findFirstSectionFileRow(t, m, changesSectionUnstaged)
	m.status.changesCursor = row

	updated, _ := sendKey(t, m, runeKey("x"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionGitDiscard {
		t.Fatalf("expected confirmAction=GitDiscard, got %d", updated.overlays.confirmAction)
	}
	if updated.overlays.confirmPath != "/home/test/.bashrc" {
		t.Fatalf("expected confirmPath set, got %q", updated.overlays.confirmPath)
	}
}

func TestDiscardKeyOnUnpushedOpensUndoConfirm(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false
	m.status.unpushedCommits = []chezmoi.GitCommit{
		{Hash: "abc1234", Message: "test commit"},
	}
	m.buildChangesRows()

	// Position cursor on the unpushed section.
	row := findFirstSectionFileRow(t, m, changesSectionUnpushed)
	m.status.changesCursor = row

	updated, _ := sendKey(t, m, runeKey("x"))

	if updated.view != ConfirmScreen {
		t.Fatalf("expected view=ConfirmScreen, got %d", updated.view)
	}
	if updated.overlays.confirmAction != chezmoiActionGitUndoCommit {
		t.Fatalf("expected confirmAction=GitUndoCommit, got %d", updated.overlays.confirmAction)
	}
}
