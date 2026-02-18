package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// --- Command Execution ---

func (m Model) executeChezmoiCommand(id chezmoiCommandID) (tea.Model, tea.Cmd) {
	switch id {
	// --- Shell-out commands (need TTY for sudo/interactive scripts) ---
	case chezmoiCmdApply:
		// Dry-run preview: show diff first, then confirm via shell-out
		m.ui.busyAction = true
		m.diff.previewApply = true
		return m, func() tea.Msg {
			output, err := m.service.DiffAll()
			return chezmoiSourceContentMsg{path: "Preview: chezmoi apply", content: output, err: err}
		}
	case chezmoiCmdUpdate:
		m.view = ConfirmScreen
		m.overlays.confirmAction = chezmoiActionUpdate
		m.overlays.confirmLabel = "update (pull from remote and apply)"
		return m, nil
	case chezmoiCmdRefreshExternals:
		m.view = ConfirmScreen
		m.overlays.confirmAction = chezmoiActionRefresh
		m.overlays.confirmLabel = "refresh externals (re-download and apply)"
		return m, nil
	case chezmoiCmdInit:
		m.view = ConfirmScreen
		m.overlays.confirmAction = chezmoiActionInit
		m.overlays.confirmLabel = "init (apply config template changes)"
		return m, nil

	// --- Confirmation-gated mutation ---
	case chezmoiCmdReAddAll:
		m.view = ConfirmScreen
		m.overlays.confirmAction = chezmoiActionReAdd
		m.overlays.confirmLabel = "re-add all (overwrite source from destination)"
		return m, nil

	// --- Inline capture (read-only commands) ---
	case chezmoiCmdStatus:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.StatusText()
			return chezmoiSourceContentMsg{path: "chezmoi status", content: output, err: err}
		}
	case chezmoiCmdDiffAll:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.DiffAll()
			return chezmoiSourceContentMsg{path: "chezmoi diff", content: output, err: err}
		}
	case chezmoiCmdCatConfig:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.CatConfig()
			return chezmoiSourceContentMsg{path: "chezmoi cat-config", content: output, err: err}
		}
	case chezmoiCmdDoctor:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.Doctor()
			return chezmoiSourceContentMsg{path: "chezmoi doctor", content: output, err: err}
		}
	case chezmoiCmdVerify:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			if err := m.service.Verify(); err != nil {
				return chezmoiActionDoneMsg{action: chezmoiActionNone, err: err}
			}
			return chezmoiActionDoneMsg{action: chezmoiActionNone, message: "verify: all files match source state"}
		}
	case chezmoiCmdData:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.Data()
			return chezmoiSourceContentMsg{path: "chezmoi data", content: output, err: err}
		}
	case chezmoiCmdGitLog:
		m.ui.busyAction = true
		return m, func() tea.Msg {
			output, err := m.service.GitLog()
			return chezmoiSourceContentMsg{path: "chezmoi git log", content: output, err: err}
		}

	// --- Confirm-gated archive ---
	case chezmoiCmdArchive:
		m.view = ConfirmScreen
		m.overlays.confirmAction = chezmoiActionArchive
		m.overlays.confirmLabel = fmt.Sprintf("create backup archive in %s", m.service.ArchiveOutputDir())
		return m, nil

	// --- Editor shell-out commands ---
	case chezmoiCmdEditSource:
		cmd := m.service.EditSourceCmd()
		return m, execCmdOrUnsupported(chezmoiActionEditSource, cmd, "chezmoi: edit not supported")
	case chezmoiCmdEditConfig:
		cmd := m.service.EditConfigCmd()
		return m, execCmdOrUnsupported(chezmoiActionEditSource, cmd, "chezmoi: config editing not supported")
	case chezmoiCmdEditConfigTemplate:
		cmd := m.service.EditConfigTemplateCmd()
		return m, execCmdOrUnsupported(chezmoiActionEditSource, cmd, "chezmoi: config template not found")
	}
	return m, nil
}

func (m Model) executeDryRun(id chezmoiCommandID) (tea.Model, tea.Cmd) {
	var cmd *exec.Cmd
	switch id {
	case chezmoiCmdApply:
		cmd = m.service.ApplyDryRunCmd()
	case chezmoiCmdRefreshExternals:
		cmd = m.service.ApplyRefreshDryRunCmd()
	default:
		return m, nil
	}
	wrapped := wrapWithPressEnter(cmd)
	if wrapped == nil {
		m.ui.message = "chezmoi: dry-run not supported"
		return m, nil
	}
	return m, tea.ExecProcess(wrapped, func(err error) tea.Msg {
		return chezmoiActionDoneMsg{action: chezmoiActionNone, err: err}
	})
}

// wrapWithPressEnter wraps an exec.Cmd so output stays visible and user
// presses Enter to return to the TUI.
func wrapWithPressEnter(cmd *exec.Cmd) *exec.Cmd {
	if cmd == nil {
		return nil
	}
	// Use shell positional parameters so untrusted data stays in argv, not shell code.
	script := `"$@"; printf '\n\033[2mPress Enter to continue...\033[0m'; read _`
	args := append([]string{"-c", script, "chezmoi-wrap"}, cmd.Path)
	args = append(args, cmd.Args[1:]...)
	wrapped := exec.Command("sh", args...)
	wrapped.Dir = cmd.Dir
	if cmd.Env != nil {
		wrapped.Env = cmd.Env
	} else {
		wrapped.Env = os.Environ()
	}
	return wrapped
}

func execCmdOrUnsupported(action chezmoiAction, cmd *exec.Cmd, msg string) tea.Cmd {
	if cmd == nil {
		return func() tea.Msg {
			return chezmoiExecDoneMsg{action: action, err: fmt.Errorf("%s", msg)}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return chezmoiExecDoneMsg{action: action, err: err}
	})
}

// commandIDFromLabel maps a command label to its internal ID.
func commandIDFromLabel(label string) chezmoiCommandID {
	switch label {
	case "Apply":
		return chezmoiCmdApply
	case "Update":
		return chezmoiCmdUpdate
	case "Refresh Externals":
		return chezmoiCmdRefreshExternals
	case "Re-Add All":
		return chezmoiCmdReAddAll
	case "Init":
		return chezmoiCmdInit
	case "Edit Source":
		return chezmoiCmdEditSource
	case "Doctor":
		return chezmoiCmdDoctor
	case "Verify":
		return chezmoiCmdVerify
	case "Status":
		return chezmoiCmdStatus
	case "Diff All":
		return chezmoiCmdDiffAll
	case "Cat Config":
		return chezmoiCmdCatConfig
	case "Edit Config":
		return chezmoiCmdEditConfig
	case "Edit Config Template":
		return chezmoiCmdEditConfigTemplate
	case "Git Log":
		return chezmoiCmdGitLog
	case "Data":
		return chezmoiCmdData
	case "Archive":
		return chezmoiCmdArchive
	default:
		return 0
	}
}

// humanSize formats a byte count as a human-readable string.
func humanSize(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
