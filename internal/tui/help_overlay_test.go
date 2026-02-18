package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpOverlayRows_ContextualByTab(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen

	m.activeTab = 1 // Files
	filesRows := m.helpOverlayRows()
	if !helpRowsContainTitle(filesRows, "Files") {
		t.Fatalf("expected Files help section in Files tab")
	}
	if !helpRowsContainTitle(filesRows, "Preview") {
		t.Fatalf("expected Preview help section in Files tab")
	}
	if helpRowsContainTitle(filesRows, "Changes") {
		t.Fatalf("did not expect Changes help section in Files tab")
	}

	m.activeTab = 3 // Commands
	commandRows := m.helpOverlayRows()
	if !helpRowsContainTitle(commandRows, "Commands") {
		t.Fatalf("expected Commands help section in Commands tab")
	}
	if helpRowsContainTitle(commandRows, "Preview") {
		t.Fatalf("did not expect Preview help section in Commands tab")
	}
}

func TestHelpOverlayRows_GlobalIncludesMouseModeToggle(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen

	rows := m.helpOverlayRows()
	found := false
	for _, row := range rows {
		for _, section := range row {
			if section.Title != "Global" {
				continue
			}
			for _, entry := range section.Entries {
				if entry.Key == "m" {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Fatal("expected Global help section to include m mouse/copy toggle")
	}
}

func TestBuildHelpOverlayResponsiveAndScrollable(t *testing.T) {
	entries := make([]HelpEntry, 30)
	for i := range entries {
		n := i + 1
		entries[i] = HelpEntry{
			Key:  fmt.Sprintf("%02d", n),
			Desc: fmt.Sprintf("item %02d", n),
		}
	}

	rows := [][]HelpSection{
		{
			{
				Title:   "Long Section",
				Entries: entries,
			},
		},
	}
	footer := "↑/↓ scroll | ?/esc close"

	maxScroll := helpOverlayMaxScroll(70, 12, footer, rows...)
	if maxScroll <= 0 {
		t.Fatalf("expected positive max scroll for constrained viewport")
	}

	top := buildHelpOverlay(70, 12, 0, footer, rows...)
	next := buildHelpOverlay(70, 12, 1, footer, rows...)
	bottom := buildHelpOverlay(70, 12, maxScroll, footer, rows...)

	assertRenderedLinesFitWidth(t, top, 70)
	assertRenderedLinesFitWidth(t, next, 70)
	assertRenderedLinesFitWidth(t, bottom, 70)

	if top == bottom {
		t.Fatalf("expected different overlay rendering when scrolled")
	}
	if top == next {
		t.Fatalf("expected one-line scroll step to change overlay output")
	}
	if !strings.Contains(top, "item 01") {
		t.Fatalf("expected first item near top of unscrolled overlay")
	}
	if strings.Contains(bottom, "item 01") {
		t.Fatalf("expected scrolled overlay to move past first item")
	}
}

func TestRenderManagedStatusBarPreviewHintsAreExplicit(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 80
	m.height = 40

	hidden := ansi.Strip(m.renderManagedStatusBar())
	if !strings.Contains(hidden, "p show preview") {
		t.Fatalf("expected explicit show-preview hint when panel hidden, got:\n%s", hidden)
	}

	m.width = 160
	m.panel.focusZone = panelFocusList
	listFocused := ansi.Strip(m.renderManagedStatusBar())
	if !strings.Contains(listFocused, "l/→ focus preview") {
		t.Fatalf("expected explicit focus-preview hint when list has focus, got:\n%s", listFocused)
	}
	if !strings.Contains(listFocused, "p hide preview") {
		t.Fatalf("expected explicit hide-preview hint when panel visible, got:\n%s", listFocused)
	}

	m.panel.focusZone = panelFocusPanel
	panelFocused := ansi.Strip(m.renderManagedStatusBar())
	if !strings.Contains(panelFocused, "h/← back to files") {
		t.Fatalf("expected explicit back-to-files hint when preview has focus, got:\n%s", panelFocused)
	}
}

func TestStyledHelpResponsiveWrapsAndFitsWidth(t *testing.T) {
	raw := "↑/↓ nav | enter open/toggle | a actions | t flat | f view/filter | r refresh | v switch diff/content | l/→ focus preview | p hide preview | ? keys | esc back"

	for _, width := range []int{120, 90, 70, 48, 34} {
		out := styledHelpResponsive(raw, width)
		lines := strings.Split(out, "\n")
		if len(lines) > 2 {
			t.Fatalf("expected at most 2 lines for width %d, got %d", width, len(lines))
		}
		for i, line := range lines {
			if got := len([]rune(strings.TrimSpace(line))); got == 0 {
				t.Fatalf("expected non-empty help line at width %d (line %d)", width, i+1)
			}
		}
		assertRenderedLinesFitWidth(t, out, width)
	}
}

func TestCompactHelpSegmentsRetainsCoreActions(t *testing.T) {
	segments := []string{
		"↑/↓ nav",
		"enter open/toggle",
		"a actions",
		"t flat",
		"f view/filter",
		"r refresh",
		"v switch diff/content",
		"l/→ focus preview",
		"p hide preview",
		"? keys",
		"esc back",
	}

	compact := compactHelpSegments(segments)

	assertSegmentWithKey := func(key string) {
		t.Helper()
		for _, seg := range compact {
			k, _ := splitKeyAction(seg)
			if k == key {
				return
			}
		}
		t.Fatalf("expected compact hints to include key %q; got %v", key, compact)
	}

	assertSegmentWithKey("enter")
	assertSegmentWithKey("a")
	assertSegmentWithKey("p")
	assertSegmentWithKey("?")
	assertSegmentWithKey("esc")
}

func helpRowsContainTitle(rows [][]HelpSection, title string) bool {
	for _, row := range rows {
		for _, section := range row {
			if section.Title == title {
				return true
			}
		}
	}
	return false
}
