package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- Status / Git async command factories ---

func (m Model) loadStatusCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		files, err := m.service.Status()
		return chezmoiStatusLoadedMsg{files: files, err: err, gen: gen}
	}
}

func (m Model) loadTemplatePathsCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		files, err := m.service.ManagedFilesWithFilter(chezmoi.EntryFilter{
			Include: []chezmoi.EntryType{chezmoi.EntryTemplates},
		})
		if err != nil {
			// Non-fatal: template detection is best-effort
			return templatePathsLoadedMsg{gen: gen}
		}
		paths := make(map[string]bool, len(files))
		for _, f := range files {
			paths[f] = true
		}
		return templatePathsLoadedMsg{paths: paths, gen: gen}
	}
}

func (m *Model) annotateTemplateFiles() {
	if m.status.templatePaths == nil {
		return
	}
	for i := range m.status.files {
		m.status.files[i].IsTemplate = m.status.templatePaths[m.status.files[i].Path]
	}
	for i := range m.status.filteredFiles {
		m.status.filteredFiles[i].IsTemplate = m.status.templatePaths[m.status.filteredFiles[i].Path]
	}
}

func (m Model) loadDiffCmd(path string) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.service.Diff(path)
		return chezmoiDiffLoadedMsg{path: path, diff: diff, err: err}
	}
}

func (m Model) reAddCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.ReAdd(path); err != nil {
			return chezmoiActionDoneMsg{action: chezmoiActionReAdd, err: err}
		}
		return chezmoiActionDoneMsg{action: chezmoiActionReAdd, message: "re-added " + shortenPath(path, m.targetPath)}
	}
}

func (m Model) reAddSelectionCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		for _, path := range paths {
			if err := m.service.ReAdd(path); err != nil {
				return chezmoiActionDoneMsg{
					action: chezmoiActionReAdd,
					err:    fmt.Errorf("re-add %s: %w", shortenPath(path, m.targetPath), err),
				}
			}
		}
		return chezmoiActionDoneMsg{
			action:  chezmoiActionReAdd,
			message: fmt.Sprintf("re-added %d selected files", len(paths)),
		}
	}
}

func (m Model) applyFileCmd(path string) tea.Cmd {
	cmd := m.service.ApplyCmd(path)
	return execCmdOrUnsupported(chezmoiActionApplyFile, cmd, "chezmoi: apply not supported")
}

func (m Model) applyAllCmd() tea.Cmd {
	cmd := m.service.ApplyAllCmd()
	return execCmdOrUnsupported(chezmoiActionApplyAll, cmd, "chezmoi: apply not supported")
}

func (m Model) updateCmd() tea.Cmd {
	cmd := m.service.UpdateCmd()
	return execCmdOrUnsupported(chezmoiActionUpdate, cmd, "chezmoi: update not supported")
}

func (m Model) commitWithMsgCmd(message string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitCommit(message); err != nil {
			return chezmoiActionDoneMsg{action: chezmoiActionCommit, err: err}
		}
		return chezmoiActionDoneMsg{action: chezmoiActionCommit, message: "committed: " + message}
	}
}

func (m Model) pushCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitPush(); err != nil {
			return chezmoiActionDoneMsg{action: chezmoiActionPush, err: err}
		}
		return chezmoiActionDoneMsg{action: chezmoiActionPush, message: "pushed to remote"}
	}
}

func (m Model) loadGitCommitsCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		unpushedRaw, err := m.service.GitLogUnpushed()
		if err != nil {
			return chezmoiGitCommitsLoadedMsg{err: err, gen: gen}
		}
		incomingRaw, err := m.service.GitLogIncoming()
		if err != nil {
			return chezmoiGitCommitsLoadedMsg{err: err, gen: gen}
		}
		return chezmoiGitCommitsLoadedMsg{
			unpushed: chezmoi.ParseGitLogOneline(unpushedRaw),
			incoming: chezmoi.ParseGitLogOneline(incomingRaw),
			gen:      gen,
		}
	}
}

func (m Model) gitFetchCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		err := m.service.GitFetch()
		return chezmoiGitFetchDoneMsg{err: err, gen: gen}
	}
}

// gitPullCmd runs git pull. Returns chezmoiActionDoneMsg (no gen field)
// because the action handler calls postActionReloadCmds which increments
// gen and starts fresh data loads.
func (m Model) gitPullCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitPull(); err != nil {
			return chezmoiActionDoneMsg{action: chezmoiActionPull, err: err}
		}
		return chezmoiActionDoneMsg{action: chezmoiActionPull, message: "pulled from remote"}
	}
}

func (m Model) loadGitStatusCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		staged, unstaged, err := m.service.GitStatus()
		if err != nil {
			return chezmoiGitStatusLoadedMsg{err: err, gen: gen}
		}
		info, _ := m.service.GitBranchInfo()
		return chezmoiGitStatusLoadedMsg{staged: staged, unstaged: unstaged, info: info, gen: gen}
	}
}

func (m Model) gitAddCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitAdd(path); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitStage, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitStage, message: "staged " + shortenPath(path, m.targetPath)}
	}
}

func (m Model) gitResetCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitReset(path); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitUnstage, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitUnstage, message: "unstaged " + shortenPath(path, m.targetPath)}
	}
}

