package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/daptify14/chezit/internal/chezmoi"
	"github.com/daptify14/chezit/internal/config"
)

// ── Service Factories ───────────────────────────────────────────────

func testService() *chezmoi.Service {
	return chezmoi.NewService(
		chezmoi.New(chezmoi.WithBinaryPath("/bin/true")),
		config.ModeWrite,
		"/home/test",
	)
}

func testServiceReadOnly() *chezmoi.Service {
	return chezmoi.NewService(
		chezmoi.New(chezmoi.WithBinaryPath("/bin/true")),
		config.ModeReadOnly,
		"/home/test",
	)
}

func testServiceWithTarget(targetPath string) *chezmoi.Service {
	return chezmoi.NewService(
		chezmoi.New(chezmoi.WithBinaryPath("/bin/true")),
		config.ModeWrite,
		targetPath,
	)
}

// ── Model Builder ───────────────────────────────────────────────────

// testModelConfig holds configuration for building a test Model.
// Options populate this struct; newTestModel reads it once to construct
// the Model. This avoids order-dependent option footguns.
type testModelConfig struct {
	service  *chezmoi.Service
	readOnly bool
	view     Screen
	tab      int
	width    int
	height   int
	iconMode IconMode
	postInit []func(*Model)
}

// TestModelOption configures a test Model via testModelConfig.
type TestModelOption func(*testModelConfig)

func WithView(v Screen) TestModelOption {
	return func(c *testModelConfig) { c.view = v }
}

func WithTab(tab int) TestModelOption {
	return func(c *testModelConfig) { c.tab = tab }
}

func WithSize(w, h int) TestModelOption {
	return func(c *testModelConfig) { c.width = w; c.height = h }
}

func WithReadOnly() TestModelOption {
	return func(c *testModelConfig) { c.readOnly = true }
}

func WithIconMode(mode IconMode) TestModelOption {
	return func(c *testModelConfig) { c.iconMode = mode }
}

func WithDriftFiles(files []chezmoi.FileStatus) TestModelOption {
	return func(c *testModelConfig) {
		c.postInit = append(c.postInit, func(m *Model) {
			m.status.files = files
			m.status.filteredFiles = files
			// NOTE: buildChangesRows is a mutation method. This is a test-only
			// exception to the project immutability style -- unavoidable when
			// populating derived state from synthetic test data.
			m.buildChangesRows()
		})
	}
}

func WithManagedFiles(files []string) TestModelOption {
	return func(c *testModelConfig) {
		c.postInit = append(c.postInit, func(m *Model) {
			m.filesTab.views[managedViewManaged].files = files
			m.filesTab.views[managedViewManaged].filteredFiles = files
			m.filesTab.views[managedViewManaged].loading = false
			m.filesTab.views[managedViewIgnored].loading = false
			m.filesTab.views[managedViewUnmanaged].loading = false
			m.rebuildFileViewTree(managedViewManaged)
		})
	}
}

func WithPanelVisible() TestModelOption {
	return func(c *testModelConfig) {
		c.postInit = append(c.postInit, func(m *Model) {
			m.panel.manualOverride = true
			m.panel.visible = true
		})
	}
}

func WithDiffContent(path, content string, lineCount int) TestModelOption {
	return func(c *testModelConfig) {
		c.view = DiffScreen
		c.postInit = append(c.postInit, func(m *Model) {
			m.diff.path = path
			m.diff.content = content
			lines := make([]string, lineCount)
			for i := range lines {
				lines[i] = fmt.Sprintf("+line %d added", i)
			}
			m.diff.lines = lines
		})
	}
}

func WithInfoContent(subView int, content string, lineCount int) TestModelOption {
	return func(c *testModelConfig) {
		c.tab = 2
		c.postInit = append(c.postInit, func(m *Model) {
			m.info.activeView = subView
			lines := make([]string, lineCount)
			for i := range lines {
				lines[i] = fmt.Sprintf("info line %d", i)
			}
			m.info.views[subView].content = content
			m.info.views[subView].lines = lines
			m.info.views[subView].loaded = true
		})
	}
}

