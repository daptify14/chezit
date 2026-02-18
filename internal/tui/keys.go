package tui

import "charm.land/bubbles/v2/key"

// ── Shared Bindings (used across most chezit views) ────────────────

type ChezSharedKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Home    key.Binding
	End     key.Binding
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Mouse   key.Binding
	Filter  key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
}

var ChezSharedKeys = ChezSharedKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Move down"),
	),
	Home: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("g", "Jump top"),
	),
	End: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("G", "Jump bottom"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Keys"),
	),
	Mouse: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "Mouse/copy mode"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Filter"),
	),
	TabNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Next tab"),
	),
	TabPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("S-Tab", "Prev tab"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "Tab 1"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "Tab 2"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "Tab 3"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "Tab 4"),
	),
}

// ── Action Menu Bindings ───────────────────────────────────────────

type ChezActionMenuKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Close  key.Binding
}

var ChezActionMenuKeys = ChezActionMenuKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Move down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Select"),
	),
	Close: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "Close"),
	),
}

// ── Filter Mode Bindings ───────────────────────────────────────────

type ChezFilterKeyMap struct {
	Cancel  key.Binding
	Confirm key.Binding
}

var ChezFilterKeys = ChezFilterKeyMap{
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Cancel"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Apply"),
	),
}

// ── Help Overlay Dismiss Bindings ──────────────────────────────────

type ChezHelpOverlayKeyMap struct {
	Close key.Binding
}

var ChezHelpOverlayKeys = ChezHelpOverlayKeyMap{
	Close: key.NewBinding(
		key.WithKeys("?", "esc", "q"),
		key.WithHelp("?/esc", "Close"),
	),
}

// ── Confirm Dialog Bindings ────────────────────────────────────────

type ChezConfirmKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

var ChezConfirmKeys = ChezConfirmKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "Confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n/esc", "Cancel"),
	),
}

// ── Changes Tab Bindings ───────────────────────────────────────────

type ChezChangesKeyMap struct {
	Edit       key.Binding
	Stage      key.Binding
	Unstage    key.Binding
	Discard    key.Binding
	StageAll   key.Binding
	UnstageAll key.Binding
	Commit     key.Binding
	Push       key.Binding
	Fetch      key.Binding
	Pull       key.Binding
	Actions    key.Binding
	Refresh    key.Binding
}

var ChezChangesKeys = ChezChangesKeyMap{
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Edit file"),
	),
	Stage: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Re-add/Stage"),
	),
	Unstage: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "Unstage"),
	),
	Discard: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "Discard/Undo"),
	),
	StageAll: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "Stage all"),
	),
	UnstageAll: key.NewBinding(
		key.WithKeys("U"),
		key.WithHelp("U", "Unstage all"),
	),
	Commit: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Commit"),
	),
	Push: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "Push"),
	),
	Fetch: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Fetch"),
	),
	Pull: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Pull"),
	),
	Actions: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "Actions"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refresh"),
	),
}

// ── Managed Tab Bindings ───────────────────────────────────────────

type ChezManagedKeyMap struct {
	TreeToggle    key.Binding
	ViewPicker    key.Binding
	FilterOverlay key.Binding
	ClearSearch   key.Binding
	Actions       key.Binding
	Refresh       key.Binding
	Expand        key.Binding
	Collapse      key.Binding
}

var ChezManagedKeys = ChezManagedKeyMap{
	TreeToggle: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Tree/flat"),
	),
	ViewPicker: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "View/filter"),
	),
	FilterOverlay: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "Filter section"),
	),
	ClearSearch: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Clear search"),
	),
	Actions: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "Actions"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refresh"),
	),
	Expand: key.NewBinding(
		key.WithKeys("l", "enter", "space"),
		key.WithHelp("l", "Expand"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "Collapse"),
	),
}

// ── Scroll Bindings (diff/config view) ─────────────────────────────

type ChezScrollKeyMap struct {
	HalfDown key.Binding
	HalfUp   key.Binding
	PageDown key.Binding
	PageUp   key.Binding
}

var ChezScrollKeys = ChezScrollKeyMap{
	HalfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("^d", "Half-page down"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("^u", "Half-page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("^f", "Full-page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("^b", "Full-page up"),
	),
}

// ── Diff View Bindings ─────────────────────────────────────────────

type ChezDiffKeyMap struct {
	Edit    key.Binding
	Actions key.Binding
}

var ChezDiffKeys = ChezDiffKeyMap{
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Edit file"),
	),
	Actions: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "Actions"),
	),
}

// ── Command Tab Bindings ───────────────────────────────────────────

type ChezCommandKeyMap struct {
	Run    key.Binding
	DryRun key.Binding
}

var ChezCommandKeys = ChezCommandKeyMap{
	Run: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Run"),
	),
	DryRun: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Dry run"),
	),
}

// ── Filter Overlay Bindings ────────────────────────────────────────

type ChezFilterOverlayKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	Apply   key.Binding
	Dismiss key.Binding
}

var ChezFilterOverlayKeys = ChezFilterOverlayKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Move down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "Toggle"),
	),
	Apply: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Apply"),
	),
	Dismiss: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Cancel"),
	),
}

// ── View Picker Bindings ───────────────────────────────────────────

type ChezViewPickerKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Dismiss key.Binding
}

var ChezViewPickerKeys = ChezViewPickerKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Move down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Select"),
	),
	Dismiss: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "Dismiss"),
	),
}

// ── Managed Flat View Bindings ─────────────────────────────────────

type ChezManagedFlatKeyMap struct {
	Actions key.Binding
}

var ChezManagedFlatKeys = ChezManagedFlatKeyMap{
	Actions: key.NewBinding(
		key.WithKeys("a", "enter"),
		key.WithHelp("a", "Actions"),
	),
}

// ── Info Tab Bindings ───────────────────────────────────────────────

type ChezInfoKeyMap struct {
	Left    key.Binding
	Right   key.Binding
	Format  key.Binding
	Refresh key.Binding
}

var ChezInfoKeys = ChezInfoKeyMap{
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "Prev view"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "Next view"),
	),
	Format: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Toggle format"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refresh"),
	),
}

// ── Panel Bindings ──────────────────────────────────────────────────

type ChezPanelKeyMap struct {
	Toggle      key.Binding
	FocusPanel  key.Binding
	FocusList   key.Binding
	ContentMode key.Binding
	ScrollDown  key.Binding
	ScrollUp    key.Binding
	HalfDown    key.Binding
	HalfUp      key.Binding
	Top         key.Binding
	Bottom      key.Binding
}

var ChezPanelKeys = ChezPanelKeyMap{
	Toggle: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Toggle panel"),
	),
	FocusPanel: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "Panel"),
	),
	FocusList: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "List"),
	),
	ContentMode: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "Panel view"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Scroll down"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Scroll up"),
	),
	HalfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("^d", "Half-page down"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("^u", "Half-page up"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "Top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "Bottom"),
	),
}

// ── Helper Functions ────────────────────────────────────────────────

// bindingsToHelpEntries converts key bindings to HelpEntry slices, filtering disabled bindings.
func bindingsToHelpEntries(bindings ...key.Binding) []HelpEntry {
	entries := make([]HelpEntry, 0, len(bindings))
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		if h.Key == "" {
			continue
		}
		entries = append(entries, HelpEntry{Key: h.Key, Desc: h.Desc})
	}
	return entries
}
