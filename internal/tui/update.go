package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Update implements tea.Model by dispatching messages to the appropriate handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.logMsg(msg)

	var cmd tea.Cmd

	if m.startupErr != nil {
		keyMsg, ok := msg.(tea.KeyPressMsg)
		if !ok {
			return m, nil
		}
		if key.Matches(keyMsg, ChezSharedKeys.Back) ||
			key.Matches(keyMsg, ChezSharedKeys.Quit) ||
			key.Matches(keyMsg, ChezSharedKeys.Enter) {
			return m, tea.Quit
		}
		return m, nil
	}

	// Terminal background detection is cross-cutting and must be processed
	// before view-specific routing (including commit-form routing).
	if bgMsg, ok := msg.(tea.BackgroundColorMsg); ok {
		SetTheme(ThemeForBackground(bgMsg.IsDark()))
		m.restyleFilterInputForTheme()
		return m, nil
	}

	// Route all messages to huh forms when commit view is active.
	if m.view == CommitScreen {
		return m.handleCommitUpdate(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		tab := m.activeTabName()
		if m.panel.shouldShow(m.width) && (tab == "Status" || tab == "Files") {
			m = m.syncPanelViewportContent()
			m, cmd = m.panelLoadForCurrentTab()
			return m, cmd
		}
		return m, nil
	case spinner.TickMsg:
		if m.isAnyLoading() {
			m.ui.loadingSpinner, cmd = m.ui.loadingSpinner.Update(msg)
			return m, cmd
		}

	// Status tab messages
	case chezmoiStatusLoadedMsg:
		return m.handleStatusLoaded(msg)
	case chezmoiGitStatusLoadedMsg:
		return m.handleGitStatusLoaded(msg)
	case chezmoiGitActionDoneMsg:
		return m.handleGitActionDone(msg)
	case chezmoiGitCommitsLoadedMsg:
		return m.handleGitCommitsLoaded(msg)
	case chezmoiGitFetchDoneMsg:
		return m.handleGitFetchDone(msg)
	case templatePathsLoadedMsg:
		return m.handleTemplatePathsLoaded(msg)

	// Files tab messages
	case chezmoiManagedLoadedMsg:
		return m.handleManagedLoaded(msg)
	case chezmoiIgnoredLoadedMsg:
		return m.handleIgnoredLoaded(msg)
	case chezmoiUnmanagedLoadedMsg:
		return m.handleUnmanagedLoaded(msg)
	case filesSearchDebouncedMsg:
		return m.handleSearchDebounced(msg)
	case filesSearchCompletedMsg:
		return m.handleSearchCompleted(msg)
	case opaqueDirPopulatedMsg:
		return m.handleOpaqueDirPopulated(msg)

	// Info tab messages
	case infoContentLoadedMsg:
		return m.handleInfoContentLoaded(msg)

	// Landing
	case landingStatsReadyMsg:
		m.landing.statsReady = true
		return m, nil

	// Cross-cutting messages
	case chezmoiDiffLoadedMsg:
		return m.handleDiffLoaded(msg)
	case chezmoiActionDoneMsg:
		return m.handleActionDone(msg)
	case chezmoiForgetDoneMsg:
		return m.handleForgetDone(msg)
	case chezmoiAddDoneMsg:
		return m.handleAddDone(msg)
	case chezmoiSourceContentMsg:
		return m.handleSourceContent(msg)
	case chezmoiCapturedOutputMsg:
		return m.handleCapturedOutput(msg)
	case chezmoiArchiveDoneMsg:
		return m.handleArchiveDone(msg)
	case sourceDirResolvedMsg:
		return m.handleSourceDirResolved(msg)
	case chezmoiExecDoneMsg:
		return m.handleExecDone(msg)
	case panelContentLoadedMsg:
		return m.handlePanelContentLoaded(msg)

	// Input messages
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// --- Root key gate ---

func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if !m.filterInput.Focused() && key.Matches(msg, ChezSharedKeys.Mouse) {
		m.toggleMouseCapture()
		return m, nil
	}

	if m.view == LandingScreen {
		return m.handleLandingKeys(msg)
	}

	if m.view == ConfirmScreen {
		return m.handleConfirmKeys(msg)
	}

	if m.overlays.showHelp {
		maxScroll := m.helpOverlayMaxScroll()
		pageStep := helpOverlayPageStep(m.width, m.height)
		switch {
		case key.Matches(msg, ChezHelpOverlayKeys.Close):
			m.overlays.showHelp = false
			m.overlays.helpScroll = 0
		case key.Matches(msg, ChezSharedKeys.Up):
			m.overlays.helpScroll = max(0, m.overlays.helpScroll-1)
		case key.Matches(msg, ChezSharedKeys.Down):
			m.overlays.helpScroll = min(maxScroll, m.overlays.helpScroll+1)
		case key.Matches(msg, ChezScrollKeys.HalfUp), key.Matches(msg, ChezScrollKeys.PageUp):
			m.overlays.helpScroll = max(0, m.overlays.helpScroll-pageStep)
		case key.Matches(msg, ChezScrollKeys.HalfDown), key.Matches(msg, ChezScrollKeys.PageDown):
			m.overlays.helpScroll = min(maxScroll, m.overlays.helpScroll+pageStep)
		case key.Matches(msg, ChezSharedKeys.Home):
			m.overlays.helpScroll = 0
		case key.Matches(msg, ChezSharedKeys.End):
			m.overlays.helpScroll = maxScroll
		}
		return m, nil
	}

	if m.actions.show {
		switch {
		case key.Matches(msg, ChezActionMenuKeys.Close):
			m.actions.show = false
			m.ui.message = ""
			return m, nil
		case key.Matches(msg, ChezActionMenuKeys.Up):
			m.actions.cursor = nextSelectableCursor(m.actions.items, m.actions.cursor, -1)
			return m, nil
		case key.Matches(msg, ChezActionMenuKeys.Down):
			m.actions.cursor = nextSelectableCursor(m.actions.items, m.actions.cursor, 1)
			return m, nil
		case key.Matches(msg, ChezActionMenuKeys.Select):
			if m.actions.cursor < len(m.actions.items) {
				item := m.actions.items[m.actions.cursor]
				if isChezmoiActionSelectable(item) {
					return m.executeStatusAction(item.action)
				}
				if item.disabled {
					m.ui.message = actionUnavailableMessage(item.unavailableReason)
				}
			}
			return m, nil
		}
		return m, nil
	}

	if m.filterInput.Focused() {
		switch {
		case key.Matches(msg, ChezSharedKeys.Filter):
			return m, nil
		case key.Matches(msg, ChezFilterKeys.Cancel):
			if m.filterInput.Value() != "" {
				m.filterInput.SetValue("")
				m.applyActiveFilter()
				m.resetFilesSearch(true)
			} else {
				m.filterInput.Blur()
			}
			return m, nil
		case key.Matches(msg, ChezFilterKeys.Confirm):
			if m.activeTabName() == "Files" &&
				m.filterInput.Value() != "" &&
				(m.filesTab.viewMode == managedViewUnmanaged || m.filesTab.viewMode == managedViewAll) {
				m.pauseFilesSearch()
				m.applyManagedFilter()
			}
			m.filterInput.Blur()
			return m, nil
		default:
			m.filterInput, _ = m.filterInput.Update(msg)
			m.applyActiveFilter()
			if cmd := m.triggerFilesSearchIfNeeded(); cmd != nil {
				return m, cmd
			}
			return m, nil
		}
	}

	tab := m.activeTabName()
	if key.Matches(msg, ChezSharedKeys.Filter) && m.view == StatusScreen && (tab == "Status" || tab == "Files") {
		// Defensively reinitialize before focus to avoid nil-cursor panics
		// if the textinput model was ever left zero-valued.
		m.filterInput = newFilterInput()
		m.filterInput.Focus()
		return m, textinput.Blink
	}

	if m.view == StatusScreen && !m.overlays.showViewPicker && !m.overlays.showFilterOverlay {
		switch {
		case key.Matches(msg, ChezSharedKeys.TabNext):
			tabCmd := m.switchTab((m.activeTab + 1) % len(m.tabNames))
			return m, tabCmd
		case key.Matches(msg, ChezSharedKeys.TabPrev):
			tabCmd := m.switchTab((m.activeTab - 1 + len(m.tabNames)) % len(m.tabNames))
			return m, tabCmd
		case key.Matches(msg, ChezSharedKeys.Tab1):
			if len(m.tabNames) > 0 {
				tabCmd := m.switchTab(0)
				return m, tabCmd
			}
		case key.Matches(msg, ChezSharedKeys.Tab2):
			if len(m.tabNames) > 1 {
				tabCmd := m.switchTab(1)
				return m, tabCmd
			}
		case key.Matches(msg, ChezSharedKeys.Tab3):
			if len(m.tabNames) > 2 {
				tabCmd := m.switchTab(2)
				return m, tabCmd
			}
		case key.Matches(msg, ChezSharedKeys.Tab4):
			if len(m.tabNames) > 3 {
				tabCmd := m.switchTab(3)
				return m, tabCmd
			}
		}
	}

	switch {
	case key.Matches(msg, ChezSharedKeys.Help):
		m.overlays.showHelp = true
		m.overlays.helpScroll = 0
		return m, nil
	case key.Matches(msg, ChezSharedKeys.Quit):
		return m, tea.Quit
	}

	if m.view == DiffScreen {
		return m.handleDiffKeys(msg)
	}

	// Panel toggle and focus switching (Status/Files tabs only)
	tab = m.activeTabName()
	if m.view == StatusScreen && (tab == "Status" || tab == "Files") {
		if key.Matches(msg, ChezPanelKeys.Toggle) {
			m.panel.toggle(m.width)
			if m.panel.shouldShow(m.width) {
				var panelCmd tea.Cmd
				m, panelCmd = m.panelLoadForCurrentTab()
				return m, panelCmd
			}
			// Panel hidden: reset focus to list
			m.panel.focusZone = panelFocusList
			return m, nil
		}

		if m.panel.shouldShow(m.width) {
			if m.panel.focusZone == panelFocusList {
				if newM, cmd, handled := m.handlePanelModeKeysFromList(msg); handled {
					return newM, cmd
				}
			}
			if key.Matches(msg, ChezPanelKeys.FocusPanel) && m.panel.focusZone == panelFocusList {
				// 'l' is used for tree expand on Files tab — let it fall through
				if msg.String() != "l" || tab != "Files" || !m.filesTab.treeView {
					m.panel.focusZone = panelFocusPanel
					return m, nil
				}
			}

			// Route keys to panel when it has focus
			if m.panel.focusZone == panelFocusPanel {
				newM, cmd, handled := m.handlePanelKeys(msg)
				if handled {
					return newM, cmd
				}
			}
		}
	}

	switch m.activeTabName() {
	case "Status":
		return m.handleStatusKeys(msg)
	case "Files":
		return m.handleFilesKeys(msg)
	case "Info":
		return m.handleInfoKeys(msg)
	case "Commands":
		return m.handleCommandsKeys(msg)
	}

	return m, nil
}

func (m Model) handlePanelModeKeysFromList(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, ChezPanelKeys.ContentMode):
		if m.panel.contentMode == panelModeDiff {
			m.panel.contentMode = panelModeContent
		} else {
			m.panel.contentMode = panelModeDiff
		}
		updated, cmd := m.panelLoadForCurrentTab()
		return updated, cmd, true
	default:
		return m, nil, false
	}
}

// --- Tab Switching ---

func (m *Model) switchTab(tab int) tea.Cmd {
	if tab < 0 || tab >= len(m.tabNames) {
		return nil
	}
	m.activeTab = tab
	m.actions.show = false
	m.ui.message = ""
	m.filterInput.SetValue("")
	m.filterInput.Blur()
	m.resetFilesSearch(true)
	m.applyActiveFilter()
	m.clearStatusSelection()
	// Reset panel for new tab
	m.panel.resetForTab(m.tabNames[tab])
	m.panel.focusZone = panelFocusList

	return m.loadDeferredForTab(m.tabNames[tab])
}
