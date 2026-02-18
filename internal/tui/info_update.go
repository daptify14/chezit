package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// infoSubViewState holds per-sub-view state for the Info tab.
type infoSubViewState struct {
	content       string
	lines         []string
	loaded        bool
	loading       bool
	viewport      viewport.Model
	viewportReady bool
	lastWidth     int
}

// ensureViewport creates or resizes the viewport to the given dimensions.
func (v *infoSubViewState) ensureViewport(width, height int) {
	if !v.viewportReady || v.lastWidth != width {
		v.viewport = viewport.New()
		v.viewport.SetWidth(width)
		v.viewport.SetHeight(height)
		v.viewportReady = true
		v.lastWidth = width
	}
	if v.viewport.Height() != height {
		v.viewport.SetHeight(height)
	}
	if v.viewport.Width() != width {
		v.viewport.SetWidth(width)
	}
}

// --- Info tab message handler ---

func (m Model) handleInfoContentLoaded(msg infoContentLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	view := &m.info.views[msg.view]
	view.loading = false
	if msg.err != nil {
		view.lines = []string{"Error: " + msg.err.Error()}
		view.loaded = true
		return m, nil
	}
	view.content = msg.content
	// Apply syntax highlighting based on sub-view type
	switch msg.view {
	case infoViewConfig:
		highlighted := highlightCode(msg.content, infoConfigFilenameHint(msg.content))
		view.lines = strings.Split(highlighted, "\n")
	case infoViewFull:
		ext := "yaml"
		if m.info.format == "json" {
			ext = "json"
		}
		highlighted := highlightCode(msg.content, "config."+ext)
		view.lines = strings.Split(highlighted, "\n")
	case infoViewData:
		ext := "yaml"
		if m.info.format == "json" {
			ext = "json"
		}
		highlighted := highlightCode(msg.content, "data."+ext)
		view.lines = strings.Split(highlighted, "\n")
	case infoViewDoctor:
		view.lines = strings.Split(msg.content, "\n")
	}
	if view.viewportReady {
		view.viewport.GotoTop()
	}
	view.loaded = true
	return m, nil
}

// --- Info tab key handler ---

func (m Model) handleInfoKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, ChezSharedKeys.Back):
		return m.escCmd()

	// Sub-view navigation
	case key.Matches(msg, ChezInfoKeys.Left):
		m.info.activeView--
		if m.info.activeView < 0 {
			m.info.activeView = infoViewCount - 1
		}
		m.ui.message = ""
		cmd := m.ensureInfoViewLoaded(m.info.activeView)
		return m, cmd
	case key.Matches(msg, ChezInfoKeys.Right):
		m.info.activeView++
		if m.info.activeView >= infoViewCount {
			m.info.activeView = 0
		}
		m.ui.message = ""
		cmd := m.ensureInfoViewLoaded(m.info.activeView)
		return m, cmd

	// Format toggle (only for Full and Data)
	case key.Matches(msg, ChezInfoKeys.Format):
		if m.info.activeView == infoViewFull || m.info.activeView == infoViewData {
			if m.info.format == "yaml" {
				m.info.format = "json"
			} else {
				m.info.format = "yaml"
			}
			// Clear and reload current sub-view with new format
			v := &m.info.views[m.info.activeView]
			v.loaded = false
			v.loading = true
			v.content = ""
			v.lines = nil
			if v.viewportReady {
				v.viewport.GotoTop()
			}
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadInfoSubViewCmd(m.info.activeView))
		}

	// Refresh
	case key.Matches(msg, ChezInfoKeys.Refresh):
		v := &m.info.views[m.info.activeView]
		v.loaded = false
		v.loading = true
		v.content = ""
		v.lines = nil
		if v.viewportReady {
			v.viewport.GotoTop()
		}
		m.ui.message = ""
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadInfoSubViewCmd(m.info.activeView))

	// Scroll — delegate to viewport via shared helper
	default:
		m = m.syncInfoViewportContent()
		if scrollViewport(&m.info.views[m.info.activeView].viewport, msg) {
			return m, nil
		}
	}
	return m, nil
}

// syncInfoViewportContent ensures the info viewport is ready and content is synced before scrolling.
func (m Model) syncInfoViewportContent() Model {
	view := &m.info.views[m.info.activeView]
	if len(view.lines) == 0 {
		return m
	}
	listHeight := m.infoViewHeight()
	view.ensureViewport(m.effectiveWidth(), listHeight)
	content := m.preRenderInfoContent(m.effectiveWidth() - 4)
	currentOffset := view.viewport.YOffset()
	view.viewport.SetContent(content)
	view.viewport.SetYOffset(currentOffset)
	return m
}

// ensureInfoViewLoaded triggers a load for the sub-view if not already loaded.
func (m *Model) ensureInfoViewLoaded(idx int) tea.Cmd {
	view := &m.info.views[idx]
	if view.loaded || view.loading {
		return nil
	}
	view.loading = true
	return tea.Batch(m.ui.loadingSpinner.Tick, m.loadInfoSubViewCmd(idx))
}

// preloadInfoViewsCmd triggers lazy loads for all Info sub-views that are not loaded yet.
func (m *Model) preloadInfoViewsCmd() tea.Cmd {
	var cmds []tea.Cmd
	for i := range infoViewCount {
		if cmd := m.ensureInfoViewLoaded(i); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// infoConfigFilenameHint inspects cat-config output to guess the file format for highlighting.
func infoConfigFilenameHint(content string) string {
	for _, line := range strings.SplitN(content, "\n", 10) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "{") {
			return "config.json"
		}
		if strings.HasPrefix(trimmed, "[") || strings.Contains(trimmed, "=") {
			return "config.toml"
		}
		if strings.Contains(trimmed, ":") {
			return "config.yaml"
		}
		break
	}
	return "config.toml" // chezmoi default
}
