package tui

import (
	"testing"
)

func TestShouldShowAutoMode(t *testing.T) {
	p := newFilePanel("")

	// Below minimum width: always hidden
	if p.shouldShow(59) {
		t.Error("expected hidden at width 59 (below panelMinWidth)")
	}

	// Below threshold: hidden in auto mode
	if p.shouldShow(80) {
		t.Error("expected hidden at width 80 (below panelAutoThreshold)")
	}

	// At threshold: visible in auto mode
	if !p.shouldShow(90) {
		t.Error("expected visible at width 90 (== panelAutoThreshold)")
	}

	// Above threshold: visible
	if !p.shouldShow(200) {
		t.Error("expected visible at width 200")
	}
}

func TestShouldShowManualOverride(t *testing.T) {
	// Show mode: visible even below threshold
	p := newFilePanel("show")
	if !p.shouldShow(100) {
		t.Error("expected visible at width 100 with show override")
	}
	// But not below minimum width
	if p.shouldShow(59) {
		t.Error("expected hidden at width 59 even with show override")
	}

	// Hide mode: hidden even above threshold
	p = newFilePanel("hide")
	if p.shouldShow(200) {
		t.Error("expected hidden at width 200 with hide override")
	}
}

func TestToggle(t *testing.T) {
	p := newFilePanel("")

	// First toggle at narrow terminal (below threshold): should show
	p.toggle(80)
	if !p.manualOverride {
		t.Error("expected manualOverride after first toggle")
	}
	if !p.visible {
		t.Error("expected visible after first toggle at narrow width")
	}

	// Second toggle: should hide
	p.toggle(80)
	if p.visible {
		t.Error("expected hidden after second toggle")
	}

	// Third toggle: should show again
	p.toggle(80)
	if !p.visible {
		t.Error("expected visible after third toggle")
	}
}

func TestToggleAtWideTerminal(t *testing.T) {
	p := newFilePanel("")

	// First toggle at wide terminal (above threshold): should hide (invert auto)
	p.toggle(200)
	if p.visible {
		t.Error("expected hidden after first toggle at wide width (inverts auto-show)")
	}
}

func TestPanelWidthFor(t *testing.T) {
	tests := []struct {
		width    int
		expected int
	}{
		{100, 40}, // 40% of 100 = 40
		{200, 80}, // 40% of 200 = 80
		{60, 30},  // 40% of 60 = 24, but min is 30
		{70, 30},  // 40% of 70 = 28, but min is 30
		{75, 30},  // 40% of 75 = 30 exactly
		{120, 48}, // 40% of 120 = 48
	}

	for _, tt := range tests {
		got := panelWidthFor(tt.width)
		if got != tt.expected {
			t.Errorf("panelWidthFor(%d) = %d, want %d", tt.width, got, tt.expected)
		}
	}
}

func TestNewFilePanelModes(t *testing.T) {
	// Default mode
	p := newFilePanel("")
	if p.manualOverride {
		t.Error("expected no manual override in default mode")
	}
	if p.contentMode != panelModeDiff {
		t.Error("expected default content mode to be diff")
	}

	// Show mode
	p = newFilePanel("show")
	if !p.manualOverride || !p.visible {
		t.Error("expected manual override + visible in show mode")
	}

	// Hide mode
	p = newFilePanel("hide")
	if !p.manualOverride || p.visible {
		t.Error("expected manual override + hidden in hide mode")
	}
}

func TestEnsureViewport(t *testing.T) {
	p := newFilePanel("")

	p.ensureViewport(80, 20)
	if !p.viewportReady {
		t.Error("expected viewportReady after ensureViewport")
	}
	if p.viewport.Width() != 80 {
		t.Errorf("expected viewport width 80, got %d", p.viewport.Width())
	}
	if p.viewport.Height() != 20 {
		t.Errorf("expected viewport height 20, got %d", p.viewport.Height())
	}

	// Resize
	p.ensureViewport(100, 30)
	if p.viewport.Width() != 100 {
		t.Errorf("expected viewport width 100 after resize, got %d", p.viewport.Width())
	}
	if p.viewport.Height() != 30 {
		t.Errorf("expected viewport height 30 after resize, got %d", p.viewport.Height())
	}
}
