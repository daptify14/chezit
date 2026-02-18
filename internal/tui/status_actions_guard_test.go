package tui

import (
	"fmt"
	"testing"
)

func TestActionRequiresWriteIncludesMutatingStatusActions(t *testing.T) {
	t.Parallel()

	writeActions := []chezmoiAction{
		chezmoiActionReAdd,
		chezmoiActionGitStage,
		chezmoiActionGitUnstage,
		chezmoiActionApplyFile,
		chezmoiActionApplyAll,
		chezmoiActionUpdate,
		chezmoiActionGitDiscard,
		chezmoiActionGitDiscardSelected,
		chezmoiActionGitUndoCommit,
	}
	for _, action := range writeActions {
		t.Run(fmt.Sprintf("action_%d_requires_write", action), func(t *testing.T) {
			t.Parallel()
			if !actionRequiresWrite(action) {
				t.Fatalf("expected action %d to require write", action)
			}
		})
	}

	nonWriteActions := []chezmoiAction{
		chezmoiActionNone,
		chezmoiActionViewDiff,
		chezmoiActionRefresh,
		chezmoiActionFetch,
		chezmoiActionPull,
	}
	for _, action := range nonWriteActions {
		t.Run(fmt.Sprintf("action_%d_not_write", action), func(t *testing.T) {
			t.Parallel()
			if actionRequiresWrite(action) {
				t.Fatalf("expected action %d to be non-mutating", action)
			}
		})
	}
}

func TestExecuteStatusActionReadOnlyBlocksMutatingActions(t *testing.T) {
	t.Parallel()

	blockedActions := []chezmoiAction{
		chezmoiActionReAdd,
		chezmoiActionGitStage,
		chezmoiActionGitUnstage,
	}
	for _, action := range blockedActions {
		t.Run(fmt.Sprintf("action_%d_blocked", action), func(t *testing.T) {
			t.Parallel()

			m := newTestModel(WithReadOnly())
			m.actions.show = true

			updatedAny, cmd := m.executeStatusAction(action)
			if cmd != nil {
				t.Fatal("expected nil cmd for blocked read-only action")
			}

			updated, ok := updatedAny.(Model)
			if !ok {
				t.Fatalf("expected Model, got %T", updatedAny)
			}
			if updated.actions.show {
				t.Fatal("expected action menu to close before read-only guard return")
			}
			if updated.ui.message != actionUnavailableMessage("read-only mode") {
				t.Fatalf("expected read-only message, got %q", updated.ui.message)
			}
		})
	}
}
