package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- Helpers ---

// setupDiffViewModel creates a Model configured for DiffScreen with 50 lines to scroll.
func setupDiffViewModel() Model {
	const lineCount = 50
	m := NewModel(Options{Service: testService()})
	m.view = DiffScreen
	m.width = 80
	m.height = 24

	lines := make([]string, lineCount)
	for i := range lineCount {
		lines[i] = fmt.Sprintf("+line %d added", i)
	}
	content := strings.Join(lines, "\n")
	m.diff.content = content
	m.diff.path = "/test/file"
	m.diff.lines = lines
	return m
}

// setupInfoTabModel creates a Model configured for the Info tab with 50 lines to scroll.
func setupInfoTabModel() Model {
	const lineCount = 50
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.activeTab = 2 // Info tab index
	m.width = 80
	m.height = 24
	m.info.activeView = 0 // config sub-view

	lines := make([]string, lineCount)
	for i := range lineCount {
		lines[i] = fmt.Sprintf("config line %d", i)
	}
	content := strings.Join(lines, "\n")
	m.info.views[0].content = content
	m.info.views[0].lines = lines
	m.info.views[0].loaded = true

	return m
}

// --- DiffScreen Tests ---

func TestDiffViewScrollDown(t *testing.T) {
	m := setupDiffViewModel()

	// First j initializes the viewport AND scrolls down by 1.
	updated, _ := sendKey(t, m, runeKey("j"))

	if !updated.diff.viewportReady {
		t.Fatal("expected viewport to be ready after first scroll key")
	}
	if updated.diff.viewport.YOffset() <= 0 {
		t.Fatalf("expected YOffset > 0 after scroll down, got %d", updated.diff.viewport.YOffset())
	}
}

func TestDiffViewRepeatScrollDownAccelerates(t *testing.T) {
	m := setupDiffViewModel()

	single, _ := sendKey(t, m, runeKey("j"))
	repeat, _ := sendKey(t, m, tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true})

	if repeat.diff.viewport.YOffset() <= single.diff.viewport.YOffset() {
		t.Fatalf(
			"expected repeat scroll down to move farther than single key press: repeat=%d single=%d",
			repeat.diff.viewport.YOffset(),
			single.diff.viewport.YOffset(),
		)
	}
}

func TestDiffViewScrollUp(t *testing.T) {
	m := setupDiffViewModel()

	// Scroll down several times to move away from top.
	updated, _ := sendKey(t, m, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))

	offsetAfterDown := updated.diff.viewport.YOffset()
	if offsetAfterDown <= 0 {
		t.Fatalf("expected YOffset > 0 after multiple scroll down, got %d", offsetAfterDown)
	}

	// Scroll up.
	updated, _ = sendKey(t, updated, runeKey("k"))

	if updated.diff.viewport.YOffset() >= offsetAfterDown {
		t.Fatalf("expected YOffset to decrease after scroll up: before=%d, after=%d",
			offsetAfterDown, updated.diff.viewport.YOffset())
	}
}

func TestDiffViewScrollUpAtTopStaysAtZero(t *testing.T) {
	m := setupDiffViewModel()

	// Initialize viewport with a down scroll, then go to top, then try scrolling up.
	updated, _ := sendKey(t, m, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("g")) // go to top
	updated, _ = sendKey(t, updated, runeKey("k")) // try scrolling up from top

	if updated.diff.viewport.YOffset() != 0 {
		t.Fatalf("expected YOffset = 0 when scrolling up at top, got %d", updated.diff.viewport.YOffset())
	}
}

func TestDiffViewHalfPageScroll(t *testing.T) {
	m := setupDiffViewModel()

	t.Run("HalfPageDown", func(t *testing.T) {
		updated, _ := sendKey(t, m, ctrlKey('d'))

		if !updated.diff.viewportReady {
			t.Fatal("expected viewport to be ready after half page down")
		}
		if updated.diff.viewport.YOffset() <= 1 {
			t.Fatalf("expected YOffset > 1 after half page down, got %d", updated.diff.viewport.YOffset())
		}
	})

	t.Run("HalfPageUp", func(t *testing.T) {
		// Scroll down first.
		updated, _ := sendKey(t, m, ctrlKey('d'))
		offsetAfterDown := updated.diff.viewport.YOffset()

		// Then half page up.
		updated, _ = sendKey(t, updated, ctrlKey('u'))

		if updated.diff.viewport.YOffset() >= offsetAfterDown {
			t.Fatalf("expected YOffset to decrease after half page up: before=%d, after=%d",
				offsetAfterDown, updated.diff.viewport.YOffset())
		}
	})
}

