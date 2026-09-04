package tui

import "testing"

func findManagedItem(items []chezmoiActionItem, action chezmoiAction) (chezmoiActionItem, bool) {
	for _, item := range items {
		if item.action == action {
			return item, true
		}
	}
	return chezmoiActionItem{}, false
}

func TestManagedActionsMenuReadOnlyKeepsViewSource(t *testing.T) {
	m := newTestModel(WithReadOnly(), WithManagedFiles([]string{"/home/u/.zshrc"}))
	m.filesTab.treeView = false

	m.openFilesActionsMenu()

	viewSource, ok := findManagedItem(m.actions.managedItems, chezmoiActionViewSource)
	if !ok {
		t.Fatalf("expected View Source item, got %+v", m.actions.managedItems)
	}
	if viewSource.disabled {
		t.Fatal("expected View Source enabled in read-only mode")
	}
	editSource, ok := findManagedItem(m.actions.managedItems, chezmoiActionEditSource)
	if !ok {
		t.Fatalf("expected Edit Source item, got %+v", m.actions.managedItems)
	}
	if !editSource.disabled {
		t.Fatal("expected Edit Source disabled in read-only mode")
	}
}

func TestExecuteFilesActionViewSourceAllowedInReadOnly(t *testing.T) {
	m := newTestModel(WithReadOnly(), WithManagedFiles([]string{"/home/u/.zshrc"}))
	m.filesTab.treeView = false

	updatedAny, cmd := m.executeFilesAction(chezmoiActionViewSource)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedAny)
	}
	if cmd == nil {
		t.Fatal("expected view source cmd in read-only mode")
	}
	if !updated.ui.busyAction {
		t.Fatal("expected busyAction set")
	}
	if updated.ui.message != "" {
		t.Fatalf("expected no message, got %q", updated.ui.message)
	}
}
