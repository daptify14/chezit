package tui

import (
	"fmt"
	"os"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// --- Status tab message handlers ---

func (m Model) handleStatusLoaded(msg chezmoiStatusLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.ui.loading = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
	} else {
		m.status.files = msg.files
		m.status.filteredFiles = msg.files
		m.applyChezmoiFilter()
		m.annotateTemplateFiles()
		m.buildChangesRows()
		m.updateCommandAvailability()
	}
	if m.allLandingStatsLoaded() && !m.landing.statsReady {
		return m, tea.Batch(nil, debounceLandingReadyCmd())
	}
	if m.panel.shouldShow(m.width) && m.activeTabName() == "Status" {
		var cmd tea.Cmd
		m, cmd = m.panelLoadForChanges()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleGitStatusLoaded(msg chezmoiGitStatusLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.status.loadingGit = false
	if msg.err != nil {
		if m.activeTabName() == "Status" {
			m.ui.message = "Error: " + msg.err.Error()
		}
	} else {
		m.status.gitStagedFiles = msg.staged
		m.status.gitUnstagedFiles = msg.unstaged
		info := msg.info
		// Preserve previously known branch info when a refresh returns an
		// empty branch (for example, when branch lookup transiently fails).
		if info.Branch == "" && m.status.gitInfo.Branch != "" {
			info = m.status.gitInfo
		}
		m.status.gitInfo = info
		m.buildChangesRows()
		m.updateCommandAvailability()
	}
	if m.allLandingStatsLoaded() && !m.landing.statsReady {
		return m, tea.Batch(nil, debounceLandingReadyCmd())
	}
	return m, nil
}

func (m Model) handleGitActionDone(msg chezmoiGitActionDoneMsg) (tea.Model, tea.Cmd) {
	m.ui.busyAction = false
	if msg.err != nil {
		m.ui.message = "Error: " + msg.err.Error()
		return m, nil
	}
	m.ui.message = msg.message
	if msg.action == chezmoiActionGitStage {
		m.status.changesCursor++
	}
	return m, tea.Batch(m.loadGitStatusCmd(), sendRefreshMsg())
}

func (m Model) handleGitCommitsLoaded(msg chezmoiGitCommitsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	if msg.err != nil {
		// Non-fatal: leave commits empty
		return m, nil
	}
	m.status.unpushedCommits = msg.unpushed
	m.status.incomingCommits = msg.incoming
	m.buildChangesRows()
	return m, nil
}

func (m Model) handleGitFetchDone(msg chezmoiGitFetchDoneMsg) (tea.Model, tea.Cmd) {
	m.status.fetchInProgress = false
	if msg.err != nil {
		m.ui.message = "Fetch error: " + msg.err.Error()
		return m, nil
	}
	m.status.lastFetchTime = time.Now()
	m.ui.message = "fetch complete"
	return m, tea.Batch(m.loadGitCommitsCmd(), m.loadGitStatusCmd())
}

func (m Model) handleTemplatePathsLoaded(msg templatePathsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen {
		return m, nil
	}
	m.status.templatePaths = msg.paths
	m.annotateTemplateFiles()
	m.buildChangesRows()
	return m, nil
}

// --- Status tab key handlers ---

func (m Model) handleStatusKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	row := m.currentChangesRow()
	if msg.IsRepeat && isStatusRepeatActionKey(msg) {
		return m, nil
	}

	if isStatusShiftUp(msg) || isStatusShiftDown(msg) {
		return m.handleStatusShiftSelection(msg)
	}

	switch {
	case key.Matches(msg, ChezSharedKeys.Back):
		m.clearStatusSelection()
		return m.escCmd()
	case key.Matches(msg, ChezSharedKeys.Up):
		m.clearStatusSelection()
		next := moveCursorUp(m.status.changesCursor, navigationStepForKey(msg))
		return m.moveStatusCursor(next)
	case key.Matches(msg, ChezSharedKeys.Down):
		m.clearStatusSelection()
		next := moveCursorDown(m.status.changesCursor, len(m.status.changesRows), navigationStepForKey(msg))
		return m.moveStatusCursor(next)
	case key.Matches(msg, ChezSharedKeys.Enter):
		return m.handleStatusEnter(row)
	case key.Matches(msg, ChezChangesKeys.Stage):
		return m.handleStatusStage(row)
	case key.Matches(msg, ChezChangesKeys.Unstage):
		return m.handleStatusUnstage(row)
	case key.Matches(msg, ChezChangesKeys.Discard):
		return m.handleStatusDiscard(row)
	case key.Matches(msg, ChezChangesKeys.StageAll):
		return m.handleStatusStageAll()
	case key.Matches(msg, ChezChangesKeys.UnstageAll):
		return m.handleStatusUnstageAll()
	case key.Matches(msg, ChezChangesKeys.Commit):
		return m.handleStatusCommit()
	case key.Matches(msg, ChezChangesKeys.Push):
		return m.handleStatusPush()
	case key.Matches(msg, ChezChangesKeys.Fetch):
		return m.handleStatusFetch(row)
	case key.Matches(msg, ChezChangesKeys.Pull):
		return m.handleStatusPull(row)
	case key.Matches(msg, ChezChangesKeys.Edit):
		return m.handleStatusEdit(row)
	case key.Matches(msg, ChezChangesKeys.Actions):
		return m.handleStatusActions(row)
	case key.Matches(msg, ChezChangesKeys.Refresh):
		return m.handleStatusRefresh()
	}
	return m, nil
}

func isStatusRepeatActionKey(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, ChezChangesKeys.Stage) ||
		key.Matches(msg, ChezChangesKeys.Unstage) ||
		key.Matches(msg, ChezChangesKeys.Discard) ||
		key.Matches(msg, ChezChangesKeys.StageAll) ||
		key.Matches(msg, ChezChangesKeys.UnstageAll) ||
		key.Matches(msg, ChezChangesKeys.Commit) ||
		key.Matches(msg, ChezChangesKeys.Push) ||
		key.Matches(msg, ChezChangesKeys.Fetch) ||
		key.Matches(msg, ChezChangesKeys.Pull) ||
		key.Matches(msg, ChezChangesKeys.Edit)
}