func TestDiffViewPageScroll(t *testing.T) {
	m := setupDiffViewModel()

	t.Run("PageDown", func(t *testing.T) {
		updated, _ := sendKey(t, m, ctrlKey('f'))

		if !updated.diff.viewportReady {
			t.Fatal("expected viewport to be ready after page down")
		}
		if updated.diff.viewport.YOffset() <= 1 {
			t.Fatalf("expected YOffset > 1 after full page down, got %d", updated.diff.viewport.YOffset())
		}
	})

	t.Run("PageUp", func(t *testing.T) {
		// Scroll down first.
		updated, _ := sendKey(t, m, ctrlKey('f'))
		offsetAfterDown := updated.diff.viewport.YOffset()

		// Then page up.
		updated, _ = sendKey(t, updated, ctrlKey('b'))

		if updated.diff.viewport.YOffset() >= offsetAfterDown {
			t.Fatalf("expected YOffset to decrease after page up: before=%d, after=%d",
				offsetAfterDown, updated.diff.viewport.YOffset())
		}
	})

	t.Run("PageDownScrollsMoreThanHalfPage", func(t *testing.T) {
		halfPage, _ := sendKey(t, m, ctrlKey('d'))
		fullPage, _ := sendKey(t, m, ctrlKey('f'))

		if fullPage.diff.viewport.YOffset() <= halfPage.diff.viewport.YOffset() {
			t.Fatalf("expected full page scroll > half page scroll: full=%d, half=%d",
				fullPage.diff.viewport.YOffset(), halfPage.diff.viewport.YOffset())
		}
	})
}

func TestDiffViewGotoTopBottom(t *testing.T) {
	m := setupDiffViewModel()

	t.Run("GotoBottomThenTop", func(t *testing.T) {
		// Scroll down first to initialize viewport.
		updated, _ := sendKey(t, m, runeKey("j"))

		// Go to bottom.
		updated, _ = sendKey(t, updated, runeKey("G"))
		bottomOffset := updated.diff.viewport.YOffset()
		if bottomOffset <= 0 {
			t.Fatalf("expected YOffset > 0 at bottom, got %d", bottomOffset)
		}

		// Go to top.
		updated, _ = sendKey(t, updated, runeKey("g"))
		if updated.diff.viewport.YOffset() != 0 {
			t.Fatalf("expected YOffset = 0 after GotoTop, got %d", updated.diff.viewport.YOffset())
		}
	})

	t.Run("GotoTopFromTop", func(t *testing.T) {
		// Initialize viewport.
		updated, _ := sendKey(t, m, runeKey("j"))
		// Immediately go to top.
		updated, _ = sendKey(t, updated, runeKey("g"))

		if updated.diff.viewport.YOffset() != 0 {
			t.Fatalf("expected YOffset = 0 after GotoTop from near-top, got %d", updated.diff.viewport.YOffset())
		}
	})

	t.Run("GotoBottomAtMaxOffset", func(t *testing.T) {
		updated, _ := sendKey(t, m, runeKey("G"))
		bottomOffset := updated.diff.viewport.YOffset()

		// Going to bottom again should not change anything.
		updated, _ = sendKey(t, updated, runeKey("G"))
		if updated.diff.viewport.YOffset() != bottomOffset {
			t.Fatalf("expected same offset at bottom: first=%d, second=%d",
				bottomOffset, updated.diff.viewport.YOffset())
		}
	})
}

func TestDiffViewEscReturnsToStatus(t *testing.T) {
	m := setupDiffViewModel()

	// Scroll to initialize viewport and populate state.
	updated, _ := sendKey(t, m, runeKey("j"))

	// Press Esc.
	updated, _ = sendKey(t, updated, specialKey(tea.KeyEscape))

	if updated.view != StatusScreen {
		t.Fatalf("expected view = StatusScreen after Esc, got %d", updated.view)
	}
	if updated.diff.content != "" {
		t.Fatalf("expected diff.content to be empty after Esc, got %q", updated.diff.content)
	}
	if updated.diff.lines != nil {
		t.Fatalf("expected diff.lines to be nil after Esc, got %d lines", len(updated.diff.lines))
	}
	if updated.diff.viewportReady {
		t.Fatal("expected viewport to be reset (viewportReady=false) after Esc")
	}
}

func TestDiffViewEscClearsActionsMenu(t *testing.T) {
	m := setupDiffViewModel()
	m.actions.show = true

	// First Esc closes the actions menu but stays in DiffView.
	updated, _ := sendKey(t, m, specialKey(tea.KeyEscape))

	if updated.actions.show {
		t.Fatal("expected actions menu to be closed after first Esc")
	}
	if updated.view != DiffScreen {
		t.Fatalf("expected to remain in DiffScreen after closing actions menu, got %d", updated.view)
	}

	// Second Esc exits DiffScreen back to StatusScreen.
	updated, _ = sendKey(t, updated, specialKey(tea.KeyEscape))

	if updated.view != StatusScreen {
		t.Fatalf("expected StatusScreen after second Esc, got %d", updated.view)
	}
}

