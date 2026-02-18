package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// newStatusModel creates a Model pre-configured for Status tab testing.
// It populates two drift files and builds changesRows. The returned model
// has width=120 (panel auto-visible), height=40, and the cursor positioned
// at the first non-header (file) row.
func newStatusModel(t *testing.T) Model {
	t.Helper()
	m := NewModel(Options{Service: testService()})
	m.view = StatusScreen
	m.activeTab = 0
	m.width = 120
	m.height = 40

	m.status.files = []chezmoi.FileStatus{
		{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		{Path: "/home/test/.zshrc", SourceStatus: 'A', DestStatus: ' '},
	}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()

	firstFileRow := findFirstFileRow(t, m)
	m.status.changesCursor = firstFileRow
	return m
}

// --- Test: Cursor Down ---

func TestStatusCursorDown(t *testing.T) {
	m := newStatusModel(t)
	firstFile := findFirstFileRow(t, m)
	secondFile := findSecondFileRow(t, m)
	m.status.changesCursor = firstFile

	updatedAny, _ := m.Update(runeKey("j"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.status.changesCursor != secondFile {
		t.Fatalf("expected cursor at %d after j, got %d", secondFile, updated.status.changesCursor)
	}
}

func TestStatusCursorDownRepeatAccelerates(t *testing.T) {
	files := make([]chezmoi.FileStatus, 8)
	for i := range files {
		files[i] = chezmoi.FileStatus{
			Path:         fmt.Sprintf("/home/test/.config/tool/file-%d.toml", i),
			SourceStatus: 'M',
			DestStatus:   ' ',
		}
	}
	m := newTestModel(WithDriftFiles(files))
	m.view = StatusScreen
	m.activeTab = 0
	m.width = 120
	m.height = 40

	start := findFirstFileRow(t, m)
	m.status.changesCursor = start

	repeatDown := tea.KeyPressMsg{Code: 'j', Text: "j", IsRepeat: true}
	updatedAny, _ := m.Update(repeatDown)
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	want := min(start+3, len(updated.status.changesRows)-1)
	if updated.status.changesCursor != want {
		t.Fatalf("expected repeat down to jump to %d, got %d", want, updated.status.changesCursor)
	}
}

func TestStatusShiftDownActivatesRangeSelection(t *testing.T) {
	m := newStatusModel(t)
	start := findFirstFileRow(t, m)
	m.status.changesCursor = start

	updatedAny, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.status.selectionActive {
		t.Fatal("expected selectionActive=true after shift+down")
	}
	if updated.status.selectionAnchor != start {
		t.Fatalf("expected selectionAnchor=%d, got %d", start, updated.status.selectionAnchor)
	}
	if updated.status.changesCursor <= start {
		t.Fatalf("expected cursor to advance after shift+down, got %d", updated.status.changesCursor)
	}
}

func TestStatusShiftRangeSelectionStaysInAnchorSection(t *testing.T) {
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			{Path: "/home/test/.zshrc", SourceStatus: 'M', DestStatus: ' '},
		}),
	)
	m.view = StatusScreen
	m.activeTab = 0
	m.width = 120
	m.height = 40
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
	}
	m.buildChangesRows()

	start := findFirstSectionFileRow(t, m, changesSectionDrift)
	m.status.changesCursor = start

	updatedAny, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift, IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.status.selectionActive {
		t.Fatal("expected selectionActive=true after shift range selection")
	}
	if updated.status.changesRows[updated.status.changesCursor].section != changesSectionDrift {
		t.Fatalf("expected cursor to remain in drift section, got section=%d", updated.status.changesRows[updated.status.changesCursor].section)
	}

	lastDrift := start
	for i, row := range updated.status.changesRows {
		if row.section == changesSectionDrift {
			lastDrift = i
		}
	}
	if updated.status.changesCursor != lastDrift {
		t.Fatalf("expected cursor to clamp at last drift row %d, got %d", lastDrift, updated.status.changesCursor)
	}
}

// --- Test: Cursor Up ---

