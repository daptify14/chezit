package tui

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestNewModelBuildsAllTabsAndCommands(t *testing.T) {
	m := NewModel(Options{Service: testService()})

	if len(m.tabNames) != 4 {
		t.Fatalf("expected 4 tabs, got %#v", m.tabNames)
	}
	if m.tabNames[0] != "Status" || m.tabNames[1] != "Files" || m.tabNames[2] != "Info" || m.tabNames[3] != "Commands" {
		t.Fatalf("unexpected tabs: %#v", m.tabNames)
	}
	if !m.ui.loading {
		t.Fatalf("expected loading=true")
	}
	if !m.status.loadingGit {
		t.Fatalf("expected loadingGit=true")
	}

	labels := make([]string, 0, len(m.cmds.items))
	for _, cmd := range m.cmds.items {
		labels = append(labels, cmd.label)
	}
	joined := strings.Join(labels, ",")
	if !strings.Contains(joined, "Apply") {
		t.Fatalf("expected Apply command, got %v", labels)
	}
}

func TestNewModelReadOnlyStripsMutatingCommands(t *testing.T) {
	m := NewModel(Options{Service: testServiceReadOnly()})

	if !m.status.loadingGit {
		t.Fatalf("expected loadingGit=true in read-only mode")
	}

	labels := make([]string, 0, len(m.cmds.items))
	for _, cmd := range m.cmds.items {
		labels = append(labels, cmd.label)
	}
	joined := strings.Join(labels, ",")
	if strings.Contains(joined, "Apply") || strings.Contains(joined, "Update") || strings.Contains(joined, "Re-Add All") || strings.Contains(joined, "Init") || strings.Contains(joined, "Edit Source") {
		t.Fatalf("unexpected mutating commands in read-only mode: %v", labels)
	}
}

func TestInitStatusIncludesGitCommitsLoad(t *testing.T) {
	m := NewModel(Options{Service: testService()})

	msgs := collectInitAndBatchMsgs(t, m.Init())
	if !containsMsgType(msgs, chezmoiGitCommitsLoadedMsg{}) {
		t.Fatalf("expected Init batch to include %T, got %T entries", chezmoiGitCommitsLoadedMsg{}, msgs)
	}
}

func TestLoadDeferredStatusIncludesGitCommitsLoad(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "Files"})
	if !m.status.gitDeferred {
		t.Fatal("precondition failed: expected gitDeferred=true for Files initial tab")
	}

	cmd := m.loadDeferredForTab("Status")
	msgs := collectInitAndBatchMsgs(t, cmd)
	if !containsMsgType(msgs, chezmoiGitCommitsLoadedMsg{}) {
		t.Fatalf("expected deferred Status load batch to include %T, got %T entries", chezmoiGitCommitsLoadedMsg{}, msgs)
	}
}

func TestAllLandingStatsLoadedWaitsForDeferredStatusAndGit(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "Files"})
	m.ui.loading = false
	m.status.loadingGit = false
	m.filesTab.views[managedViewManaged].loading = false

	if m.allLandingStatsLoaded() {
		t.Fatal("expected landing stats to remain not-ready while deferred status/git loads are pending")
	}

	m.status.statusDeferred = false
	m.status.gitDeferred = false

	if !m.allLandingStatsLoaded() {
		t.Fatal("expected landing stats to be ready once deferred status/git loads are resolved")
	}
}

