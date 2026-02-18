package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCommandsCursorDownRepeatAccelerates(t *testing.T) {
	m := newTestModel(WithTab(3))
	m.view = StatusScreen
	m.activeTab = 3 // Commands
	m.cmds.items = []chezmoiCommandItem{
		{label: "One", id: chezmoiCmdStatus, available: true},
		{label: "Two", id: chezmoiCmdStatus, available: true},
		{label: "Three", id: chezmoiCmdStatus, available: true},
		{label: "Four", id: chezmoiCmdStatus, available: true},
		{label: "Five", id: chezmoiCmdStatus, available: true},
	}
	m.cmds.cursor = 0

	updatedAny, _ := m.handleCommandsKeys(tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model from handleCommandsKeys")
	}

	want := min(3, len(m.cmds.items)-1)
	if updated.cmds.cursor != want {
		t.Fatalf("expected repeat down to jump to %d, got %d", want, updated.cmds.cursor)
	}
}

func TestCommandsRunRepeatIgnored(t *testing.T) {
	m := newTestModel(WithTab(3))
	m.view = StatusScreen
	m.activeTab = 3 // Commands
	m.cmds.items = []chezmoiCommandItem{
		{label: "Status", id: chezmoiCmdStatus, available: true},
	}
	m.cmds.cursor = 0

	updatedAny, cmd := m.handleCommandsKeys(tea.KeyPressMsg{Code: tea.KeyEnter, IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model from handleCommandsKeys")
	}

	if cmd != nil {
		t.Fatal("expected repeat enter to be ignored (nil cmd)")
	}
	if updated.ui.busyAction {
		t.Fatal("expected busyAction to remain false when repeat run is ignored")
	}
}