func (m Model) handleStatusShiftSelection(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.beginStatusSelectionIfNeeded()
	step := navigationStepForKey(msg)
	var next int
	if isStatusShiftUp(msg) {
		next = moveCursorUp(m.status.changesCursor, step)
	} else {
		next = moveCursorDown(m.status.changesCursor, len(m.status.changesRows), step)
	}
	next = m.clampStatusSelectionCursor(next)
	return m.moveStatusCursor(next)
}

func (m Model) moveStatusCursor(next int) (tea.Model, tea.Cmd) {
	if next == m.status.changesCursor {
		return m, nil
	}
	m.status.changesCursor = next
	if m.panel.shouldShow(m.width) {
		var panelCmd tea.Cmd
		m, panelCmd = m.panelLoadForChanges()
		return m, panelCmd
	}
	return m, nil
}

func (m Model) handleStatusEnter(row changesRow) (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if row.isHeader {
		m.status.sectionCollapsed[row.section] = !m.status.sectionCollapsed[row.section]
		m.buildChangesRows()
		return m, nil
	}
	switch row.section {
	case changesSectionDrift:
		if row.driftFile != nil {
			m.diff.sourceSection = changesSectionDrift
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadDiffCmd(row.driftFile.Path))
		}
	case changesSectionUnstaged:
		if row.gitFile != nil {
			m.diff.sourceSection = changesSectionUnstaged
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadGitDiffCmd(row.gitFile.Path, false))
		}
	case changesSectionStaged:
		if row.gitFile != nil {
			m.diff.sourceSection = changesSectionStaged
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadGitDiffCmd(row.gitFile.Path, true))
		}
	case changesSectionUnpushed, changesSectionIncoming:
		if row.commit != nil {
			m.diff.sourceSection = row.section
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, func() tea.Msg {
				content, err := m.service.GitShow(row.commit.Hash)
				return chezmoiDiffLoadedMsg{path: row.commit.Hash, diff: content, err: err}
			})
		}
	}
	return m, nil
}

func (m Model) handleStatusStage(row changesRow) (tea.Model, tea.Cmd) {
	if m.status.selectionActive {
		driftPaths, unstagedPaths := m.selectedStageTargets()
		total := len(driftPaths) + len(unstagedPaths)
		m.clearStatusSelection()
		if total == 0 {
			m.ui.message = "No stageable files in selection"
			return m, nil
		}
		m.ui.busyAction = true
		m.ui.message = ""
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitStageSelectionCmd(driftPaths, unstagedPaths))
	}
	if row.isHeader {
		return m, nil
	}
	switch row.section {
	case changesSectionDrift:
		if row.driftFile != nil {
			if !m.canReAddDriftFile(*row.driftFile) {
				m.ui.message = "Re-add unavailable for this drift entry"
				return m, nil
			}
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.reAddCmd(row.driftFile.Path))
		}
	case changesSectionUnstaged:
		if row.gitFile != nil {
			m.ui.busyAction = true
			m.ui.message = ""
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitAddCmd(row.gitFile.Path))
		}
	}
	return m, nil
}

