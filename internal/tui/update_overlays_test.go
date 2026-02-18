package tui

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// ── Help Overlay Tests ──────────────────────────────────────────────

func TestHelpOverlayInterceptsKeys(t *testing.T) {
	t.Run("j scrolls help down", func(t *testing.T) {
		m := newTestModel()
		// Use a small height so that help content exceeds the viewport,
		// producing a maxScroll > 0 that allows downward scrolling.
		m.height = 15
		m.overlays.showHelp = true
		m.overlays.helpScroll = 0

		updated, cmd := sendKey(t, m, runeKey("j"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.helpScroll != 1 {
			t.Fatalf("expected helpScroll=1, got %d", updated.overlays.helpScroll)
		}
		if !updated.overlays.showHelp {
			t.Fatal("expected showHelp to remain true")
		}
	})

	t.Run("k scrolls help up", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.overlays.helpScroll = 5

		updated, cmd := sendKey(t, m, runeKey("k"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.helpScroll != 4 {
			t.Fatalf("expected helpScroll=4, got %d", updated.overlays.helpScroll)
		}
	})

	t.Run("k does not scroll below zero", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.overlays.helpScroll = 0

		updated, cmd := sendKey(t, m, runeKey("k"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.helpScroll != 0 {
			t.Fatalf("expected helpScroll=0, got %d", updated.overlays.helpScroll)
		}
	})

	t.Run("question mark closes help", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.overlays.helpScroll = 3

		updated, cmd := sendKey(t, m, runeKey("?"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.showHelp {
			t.Fatal("expected showHelp=false after pressing ?")
		}
		if updated.overlays.helpScroll != 0 {
			t.Fatalf("expected helpScroll=0 after close, got %d", updated.overlays.helpScroll)
		}
	})

	t.Run("esc closes help", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.overlays.helpScroll = 2

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEsc))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.showHelp {
			t.Fatal("expected showHelp=false after Esc")
		}
		if updated.overlays.helpScroll != 0 {
			t.Fatalf("expected helpScroll=0 after close, got %d", updated.overlays.helpScroll)
		}
	})

	t.Run("q closes help", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.overlays.helpScroll = 1

		updated, cmd := sendKey(t, m, runeKey("q"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.showHelp {
			t.Fatal("expected showHelp=false after q")
		}
		if updated.overlays.helpScroll != 0 {
			t.Fatalf("expected helpScroll=0 after close, got %d", updated.overlays.helpScroll)
		}
	})

	t.Run("tab does not switch tabs when help is open", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.activeTab = 0

		updated, cmd := sendKey(t, m, specialKey(tea.KeyTab))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.activeTab != 0 {
			t.Fatalf("expected activeTab=0, got %d (tab switch should be blocked)", updated.activeTab)
		}
		if !updated.overlays.showHelp {
			t.Fatal("expected showHelp to remain true")
		}
	})

	t.Run("a does not open actions menu when help is open", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true

		updated, cmd := sendKey(t, m, runeKey("a"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.actions.show {
			t.Fatal("expected actions.show=false (actions menu should not open through help)")
		}
	})

	t.Run("number 1 does not switch tabs when help is open", func(t *testing.T) {
		m := newTestModel()
		m.overlays.showHelp = true
		m.activeTab = 2

		updated, cmd := sendKey(t, m, runeKey("1"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.activeTab != 2 {
			t.Fatalf("expected activeTab=2, got %d (number key tab switch blocked by help)", updated.activeTab)
		}
	})
}

// ── Actions Menu Tests ──────────────────────────────────────────────

func setupActionsMenuModel() Model {
	m := newTestModel()
	m.status.files = []chezmoi.FileStatus{{Path: "/tmp/test", SourceStatus: 'M', DestStatus: ' '}}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 2 // first drift file row (after Incoming header + Drift header)
	m.openStatusActionsMenu()
	return m
}

func TestActionsMenuInterceptsKeys(t *testing.T) {
	t.Run("j moves cursor down in actions menu", func(t *testing.T) {
		m := setupActionsMenuModel()
		startCursor := m.actions.cursor

		updated, cmd := sendKey(t, m, runeKey("j"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if !updated.actions.show {
			t.Fatal("expected actions.show to remain true")
		}
		// cursor should have moved (or stayed if at boundary)
		nextExpected := nextSelectableCursor(m.actions.items, startCursor, 1)
		if updated.actions.cursor != nextExpected {
			t.Fatalf("expected cursor=%d, got %d", nextExpected, updated.actions.cursor)
		}
	})

	t.Run("k moves cursor up in actions menu", func(t *testing.T) {
		m := setupActionsMenuModel()
		// Move cursor down first so we can move up
		if len(m.actions.items) > 1 {
			m.actions.cursor = nextSelectableCursor(m.actions.items, m.actions.cursor, 1)
		}
		startCursor := m.actions.cursor

		updated, cmd := sendKey(t, m, runeKey("k"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if !updated.actions.show {
			t.Fatal("expected actions.show to remain true")
		}
		nextExpected := nextSelectableCursor(m.actions.items, startCursor, -1)
		if updated.actions.cursor != nextExpected {
			t.Fatalf("expected cursor=%d, got %d", nextExpected, updated.actions.cursor)
		}
	})

	t.Run("esc closes actions menu", func(t *testing.T) {
		m := setupActionsMenuModel()

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEsc))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.actions.show {
			t.Fatal("expected actions.show=false after Esc")
		}
	})

	t.Run("q closes actions menu", func(t *testing.T) {
		m := setupActionsMenuModel()

		updated, cmd := sendKey(t, m, runeKey("q"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.actions.show {
			t.Fatal("expected actions.show=false after q")
		}
	})

	t.Run("question mark does not open help when actions menu is open", func(t *testing.T) {
		m := setupActionsMenuModel()

		updated, cmd := sendKey(t, m, runeKey("?"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.showHelp {
			t.Fatal("expected showHelp=false (help should not open through actions menu)")
		}
		if !updated.actions.show {
			t.Fatal("expected actions.show to remain true (? is unhandled in menu, falls through to return m, nil)")
		}
	})
}

// ── Filter Input Tests ──────────────────────────────────────────────

func TestFilterInputInterceptsKeys(t *testing.T) {
	t.Run("slash on files/all safely reinitializes zero-value filter input", func(t *testing.T) {
		m := newTestModel(WithTab(1))
		m.filesTab.viewMode = managedViewAll
		m.filterInput = textinput.Model{} // simulate old/corrupt zero-value state

		updated, cmd := sendKey(t, m, runeKey("/"))

		if cmd == nil {
			t.Fatalf("expected non-nil cmd when entering filter mode")
		}
		if !updated.filterInput.Focused() {
			t.Fatal("expected filter to be focused after /")
		}
		if updated.filterInput.Value() != "" {
			t.Fatalf("expected empty filter value after /, got %q", updated.filterInput.Value())
		}
	})

	t.Run("esc with empty value blurs filter", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("")

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEsc))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.filterInput.Focused() {
			t.Fatal("expected filter to be blurred after Esc with empty value")
		}
	})

	t.Run("esc with non-empty value clears value first", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("hello")

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEsc))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.filterInput.Value() != "" {
			t.Fatalf("expected filter value to be cleared, got %q", updated.filterInput.Value())
		}
		if !updated.filterInput.Focused() {
			t.Fatal("expected filter to remain focused after first Esc (value cleared)")
		}
	})

	t.Run("second esc after clearing blurs filter", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("hello")

		// First Esc clears value
		m, _ = sendKey(t, m, specialKey(tea.KeyEsc))
		// Second Esc blurs
		updated, cmd := sendKey(t, m, specialKey(tea.KeyEsc))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.filterInput.Focused() {
			t.Fatal("expected filter to be blurred after second Esc")
		}
	})

	t.Run("enter blurs filter", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("test")

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEnter))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.filterInput.Focused() {
			t.Fatal("expected filter to be blurred after Enter")
		}
	})

	t.Run("enter on files deep-search views pauses active search", func(t *testing.T) {
		m := newTestModel(WithTab(1)) // Files
		m.filesTab.viewMode = managedViewUnmanaged
		m.filterInput.Focus()
		m.filterInput.SetValue("token")
		m.filesTab.search.searching = true
		canceled := false
		m.filesTab.search.cancel = func() {
			canceled = true
		}

		updated, cmd := sendKey(t, m, specialKey(tea.KeyEnter))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.filterInput.Focused() {
			t.Fatal("expected filter to be blurred after Enter")
		}
		if !updated.filesTab.search.paused {
			t.Fatal("expected search to be paused after blurring filter")
		}
		if updated.filesTab.search.searching {
			t.Fatal("expected searching=false after pausing")
		}
		if !canceled {
			t.Fatal("expected in-flight search to be canceled")
		}
		if updated.filesTab.search.query != "token" {
			t.Fatalf("expected paused query token, got %q", updated.filesTab.search.query)
		}
	})

	t.Run("q is typed into filter not treated as quit", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("")

		updated, cmd := sendKey(t, m, runeKey("q"))

		if isQuitCmd(cmd) {
			t.Fatal("expected non-quit cmd, but got tea.Quit")
		}
		if !updated.filterInput.Focused() {
			t.Fatal("expected filter to remain focused")
		}
	})

	t.Run("question mark is typed into filter not treated as help", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("")

		updated, cmd := sendKey(t, m, runeKey("?"))

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.overlays.showHelp {
			t.Fatal("expected showHelp=false (? should be consumed by filter input)")
		}
		if !updated.filterInput.Focused() {
			t.Fatal("expected filter to remain focused")
		}
	})
}

