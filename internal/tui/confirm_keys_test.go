package tui

import (
	"os/exec"
	"path/filepath"
	"strings"
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

// --- Apply confirm selector tests ---

func newApplyConfirmModel(action chezmoiAction, label string) Model {
	m := newConfirmModel(action, label)
	m.overlays.applyForce = true // default for apply actions
	return m
}

func TestApplyConfirmDefaults(t *testing.T) {
	t.Run("apply_action_defaults_to_force", func(t *testing.T) {
		m := NewModel(Options{Service: testService()})
		m.width, m.height = 120, 40
		m = m.showConfirmScreen(chezmoiActionApplyAll, "apply all")
		if !m.overlays.applyForce || m.view != ConfirmScreen {
			t.Fatalf("expected applyForce=true and ConfirmScreen, got force=%v view=%d", m.overlays.applyForce, m.view)
		}
	})
	t.Run("non_apply_action_no_force", func(t *testing.T) {
		m := NewModel(Options{Service: testService()})
		m.width, m.height = 120, 40
		m = m.showConfirmScreen(chezmoiActionPush, "push")
		if m.overlays.applyForce {
			t.Fatal("expected applyForce=false for non-apply action")
		}
	})
}

func TestApplyConfirmNavigation(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		want bool
	}{
		{"right_selects_interactive", specialKey(tea.KeyRight), false},
		{"left_selects_force", specialKey(tea.KeyLeft), true},
		{"h_selects_force", runeKey("h"), true},
		{"l_selects_interactive", runeKey("l"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newApplyConfirmModel(chezmoiActionApplyAll, "apply all")
			m.overlays.applyForce = !tc.want // start opposite so the key has to change it
			updated, _ := sendKey(t, m, tc.key)
			if updated.overlays.applyForce != tc.want {
				t.Fatalf("expected applyForce=%v, got %v", tc.want, updated.overlays.applyForce)
			}
		})
	}
	t.Run("tab_toggles", func(t *testing.T) {
		m := newApplyConfirmModel(chezmoiActionApplyAll, "apply all")
		updated, _ := sendKey(t, m, specialKey(tea.KeyTab))
		if updated.overlays.applyForce {
			t.Fatal("expected applyForce=false after first Tab")
		}
		updated, _ = sendKey(t, updated, specialKey(tea.KeyTab))
		if !updated.overlays.applyForce {
			t.Fatal("expected applyForce=true after second Tab")
		}
	})
}

func TestApplyConfirmCancel(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"esc", specialKey(tea.KeyEscape)},
		{"n", runeKey("n")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newApplyConfirmModel(chezmoiActionApplyAll, "apply all")
			updated, _ := sendKey(t, m, tc.key)
			if updated.view != StatusScreen {
				t.Fatalf("expected StatusScreen, got %d", updated.view)
			}
			if updated.overlays.applyForce || updated.overlays.confirmAction != chezmoiActionNone {
				t.Fatalf("expected overlay state cleared, got force=%v action=%d", updated.overlays.applyForce, updated.overlays.confirmAction)
			}
			if updated.overlays.applyWrapTTY {
				t.Fatal("expected applyWrapTTY=false after cancel")
			}
		})
	}
}

func TestApplyConfirmIgnoresKeys(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{runeKey("y"), runeKey("j"), runeKey("q")} {
		m := newApplyConfirmModel(chezmoiActionApplyAll, "apply all")
		updated, _ := sendKey(t, m, k)
		if updated.view != ConfirmScreen || updated.overlays.confirmAction != chezmoiActionApplyAll {
			t.Fatalf("key %v should be ignored in apply confirm, got view=%d action=%d", k, updated.view, updated.overlays.confirmAction)
		}
	}
}

func TestApplyConfirmEnterDispatches(t *testing.T) {
	m := newApplyConfirmModel(chezmoiActionApplyAll, "apply all")
	updated, cmd := sendKey(t, m, specialKey(tea.KeyEnter))
	if updated.view != StatusScreen || updated.overlays.confirmAction != chezmoiActionNone || updated.overlays.applyForce {
		t.Fatalf("expected clean state after confirm, got view=%d action=%d force=%v", updated.view, updated.overlays.confirmAction, updated.overlays.applyForce)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd after apply confirm")
	}
}