func (m Model) handleStatusUnstage(row changesRow) (tea.Model, tea.Cmd) {
	if m.status.selectionActive {
		paths := m.selectedUnstageTargets()
		m.clearStatusSelection()
		if len(paths) == 0 {
			m.ui.message = "No staged files in selection"
			return m, nil
		}
		m.ui.busyAction = true
		m.ui.message = ""
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitUnstageSelectionCmd(paths))
	}
	if !row.isHeader && row.section == changesSectionStaged && row.gitFile != nil {
		m.ui.busyAction = true
		m.ui.message = ""
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitResetCmd(row.gitFile.Path))
	}
	return m, nil
}

func (m Model) handleStatusDiscard(row changesRow) (tea.Model, tea.Cmd) {
	if m.service.IsReadOnly() {
		return m, nil
	}
	if m.status.selectionActive {
		paths := m.selectedDiscardTargets()
		m.clearStatusSelection()
		if len(paths) == 0 {
			m.ui.message = "No discardable files in selection"
			return m, nil
		}
		m.overlays.confirmAction = chezmoiActionGitDiscardSelected
		m.overlays.confirmLabel = fmt.Sprintf("discard changes in %d selected files", len(paths))
		m.overlays.confirmPaths = paths
		m.view = ConfirmScreen
		return m, nil
	}
	switch row.section {
	case changesSectionUnstaged:
		if !row.isHeader && row.gitFile != nil && row.gitFile.StatusCode != "U" {
			m.overlays.confirmAction = chezmoiActionGitDiscard
			m.overlays.confirmLabel = "discard changes to " + shortenPath(row.gitFile.Path, m.targetPath)
			m.overlays.confirmPath = row.gitFile.Path
			m.view = ConfirmScreen
		}
	case changesSectionUnpushed:
		if len(m.status.unpushedCommits) > 0 {
			m.overlays.confirmAction = chezmoiActionGitUndoCommit
			m.overlays.confirmLabel = "undo last commit (changes return to staged)"
			m.view = ConfirmScreen
		}
	}
	return m, nil
}

func (m Model) handleStatusStageAll() (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if len(m.status.gitUnstagedFiles) > 0 {
		m.overlays.confirmAction = chezmoiActionGitStageAll
		m.overlays.confirmLabel = fmt.Sprintf("stage all %d unstaged files", len(m.status.gitUnstagedFiles))
		m.view = ConfirmScreen
	}
	return m, nil
}

func (m Model) handleStatusUnstageAll() (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if len(m.status.gitStagedFiles) > 0 {
		m.overlays.confirmAction = chezmoiActionGitUnstageAll
		m.overlays.confirmLabel = fmt.Sprintf("unstage all %d staged files", len(m.status.gitStagedFiles))
		m.view = ConfirmScreen
	}
	return m, nil
}

