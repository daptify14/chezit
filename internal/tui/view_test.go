package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestViewEnablesAltScreenAndMouseCaptureByDefault(t *testing.T) {
	m := newTestModel()
	v := m.View()

	if !v.AltScreen {
		t.Fatalf("expected AltScreen=true")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("expected MouseModeCellMotion, got %v", v.MouseMode)
	}
	if !v.KeyboardEnhancements.ReportEventTypes {
		t.Fatalf("expected KeyboardEnhancements.ReportEventTypes=true")
	}
}

func TestViewDisablesMouseCaptureWhenToggledOff(t *testing.T) {
	m := newTestModel()
	m.ui.mouseCapture = false
	v := m.View()

	if v.MouseMode != tea.MouseModeNone {
		t.Fatalf("expected MouseModeNone, got %v", v.MouseMode)
	}
	if !v.KeyboardEnhancements.ReportEventTypes {
		t.Fatalf("expected KeyboardEnhancements.ReportEventTypes=true")
	}
}

func TestViewDoesNotForceBackgroundColor(t *testing.T) {
	m := newTestModel()
	v := m.View()

	if v.BackgroundColor != nil {
		t.Fatalf("expected BackgroundColor=nil to inherit terminal background, got %v", v.BackgroundColor)
	}
}