func TestApplyConfirmExecCmd(t *testing.T) {
	tests := []struct {
		name     string
		action   chezmoiAction
		path     string
		force    bool
		wantArgs string
	}{
		{"force_apply_all", chezmoiActionApplyAll, "", true, "apply --force"},
		{"interactive_apply_all", chezmoiActionApplyAll, "", false, "apply"},
		{"force_managed", chezmoiActionApplyManaged, "/home/test/.bashrc", true, "apply --force /home/test/.bashrc"},
		{"interactive_managed", chezmoiActionApplyManaged, "/home/test/.bashrc", false, "apply /home/test/.bashrc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newApplyConfirmModel(tc.action, "apply")
			m.overlays.confirmPath = tc.path
			cmd := m.applyConfirmExecCmd(tc.action, tc.path, tc.force)
			if cmd == nil {
				t.Fatal("expected non-nil exec.Cmd")
				return
			}
			got := strings.Join(cmd.Args[1:], " ")
			if got != tc.wantArgs {
				t.Fatalf("expected args %q, got %q", tc.wantArgs, got)
			}
		})
	}
}

func TestApplyConfirmDescriptions(t *testing.T) {
	tests := []struct {
		action chezmoiAction
		force  bool
		want   string
	}{
		{chezmoiActionApplyAll, true, "Applies all changes with --force (skips chezmoi prompts)"},
		{chezmoiActionApplyAll, false, "Runs plain chezmoi apply and lets chezmoi prompt as needed"},
		{chezmoiActionApplyFile, true, "Applies the selected file with --force (skips chezmoi prompts)"},
		{chezmoiActionApplyFile, false, "Runs plain chezmoi apply for the selected file and lets chezmoi prompt if needed"},
		{chezmoiActionApplyManaged, true, "Applies the selected file with --force (skips chezmoi prompts)"},
	}
	for _, tc := range tests {
		if got := applyConfirmDescription(tc.action, tc.force); got != tc.want {
			t.Fatalf("applyConfirmDescription(%d, %v) = %q, want %q", tc.action, tc.force, got, tc.want)
		}
	}
}

func TestWrapApplyConfirmCmd(t *testing.T) {
	cmd := exec.Command("/bin/true")

	unwrapped := wrapApplyConfirmCmd(cmd, false)
	if unwrapped != cmd {
		t.Fatal("expected wrapApplyConfirmCmd(false) to return original cmd")
	}

	wrapped := wrapApplyConfirmCmd(cmd, true)
	if wrapped == nil {
		t.Fatal("expected wrapped cmd")
		return
	}
	if filepath.Base(wrapped.Path) != "sh" {
		t.Fatalf("expected wrapped command to use sh, got %q", wrapped.Path)
	}
	if !containsAny(strings.Join(wrapped.Args, " "), "chezmoi-wrap", "Press Enter to continue") {
		t.Fatalf("expected wrap args to include shell wrapper, got %v", wrapped.Args)
	}
}

func TestPreviewApplyEnterOpensSelector(t *testing.T) {
	m := newTestModel(WithTab(3))
	m.view = DiffScreen
	m.activeTab = 3
	m.diff.previewApply = true
	m.diff.content = "preview"
	m.diff.lines = []string{"+line 1"}

	updated, cmd := sendKey(t, m, specialKey(tea.KeyEnter))

	if cmd != nil {
		t.Fatal("expected no cmd — should transition to selector only")
	}
	if updated.view != ConfirmScreen || updated.overlays.confirmAction != chezmoiActionApplyAll {
		t.Fatalf("expected ConfirmScreen with ApplyAll, got view=%d action=%d", updated.view, updated.overlays.confirmAction)
	}
	if !updated.overlays.applyForce {
		t.Fatal("expected applyForce=true by default")
	}
	if !updated.overlays.applyWrapTTY {
		t.Fatal("expected preview apply selector to preserve wrapped output")
	}
	if updated.diff.previewApply || updated.diff.lines != nil {
		t.Fatal("expected diff state cleared")
	}
}