// ── Global Help Key Tests ───────────────────────────────────────────

func TestGlobalHelpKeyOpensOverlay(t *testing.T) {
	m := newTestModel()
	m.overlays.showHelp = false

	updated, cmd := sendKey(t, m, runeKey("?"))

	if cmd != nil {
		t.Fatalf("expected nil cmd, got non-nil")
	}
	if !updated.overlays.showHelp {
		t.Fatal("expected showHelp=true after pressing ?")
	}
	if updated.overlays.helpScroll != 0 {
		t.Fatalf("expected helpScroll=0, got %d", updated.overlays.helpScroll)
	}
}

func TestGlobalMouseToggleKey(t *testing.T) {
	t.Run("m toggles mouse capture when filter is not focused", func(t *testing.T) {
		m := newTestModel()
		if !m.ui.mouseCapture {
			t.Fatal("precondition: expected mouseCapture=true by default")
		}

		updated, cmd := sendKey(t, m, runeKey("m"))
		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.ui.mouseCapture {
			t.Fatal("expected mouseCapture=false after pressing m")
		}
		if updated.ui.message == "" {
			t.Fatal("expected user-visible toggle message")
		}
	})

	t.Run("m is typed into focused filter and does not toggle mouse capture", func(t *testing.T) {
		m := newTestModel()
		m.filterInput.Focus()
		m.filterInput.SetValue("")

		updated, cmd := sendKey(t, m, runeKey("m"))
		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if !updated.ui.mouseCapture {
			t.Fatal("expected mouseCapture to remain true while typing in filter")
		}
		if updated.filterInput.Value() != "m" {
			t.Fatalf("expected filter input to capture typed m, got %q", updated.filterInput.Value())
		}
	})
}