// newTestModel creates a default Model (StatusScreen, tab=0, 120x40, write mode).
// Apply options to customize. Existing zero-arg calls remain valid.
//
// Options are order-independent: the config struct is populated first, then the
// Model is built once from the final config. Post-construction mutations
// (WithDriftFiles, WithPanelVisible, etc.) run after Model creation.
func newTestModel(opts ...TestModelOption) Model {
	cfg := &testModelConfig{
		view:     StatusScreen,
		tab:      0,
		width:    120,
		height:   40,
		iconMode: IconModeNone,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	svc := cfg.service
	if svc == nil {
		if cfg.readOnly {
			svc = testServiceReadOnly()
		} else {
			svc = testService()
		}
	}

	m := NewModel(Options{Service: svc, IconMode: cfg.iconMode})
	m.view = cfg.view
	m.activeTab = cfg.tab
	m.width = cfg.width
	m.height = cfg.height

	for _, fn := range cfg.postInit {
		fn(&m)
	}

	return m
}

// ── Key Factories ───────────────────────────────────────────────────

// runeKey creates a tea.KeyPressMsg for a rune string (e.g., "j", "?", "G").
func runeKey(r string) tea.KeyPressMsg {
	runes := []rune(r)
	return tea.KeyPressMsg{Code: runes[0], Text: r}
}

// specialKey creates a tea.KeyPressMsg for a special key code (e.g., tea.KeyEsc).
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// ctrlKey creates a tea.KeyPressMsg for a ctrl+key combo (e.g., ctrlKey('d') for ctrl+d).
func ctrlKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
}

// shiftKey creates a tea.KeyPressMsg for a shift+key combo (e.g., shiftKey(tea.KeyTab) for shift+tab).
func shiftKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModShift}
}

// ── Dispatch Helpers ────────────────────────────────────────────────

// sendKey dispatches a tea.KeyPressMsg through Model.Update and asserts the
// returned value is a Model.
func sendKey(t *testing.T, m Model, key tea.KeyPressMsg) (Model, tea.Cmd) {
	t.Helper()
	result, cmd := m.Update(key)
	updated, ok := result.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", result)
	}
	return updated, cmd
}

// sendMsg dispatches any tea.Msg through Model.Update and asserts the
// returned value is a Model.
func sendMsg(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	result, cmd := m.Update(msg)
	updated, ok := result.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", result)
	}
	return updated, cmd
}

// ── Assertion Helpers ───────────────────────────────────────────────

// isQuitCmd checks whether a tea.Cmd produces a tea.QuitMsg.
func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	return ok
}

// assertRenderedLinesFitWidth checks that no ANSI-aware line exceeds width.
func assertRenderedLinesFitWidth(t *testing.T, output string, width int) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for i, line := range lines {
		if got := ansi.StringWidth(line); got > width {
			t.Fatalf("line %d width=%d exceeds maxWidth=%d: %q", i+1, got, width, line)
		}
	}
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ── Golden Test Helpers ─────────────────────────────────────────────

// stripForGolden removes ANSI escape codes and trailing whitespace from
// rendered output. Lipgloss often pads lines to full width with spaces;
// stripping trailing whitespace prevents golden file mismatches from
// invisible padding changes.
func stripForGolden(s string) string {
	s = ansi.Strip(s)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// ── Status Tab Helpers ──────────────────────────────────────────────

// findFirstFileRow finds the index of the first non-header row in changesRows.
func findFirstFileRow(t *testing.T, m Model) int {
	t.Helper()
	for i, row := range m.status.changesRows {
		if !row.isHeader {
			return i
		}
	}
	t.Fatal("no non-header row found in changesRows")
	return -1
}

// findSecondFileRow finds the index of the second non-header row.
func findSecondFileRow(t *testing.T, m Model) int {
	t.Helper()
	count := 0
	for i, row := range m.status.changesRows {
		if !row.isHeader {
			count++
			if count == 2 {
				return i
			}
		}
	}
	t.Fatal("fewer than 2 non-header rows in changesRows")
	return -1
}

// findFirstHeaderRow finds the index of the first header row.
func findFirstHeaderRow(t *testing.T, m Model) int {
	t.Helper()
	for i, row := range m.status.changesRows {
		if row.isHeader {
			return i
		}
	}
	t.Fatal("no header row found in changesRows")
	return -1
}

// findFirstSectionFileRow finds the first non-header row for a given section.
func findFirstSectionFileRow(t *testing.T, m Model, section changesSection) int {
	t.Helper()
	for i, row := range m.status.changesRows {
		if !row.isHeader && row.section == section {
			return i
		}
	}
	t.Fatalf("no non-header row found for section %d", section)
	return -1
}