func TestDiffViewEmptyLinesNoScroll(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = DiffScreen
	m.width = 80
	m.height = 24
	m.diff.path = "/test/file"
	m.diff.lines = nil // empty
	m.diff.content = ""

	// Scrolling with empty lines should not panic.
	updated, _ := sendKey(t, m, runeKey("j"))

	// Viewport may or may not be initialized but should not panic.
	if updated.view != DiffScreen {
		t.Fatalf("expected to remain in DiffView, got %d", updated.view)
	}
}

func TestDiffViewHomeEndKeys(t *testing.T) {
	m := setupDiffViewModel()

	t.Run("HomeKey", func(t *testing.T) {
		// Scroll down first.
		updated, _ := sendKey(t, m, runeKey("j"))
		updated, _ = sendKey(t, updated, runeKey("j"))

		// Home key.
		updated, _ = sendKey(t, updated, tea.KeyPressMsg{Code: tea.KeyHome})
		if updated.diff.viewport.YOffset() != 0 {
			t.Fatalf("expected YOffset = 0 after Home key, got %d", updated.diff.viewport.YOffset())
		}
	})

	t.Run("EndKey", func(t *testing.T) {
		// Initialize viewport.
		updated, _ := sendKey(t, m, runeKey("j"))

		// End key.
		updated, _ = sendKey(t, updated, tea.KeyPressMsg{Code: tea.KeyEnd})
		if updated.diff.viewport.YOffset() <= 0 {
			t.Fatalf("expected YOffset > 0 after End key, got %d", updated.diff.viewport.YOffset())
		}
	})
}

// --- Info Tab Tests ---

func TestInfoTabScrollDown(t *testing.T) {
	m := setupInfoTabModel()

	// First j initializes the viewport AND scrolls down.
	updated, _ := sendKey(t, m, runeKey("j"))

	view := updated.info.views[0]
	if !view.viewportReady {
		t.Fatal("expected viewport to be ready after first scroll key")
	}
	if view.viewport.YOffset() <= 0 {
		t.Fatalf("expected YOffset > 0 after scroll down, got %d", view.viewport.YOffset())
	}
}

func TestInfoTabRepeatScrollDownAccelerates(t *testing.T) {
	m := setupInfoTabModel()

	single, _ := sendKey(t, m, runeKey("j"))
	repeat, _ := sendKey(t, m, tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true})

	if repeat.info.views[0].viewport.YOffset() <= single.info.views[0].viewport.YOffset() {
		t.Fatalf(
			"expected repeat scroll down to move farther than single key press: repeat=%d single=%d",
			repeat.info.views[0].viewport.YOffset(),
			single.info.views[0].viewport.YOffset(),
		)
	}
}

func TestInfoTabScrollUp(t *testing.T) {
	m := setupInfoTabModel()

	// Scroll down several times.
	updated, _ := sendKey(t, m, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))

	offsetAfterDown := updated.info.views[0].viewport.YOffset()
	if offsetAfterDown <= 0 {
		t.Fatalf("expected YOffset > 0 after multiple scroll down, got %d", offsetAfterDown)
	}

	// Scroll up.
	updated, _ = sendKey(t, updated, runeKey("k"))

	if updated.info.views[0].viewport.YOffset() >= offsetAfterDown {
		t.Fatalf("expected YOffset to decrease after scroll up: before=%d, after=%d",
			offsetAfterDown, updated.info.views[0].viewport.YOffset())
	}
}

func TestInfoTabGotoTop(t *testing.T) {
	m := setupInfoTabModel()

	// Scroll down first.
	updated, _ := sendKey(t, m, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))

	if updated.info.views[0].viewport.YOffset() <= 0 {
		t.Fatalf("expected YOffset > 0 after scrolling down, got %d", updated.info.views[0].viewport.YOffset())
	}

	// Go to top.
	updated, _ = sendKey(t, updated, runeKey("g"))

	if updated.info.views[0].viewport.YOffset() != 0 {
		t.Fatalf("expected YOffset = 0 after GotoTop, got %d", updated.info.views[0].viewport.YOffset())
	}
}

