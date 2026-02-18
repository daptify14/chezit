package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- Status tab action menus ---

func driftAllowsReAdd(src, dest rune) bool {
	return src == 'M' || dest == 'M' || src == 'D'
}

func (m Model) canReAddDriftFile(f chezmoi.FileStatus) bool {
	if !driftAllowsReAdd(f.SourceStatus, f.DestStatus) {
		return false
	}
	return !m.status.templatePaths[f.Path]
}

func driftApplyLabel(f chezmoi.FileStatus) string {
	if f.IsScript() {
		return "Run Script"
	}
	return "Apply File"
}

func (m Model) driftFileByPath(path string) *chezmoi.FileStatus {
	for i := range m.status.files {
		if m.status.files[i].Path == path {
			return &m.status.files[i]
		}
	}
	return nil
}

func (m *Model) openStatusActionsMenu() {
	if m.status.selectionActive {
		m.openStatusSelectionActionsMenu()
		return
	}

	row := m.currentChangesRow()
	if row.isHeader {
		return
	}
	m.actions.items = nil

	switch row.section {
	case changesSectionDrift:
		if row.driftFile == nil {
			return
		}
		f := *row.driftFile
		m.actions.items = append(m.actions.items, chezmoiActionItem{label: "View Diff", action: chezmoiActionViewDiff})

		if driftAllowsReAdd(f.SourceStatus, f.DestStatus) {
			isTemplate := m.status.templatePaths[f.Path]
			m.actions.items = appendActionItem(
				m.actions.items,
				"Re-add to Source",
				chezmoiActionReAdd,
				"", !isTemplate,
				"template",
			)
		}

		m.actions.items = appendActionItem(
			m.actions.items,
			driftApplyLabel(f),
			chezmoiActionApplyFile,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
		m.actions.items = append(m.actions.items, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
		m.actions.items = appendActionItem(
			m.actions.items,
			"Apply All",
			chezmoiActionApplyAll,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
		m.actions.items = appendActionItem(
			m.actions.items,
			"Update (pull + apply)",
			chezmoiActionUpdate,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
		m.actions.items = appendActionItem(
			m.actions.items,
			"Refresh",
			chezmoiActionRefresh,
			"", true,
			"",
		)

	case changesSectionUnstaged:
		if row.gitFile == nil {
			return
		}
		m.actions.items = appendActionItem(
			m.actions.items,
			"Stage File",
			chezmoiActionGitStage,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
		canDiscard := row.gitFile.StatusCode != "U"
		m.actions.items = appendActionItem(
			m.actions.items,
			"Discard Changes",
			chezmoiActionGitDiscard,
			"", canDiscard && !m.service.IsReadOnly(),
			"read-only mode",
		)

	case changesSectionUnpushed:
		if len(m.status.unpushedCommits) == 0 {
			return
		}
		m.actions.items = appendActionItem(
			m.actions.items,
			"Undo Last Commit",
			chezmoiActionGitUndoCommit,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)

	default:
		return
	}

	m.actions.cursor = firstSelectableCursor(m.actions.items)
	m.actions.show = true
}

func (m *Model) openStatusSelectionActionsMenu() {
	selectedCount := m.selectedActionableCount()
	if selectedCount == 0 {
		m.ui.message = "No actionable files in selection"
		return
	}

	section, ok := m.statusSelectionSection()
	if !ok {
		m.ui.message = "No actionable files in selection"
		return
	}

	m.actions.items = nil

	switch section {
	case changesSectionDrift:
		paths := m.selectedReAddTargets()
		if len(paths) == 0 {
			m.ui.message = "No re-addable files in selection"
			return
		}
		m.actions.items = append(m.actions.items, chezmoiActionItem{
			label:  "Re-add selected",
			action: chezmoiActionReAdd,
		})
	case changesSectionUnstaged:
		_, unstagedPaths := m.selectedStageTargets()
		discardPaths := m.selectedDiscardTargets()
		if len(unstagedPaths) == 0 && len(discardPaths) == 0 {
			m.ui.message = "No actionable files in selection"
			return
		}
		if len(unstagedPaths) > 0 {
			m.actions.items = append(m.actions.items, chezmoiActionItem{
				label:  "Stage selected",
				action: chezmoiActionGitStage,
			})
		}
		if len(discardPaths) > 0 {
			m.actions.items = append(m.actions.items, chezmoiActionItem{
				label:  "Discard selected",
				action: chezmoiActionGitDiscardSelected,
			})
		}
	case changesSectionStaged:
		paths := m.selectedUnstageTargets()
		if len(paths) == 0 {
			m.ui.message = "No staged files in selection"
			return
		}
		m.actions.items = append(m.actions.items, chezmoiActionItem{
			label:  "Unstage selected",
			action: chezmoiActionGitUnstage,
		})
	default:
		m.ui.message = "No bulk actions for selected section"
		return
	}

	m.actions.cursor = firstSelectableCursor(m.actions.items)
	m.actions.show = true
}

func (m *Model) openDiffActionsMenu() {
	m.actions.items = nil

	switch m.diff.sourceSection {
	case changesSectionDrift:
		applyLabel := "Apply File"
		if file := m.driftFileByPath(m.diff.path); file != nil {
			if driftAllowsReAdd(file.SourceStatus, file.DestStatus) {
				isTemplate := m.status.templatePaths[m.diff.path]
				m.actions.items = appendActionItem(
					m.actions.items,
					"Re-add to Source",
					chezmoiActionReAdd,
					"", !isTemplate,
					"template",
				)
			}
			applyLabel = driftApplyLabel(*file)
		} else if (chezmoi.FileStatus{Path: m.diff.path}).IsScript() {
			applyLabel = "Run Script"
		}
		m.actions.items = appendActionItem(
			m.actions.items,
			applyLabel,
			chezmoiActionApplyFile,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
	case changesSectionUnstaged:
		m.actions.items = appendActionItem(
			m.actions.items,
			"Stage File",
			chezmoiActionGitStage,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
		m.actions.items = appendActionItem(
			m.actions.items,
			"Discard Changes",
			chezmoiActionGitDiscard,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
	case changesSectionUnpushed:
		m.actions.items = appendActionItem(
			m.actions.items,
			"Undo Last Commit",
			chezmoiActionGitUndoCommit,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
	case changesSectionStaged:
		m.actions.items = appendActionItem(
			m.actions.items,
			"Unstage File",
			chezmoiActionGitUnstage,
			"", !m.service.IsReadOnly(),
			"read-only mode",
		)
	}

	m.actions.cursor = firstSelectableCursor(m.actions.items)
	m.actions.show = true
}

func (m Model) executeStatusAction(action chezmoiAction) (tea.Model, tea.Cmd) {
	m.actions.show = false

	if actionRequiresWrite(action) && m.service.IsReadOnly() {
		m.ui.message = actionUnavailableMessage("read-only mode")
		return m, nil
	}

	switch action {
	case chezmoiActionViewDiff:
		row := m.currentChangesRow()
		if !row.isHeader && row.section == changesSectionDrift && row.driftFile != nil {
			m.diff.sourceSection = changesSectionDrift
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadDiffCmd(row.driftFile.Path))
		}

	case chezmoiActionReAdd:
		if m.status.selectionActive {
			paths := m.selectedReAddTargets()
			return m.executeBulkAction(paths, "No re-addable files in selection", m.reAddSelectionCmd(paths))
		}
		path := m.currentFilePath()
		if path != "" {
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.reAddCmd(path))
		}

	case chezmoiActionApplyFile:
		path := m.currentFilePath()
		if path != "" {
			m = m.showConfirmScreen(chezmoiActionApplyFile, "apply "+path)
		}
		return m, nil

	case chezmoiActionApplyAll:
		return m.showConfirmScreen(chezmoiActionApplyAll, "apply all changes to destination"), nil

	case chezmoiActionUpdate:
		return m.showConfirmScreen(chezmoiActionUpdate, "update (pull from remote and apply)"), nil

	case chezmoiActionRefresh:
		m.ui.loading = true
		m.status.loadingGit = true
		m.ui.message = ""
		m.nextGen()
		reloadCmds := m.reloadStatusAndGitCmds()
		if len(reloadCmds) == 0 {
			return m, nil
		}
		return m, tea.Batch(append([]tea.Cmd{m.ui.loadingSpinner.Tick}, reloadCmds...)...)

	case chezmoiActionGitStage:
		if m.status.selectionActive {
			driftPaths, unstagedPaths := m.selectedStageTargets()
			return m.executeBulkAction(
				append(driftPaths, unstagedPaths...),
				"No stageable files in selection",
				m.gitStageSelectionCmd(driftPaths, unstagedPaths),
			)
		}
		if m.diff.path != "" {
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitAddCmd(m.diff.path))
		}

	case chezmoiActionGitUnstage:
		if m.status.selectionActive {
			paths := m.selectedUnstageTargets()
			return m.executeBulkAction(paths, "No staged files in selection", m.gitUnstageSelectionCmd(paths))
		}
		if m.diff.path != "" {
			m.ui.busyAction = true
			return m, tea.Batch(m.ui.loadingSpinner.Tick, m.gitResetCmd(m.diff.path))
		}

	case chezmoiActionGitDiscard:
		path := m.diff.path
		if path != "" {
			m.overlays.confirmPath = path
			m = m.showConfirmScreen(chezmoiActionGitDiscard, "discard changes to "+shortenPath(path, m.targetPath))
		}
		return m, nil

	case chezmoiActionGitDiscardSelected:
		paths := m.selectedDiscardTargets()
		m.clearStatusSelection()
		if len(paths) == 0 {
			m.ui.message = "No discardable files in selection"
			return m, nil
		}
		m.overlays.confirmPaths = paths
		m = m.showConfirmScreen(chezmoiActionGitDiscardSelected, fmt.Sprintf("discard changes in %d selected files", len(paths)))
		return m, nil

	case chezmoiActionGitUndoCommit:
		return m.showConfirmScreen(chezmoiActionGitUndoCommit, "undo last commit (changes return to staged)"), nil
	}

	return m, nil
}

// actionRequiresWrite returns true for actions that mutate state and are blocked
// in read-only mode.
func actionRequiresWrite(action chezmoiAction) bool {
	switch action {
	case chezmoiActionReAdd,
		chezmoiActionGitStage,
		chezmoiActionGitUnstage,
		chezmoiActionApplyFile,
		chezmoiActionApplyAll,
		chezmoiActionUpdate,
		chezmoiActionGitDiscard,
		chezmoiActionGitDiscardSelected,
		chezmoiActionGitUndoCommit:
		return true
	}
	return false
}

// showConfirmScreen sets the confirm overlay for an action and transitions to
// ConfirmScreen. Callers set confirmPath/confirmPaths directly when needed.
func (m Model) showConfirmScreen(action chezmoiAction, label string) Model {
	m.overlays.confirmAction = action
	m.overlays.confirmLabel = label
	m.view = ConfirmScreen
	return m
}

// executeBulkAction handles the common pattern for bulk selection actions:
// clear selection, guard against empty targets, set busy, and batch the command.
func (m Model) executeBulkAction(paths []string, emptyMsg string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.clearStatusSelection()
	if len(paths) == 0 {
		m.ui.message = emptyMsg
		return m, nil
	}
	m.ui.busyAction = true
	m.ui.message = ""
	return m, tea.Batch(m.ui.loadingSpinner.Tick, cmd)
}

// --- Status state helpers ---

// updateCommandAvailability updates the available flag on commands based on current state.
// Apply is always available (chezmoi also runs scripts that don't appear in drift).
// Re-Add All is only available when there's file drift.
func (m *Model) updateCommandAvailability() {
	hasDrift := len(m.status.filteredFiles) > 0
	for i := range m.cmds.items {
		switch m.cmds.items[i].id {
		case chezmoiCmdReAddAll:
			m.cmds.items[i].available = hasDrift
		default:
			m.cmds.items[i].available = true
		}
	}
}

func (m *Model) buildChangesRows() {
	m.status.changesRows = nil

	// Incoming first: "what's arriving from remote"
	m.status.changesRows = append(m.status.changesRows, changesRow{isHeader: true, section: changesSectionIncoming})
	if !m.status.sectionCollapsed[changesSectionIncoming] {
		for i := range m.status.incomingCommits {
			m.status.changesRows = append(m.status.changesRows, changesRow{
				section: changesSectionIncoming,
				commit:  &m.status.incomingCommits[i],
			})
		}
	}

	// Local work pipeline: drift → unstaged → staged
	m.status.changesRows = append(m.status.changesRows, changesRow{isHeader: true, section: changesSectionDrift})
	if !m.status.sectionCollapsed[changesSectionDrift] {
		for i := range m.status.filteredFiles {
			m.status.changesRows = append(m.status.changesRows, changesRow{
				section:   changesSectionDrift,
				driftFile: &m.status.filteredFiles[i],
			})
		}
	}

	m.status.changesRows = append(m.status.changesRows, changesRow{isHeader: true, section: changesSectionUnstaged})
	if !m.status.sectionCollapsed[changesSectionUnstaged] {
		for i := range m.status.gitUnstagedFiles {
			m.status.changesRows = append(m.status.changesRows, changesRow{
				section: changesSectionUnstaged,
				gitFile: &m.status.gitUnstagedFiles[i],
			})
		}
	}

	m.status.changesRows = append(m.status.changesRows, changesRow{isHeader: true, section: changesSectionStaged})
	if !m.status.sectionCollapsed[changesSectionStaged] {
		for i := range m.status.gitStagedFiles {
			m.status.changesRows = append(m.status.changesRows, changesRow{
				section: changesSectionStaged,
				gitFile: &m.status.gitStagedFiles[i],
			})
		}
	}

	// Unpushed last: "what's departing to remote"
	m.status.changesRows = append(m.status.changesRows, changesRow{isHeader: true, section: changesSectionUnpushed})
	if !m.status.sectionCollapsed[changesSectionUnpushed] {
		for i := range m.status.unpushedCommits {
			m.status.changesRows = append(m.status.changesRows, changesRow{
				section: changesSectionUnpushed,
				commit:  &m.status.unpushedCommits[i],
			})
		}
	}

	if m.status.changesCursor >= len(m.status.changesRows) {
		m.status.changesCursor = max(0, len(m.status.changesRows)-1)
	}
	if len(m.status.changesRows) == 0 {
		m.status.selectionActive = false
		m.status.selectionAnchor = 0
		return
	}
	if m.status.selectionActive {
		m.status.selectionAnchor = clampStatusRowIndex(m.status.selectionAnchor, len(m.status.changesRows))
	}
}

func (m Model) currentChangesRow() changesRow {
	if m.status.changesCursor >= 0 && m.status.changesCursor < len(m.status.changesRows) {
		return m.status.changesRows[m.status.changesCursor]
	}
	return changesRow{}
}

func (m Model) currentFilePath() string {
	if m.view == DiffScreen && m.diff.path != "" {
		return m.diff.path
	}
	row := m.currentChangesRow()
	if !row.isHeader && row.section == changesSectionDrift && row.driftFile != nil {
		return row.driftFile.Path
	}
	return ""
}