func TestStatusCursorUp(t *testing.T) {
	m := newStatusModel(t)
	firstFile := findFirstFileRow(t, m)
	secondFile := findSecondFileRow(t, m)
	m.status.changesCursor = secondFile

	updatedAny, _ := m.Update(runeKey("k"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.status.changesCursor != firstFile {
		t.Fatalf("expected cursor at %d after k, got %d", firstFile, updated.status.changesCursor)
	}
}

func TestStatusPlainDownClearsRangeSelection(t *testing.T) {
	m := newStatusModel(t)
	start := findFirstFileRow(t, m)
	m.status.changesCursor = start
	m.status.selectionActive = true
	m.status.selectionAnchor = start

	updatedAny, _ := m.Update(runeKey("j"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.status.selectionActive {
		t.Fatal("expected plain cursor movement to clear active selection")
	}
}

// --- Test: Cursor Down with Panel Visible ---

func TestStatusCursorDownWithPanelVisible(t *testing.T) {
	m := newStatusModel(t)
	// Width=120 is above panelAutoThreshold (90), so panel should be visible.
	m.width = 120
	firstFile := findFirstFileRow(t, m)
	m.status.changesCursor = firstFile

	if !m.panel.shouldShow(m.width) {
		t.Fatal("expected panel to be auto-visible at width=120")
	}

	_, cmd := m.Update(runeKey("j"))

	if cmd == nil {
		t.Fatal("expected non-nil cmd (panel reload) when panel is visible and cursor moves")
	}
}

// --- Test: Cursor Move with Panel Hidden ---

func TestStatusCursorMoveWithPanelHidden(t *testing.T) {
	m := newStatusModel(t)
	m.width = 80 // Below panelAutoThreshold (90), panel hidden.
	firstFile := findFirstFileRow(t, m)
	m.status.changesCursor = firstFile

	if m.panel.shouldShow(m.width) {
		t.Fatal("expected panel to be hidden at width=80")
	}

	_, cmd := m.Update(runeKey("j"))

	if cmd != nil {
		t.Fatal("expected nil cmd (no panel reload) when panel is hidden")
	}
}

// --- Test: Panel Toggle on Status Tab ---

func TestPanelToggleOnStatusTab(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120

	if !m.panel.shouldShow(m.width) {
		t.Fatal("expected panel visible at width=120 before toggle")
	}

	// First toggle: at wide width, first toggle hides the panel.
	// toggle() behavior: when not manualOverride, sets manualOverride=true,
	// visible = (termWidth < panelAutoThreshold). At 120 >= 90, so visible=false.
	updatedAny, _ := m.Update(runeKey("p"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.shouldShow(updated.width) {
		t.Fatal("expected panel hidden after first toggle at wide width")
	}
	if !updated.panel.manualOverride {
		t.Fatal("expected manualOverride=true after toggle")
	}

	// Second toggle: should show again (manualOverride is true, visible flips to true).
	updatedAny2, _ := updated.Update(runeKey("p"))
	updated2, ok2 := updatedAny2.(Model)
	if !ok2 {
		t.Fatal("expected Model type assertion")
	}

	if !updated2.panel.shouldShow(updated2.width) {
		t.Fatal("expected panel visible after second toggle")
	}
}

// --- Test: Panel Toggle Resets Focus When Hidden ---

func TestPanelToggleResetsFocusWhenHidden(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.manualOverride = true
	m.panel.visible = true
	m.panel.focusZone = panelFocusPanel

	if m.panel.focusZone != panelFocusPanel {
		t.Fatal("precondition: expected focus on panel")
	}

	// Toggle panel off.
	updatedAny, _ := m.Update(runeKey("p"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.shouldShow(updated.width) {
		t.Fatal("expected panel to be hidden after toggle")
	}
	if updated.panel.focusZone != panelFocusList {
		t.Fatalf("expected focusZone reset to panelFocusList (0), got %d", updated.panel.focusZone)
	}
}

// --- Test: Panel Focus via Right Arrow ---

func TestPanelFocusViaRightArrow(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.focusZone = panelFocusList

	if !m.panel.shouldShow(m.width) {
		t.Fatal("precondition: expected panel visible at width=120")
	}

	updatedAny, _ := m.Update(specialKey(tea.KeyRight))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.focusZone != panelFocusPanel {
		t.Fatalf("expected focusZone=panelFocusPanel (1) after Right arrow, got %d", updated.panel.focusZone)
	}
}

// --- Test: Panel Focus Return via Left Arrow ---

func TestPanelFocusReturnViaLeftArrow(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.focusZone = panelFocusPanel

	if !m.panel.shouldShow(m.width) {
		t.Fatal("precondition: expected panel visible at width=120")
	}

	updatedAny, _ := m.Update(specialKey(tea.KeyLeft))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.focusZone != panelFocusList {
		t.Fatalf("expected focusZone=panelFocusList (0) after Left arrow, got %d", updated.panel.focusZone)
	}
}

// --- Test: Panel Content Mode Cycling from List ---

func TestPanelContentModeCyclingFromList(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.focusZone = panelFocusList
	m.panel.contentMode = panelModeDiff

	if !m.panel.shouldShow(m.width) {
		t.Fatal("precondition: expected panel visible")
	}

	// First v: diff -> content
	updatedAny, _ := m.Update(runeKey("v"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.contentMode != panelModeContent {
		t.Fatalf("expected contentMode=panelModeContent (1) after first v, got %d", updated.panel.contentMode)
	}

	// Second v: content -> diff (toggles back)
	updatedAny2, _ := updated.Update(runeKey("v"))
	updated2, ok2 := updatedAny2.(Model)
	if !ok2 {
		t.Fatal("expected Model type assertion")
	}

	if updated2.panel.contentMode != panelModeDiff {
		t.Fatalf("expected contentMode=panelModeDiff (0) after second v, got %d", updated2.panel.contentMode)
	}
}

// --- Test: Actions Menu Opens on A Key ---

func TestActionsMenuOpensOnAKey(t *testing.T) {
	m := newStatusModel(t)
	firstFile := findFirstFileRow(t, m)
	m.status.changesCursor = firstFile

	// Verify the cursor is on a drift file row.
	row := m.currentChangesRow()
	if row.isHeader {
		t.Fatal("precondition: expected cursor on a file row, not a header")
	}
	if row.section != changesSectionDrift {
		t.Fatalf("precondition: expected drift section, got %d", row.section)
	}

	updatedAny, _ := m.Update(runeKey("a"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.actions.show {
		t.Fatal("expected actions.show=true after pressing 'a'")
	}
	if len(updated.actions.items) == 0 {
		t.Fatal("expected actions.items to be non-empty")
	}
}

func TestActionsMenuOnSelectedDriftShowsBulkReAdd(t *testing.T) {
	m := newStatusModel(t)
	start := findFirstFileRow(t, m)
	end := findSecondFileRow(t, m)
	m.status.selectionActive = true
	m.status.selectionAnchor = start
	m.status.changesCursor = end

	updatedAny, _ := m.Update(runeKey("a"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.actions.show {
		t.Fatal("expected actions.show=true for selected drift range")
	}
	if len(updated.actions.items) != 1 {
		t.Fatalf("expected 1 bulk action item, got %d", len(updated.actions.items))
	}
	item := updated.actions.items[0]
	if item.label != "Re-add selected" {
		t.Fatalf("expected label %q, got %q", "Re-add selected", item.label)
	}
	if item.action != chezmoiActionReAdd {
		t.Fatalf("expected action=%v, got %v", chezmoiActionReAdd, item.action)
	}
}

func TestActionsMenuOnSelectedUnstagedShowsBulkStage(t *testing.T) {
	m := newStatusModel(t)
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
		{Path: "/home/test/.profile", StatusCode: "M"},
	}
	m.buildChangesRows()
	start := findFirstSectionFileRow(t, m, changesSectionUnstaged)
	m.status.selectionActive = true
	m.status.selectionAnchor = start
	m.status.changesCursor = min(start+1, len(m.status.changesRows)-1)

	updatedAny, _ := m.Update(runeKey("a"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.actions.show {
		t.Fatal("expected actions.show=true for selected unstaged range")
	}
	if len(updated.actions.items) != 2 {
		t.Fatalf("expected 2 bulk action items, got %d", len(updated.actions.items))
	}
	item := updated.actions.items[0]
	if item.label != "Stage selected" {
		t.Fatalf("expected label %q, got %q", "Stage selected", item.label)
	}
	if item.action != chezmoiActionGitStage {
		t.Fatalf("expected action=%v, got %v", chezmoiActionGitStage, item.action)
	}
	discardItem := updated.actions.items[1]
	if discardItem.label != "Discard selected" {
		t.Fatalf("expected label %q, got %q", "Discard selected", discardItem.label)
	}
	if discardItem.action != chezmoiActionGitDiscardSelected {
		t.Fatalf("expected action=%v, got %v", chezmoiActionGitDiscardSelected, discardItem.action)
	}
}

func TestActionsMenuOnSelectedStagedShowsBulkUnstage(t *testing.T) {
	m := newStatusModel(t)
	m.status.gitStagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
		{Path: "/home/test/.profile", StatusCode: "M"},
	}
	m.buildChangesRows()
	start := findFirstSectionFileRow(t, m, changesSectionStaged)
	m.status.selectionActive = true
	m.status.selectionAnchor = start
	m.status.changesCursor = min(start+1, len(m.status.changesRows)-1)

	updatedAny, _ := m.Update(runeKey("a"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.actions.show {
		t.Fatal("expected actions.show=true for selected staged range")
	}
	if len(updated.actions.items) != 1 {
		t.Fatalf("expected 1 bulk action item, got %d", len(updated.actions.items))
	}
	item := updated.actions.items[0]
	if item.label != "Unstage selected" {
		t.Fatalf("expected label %q, got %q", "Unstage selected", item.label)
	}
	if item.action != chezmoiActionGitUnstage {
		t.Fatalf("expected action=%v, got %v", chezmoiActionGitUnstage, item.action)
	}
}

// --- Test: Enter on Section Header Toggles Collapse ---

func TestEnterOnSectionHeaderTogglesCollapse(t *testing.T) {
	m := newStatusModel(t)
	headerRow := findFirstHeaderRow(t, m)
	m.status.changesCursor = headerRow

	section := m.status.changesRows[headerRow].section
	collapsedBefore := m.status.sectionCollapsed[section]

	updatedAny, _ := m.Update(specialKey(tea.KeyEnter))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	collapsedAfter := updated.status.sectionCollapsed[section]
	if collapsedAfter == collapsedBefore {
		t.Fatalf("expected sectionCollapsed[%d] to toggle from %v to %v", section, collapsedBefore, !collapsedBefore)
	}

	// Toggle back.
	updatedAny2, _ := updated.Update(specialKey(tea.KeyEnter))
	updated2, ok2 := updatedAny2.(Model)
	if !ok2 {
		t.Fatal("expected Model type assertion")
	}

	collapsedReset := updated2.status.sectionCollapsed[section]
	if collapsedReset != collapsedBefore {
		t.Fatalf("expected sectionCollapsed[%d] to return to %v after second toggle, got %v", section, collapsedBefore, collapsedReset)
	}
}

// --- Test: Enter on File Row Loads Diff ---

func TestEnterOnFileRowLoadsDiff(t *testing.T) {
	m := newStatusModel(t)
	firstFile := findFirstFileRow(t, m)
	m.status.changesCursor = firstFile

	row := m.currentChangesRow()
	if row.isHeader {
		t.Fatal("precondition: expected cursor on a file row")
	}
	if row.section != changesSectionDrift {
		t.Fatalf("precondition: expected drift section, got %d", row.section)
	}

	_, cmd := m.Update(specialKey(tea.KeyEnter))

	if cmd == nil {
		t.Fatal("expected non-nil cmd (diff load) when pressing Enter on a drift file row")
	}
}

// --- Test: Esc on Status Tab ---

func TestEscOnStatusTabWithEscQuit(t *testing.T) {
	// EscQuit (default, = 0): pressing Esc on Status tab should go to LandingScreen.
	m := newStatusModel(t)
	m.opts.EscBehavior = EscQuit

	updatedAny, cmd := m.Update(specialKey(tea.KeyEscape))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	// With EscQuit, escCmd() sets view=LandingScreen and returns nil cmd.
	if updated.view != LandingScreen {
		t.Fatalf("expected view=LandingScreen after Esc with EscQuit, got %d", updated.view)
	}
	if cmd != nil {
		t.Fatal("expected nil cmd from Esc with EscQuit (goes to landing, no quit)")
	}
}

func TestEscOnStatusTabWithEscBack(t *testing.T) {
	// EscBack (= 1): pressing Esc should produce an ExitMsg command.
	m := newStatusModel(t)
	m.opts.EscBehavior = EscBack

	_, cmd := m.Update(specialKey(tea.KeyEscape))

	if cmd == nil {
		t.Fatal("expected non-nil cmd from Esc with EscBack")
	}
	msg := cmd()
	if _, ok := msg.(ExitMsg); !ok {
		t.Fatalf("expected ExitMsg from Esc with EscBack, got %T", msg)
	}
}

// --- Test: Cursor Down at Bottom Boundary ---

func TestStatusCursorDownAtBottomDoesNotOverflow(t *testing.T) {
	m := newStatusModel(t)
	lastIdx := len(m.status.changesRows) - 1
	m.status.changesCursor = lastIdx

	updatedAny, _ := m.Update(runeKey("j"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.status.changesCursor != lastIdx {
		t.Fatalf("expected cursor to stay at %d at bottom boundary, got %d", lastIdx, updated.status.changesCursor)
	}
}

// --- Test: Cursor Up at Top Boundary ---

func TestStatusCursorUpAtTopDoesNotUnderflow(t *testing.T) {
	m := newStatusModel(t)
	m.status.changesCursor = 0

	updatedAny, _ := m.Update(runeKey("k"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.status.changesCursor != 0 {
		t.Fatalf("expected cursor to stay at 0 at top boundary, got %d", updated.status.changesCursor)
	}
}

// --- Test: Panel Not Visible at Narrow Width ---

func TestPanelNotVisibleAtNarrowWidth(t *testing.T) {
	m := newStatusModel(t)
	m.width = 80

	if m.panel.shouldShow(m.width) {
		t.Fatal("expected panel to be hidden at width=80")
	}
}

// --- Test: Panel Visible at Wide Width ---

func TestPanelVisibleAtWideWidth(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120

	if !m.panel.shouldShow(m.width) {
		t.Fatal("expected panel to be visible at width=120")
	}
}

// --- Test: Right Arrow No-op When Panel Hidden ---

func TestRightArrowNoOpWhenPanelHidden(t *testing.T) {
	m := newStatusModel(t)
	m.width = 80
	m.panel.focusZone = panelFocusList

	updatedAny, _ := m.Update(specialKey(tea.KeyRight))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	// When panel is hidden, Right arrow should not change focus zone.
	if updated.panel.focusZone != panelFocusList {
		t.Fatalf("expected focusZone to remain panelFocusList when panel hidden, got %d", updated.panel.focusZone)
	}
}

// --- Test: V Key No-op When Panel Hidden ---

func TestVKeyNoOpWhenPanelHidden(t *testing.T) {
	m := newStatusModel(t)
	m.width = 80
	m.panel.contentMode = panelModeDiff

	updatedAny, _ := m.Update(runeKey("v"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	// When panel is hidden, 'v' should not cycle content mode.
	if updated.panel.contentMode != panelModeDiff {
		t.Fatalf("expected contentMode unchanged when panel hidden, got %d", updated.panel.contentMode)
	}
}

// --- Test: Refresh Key Triggers Reload ---

func TestRefreshKeyTriggersReload(t *testing.T) {
	m := newStatusModel(t)
	m.ui.loading = false
	m.status.loadingGit = false

	updatedAny, cmd := m.Update(runeKey("r"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.ui.loading {
		t.Fatal("expected ui.loading=true after refresh")
	}
	if !updated.status.loadingGit {
		t.Fatal("expected loadingGit=true after refresh")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from refresh")
	}
}

func TestStatusStageRepeatIgnored(t *testing.T) {
	m := newStatusModel(t)
	m.status.changesCursor = findFirstFileRow(t, m)

	updatedAny, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s", IsRepeat: true})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if cmd != nil {
		t.Fatal("expected repeat stage key to be ignored (nil cmd)")
	}
	if updated.ui.busyAction {
		t.Fatal("expected busyAction to remain false when repeat stage is ignored")
	}
}

func TestStatusStageSelectedRangeRunsBatchStage(t *testing.T) {
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			{Path: "/home/test/.zshrc", SourceStatus: 'M', DestStatus: ' '},
		}),
	)
	m.view = StatusScreen
	m.activeTab = 0
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
	}
	m.buildChangesRows()

	m.status.selectionActive = true
	m.status.selectionAnchor = findFirstSectionFileRow(t, m, changesSectionDrift)
	m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionUnstaged)

	updatedAny, cmd := m.Update(runeKey("s"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd for staging selected range")
	}
	if !updated.ui.busyAction {
		t.Fatal("expected busyAction=true for staging selected range")
	}
	if updated.status.selectionActive {
		t.Fatal("expected selection to clear after staging selected range")
	}
}

func TestStatusUnstageSelectedRangeRunsBatchUnstage(t *testing.T) {
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
		}),
	)
	m.view = StatusScreen
	m.activeTab = 0
	m.status.gitStagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
		{Path: "/home/test/.profile", StatusCode: "M"},
	}
	m.buildChangesRows()

	start := findFirstSectionFileRow(t, m, changesSectionStaged)
	m.status.selectionActive = true
	m.status.selectionAnchor = start
	m.status.changesCursor = min(start+1, len(m.status.changesRows)-1)

	updatedAny, cmd := m.Update(runeKey("u"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd for unstaging selected range")
	}
	if !updated.ui.busyAction {
		t.Fatal("expected busyAction=true for unstaging selected range")
	}
	if updated.status.selectionActive {
		t.Fatal("expected selection to clear after unstaging selected range")
	}
}

func TestStatusBarShowsSelectedCountWhenRangeActive(t *testing.T) {
	m := newStatusModel(t)
	start := findFirstFileRow(t, m)
	end := findSecondFileRow(t, m)
	m.status.selectionActive = true
	m.status.selectionAnchor = start
	m.status.changesCursor = end

	bar := ansi.Strip(m.renderChangesStatusBar())
	if !containsAny(bar, "selected") {
		t.Fatalf("expected selected count in status bar, got %q", bar)
	}
}

func TestStatusBarShowsDriftSubtypeForSelectedRow(t *testing.T) {
	tests := []struct {
		name string
		file chezmoi.FileStatus
		want string
	}{
		{
			name: "source only",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			want: "pending apply",
		},
		{
			name: "target only",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: ' ', DestStatus: 'M'},
			want: "target changed",
		},
		{
			name: "both sides",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: 'M'},
			want: "diverged",
		},
		{
			name: "script run pending",
			file: chezmoi.FileStatus{Path: "/home/test/.chezmoiscripts/run_once_install.sh", SourceStatus: 'R', DestStatus: ' '},
			want: "pending script run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(WithDriftFiles([]chezmoi.FileStatus{tt.file}))
			m.view = StatusScreen
			m.activeTab = 0
			m.width = 120
			m.height = 40
			m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionDrift)

			bar := ansi.Strip(m.renderChangesStatusBar())
			if !strings.Contains(bar, tt.want) {
				t.Fatalf("expected status bar to contain %q, got %q", tt.want, bar)
			}
		})
	}
}

func TestStatusBarDoesNotShowDriftSubtypeForGitRows(t *testing.T) {
	m := newStatusModel(t)
	m.status.gitUnstagedFiles = []chezmoi.GitFile{
		{Path: "/home/test/.gitconfig", StatusCode: "M"},
	}
	m.buildChangesRows()
	m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionUnstaged)

	bar := ansi.Strip(m.renderChangesStatusBar())
	if containsAny(bar, "pending apply", "target changed", "diverged") {
		t.Fatalf("expected no drift subtype labels on unstaged row, got %q", bar)
	}
}

func TestRenderChangesDriftRowShowsSourceTargetSlots(t *testing.T) {
	m := newStatusModel(t)
	tests := []struct {
		name string
		file chezmoi.FileStatus
		want string
	}{
		{
			name: "source only",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' '},
			want: "M·",
		},
		{
			name: "target only",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: ' ', DestStatus: 'M'},
			want: "·M",
		},
		{
			name: "both sides",
			file: chezmoi.FileStatus{Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: 'M'},
			want: "MM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := ansi.Strip(m.renderChangesDriftRow(tt.file, false, 100))
			if !strings.Contains(line, tt.want) {
				t.Fatalf("expected drift row to contain %q, got %q", tt.want, line)
			}
		})
	}
}

// --- Test: Panel Content Mode Cycling from Panel Focus ---

func TestPanelContentModeCyclingFromPanelFocus(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.focusZone = panelFocusPanel
	m.panel.contentMode = panelModeDiff

	// v key when panel has focus should also cycle content mode.
	updatedAny, _ := m.Update(runeKey("v"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.contentMode != panelModeContent {
		t.Fatalf("expected contentMode=panelModeContent from panel focus, got %d", updated.panel.contentMode)
	}
}

// --- Test: Left Arrow No-op When Already on List Focus ---

func TestLeftArrowNoOpWhenAlreadyOnListFocus(t *testing.T) {
	m := newStatusModel(t)
	m.width = 120
	m.panel.focusZone = panelFocusList

	// Left arrow when already on list focus should be handled by panel keys
	// but FocusList only matches when focusZone == panelFocusPanel.
	// The key falls through to handleStatusKeys which has no Left handler.
	updatedAny, _ := m.Update(specialKey(tea.KeyLeft))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.focusZone != panelFocusList {
		t.Fatalf("expected focusZone to remain panelFocusList, got %d", updated.panel.focusZone)
	}
}

// --- Test: Enter on Drift File Sets BusyAction ---

func TestEnterOnDriftFileSetssBusyAction(t *testing.T) {
	m := newStatusModel(t)
	firstFile := findFirstFileRow(t, m)
	m.status.changesCursor = firstFile
	m.ui.busyAction = false

	updatedAny, cmd := m.Update(specialKey(tea.KeyEnter))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if !updated.ui.busyAction {
		t.Fatal("expected busyAction=true when loading diff for drift file")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd when loading diff for drift file")
	}
}

// --- Test: Actions Menu on Header Row Does Not Open ---

func TestActionsMenuDoesNotOpenOnHeaderRow(t *testing.T) {
	m := newStatusModel(t)
	headerRow := findFirstHeaderRow(t, m)
	m.status.changesCursor = headerRow

	updatedAny, _ := m.Update(runeKey("a"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.actions.show {
		t.Fatal("expected actions.show=false when pressing 'a' on a header row")
	}
}

// --- Test: Actions Menu Disables Re-add for Template ---

func TestActionsMenuDisablesReAddForTemplate(t *testing.T) {
	driftFile := chezmoi.FileStatus{
		Path: "/home/test/.bashrc", SourceStatus: 'M', DestStatus: ' ', IsTemplate: true,
	}
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{driftFile}),
	)
	m.status.templatePaths = map[string]bool{"/home/test/.bashrc": true}

	// Move cursor to the drift file row.
	fileRow := findFirstFileRow(t, m)
	m.status.changesCursor = fileRow

	m.openStatusActionsMenu()

	if !m.actions.show {
		t.Fatal("expected actions menu to be shown")
	}

	// Find the Re-add item and verify it is disabled.
	found := false
	for _, item := range m.actions.items {
		if item.action == chezmoiActionReAdd {
			found = true
			if !item.disabled {
				t.Error("expected Re-add action to be disabled for template file")
			}
			if item.unavailableReason != "template" {
				t.Errorf("expected unavailableReason=%q, got %q", "template", item.unavailableReason)
			}
		}
	}
	if !found {
		t.Fatal("Re-add action item not found in actions menu")
	}
}

func TestActionsMenuScriptDriftUsesRunScriptLabel(t *testing.T) {
	const scriptPath = "/home/test/.chezmoiscripts/run_once_install.sh"
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: scriptPath, SourceStatus: 'R', DestStatus: ' '},
		}),
	)
	m.status.changesCursor = findFirstSectionFileRow(t, m, changesSectionDrift)

	m.openStatusActionsMenu()

	applyItem, ok := findActionItemByAction(m.actions.items, chezmoiActionApplyFile)
	if !ok {
		t.Fatal("expected Apply action item for script drift row")
	}
	if applyItem.label != "Run Script" {
		t.Fatalf("expected apply label %q, got %q", "Run Script", applyItem.label)
	}
	if _, hasReAdd := findActionItemByAction(m.actions.items, chezmoiActionReAdd); hasReAdd {
		t.Fatal("expected Re-add action to be hidden for script R drift row")
	}
}

func TestDiffActionsMirrorReAddEligibilityForScriptR(t *testing.T) {
	const scriptPath = "/home/test/.chezmoiscripts/run_once_install.sh"
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: scriptPath, SourceStatus: 'R', DestStatus: ' '},
		}),
	)
	m.diff.sourceSection = changesSectionDrift
	m.diff.path = scriptPath

	m.openDiffActionsMenu()

	applyItem, ok := findActionItemByAction(m.actions.items, chezmoiActionApplyFile)
	if !ok {
		t.Fatal("expected Apply action item in diff actions")
	}
	if applyItem.label != "Run Script" {
		t.Fatalf("expected apply label %q, got %q", "Run Script", applyItem.label)
	}
	if _, hasReAdd := findActionItemByAction(m.actions.items, chezmoiActionReAdd); hasReAdd {
		t.Fatal("expected Re-add action to be hidden in diff actions for script R drift row")
	}
}

func TestDiffActionsShowReAddForModifiedDrift(t *testing.T) {
	const path = "/home/test/.bashrc"
	m := newTestModel(
		WithDriftFiles([]chezmoi.FileStatus{
			{Path: path, SourceStatus: 'M', DestStatus: ' '},
		}),
	)
	m.diff.sourceSection = changesSectionDrift
	m.diff.path = path

	m.openDiffActionsMenu()

	applyItem, ok := findActionItemByAction(m.actions.items, chezmoiActionApplyFile)
	if !ok {
		t.Fatal("expected Apply action item in diff actions")
	}
	if applyItem.label != "Apply File" {
		t.Fatalf("expected apply label %q, got %q", "Apply File", applyItem.label)
	}
	if _, hasReAdd := findActionItemByAction(m.actions.items, chezmoiActionReAdd); !hasReAdd {
		t.Fatal("expected Re-add action in diff actions for modified drift row")
	}
}

func findActionItemByAction(items []chezmoiActionItem, action chezmoiAction) (chezmoiActionItem, bool) {
	for _, item := range items {
		if item.action == action {
			return item, true
		}
	}
	return chezmoiActionItem{}, false
}

// --- Test: Panel Toggle Below Min Width ---

func TestPanelToggleBelowMinWidth(t *testing.T) {
	m := newStatusModel(t)
	m.width = 50 // Below panelMinWidth (60)

	// Even after toggling, shouldShow returns false when below panelMinWidth.
	updatedAny, _ := m.Update(runeKey("p"))
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatal("expected Model type assertion")
	}

	if updated.panel.shouldShow(updated.width) {
		t.Fatal("expected panel to remain hidden when width is below panelMinWidth")
	}
	// Focus should still be on list.
	if updated.panel.focusZone != panelFocusList {
		t.Fatalf("expected focusZone=panelFocusList after toggle below min width, got %d", updated.panel.focusZone)
	}
}