func collectInitAndBatchMsgs(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		msgs := make([]tea.Msg, 0, len(batch))
		for _, child := range batch {
			if child == nil {
				continue
			}
			childMsg := child()
			if childMsg != nil {
				msgs = append(msgs, childMsg)
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func containsMsgType(msgs []tea.Msg, example tea.Msg) bool {
	targetType := reflect.TypeOf(example)
	for _, msg := range msgs {
		if reflect.TypeOf(msg) == targetType {
			return true
		}
	}
	return false
}

func TestAllLandingStatsLoadedIgnoresDeferredGitInReadOnlyMode(t *testing.T) {
	m := NewModel(Options{Service: testServiceReadOnly(), InitialTab: "Files"})
	m.ui.loading = false
	m.filesTab.views[managedViewManaged].loading = false
	m.status.statusDeferred = false

	if !m.status.gitDeferred {
		t.Fatal("expected gitDeferred to be true for direct files startup")
	}
	if !m.allLandingStatsLoaded() {
		t.Fatal("expected landing stats to be ready in read-only mode even with deferred git")
	}
}

func TestApplyAllCmdReturnsReadOnlyErrorWhenNil(t *testing.T) {
	m := NewModel(Options{Service: testServiceReadOnly()})

	msg := m.applyAllCmd()()
	done, ok := msg.(chezmoiExecDoneMsg)
	if !ok {
		t.Fatalf("expected chezmoiExecDoneMsg, got %T", msg)
	}
	if !errors.Is(done.err, chezmoi.ErrReadOnly) {
		t.Fatalf("expected read-only error, got %v", done.err)
	}
}

func TestHandleExecDoneReadOnlyShowsUnavailableWithoutReload(t *testing.T) {
	m := NewModel(Options{Service: testServiceReadOnly()})
	m.ui.busyAction = true
	gen := m.gen

	updatedAny, cmd := m.Update(chezmoiExecDoneMsg{action: chezmoiActionApplyAll, err: chezmoi.ErrReadOnly})
	updated, ok := updatedAny.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updatedAny)
	}
	if cmd != nil {
		t.Fatal("expected no reload cmd after a read-only refusal")
	}
	if updated.ui.message != actionUnavailableMessage("read-only mode") {
		t.Fatalf("expected read-only message, got %q", updated.ui.message)
	}
	if updated.gen != gen {
		t.Fatalf("expected gen unchanged at %d, got %d", gen, updated.gen)
	}
	if updated.ui.busyAction {
		t.Fatal("expected busyAction cleared")
	}
}

func TestOpenChangesActionsDisablesUnsupportedActions(t *testing.T) {
	m := NewModel(Options{Service: testServiceReadOnly()})
	m.status.files = []chezmoi.FileStatus{{Path: "/tmp/test", SourceStatus: 'M', DestStatus: ' '}}
	m.status.filteredFiles = m.status.files
	m.buildChangesRows()
	m.status.changesCursor = 2 // first drift file row (after Incoming header + Drift header)
	m.openStatusActionsMenu()

	var applyItem, updateItem *chezmoiActionItem
	for i := range m.actions.items {
		item := &m.actions.items[i]
		switch item.action {
		case chezmoiActionApplyFile:
			applyItem = item
		case chezmoiActionUpdate:
			updateItem = item
		}
	}
	if applyItem == nil || !applyItem.disabled {
		t.Fatalf("expected Apply File to be disabled, got %#v", applyItem)
	}
	if updateItem == nil || !updateItem.disabled {
		t.Fatalf("expected Update to be disabled, got %#v", updateItem)
	}
}

func TestSelectedManagedPathForOpenResolvesDirectoryToAbsolutePath(t *testing.T) {
	const targetPath = "/home/test"

	managedFile := filepath.Join(targetPath, ".config", "myapp", "config.yml")
	m := NewModel(Options{Service: testService()})
	m.filesTab.views[managedViewManaged].files = []string{managedFile}
	m.filesTab.views[managedViewManaged].filteredFiles = m.filesTab.views[managedViewManaged].files
	m.rebuildFileViewTree(managedViewManaged)
	m.filesTab.treeView = true
	m.filesTab.cursor = 0

	if got := m.selectedManagedPath(); got != ".config" {
		t.Fatalf("expected selectedManagedPath=.config, got %q", got)
	}
	want := filepath.Join(targetPath, ".config")
	if got := m.selectedManagedPathForOpen(); got != want {
		t.Fatalf("expected selectedManagedPathForOpen=%q, got %q", want, got)
	}
}

func TestEscFromLandingQuits(t *testing.T) {
	m := NewModel(Options{Service: testService(), EscBehavior: EscQuit})
	m.view = LandingScreen

	updatedModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if _, ok := updatedModel.(Model); !ok {
		t.Fatalf("expected Model after update, got %T", updatedModel)
	}
	if cmd == nil {
		t.Fatal("expected quit command for q on landing view")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestNewModelAutoFetchPresetsInFlight(t *testing.T) {
	m := NewModel(Options{Service: testService(), AutoFetch: true})
	if !m.autoFetch {
		t.Fatal("expected autoFetch to be recorded on the model")
	}
	if !m.status.fetchInProgress {
		t.Fatal("expected fetch to be preset in flight when Init will dispatch it")
	}
	if m.status.fetchOutcome != fetchNotStarted {
		t.Fatalf("expected outcome fetchNotStarted, got %d", m.status.fetchOutcome)
	}
}

func TestNewModelAutoFetchOffSetsOutcome(t *testing.T) {
	m := NewModel(Options{Service: testService(), AutoFetch: false})
	if m.status.fetchOutcome != fetchOff {
		t.Fatalf("expected outcome fetchOff, got %d", m.status.fetchOutcome)
	}
	if m.status.fetchInProgress {
		t.Fatal("expected no fetch in flight when auto_fetch is off")
	}
}

func TestInitIncludesFetchOnlyWhenAutoFetch(t *testing.T) {
	on := NewModel(Options{Service: testService(), AutoFetch: true})
	if !containsMsgType(collectInitAndBatchMsgs(t, on.Init()), chezmoiGitFetchDoneMsg{}) {
		t.Fatal("expected Init batch to include the fetch when auto_fetch is on")
	}

	off := NewModel(Options{Service: testService(), AutoFetch: false})
	if containsMsgType(collectInitAndBatchMsgs(t, off.Init()), chezmoiGitFetchDoneMsg{}) {
		t.Fatal("expected Init batch to omit the fetch when auto_fetch is off")
	}
}

func TestNewModelFilesTabDefersFetch(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "Files", AutoFetch: true})
	if m.status.fetchInProgress {
		t.Fatal("expected no fetch in flight when git is deferred")
	}
	if containsMsgType(collectInitAndBatchMsgs(t, m.Init()), chezmoiGitFetchDoneMsg{}) {
		t.Fatal("expected Init batch to omit the fetch when git is deferred")
	}

	cmd := m.loadDeferredForTab("Status")
	if !m.status.fetchInProgress {
		t.Fatal("expected deferred Status load to start the fetch")
	}
	if !containsMsgType(collectInitAndBatchMsgs(t, cmd), chezmoiGitFetchDoneMsg{}) {
		t.Fatal("expected deferred Status load batch to include the fetch")
	}

	// A second deferred load must not dispatch another fetch.
	if cmd := m.loadDeferredForTab("Status"); cmd != nil {
		t.Fatal("expected no further loads once deferred flags are cleared")
	}
}

func TestLoadDeferredSkipsFetchWhenAutoFetchOff(t *testing.T) {
	m := NewModel(Options{Service: testService(), InitialTab: "Files", AutoFetch: false})
	cmd := m.loadDeferredForTab("Status")
	if m.status.fetchInProgress {
		t.Fatal("expected no fetch when auto_fetch is off")
	}
	if containsMsgType(collectInitAndBatchMsgs(t, cmd), chezmoiGitFetchDoneMsg{}) {
		t.Fatal("expected deferred Status load batch to omit the fetch when auto_fetch is off")
	}
}