func TestInfoTabGotoBottom(t *testing.T) {
	m := setupInfoTabModel()

	// Initialize viewport.
	updated, _ := sendKey(t, m, runeKey("j"))

	// Go to bottom.
	updated, _ = sendKey(t, updated, runeKey("G"))

	if updated.info.views[0].viewport.YOffset() <= 0 {
		t.Fatalf("expected YOffset > 0 after GotoBottom, got %d", updated.info.views[0].viewport.YOffset())
	}
}

func TestInfoTabHalfPageScroll(t *testing.T) {
	m := setupInfoTabModel()

	t.Run("HalfPageDown", func(t *testing.T) {
		updated, _ := sendKey(t, m, ctrlKey('d'))

		if !updated.info.views[0].viewportReady {
			t.Fatal("expected viewport to be ready after half page down")
		}
		if updated.info.views[0].viewport.YOffset() <= 1 {
			t.Fatalf("expected YOffset > 1 after half page down, got %d", updated.info.views[0].viewport.YOffset())
		}
	})

	t.Run("HalfPageUp", func(t *testing.T) {
		// Scroll down.
		updated, _ := sendKey(t, m, ctrlKey('d'))
		offsetAfterDown := updated.info.views[0].viewport.YOffset()

		// Half page up.
		updated, _ = sendKey(t, updated, ctrlKey('u'))

		if updated.info.views[0].viewport.YOffset() >= offsetAfterDown {
			t.Fatalf("expected YOffset to decrease after half page up: before=%d, after=%d",
				offsetAfterDown, updated.info.views[0].viewport.YOffset())
		}
	})
}

func TestInfoTabPageScroll(t *testing.T) {
	m := setupInfoTabModel()

	t.Run("PageDown", func(t *testing.T) {
		updated, _ := sendKey(t, m, ctrlKey('f'))

		if !updated.info.views[0].viewportReady {
			t.Fatal("expected viewport to be ready after page down")
		}
		if updated.info.views[0].viewport.YOffset() <= 1 {
			t.Fatalf("expected YOffset > 1 after full page down, got %d", updated.info.views[0].viewport.YOffset())
		}
	})

	t.Run("PageUp", func(t *testing.T) {
		// Scroll down.
		updated, _ := sendKey(t, m, ctrlKey('f'))
		offsetAfterDown := updated.info.views[0].viewport.YOffset()

		// Page up.
		updated, _ = sendKey(t, updated, ctrlKey('b'))

		if updated.info.views[0].viewport.YOffset() >= offsetAfterDown {
			t.Fatalf("expected YOffset to decrease after page up: before=%d, after=%d",
				offsetAfterDown, updated.info.views[0].viewport.YOffset())
		}
	})
}

func TestInfoTabEmptyLinesNoScroll(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.activeTab = 2
	m.width = 80
	m.height = 24
	m.info.activeView = 0
	m.info.views[0].content = ""
	m.info.views[0].lines = nil
	m.info.views[0].loaded = true

	// Should not panic with empty lines.
	updated, _ := sendKey(t, m, runeKey("j"))

	if updated.view != StatusScreen {
		t.Fatalf("expected to remain in StatusScreen, got %d", updated.view)
	}
}

func TestInfoTabHomeEndKeys(t *testing.T) {
	m := setupInfoTabModel()

	t.Run("HomeKey", func(t *testing.T) {
		updated, _ := sendKey(t, m, runeKey("j"))
		updated, _ = sendKey(t, updated, runeKey("j"))

		updated, _ = sendKey(t, updated, tea.KeyPressMsg{Code: tea.KeyHome})
		if updated.info.views[0].viewport.YOffset() != 0 {
			t.Fatalf("expected YOffset = 0 after Home key, got %d", updated.info.views[0].viewport.YOffset())
		}
	})

	t.Run("EndKey", func(t *testing.T) {
		updated, _ := sendKey(t, m, runeKey("j"))

		updated, _ = sendKey(t, updated, tea.KeyPressMsg{Code: tea.KeyEnd})
		if updated.info.views[0].viewport.YOffset() <= 0 {
			t.Fatalf("expected YOffset > 0 after End key, got %d", updated.info.views[0].viewport.YOffset())
		}
	})
}

func TestInfoTabViewportPreservesSubView(t *testing.T) {
	m := setupInfoTabModel()

	// Scroll down in sub-view 0.
	updated, _ := sendKey(t, m, runeKey("j"))
	updated, _ = sendKey(t, updated, runeKey("j"))

	// Verify we are still on the correct sub-view.
	if updated.info.activeView != 0 {
		t.Fatalf("expected activeView = 0, got %d", updated.info.activeView)
	}
	if updated.info.views[0].viewport.YOffset() <= 0 {
		t.Fatalf("expected YOffset > 0 in sub-view 0, got %d", updated.info.views[0].viewport.YOffset())
	}
}
