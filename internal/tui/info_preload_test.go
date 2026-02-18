package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- preloadInfoViewsCmd ---

func TestPreloadInfoViewsCmdLoadsAllUnloaded(t *testing.T) {
	m := newTestModel(WithTab(2))

	// No views loaded or loading — preloadInfoViewsCmd should return a non-nil command.
	cmd := m.preloadInfoViewsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command from preloadInfoViewsCmd when no views are loaded")
	}

	// All views should now be marked loading.
	for i := range infoViewCount {
		if !m.info.views[i].loading {
			t.Fatalf("expected view %d loading=true after preloadInfoViewsCmd", i)
		}
	}
}

func TestPreloadInfoViewsCmdSkipsAlreadyLoaded(t *testing.T) {
	m := newTestModel(WithTab(2))

	// Mark all views as loaded.
	for i := range infoViewCount {
		m.info.views[i].loaded = true
	}

	cmd := m.preloadInfoViewsCmd()
	if cmd != nil {
		t.Fatal("expected nil command from preloadInfoViewsCmd when all views are already loaded")
	}
}

func TestPreloadInfoViewsCmdSkipsLoadingViews(t *testing.T) {
	m := newTestModel(WithTab(2))

	// Mark all views as loading.
	for i := range infoViewCount {
		m.info.views[i].loading = true
	}

	cmd := m.preloadInfoViewsCmd()
	if cmd != nil {
		t.Fatal("expected nil command from preloadInfoViewsCmd when all views are already loading")
	}
}

func TestPreloadInfoViewsCmdLoadsOnlyMissing(t *testing.T) {
	m := newTestModel(WithTab(2))

	// Mark first two as loaded, leave the rest.
	m.info.views[infoViewConfig].loaded = true
	m.info.views[infoViewFull].loading = true

	cmd := m.preloadInfoViewsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command for partially loaded views")
	}

	// Data and Doctor should now be loading.
	if !m.info.views[infoViewData].loading {
		t.Fatal("expected Data view loading=true")
	}
	if !m.info.views[infoViewDoctor].loading {
		t.Fatal("expected Doctor view loading=true")
	}
}

// --- NewModel with InitialTab="info" ---

func TestNewModelInfoTabSetsAllViewsLoading(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "info"})

	for i := range infoViewCount {
		if !m.info.views[i].loading {
			t.Fatalf("expected view %d loading=true when InitialTab=info", i)
		}
	}
}

func TestNewModelInfoTabDefersStatusAndGit(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "info"})

	if !m.status.statusDeferred {
		t.Fatal("expected statusDeferred=true when InitialTab=info")
	}
	if !m.status.gitDeferred {
		t.Fatal("expected gitDeferred=true when InitialTab=info")
	}
	if !m.filesTab.managedDeferred {
		t.Fatal("expected managedDeferred=true when InitialTab=info")
	}
}

func TestNewModelDefaultTabDoesNotSetInfoViewsLoading(t *testing.T) {
	m := NewModel(Options{Service: testService()})

	for i := range infoViewCount {
		if m.info.views[i].loading {
			t.Fatalf("expected view %d loading=false for default (Status) tab", i)
		}
	}
}

// --- loadDeferredForTab("Info") ---

func TestLoadDeferredForInfoPreloadsAllViews(t *testing.T) {
	m := newTestModel(WithTab(2))

	cmd := m.loadDeferredForTab("Info")
	if cmd == nil {
		t.Fatal("expected non-nil command from loadDeferredForTab(Info)")
	}

	for i := range infoViewCount {
		if !m.info.views[i].loading {
			t.Fatalf("expected view %d loading=true after loadDeferredForTab(Info)", i)
		}
	}
}

func TestLoadDeferredForInfoNoOpWhenAllLoaded(t *testing.T) {
	m := newTestModel(WithTab(2))

	for i := range infoViewCount {
		m.info.views[i].loaded = true
	}

	cmd := m.loadDeferredForTab("Info")
	if cmd != nil {
		t.Fatal("expected nil command from loadDeferredForTab(Info) when all views loaded")
	}
}

// --- Arrow navigation with preloaded views ---

func TestInfoArrowKeyNoCommandWhenPreloaded(t *testing.T) {
	// Simulate all views preloaded (loaded=true).
	m := newTestModel(WithTab(2))
	for i := range infoViewCount {
		m.info.views[i].loaded = true
		m.info.views[i].lines = []string{"line 1"}
	}
	m.info.activeView = infoViewConfig

	// Press right arrow to go to Full view.
	updated, cmd := sendKey(t, m, specialKey(tea.KeyRight))

	if updated.info.activeView != infoViewFull {
		t.Fatalf("expected activeView=infoViewFull (%d), got %d", infoViewFull, updated.info.activeView)
	}
	// ensureInfoViewLoaded should return nil since the view is already loaded.
	if cmd != nil {
		t.Fatal("expected nil command when navigating to already-loaded view")
	}
}

func TestInfoArrowKeyTriggersLoadWhenNotPreloaded(t *testing.T) {
	m := newTestModel(WithTab(2))
	m.info.views[infoViewConfig].loaded = true
	m.info.views[infoViewConfig].lines = []string{"line 1"}
	m.info.activeView = infoViewConfig

	// Full view is not loaded — pressing right should trigger a load.
	updated, cmd := sendKey(t, m, specialKey(tea.KeyRight))

	if updated.info.activeView != infoViewFull {
		t.Fatalf("expected activeView=infoViewFull (%d), got %d", infoViewFull, updated.info.activeView)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command when navigating to unloaded view")
	}
}