func (m Model) handleStatusCommit() (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if len(m.status.gitStagedFiles) > 0 {
		cmd := m.openCommitScreen()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleStatusPush() (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	m.overlays.confirmAction = chezmoiActionPush
	m.overlays.confirmLabel = "push committed changes to remote"
	m.view = ConfirmScreen
	return m, nil
}

func (m Model) handleStatusFetch(row changesRow) (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if row.section == changesSectionIncoming {
		switch {
		case m.status.fetchInProgress:
			m.ui.message = "fetch already in progress"
		case time.Since(m.status.lastFetchTime) < 5*time.Minute:
			elapsed := time.Since(m.status.lastFetchTime).Round(time.Second)
			m.ui.message = fmt.Sprintf("fetch cooldown (last fetch %s ago)", elapsed)
		default:
			m.status.fetchInProgress = true
			m.ui.message = "fetching..."
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitFetchCmd())
		}
	}
	return m, nil
}

func (m Model) handleStatusPull(row changesRow) (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if row.section == changesSectionIncoming && !m.service.IsReadOnly() {
		m.overlays.confirmAction = chezmoiActionPull
		m.overlays.confirmLabel = "pull changes from remote"
		m.view = ConfirmScreen
	}
	return m, nil
}

func (m Model) handleStatusEdit(row changesRow) (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	if row.isHeader || m.service.IsReadOnly() {
		return m, nil
	}
	path := statusEditablePath(row)
	if path != "" {
		return m, m.editSourceCmd(path)
	}
	return m, nil
}

func statusEditablePath(row changesRow) string {
	switch row.section {
	case changesSectionDrift:
		if row.driftFile != nil {
			return row.driftFile.Path
		}
	case changesSectionUnstaged, changesSectionStaged:
		if row.gitFile != nil {
			return row.gitFile.Path
		}
	}
	return ""
}

func (m Model) handleStatusActions(row changesRow) (tea.Model, tea.Cmd) {
	if m.status.selectionActive {
		m.openStatusActionsMenu()
		return m, nil
	}
	m.clearStatusSelection()
	switch {
	case row.isHeader:
		// no actions on headers
	case row.section == changesSectionDrift,
		row.section == changesSectionUnstaged,
		row.section == changesSectionUnpushed:
		m.openStatusActionsMenu()
	}
	return m, nil
}

func (m Model) handleStatusRefresh() (tea.Model, tea.Cmd) {
	m.clearStatusSelection()
	m.ui.loading = true
	m.status.loadingGit = true
	m.ui.message = ""
	m.panel.clearCache()
	m.nextGen()
	reloadCmds := m.reloadStatusAndGitCmds()
	return m, tea.Batch(append([]tea.Cmd{m.ui.loadingSpinner.Tick}, reloadCmds...)...)
}

// --- Confirm dialog key handler ---

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, ChezConfirmKeys.Confirm):
		m.view = StatusScreen
		action := m.overlays.confirmAction
		savedPath := m.overlays.confirmPath
		savedPaths := m.overlays.confirmPaths
		m.overlays.confirmAction = chezmoiActionNone
		m.overlays.confirmLabel = ""
		m.overlays.confirmPath = ""
		m.overlays.confirmPaths = nil
		switch action {
		case chezmoiActionUpdate:
			return m, m.updateCmd()
		case chezmoiActionApplyAll:
			return m, m.applyAllCmd()
		case chezmoiActionApplyFile:
			path := m.currentFilePath()
			if path != "" {
				return m, m.applyFileCmd(path)
			}
		case chezmoiActionForgetFile:
			if savedPath != "" {
				m.ui.busyAction = true
				return m, tea.Batch(m.ui.loadingSpinner.Tick, m.forgetFileCmd(savedPath))
			}
		case chezmoiActionApplyManaged:
			if savedPath != "" {
				return m, m.applyFileCmd(savedPath)
			}
		case chezmoiActionPush:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.pushCmd())
		case chezmoiActionPull:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitPullCmd())
		case chezmoiActionGitStageAll:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitAddAllCmd())
		case chezmoiActionGitUnstageAll:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitResetAllCmd())
		case chezmoiActionRefresh:
			cmd := m.service.ApplyRefreshCmd()
			wrapped := wrapWithPressEnter(cmd)
			return m, execCmdOrUnsupported(chezmoiActionRefresh, wrapped, "chezmoi: refresh not supported")
		case chezmoiActionInit:
			cmd := m.service.InitCmd()
			wrapped := wrapWithPressEnter(cmd)
			return m, execCmdOrUnsupported(chezmoiActionInit, wrapped, "chezmoi: init not supported")
		case chezmoiActionReAdd:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, func() tea.Msg {
				output, err := m.service.ReAddAll()
				return chezmoiCapturedOutputMsg{action: chezmoiActionReAdd, label: "chezmoi re-add", output: output, err: err}
			})
		case chezmoiActionArchive:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, func() tea.Msg {
				outputPath, err := m.service.Archive()
				if err != nil {
					return chezmoiArchiveDoneMsg{path: outputPath, size: -1, err: err}
				}
				size := int64(-1)
				if info, statErr := os.Stat(outputPath); statErr == nil {
					size = info.Size()
				}
				return chezmoiArchiveDoneMsg{path: outputPath, size: size}
			})
		case chezmoiActionGitDiscard:
			if savedPath != "" {
				m.ui.busyAction = true
				return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitCheckoutCmd(savedPath))
			}
		case chezmoiActionGitDiscardSelected:
			if len(savedPaths) > 0 {
				m.ui.busyAction = true
				return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitDiscardSelectionCmd(savedPaths))
			}
		case chezmoiActionGitUndoCommit:
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitSoftResetCmd())
		}
		return m, nil
	case key.Matches(msg, ChezConfirmKeys.Cancel):
		m.view = StatusScreen
		m.overlays.confirmAction = chezmoiActionNone
		m.overlays.confirmLabel = ""
		m.overlays.confirmPath = ""
		m.overlays.confirmPaths = nil
		return m, nil
	}
	return m, nil
}