// ── Global Quit Key Tests ───────────────────────────────────────────

func TestGlobalQuitKey(t *testing.T) {
	t.Run("q returns quit cmd", func(t *testing.T) {
		m := newTestModel()

		_, cmd := sendKey(t, m, runeKey("q"))

		if !isQuitCmd(cmd) {
			t.Fatal("expected tea.Quit cmd from q key")
		}
	})

	t.Run("ctrl+c returns quit cmd", func(t *testing.T) {
		m := newTestModel()

		_, cmd := sendKey(t, m, ctrlKey('c'))

		if !isQuitCmd(cmd) {
			t.Fatal("expected tea.Quit cmd from ctrl+c")
		}
	})
}

// ── Tab Switching Key Tests ─────────────────────────────────────────

func TestTabSwitchingKeys(t *testing.T) {
	t.Run("tab key advances to next tab", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0

		updated, _ := sendKey(t, m, specialKey(tea.KeyTab))

		if updated.activeTab != 1 {
			t.Fatalf("expected activeTab=1, got %d", updated.activeTab)
		}
	})

	t.Run("shift+tab goes to previous tab with wraparound", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0

		updated, _ := sendKey(t, m, shiftKey(tea.KeyTab))

		expected := len(m.tabNames) - 1
		if updated.activeTab != expected {
			t.Fatalf("expected activeTab=%d (wrap around), got %d", expected, updated.activeTab)
		}
	})

	t.Run("number 1 switches to tab 0", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 2

		updated, _ := sendKey(t, m, runeKey("1"))

		if updated.activeTab != 0 {
			t.Fatalf("expected activeTab=0, got %d", updated.activeTab)
		}
	})

	t.Run("number 2 switches to tab 1", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0

		updated, _ := sendKey(t, m, runeKey("2"))

		if updated.activeTab != 1 {
			t.Fatalf("expected activeTab=1, got %d", updated.activeTab)
		}
	})

	t.Run("number 3 switches to tab 2", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0

		updated, _ := sendKey(t, m, runeKey("3"))

		if updated.activeTab != 2 {
			t.Fatalf("expected activeTab=2, got %d", updated.activeTab)
		}
	})

	t.Run("number 4 switches to tab 3", func(t *testing.T) {
		m := newTestModel()
		m.activeTab = 0

		updated, _ := sendKey(t, m, runeKey("4"))

		if updated.activeTab != 3 {
			t.Fatalf("expected activeTab=3, got %d", updated.activeTab)
		}
	})
}

