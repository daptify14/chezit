package tui

import (
	"fmt"
	"strings"
)

const (
	cmdColLabel = 20
	cmdColCLI   = 35
)

func (m Model) renderCommandsTabContent() string {
	var b strings.Builder

	maxWidth := m.effectiveWidth() - 2
	descWidth := max(maxWidth-cmdColLabel-cmdColCLI-8, 10)

	listHeight := m.chezmoiCommandsListHeight()

	// Build display rows: headers + commands
	type displayRow struct {
		isHeader bool
		category string
		cmdIdx   int // index into m.cmds.items
	}
	var rows []displayRow
	lastCategory := ""
	for i, cmd := range m.cmds.items {
		if cmd.category != lastCategory {
			rows = append(rows, displayRow{isHeader: true, category: cmd.category})
			lastCategory = cmd.category
		}
		rows = append(rows, displayRow{cmdIdx: i})
	}

	// Map commandCursor to display row index
	cursorDisplayIdx := 0
	for i, row := range rows {
		if !row.isHeader && row.cmdIdx == m.cmds.cursor {
			cursorDisplayIdx = i
			break
		}
	}

	start, end := visibleRange(len(rows), cursorDisplayIdx, listHeight)
	visible := rows[start:end]

	for i, row := range visible {
		if row.isHeader {
			b.WriteString(m.renderCommandSectionHeader(row.category, maxWidth))
		} else {
			cmd := m.cmds.items[row.cmdIdx]
			isSelected := row.cmdIdx == m.cmds.cursor
			b.WriteString(m.renderCommandRow(cmd, isSelected, descWidth, maxWidth))
		}
		if i < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

var categoryLabels = map[string]string{
	"apply": "Apply & Sync",
	"info":  "Info & Diagnostics",
	"edit":  "Edit",
}

func (m Model) renderCommandSectionHeader(category string, maxWidth int) string {
	label := categoryLabels[category]
	if label == "" {
		label = strings.ToUpper(category[:1]) + category[1:]
	}
	header := fmt.Sprintf("  ── %s ──", label)
	return activeTheme.DimText.Render(visualTruncate(header, maxWidth))
}

func (m Model) renderCommandRow(cmd chezmoiCommandItem, selected bool, descWidth, maxWidth int) string {
	label := visualPad(visualTruncate(cmd.label, cmdColLabel), cmdColLabel)
	desc := visualPad(visualTruncate(cmd.description, descWidth), descWidth)
	cli := visualPad(visualTruncate(cmd.command, cmdColCLI), cmdColCLI)

	line := fmt.Sprintf("  %s  %s  %s", label, desc, cli)

	if !cmd.available {
		return activeTheme.DimText.Render(line)
	}

	if selected {
		return activeTheme.Selected.Width(maxWidth).Render(line)
	}

	dimDesc := activeTheme.DimText.Render(visualPad(visualTruncate(cmd.description, descWidth), descWidth))
	accentCLI := activeTheme.AccentFg.Render(visualPad(visualTruncate(cmd.command, cmdColCLI), cmdColCLI))
	return fmt.Sprintf("  %s  %s  %s", label, dimDesc, accentCLI)
}

func (m Model) renderCommandsStatusBar() string {
	status := fmt.Sprintf(" command %d/%d ", m.cmds.cursor+1, len(m.cmds.items))
	if m.ui.message != "" {
		status = " " + m.ui.message + " "
	}
	statusBar := activeTheme.StatusBar.Width(m.effectiveWidth()).Render(status)
	helpText := "↑/↓ navigate | enter run"
	if m.cmds.cursor < len(m.cmds.items) && m.cmds.items[m.cmds.cursor].supportsDryRun {
		helpText += " | d dry run"
	}
	helpText += " | tab switch | ? keys | esc quit"
	help := m.helpHint(helpText)
	return statusBar + "\n" + help
}
