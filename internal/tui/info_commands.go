package tui

import tea "charm.land/bubbletea/v2"

// --- Info tab async command factory ---

// loadInfoSubViewCmd fetches content for the given Info sub-view index.
func (m Model) loadInfoSubViewCmd(view int) tea.Cmd {
	mgr := m.service
	format := m.info.format
	gen := m.gen
	return func() tea.Msg {
		var content string
		var err error
		switch view {
		case infoViewConfig:
			content, err = mgr.CatConfig()
		case infoViewFull:
			if format == "json" {
				content, err = mgr.DumpConfigJSON()
			} else {
				content, err = mgr.DumpConfig()
			}
		case infoViewData:
			if format == "json" {
				content, err = mgr.DataJSON()
			} else {
				content, err = mgr.Data()
			}
		case infoViewDoctor:
			content, err = mgr.Doctor()
		}
		return infoContentLoadedMsg{view: view, content: content, err: err, gen: gen}
	}
}