// ── Tab Switching Blocked by ViewPicker ─────────────────────────────

func TestTabSwitchingBlockedByViewPicker(t *testing.T) {
	m := newTestModel()
	m.activeTab = 1 // Files tab so viewPicker is relevant
	m.overlays.showViewPicker = true

	updated, _ := sendKey(t, m, specialKey(tea.KeyTab))

	if updated.activeTab != 1 {
		t.Fatalf("expected activeTab=1 (blocked by viewPicker), got %d", updated.activeTab)
	}
}

// ── Tab Switching Blocked by FilterOverlay ──────────────────────────

func TestTabSwitchingBlockedByFilterOverlay(t *testing.T) {
	m := newTestModel()
	m.activeTab = 1 // Files tab so filterOverlay is relevant
	m.overlays.showFilterOverlay = true

	updated, _ := sendKey(t, m, specialKey(tea.KeyTab))

	if updated.activeTab != 1 {
		t.Fatalf("expected activeTab=1 (blocked by filterOverlay), got %d", updated.activeTab)
	}
}

// ── Tab Switching Blocked in DiffScreen ───────────────────────────────

func TestTabSwitchingBlockedInDiffView(t *testing.T) {
	m := newTestModel()
	m.view = DiffScreen
	m.activeTab = 0

	updated, _ := sendKey(t, m, specialKey(tea.KeyTab))

	if updated.activeTab != 0 {
		t.Fatalf("expected activeTab=0 (DiffScreen handled before tab switching), got %d", updated.activeTab)
	}
}