func (m Model) gitStageSelectionCmd(driftPaths, unstagedPaths []string) tea.Cmd {
	return func() tea.Msg {
		for _, path := range driftPaths {
			if err := m.service.ReAdd(path); err != nil {
				return chezmoiGitActionDoneMsg{
					action: chezmoiActionGitStageSelected,
					err:    fmt.Errorf("re-add %s: %w", shortenPath(path, m.targetPath), err),
				}
			}
		}
		for _, path := range unstagedPaths {
			if err := m.service.GitAdd(path); err != nil {
				return chezmoiGitActionDoneMsg{
					action: chezmoiActionGitStageSelected,
					err:    fmt.Errorf("stage %s: %w", shortenPath(path, m.targetPath), err),
				}
			}
		}

		total := len(driftPaths) + len(unstagedPaths)
		parts := make([]string, 0, 2)
		if len(driftPaths) > 0 {
			parts = append(parts, fmt.Sprintf("%d drift", len(driftPaths)))
		}
		if len(unstagedPaths) > 0 {
			parts = append(parts, fmt.Sprintf("%d unstaged", len(unstagedPaths)))
		}
		message := fmt.Sprintf("staged %d selected files", total)
		if len(parts) > 0 {
			message += " (" + strings.Join(parts, ", ") + ")"
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitStageSelected, message: message}
	}
}

func (m Model) gitUnstageSelectionCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		for _, path := range paths {
			if err := m.service.GitReset(path); err != nil {
				return chezmoiGitActionDoneMsg{
					action: chezmoiActionGitUnstageSelected,
					err:    fmt.Errorf("unstage %s: %w", shortenPath(path, m.targetPath), err),
				}
			}
		}
		return chezmoiGitActionDoneMsg{
			action:  chezmoiActionGitUnstageSelected,
			message: fmt.Sprintf("unstaged %d selected files", len(paths)),
		}
	}
}

func (m Model) gitCheckoutCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitCheckoutFile(path); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitDiscard, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitDiscard, message: "discarded " + shortenPath(path, m.targetPath)}
	}
}

func (m Model) gitDiscardSelectionCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		for _, path := range paths {
			if err := m.service.GitCheckoutFile(path); err != nil {
				return chezmoiGitActionDoneMsg{
					action: chezmoiActionGitDiscardSelected,
					err:    fmt.Errorf("discard %s: %w", shortenPath(path, m.targetPath), err),
				}
			}
		}
		return chezmoiGitActionDoneMsg{
			action:  chezmoiActionGitDiscardSelected,
			message: fmt.Sprintf("discarded %d selected files", len(paths)),
		}
	}
}

func (m Model) gitSoftResetCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitSoftReset(); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitUndoCommit, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitUndoCommit, message: "undid last commit (changes returned to staged)"}
	}
}

func (m Model) gitAddAllCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitAddAll(); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitStageAll, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitStageAll, message: "staged all files"}
	}
}

func (m Model) gitResetAllCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.GitResetAll(); err != nil {
			return chezmoiGitActionDoneMsg{action: chezmoiActionGitUnstageAll, err: err}
		}
		return chezmoiGitActionDoneMsg{action: chezmoiActionGitUnstageAll, message: "unstaged all files"}
	}
}

func (m Model) loadGitDiffCmd(path string, staged bool) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.service.GitDiff(path, staged)
		return chezmoiDiffLoadedMsg{path: path, diff: diff, err: err}
	}
}

func (m *Model) reloadStatusAndGitCmds() []tea.Cmd {
	cmds := []tea.Cmd{m.loadStatusCmd(), m.loadTemplatePathsCmd()}
	cmds = append(cmds, m.loadGitStatusCmd(), m.loadGitCommitsCmd())
	return cmds
}

// --- Editor command factories ---

func (m Model) editSourceCmd(path string) tea.Cmd {
	cmd := m.service.EditCmd(path)
	return execCmdOrUnsupported(chezmoiActionEditSource, cmd, "chezmoi: edit not supported")
}

// loadIgnoreFileContentCmd reads the .chezmoiignore file from the source directory.
func (m Model) loadIgnoreFileContentCmd() tea.Cmd {
	return func() tea.Msg {
		sourceDir, err := m.service.SourceDir()
		if err != nil {
			return chezmoiSourceContentMsg{path: ".chezmoiignore", err: fmt.Errorf("cannot find source dir: %w", err)}
		}
		ignorePath := filepath.Join(sourceDir, ".chezmoiignore")
		content, readErr := os.ReadFile(ignorePath)
		if readErr != nil {
			return chezmoiSourceContentMsg{path: ".chezmoiignore", err: readErr}
		}
		return chezmoiSourceContentMsg{path: ".chezmoiignore", content: string(content)}
	}
}

// resolveEditor returns the editor command string to use.
// Resolution order: Options.Editor > $EDITOR env var > "vi".
func (m Model) resolveEditor() string {
	if m.opts.Editor != "" {
		return m.opts.Editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}

// editorCmd builds an exec.Cmd for the resolved editor with the given file path.
// The editor string is split on whitespace to support arguments (e.g., "code --wait").
func (m Model) editorCmd(filePath string) *exec.Cmd {
	editor := m.resolveEditor()
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	return exec.Command(parts[0], append(parts[1:], filePath)...)
}

// editTargetCmd opens a target-path file in the configured editor.
func (m Model) editTargetCmd(filePath string) tea.Cmd {
	cmd := m.editorCmd(filePath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return chezmoiExecDoneMsg{action: chezmoiActionEditTarget, err: err}
	})
}

// resolveIgnoreFilePathCmd resolves the .chezmoiignore path asynchronously.
func (m Model) resolveIgnoreFilePathCmd() tea.Cmd {
	return func() tea.Msg {
		sourceDir, err := m.service.SourceDir()
		if err != nil {
			return sourceDirResolvedMsg{
				action: chezmoiActionEditIgnoreFile,
				err:    fmt.Errorf("cannot find source dir: %w", err),
			}
		}
		return sourceDirResolvedMsg{
			path:   filepath.Join(sourceDir, ".chezmoiignore"),
			action: chezmoiActionEditIgnoreFile,
		}
	}
}
