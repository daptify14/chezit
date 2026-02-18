package tui

import "testing"

func TestEnterTabFromLandingResetsPanelModeForFiles(t *testing.T) {
	m := NewModel(Options{
		Service:     testService(),
		EscBehavior: EscQuit,
	})
	if len(m.tabNames) < 2 {
		t.Fatalf("expected at least Status and Files tabs, got %v", m.tabNames)
	}
	if m.tabNames[1] != "Files" {
		t.Fatalf("expected Files tab at index 1, got %q", m.tabNames[1])
	}
	if m.panel.contentMode != panelModeDiff {
		t.Fatalf("expected initial panel mode diff on landing (Status default), got %v", m.panel.contentMode)
	}

	updatedModel, _ := m.enterTabFromLanding(1)
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}

	if updated.view != StatusScreen {
		t.Fatalf("expected status view after landing selection, got %v", updated.view)
	}
	if updated.activeTabName() != "Files" {
		t.Fatalf("expected active tab Files, got %q", updated.activeTabName())
	}
	if updated.panel.contentMode != panelModeContent {
		t.Fatalf("expected panel mode content for Files tab, got %v", updated.panel.contentMode)
	}
}
