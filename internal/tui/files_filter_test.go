package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestHandleChezmoiManagedKeys_FOpensViewPicker(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.overlays.showViewPicker = false

	updatedModel, _ := m.handleFilesKeys(tea.KeyPressMsg{Code: 'f', Text: "f"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if !updated.overlays.showViewPicker {
		t.Fatal("expected f to open view picker")
	}
}

func TestHandleChezmoiManagedKeys_ViewPickerTabShortcutSelectsMode(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.overlays.showViewPicker = true
	m.filesTab.viewMode = managedViewAll
	m.overlays.viewPickerItems = []viewPickerItem{
		{mode: managedViewManaged, label: "Managed"},
		{mode: managedViewAll, label: "All"},
	}
	m.overlays.viewPickerCursor = 1

	updatedModel, _ := m.handleFilesKeys(tea.KeyPressMsg{Code: '1', Text: "1"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if updated.overlays.showViewPicker {
		t.Fatal("expected view picker to close after 1 quick-select")
	}
	if updated.filesTab.viewMode != managedViewManaged {
		t.Fatalf("expected managed view mode after 1 quick-select, got %v", updated.filesTab.viewMode)
	}
}

func TestHandleChezmoiManagedKeys_ViewPickerCapitalFFocusesFilterRows(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.overlays.showViewPicker = true
	m.overlays.viewPickerItems = []viewPickerItem{{mode: managedViewManaged, label: "Managed"}}
	m.overlays.filterCategories = []filterCategory{
		{entryType: "", label: "Reset type filters", enabled: true},
		{entryType: chezmoi.EntryFiles, label: "files", enabled: true},
	}
	m.overlays.viewPickerCursor = 0

	updatedModel, _ := m.handleFilesKeys(tea.KeyPressMsg{Code: 'F', Text: "F"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if !updated.overlays.showViewPicker {
		t.Fatal("expected view picker to stay open")
	}
	if updated.overlays.viewPickerCursor != len(updated.overlays.viewPickerItems) {
		t.Fatalf("expected cursor to jump to first filter row, got %d", updated.overlays.viewPickerCursor)
	}
}

func TestHandleChezmoiManagedKeys_ViewPickerToggleFilterAndApply(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.openViewPicker()
	if len(m.overlays.filterCategories) < 2 {
		t.Fatalf("expected reset row + at least one filter row, got %d", len(m.overlays.filterCategories))
	}

	filterRow := len(m.overlays.viewPickerItems) + 1 // first concrete type after "reset"
	m.overlays.viewPickerCursor = filterRow
	cat := m.overlays.filterCategories[1]
	if cat.entryType == "" {
		t.Fatal("expected concrete entry type row")
	}
	if !cat.enabled {
		t.Fatal("expected filter row enabled by default")
	}

	updatedModel, _ := m.handleFilesKeys(tea.KeyPressMsg{Code: ' ', Text: " "})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if updated.overlays.filterCategories[1].enabled {
		t.Fatal("expected space to toggle filter row off")
	}

	appliedModel, _ := updated.handleFilesKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	applied, ok := appliedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", appliedModel)
	}
	if applied.overlays.showViewPicker {
		t.Fatal("expected enter to apply and close view/filter overlay")
	}
	if len(applied.filesTab.entryFilter.Exclude) == 0 {
		t.Fatal("expected applied entry filter to exclude toggled type")
	}
	if applied.filesTab.entryFilter.Exclude[0] != cat.entryType {
		t.Fatalf("expected excluded type %q, got %q", cat.entryType, applied.filesTab.entryFilter.Exclude[0])
	}
}

func TestHandleChezmoiKey_ViewPickerQuickViewKeysNotInterceptedByTabSwitch(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	if len(m.tabNames) < 2 {
		t.Fatalf("expected files tab to exist, got %v", m.tabNames)
	}
	m.view = StatusScreen
	m.activeTab = 1 // Files
	m.overlays.showViewPicker = true
	m.filesTab.viewMode = managedViewAll
	m.overlays.viewPickerPendingMode = managedViewAll
	m.overlays.viewPickerItems = []viewPickerItem{
		{mode: managedViewManaged, label: "Managed"},
		{mode: managedViewAll, label: "All"},
	}

	updatedModel, _ := m.handleKeyMsg(tea.KeyPressMsg{Code: '1', Text: "1"})
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedModel)
	}
	if updated.activeTab != 1 {
		t.Fatalf("expected to stay on Files tab, got activeTab=%d", updated.activeTab)
	}
	if updated.overlays.showViewPicker {
		t.Fatal("expected quick-view key to apply and close overlay")
	}
	if updated.filesTab.viewMode != managedViewManaged {
		t.Fatalf("expected quick-view key 1 to switch pending mode to Managed, got %v", updated.filesTab.viewMode)
	}
}

func TestRenderViewPickerMenuDoesNotShowCurrentAndPendingModes(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.overlays.viewPickerItems = []viewPickerItem{
		{mode: managedViewManaged, label: "Managed"},
		{mode: managedViewAll, label: "All"},
	}
	m.filesTab.viewMode = managedViewAll
	m.overlays.viewPickerPendingMode = managedViewManaged

	out := m.renderViewPickerMenu()
	if strings.Contains(out, "Current:") {
		t.Fatalf("overlay should not include current mode banner, got:\n%s", out)
	}
	if strings.Contains(out, "Pending:") {
		t.Fatalf("overlay should not include pending mode banner, got:\n%s", out)
	}
}

func TestRenderManagedStatusBarShowsViewAndFilterChips(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.filesTab.viewMode = managedViewAll
	m.filesTab.entryFilter = chezmoi.EntryFilter{Exclude: []chezmoi.EntryType{chezmoi.EntryFiles, chezmoi.EntryScripts}}

	out := m.renderManagedStatusBar()
	if !strings.Contains(out, "[view:all]") {
		t.Fatalf("expected status bar to include view chip, got:\n%s", out)
	}
	if !strings.Contains(out, "[filter:2 excluded]") {
		t.Fatalf("expected status bar to include filter chip, got:\n%s", out)
	}
}

func TestRenderManagedStatusBarShowsSearchPausedChip(t *testing.T) {
	m := NewModel(Options{Service: testService()})
	m.width = 120
	m.filesTab.viewMode = managedViewAll
	m.filesTab.search.paused = true
	m.filterInput.SetValue("token")

	out := m.renderManagedStatusBar()
	if !strings.Contains(out, "[search paused]") {
		t.Fatalf("expected status bar to include paused search chip, got:\n%s", out)
	}
}
