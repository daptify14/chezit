package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

var defaultCommitPresets = []string{
	"update dotfiles",
	"chore: update dotfiles",
	"feat: add new config",
	"fix: fix configuration",
	"chore: sync changes",
}

// commitComposeKey is the sentinel value identifying the "Compose..." option in the preset select form.
const commitComposeKey = "__compose__"

// openCommitScreen builds the preset select form and returns its Init cmd.
func (m *Model) openCommitScreen() tea.Cmd {
	m.view = CommitScreen
	m.commit.composing = false
	m.commit.presetForm = m.buildPresetForm()
	m.commit.composeForm = nil
	return m.commit.presetForm.Init()
}

// buildPresetForm creates a huh.Select with expanded presets + "Compose...".
func (m Model) buildPresetForm() *huh.Form {
	opts := make([]huh.Option[string], 0, len(m.commit.presets)+1)
	for _, p := range m.commit.presets {
		opts = append(opts, huh.NewOption(p, p))
	}
	opts = append(opts, huh.NewOption("Compose...", commitComposeKey))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("choice").
				Title("Commit Message").
				Options(opts...),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin)).
		WithWidth(56).
		WithShowHelp(false)
}

// buildComposeForm creates a huh.Input for free-text message entry.
func (m Model) buildComposeForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("message").
				Title("Commit Message").
				Placeholder("Enter commit message...").
				CharLimit(200).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil // allow empty during typing; checked at submit
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin)).
		WithWidth(56).
		WithShowHelp(false)
}

// handleCommitUpdate routes messages to the active huh form and handles completion/abort.
func (m Model) handleCommitUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.commit.composing {
		return m.handleComposeUpdate(msg)
	}
	return m.handlePresetUpdate(msg)
}

func (m Model) handlePresetUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.commit.presetForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.commit.presetForm = f
	}

	switch m.commit.presetForm.State {
	case huh.StateCompleted:
		choice := m.commit.presetForm.GetString("choice")
		if choice == commitComposeKey {
			m.commit.composing = true
			m.commit.composeForm = m.buildComposeForm()
			return m, m.commit.composeForm.Init()
		}
		m.view = StatusScreen
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.commitWithMsgCmd(choice))
	case huh.StateAborted:
		m.view = StatusScreen
		return m, nil
	}

	return m, cmd
}

func (m Model) handleComposeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.commit.composeForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.commit.composeForm = f
	}

	switch m.commit.composeForm.State {
	case huh.StateCompleted:
		message := strings.TrimSpace(m.commit.composeForm.GetString("message"))
		if message == "" {
			// Rebuild compose form if submitted empty.
			m.commit.composeForm = m.buildComposeForm()
			return m, m.commit.composeForm.Init()
		}
		m.view = StatusScreen
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.commitWithMsgCmd(message))
	case huh.StateAborted:
		m.commit.composing = false
		m.commit.presetForm = m.buildPresetForm()
		return m, m.commit.presetForm.Init()
	}

	return m, cmd
}

// renderCommitScreen wraps the active form's View() in a centered box.
func (m Model) renderCommitScreen() string {
	var formView string
	if m.commit.composing {
		formView = m.commit.composeForm.View()
	} else {
		formView = m.commit.presetForm.View()
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeTheme.Primary).
		Padding(1, 2).
		Width(60)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(formView))
}
