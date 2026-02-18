package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestChezSharedKeys_MatchRuneKeys(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
		want    bool
	}{
		{"j matches Down", tea.KeyPressMsg{Code: 'j', Text: "j"}, ChezSharedKeys.Down, true},
		{"k matches Up", tea.KeyPressMsg{Code: 'k', Text: "k"}, ChezSharedKeys.Up, true},
		{"g matches Home", tea.KeyPressMsg{Code: 'g', Text: "g"}, ChezSharedKeys.Home, true},
		{"G matches End", tea.KeyPressMsg{Code: 'G', Text: "G"}, ChezSharedKeys.End, true},
		{"q matches Quit", tea.KeyPressMsg{Code: 'q', Text: "q"}, ChezSharedKeys.Quit, true},
		{"? matches Help", tea.KeyPressMsg{Code: '?', Text: "?"}, ChezSharedKeys.Help, true},
		{"m matches Mouse", tea.KeyPressMsg{Code: 'm', Text: "m"}, ChezSharedKeys.Mouse, true},
		{"/ matches Filter", tea.KeyPressMsg{Code: '/', Text: "/"}, ChezSharedKeys.Filter, true},
		{"1 matches Tab1", tea.KeyPressMsg{Code: '1', Text: "1"}, ChezSharedKeys.Tab1, true},
		{"2 matches Tab2", tea.KeyPressMsg{Code: '2', Text: "2"}, ChezSharedKeys.Tab2, true},
		{"3 matches Tab3", tea.KeyPressMsg{Code: '3', Text: "3"}, ChezSharedKeys.Tab3, true},
		{"4 matches Tab4", tea.KeyPressMsg{Code: '4', Text: "4"}, ChezSharedKeys.Tab4, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := key.Matches(tt.msg, tt.binding); got != tt.want {
				t.Errorf("key.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChezSharedKeys_MatchSpecialKeys(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
		want    bool
	}{
		{"KeyDown matches Down", tea.KeyPressMsg{Code: tea.KeyDown}, ChezSharedKeys.Down, true},
		{"KeyUp matches Up", tea.KeyPressMsg{Code: tea.KeyUp}, ChezSharedKeys.Up, true},
		{"KeyEnter matches Enter", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezSharedKeys.Enter, true},
		{"KeyEscape matches Back", tea.KeyPressMsg{Code: tea.KeyEscape}, ChezSharedKeys.Back, true},
		{"KeyTab matches TabNext", tea.KeyPressMsg{Code: tea.KeyTab}, ChezSharedKeys.TabNext, true},
		{"KeyShiftTab matches TabPrev", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, ChezSharedKeys.TabPrev, true},
		{"KeyHome matches Home", tea.KeyPressMsg{Code: tea.KeyHome}, ChezSharedKeys.Home, true},
		{"KeyEnd matches End", tea.KeyPressMsg{Code: tea.KeyEnd}, ChezSharedKeys.End, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := key.Matches(tt.msg, tt.binding); got != tt.want {
				t.Errorf("key.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChezSharedKeys_NoCrossContamination(t *testing.T) {
	jMsg := tea.KeyPressMsg{Code: 'j', Text: "j"}
	kMsg := tea.KeyPressMsg{Code: 'k', Text: "k"}

	if key.Matches(jMsg, ChezSharedKeys.Up) {
		t.Error("j should not match Up")
	}
	if key.Matches(kMsg, ChezSharedKeys.Down) {
		t.Error("k should not match Down")
	}
	if key.Matches(jMsg, ChezSharedKeys.Quit) {
		t.Error("j should not match Quit")
	}
}

func TestChezHelpOverlayKeys_MatchAllDismissKeys(t *testing.T) {
	msgs := []tea.KeyPressMsg{
		{Code: '?', Text: "?"},
		{Code: tea.KeyEscape},
		{Code: 'q', Text: "q"},
	}
	for _, msg := range msgs {
		if !key.Matches(msg, ChezHelpOverlayKeys.Close) {
			t.Errorf("expected %q to match ChezHelpOverlayKeys.Close", msg.String())
		}
	}
}

func TestChezActionMenuKeys_Match(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"k matches Up", tea.KeyPressMsg{Code: 'k', Text: "k"}, ChezActionMenuKeys.Up},
		{"j matches Down", tea.KeyPressMsg{Code: 'j', Text: "j"}, ChezActionMenuKeys.Down},
		{"enter matches Select", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezActionMenuKeys.Select},
		{"esc matches Close", tea.KeyPressMsg{Code: tea.KeyEscape}, ChezActionMenuKeys.Close},
		{"q matches Close", tea.KeyPressMsg{Code: 'q', Text: "q"}, ChezActionMenuKeys.Close},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezChangesKeys_MatchRuneKeys(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"s matches Stage", tea.KeyPressMsg{Code: 's', Text: "s"}, ChezChangesKeys.Stage},
		{"u matches Unstage", tea.KeyPressMsg{Code: 'u', Text: "u"}, ChezChangesKeys.Unstage},
		{"S matches StageAll", tea.KeyPressMsg{Code: 'S', Text: "S"}, ChezChangesKeys.StageAll},
		{"U matches UnstageAll", tea.KeyPressMsg{Code: 'U', Text: "U"}, ChezChangesKeys.UnstageAll},
		{"c matches Commit", tea.KeyPressMsg{Code: 'c', Text: "c"}, ChezChangesKeys.Commit},
		{"P matches Push", tea.KeyPressMsg{Code: 'P', Text: "P"}, ChezChangesKeys.Push},
		{"a matches Actions", tea.KeyPressMsg{Code: 'a', Text: "a"}, ChezChangesKeys.Actions},
		{"r matches Refresh", tea.KeyPressMsg{Code: 'r', Text: "r"}, ChezChangesKeys.Refresh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezChangesKeys_CaseSensitivity(t *testing.T) {
	sMsg := tea.KeyPressMsg{Code: 's', Text: "s"}
	bigSMsg := tea.KeyPressMsg{Code: 'S', Text: "S"}
	pMsg := tea.KeyPressMsg{Code: 'p', Text: "p"}
	bigPMsg := tea.KeyPressMsg{Code: 'P', Text: "P"}

	if key.Matches(sMsg, ChezChangesKeys.StageAll) {
		t.Error("lowercase s should not match StageAll (capital S)")
	}
	if !key.Matches(bigSMsg, ChezChangesKeys.StageAll) {
		t.Error("capital S should match StageAll")
	}
	if key.Matches(pMsg, ChezChangesKeys.Push) {
		t.Error("lowercase p should not match Push (capital P)")
	}
	if !key.Matches(bigPMsg, ChezChangesKeys.Push) {
		t.Error("capital P should match Push")
	}
}

func TestChezManagedKeys_MatchRuneKeys(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"t matches TreeToggle", tea.KeyPressMsg{Code: 't', Text: "t"}, ChezManagedKeys.TreeToggle},
		{"f matches ViewPicker", tea.KeyPressMsg{Code: 'f', Text: "f"}, ChezManagedKeys.ViewPicker},
		{"F matches FilterOverlay", tea.KeyPressMsg{Code: 'F', Text: "F"}, ChezManagedKeys.FilterOverlay},
		{"c matches ClearSearch", tea.KeyPressMsg{Code: 'c', Text: "c"}, ChezManagedKeys.ClearSearch},
		{"a matches Actions", tea.KeyPressMsg{Code: 'a', Text: "a"}, ChezManagedKeys.Actions},
		{"r matches Refresh", tea.KeyPressMsg{Code: 'r', Text: "r"}, ChezManagedKeys.Refresh},
		{"l matches Expand", tea.KeyPressMsg{Code: 'l', Text: "l"}, ChezManagedKeys.Expand},
		{"enter matches Expand", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezManagedKeys.Expand},
		{"space matches Expand", tea.KeyPressMsg{Code: ' ', Text: " "}, ChezManagedKeys.Expand},
		{"h matches Collapse", tea.KeyPressMsg{Code: 'h', Text: "h"}, ChezManagedKeys.Collapse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezScrollKeys_Match(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"ctrl+d matches HalfDown", tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, ChezScrollKeys.HalfDown},
		{"ctrl+u matches HalfUp", tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, ChezScrollKeys.HalfUp},
		{"ctrl+f matches PageDown", tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}, ChezScrollKeys.PageDown},
		{"ctrl+b matches PageUp", tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}, ChezScrollKeys.PageUp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezDiffKeys_MatchRuneKeys(t *testing.T) {
	msg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	if !key.Matches(msg, ChezDiffKeys.Actions) {
		t.Error("a should match ChezDiffKeys.Actions")
	}
}

func TestChezFilterOverlayKeys_Match(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"space matches Toggle", tea.KeyPressMsg{Code: ' ', Text: " "}, ChezFilterOverlayKeys.Toggle},
		{"enter matches Apply", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezFilterOverlayKeys.Apply},
		{"esc matches Dismiss", tea.KeyPressMsg{Code: tea.KeyEscape}, ChezFilterOverlayKeys.Dismiss},
		{"k matches Up", tea.KeyPressMsg{Code: 'k', Text: "k"}, ChezFilterOverlayKeys.Up},
		{"j matches Down", tea.KeyPressMsg{Code: 'j', Text: "j"}, ChezFilterOverlayKeys.Down},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezViewPickerKeys_Match(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"k matches Up", tea.KeyPressMsg{Code: 'k', Text: "k"}, ChezViewPickerKeys.Up},
		{"j matches Down", tea.KeyPressMsg{Code: 'j', Text: "j"}, ChezViewPickerKeys.Down},
		{"enter matches Select", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezViewPickerKeys.Select},
		{"esc matches Dismiss", tea.KeyPressMsg{Code: tea.KeyEscape}, ChezViewPickerKeys.Dismiss},
		{"q matches Dismiss", tea.KeyPressMsg{Code: 'q', Text: "q"}, ChezViewPickerKeys.Dismiss},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestChezManagedFlatKeys_Match(t *testing.T) {
	tests := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"a matches Actions", tea.KeyPressMsg{Code: 'a', Text: "a"}, ChezManagedFlatKeys.Actions},
		{"enter matches Actions", tea.KeyPressMsg{Code: tea.KeyEnter}, ChezManagedFlatKeys.Actions},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !key.Matches(tt.msg, tt.binding) {
				t.Errorf("expected match for %q", tt.msg.String())
			}
		})
	}
}

func TestBindingsToHelpEntries(t *testing.T) {
	entries := bindingsToHelpEntries(ChezSharedKeys.Up, ChezSharedKeys.Down)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "↑/k" {
		t.Errorf("expected Key=↑/k, got %s", entries[0].Key)
	}
	if entries[0].Desc != "Move up" {
		t.Errorf("expected Desc=Move up, got %s", entries[0].Desc)
	}
	if entries[1].Key != "↓/j" {
		t.Errorf("expected Key=↓/j, got %s", entries[1].Key)
	}
	if entries[1].Desc != "Move down" {
		t.Errorf("expected Desc=Move down, got %s", entries[1].Desc)
	}
}

func TestBindingsToHelpEntries_FiltersDisabled(t *testing.T) {
	disabled := key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "test"),
		key.WithDisabled(),
	)
	entries := bindingsToHelpEntries(ChezSharedKeys.Up, disabled, ChezSharedKeys.Down)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (disabled filtered), got %d", len(entries))
	}
	if entries[0].Key != "↑/k" {
		t.Errorf("expected first entry Key=↑/k, got %s", entries[0].Key)
	}
	if entries[1].Key != "↓/j" {
		t.Errorf("expected second entry Key=↓/j, got %s", entries[1].Key)
	}
}

func TestChezCommandKeys_Match(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	if !key.Matches(msg, ChezCommandKeys.Run) {
		t.Error("enter should match ChezCommandKeys.Run")
	}
}

func TestChezFilterKeys_Match(t *testing.T) {
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}

	if !key.Matches(escMsg, ChezFilterKeys.Cancel) {
		t.Error("esc should match ChezFilterKeys.Cancel")
	}
	if !key.Matches(enterMsg, ChezFilterKeys.Confirm) {
		t.Error("enter should match ChezFilterKeys.Confirm")
	}
}
