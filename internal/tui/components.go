package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func styledHelp(raw string) string {
	segments, segmented := splitHelpSegments(raw)
	if !segmented {
		return activeTheme.DimText.Render(raw)
	}
	return styledHelpFromSegments(segments)
}

func styledHelpResponsive(raw string, width int) string {
	segments, segmented := splitHelpSegments(raw)
	if !segmented || width <= 0 {
		return styledHelp(raw)
	}

	if wrapped, ok := wrapStyledHelpSegments(segments, width, 2, false); ok {
		return wrapped
	}

	compactSegments := compactHelpSegments(segments)
	if wrapped, ok := wrapStyledHelpSegments(compactSegments, width, 2, true); ok {
		return wrapped
	}

	return styledHelpFromSegments(compactSegments)
}

func splitHelpSegments(raw string) ([]string, bool) {
	if strings.Contains(raw, "|") {
		return strings.Split(raw, "|"), true
	}
	if strings.Contains(raw, "\u2022") {
		return strings.Split(raw, "\u2022"), true
	}
	return nil, false
}

func styledHelpFromSegments(segments []string) string {
	parts := renderHelpSegments(segments)
	sep := activeTheme.DimText.Render("  ")
	return strings.Join(parts, sep)
}

func renderHelpSegments(segments []string) []string {
	t := &activeTheme
	var parts []string
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		key, action := splitKeyAction(seg)
		if action == "" {
			parts = append(parts, t.DimText.Render(seg))
		} else {
			k := t.BoldPrimary.Render(key)
			parts = append(parts, k+" "+t.DimText.Render(action))
		}
	}
	return parts
}

func wrapStyledHelpSegments(segments []string, width, maxLines int, truncateOverflow bool) (string, bool) {
	parts := renderHelpSegments(segments)
	if len(parts) == 0 {
		return "", true
	}

	sep := activeTheme.DimText.Render("  ")
	sepWidth := ansi.StringWidth(sep)
	if width <= 0 {
		width = 1
	}

	lines := make([]string, 0, maxLines)
	currentParts := make([]string, 0, len(parts))
	currentWidth := 0

	for _, part := range parts {
		partWidth := ansi.StringWidth(part)
		if partWidth > width {
			part = ansi.Truncate(part, width, "…")
			partWidth = ansi.StringWidth(part)
		}

		if len(currentParts) == 0 {
			currentParts = append(currentParts, part)
			currentWidth = partWidth
			continue
		}

		if currentWidth+sepWidth+partWidth <= width {
			currentParts = append(currentParts, part)
			currentWidth += sepWidth + partWidth
			continue
		}

		lines = append(lines, strings.Join(currentParts, sep))
		if len(lines) >= maxLines {
			if !truncateOverflow {
				return "", false
			}
			lines[len(lines)-1] = ansi.Truncate(lines[len(lines)-1], width, "…")
			return strings.Join(lines, "\n"), true
		}

		currentParts = currentParts[:0]
		currentParts = append(currentParts, part)
		currentWidth = partWidth
	}

	if len(currentParts) > 0 {
		lines = append(lines, strings.Join(currentParts, sep))
	}

	if len(lines) > maxLines {
		if !truncateOverflow {
			return "", false
		}
		lines = lines[:maxLines]
		lines[maxLines-1] = ansi.Truncate(lines[maxLines-1], width, "…")
	}

	return strings.Join(lines, "\n"), true
}

func compactHelpSegments(segments []string) []string {
	clean := make([]string, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg != "" {
			clean = append(clean, seg)
		}
	}
	if len(clean) <= 5 {
		return clean
	}

	const maxCompactSegments = 7
	compact := make([]string, 0, maxCompactSegments)
	seen := make(map[string]struct{}, maxCompactSegments)
	add := func(seg string) {
		if seg == "" || len(compact) >= maxCompactSegments {
			return
		}
		if _, ok := seen[seg]; ok {
			return
		}
		compact = append(compact, seg)
		seen[seg] = struct{}{}
	}

	addByKey := func(keys ...string) {
		for _, want := range keys {
			if len(compact) >= maxCompactSegments {
				return
			}
			for _, seg := range clean {
				key, _ := splitKeyAction(seg)
				if key == want {
					add(seg)
					break
				}
			}
		}
	}

	// Keep core navigation and actions visible in compact mode whenever present.
	addByKey("↑/↓")
	addByKey("enter", "a", "p", "?", "esc")

	// Add common secondary controls based on availability.
	addByKey("v", "f", "/", "t", "r")

	// Fill remaining slots in original order for context.
	if len(compact) < maxCompactSegments {
		for _, seg := range clean {
			add(seg)
			if len(compact) >= maxCompactSegments {
				break
			}
		}
	}

	return compact
}

func splitKeyAction(seg string) (string, string) {
	words := strings.Fields(seg)
	if len(words) <= 1 {
		return seg, ""
	}
	keyEnd := 1
	return strings.Join(words[:keyEnd], " "), strings.Join(words[keyEnd:], " ")
}

func renderTabs(names []string, active int) string {
	var parts []string
	for i, name := range names {
		label := fmt.Sprintf(" %d %s ", i+1, name)
		if i == active {
			parts = append(parts, activeTheme.ActiveTab.Render(label))
		} else {
			parts = append(parts, activeTheme.InactiveTab.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

type menuItem struct {
	label       string
	description string
	disabled    bool
	separator   bool
}

func renderActionsMenu(title string, items []menuItem, cursor int) string {
	var b strings.Builder
	b.WriteString(activeTheme.MenuTitle.Render(title))
	b.WriteString("\n")
	for i, item := range items {
		if item.separator {
			b.WriteString(item.label)
			b.WriteString("\n")
			continue
		}
		cur := "  "
		style := activeTheme.Normal
		if item.disabled {
			style = activeTheme.DimText
		}
		if i == cursor {
			cur = "> "
			if !item.disabled {
				style = activeTheme.Selected
			}
		}
		b.WriteString(style.Render(cur + item.label))
		b.WriteString("\n")
	}
	if cursor >= 0 && cursor < len(items) {
		if desc := items[cursor].description; desc != "" {
			b.WriteString(activeTheme.HintText.Render(desc))
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// HelpEntry represents a single key-description pair in the help overlay.
type HelpEntry struct {
	Key  string
	Desc string
}

// HelpSection groups related help entries under a titled section.
type HelpSection struct {
	Title   string
	Entries []HelpEntry
	Notes   []string
}

func helpOverlayViewport(width, height int) (contentWidth, contentHeight int) {
	totalWidth := max(1, width-4)
	totalWidth = min(totalWidth, 110)
	totalHeight := max(1, height-4)

	frameW := activeTheme.HelpOverlay.GetHorizontalFrameSize()
	frameH := activeTheme.HelpOverlay.GetVerticalFrameSize()

	contentWidth = max(1, totalWidth-frameW)
	contentHeight = max(1, totalHeight-frameH)
	return contentWidth, contentHeight
}

func helpOverlayPageStep(width, height int) int {
	_, h := helpOverlayViewport(width, height)
	return max(1, h/2)
}

func buildHelpOverlayLines(footer string, rows ...[]HelpSection) []string {
	const colGap = "    "
	const indent = "  "

	var lines []string

	for ri, row := range rows {
		if len(row) == 0 {
			continue
		}
		if ri > 0 {
			lines = append(lines, "")
		}

		kw := helpKeyWidths(row)
		cw := helpColWidths(row, kw)
		lines = append(lines,
			helpTitleLine(row, cw, indent, colGap),
			helpSepLine(row, cw, indent, colGap),
		)
		lines = append(lines, helpEntryLines(row, cw, kw, indent, colGap)...)
		lines = append(lines, helpNoteLines(row, cw, indent, colGap)...)
	}

	if footer != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, indent+footer)
	}

	return lines
}

func helpKeyWidths(row []HelpSection) []int {
	widths := make([]int, len(row))
	for si, sec := range row {
		for _, e := range sec.Entries {
			if w := ansi.StringWidth(e.Key); w > widths[si] {
				widths[si] = w
			}
		}
	}
	return widths
}

func helpColWidths(row []HelpSection, keyWidths []int) []int {
	widths := make([]int, len(row))
	for si, sec := range row {
		if tw := ansi.StringWidth(sec.Title); tw > widths[si] {
			widths[si] = tw
		}
		for _, e := range sec.Entries {
			if ew := keyWidths[si] + 2 + ansi.StringWidth(e.Desc); ew > widths[si] {
				widths[si] = ew
			}
		}
		for _, n := range sec.Notes {
			if nw := ansi.StringWidth(n); nw > widths[si] {
				widths[si] = nw
			}
		}
	}
	return widths
}

func helpTitleLine(row []HelpSection, colWidths []int, indent, colGap string) string {
	var b strings.Builder
	b.WriteString(indent)
	for si, sec := range row {
		if si > 0 {
			b.WriteString(colGap)
		}
		b.WriteString(sec.Title)
		if si < len(row)-1 {
			if pad := colWidths[si] - ansi.StringWidth(sec.Title); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	return b.String()
}

func helpSepLine(row []HelpSection, colWidths []int, indent, colGap string) string {
	var b strings.Builder
	b.WriteString(indent)
	for si, sec := range row {
		if si > 0 {
			b.WriteString(colGap)
		}
		b.WriteString(strings.Repeat("─", max(ansi.StringWidth(sec.Title), colWidths[si])))
	}
	return b.String()
}

func helpEntryLines(row []HelpSection, colWidths, keyWidths []int, indent, colGap string) []string {
	maxEntries := 0
	for _, sec := range row {
		if n := len(sec.Entries); n > maxEntries {
			maxEntries = n
		}
	}
	lines := make([]string, 0, maxEntries)
	for ei := range maxEntries {
		var b strings.Builder
		b.WriteString(indent)
		for si, sec := range row {
			if si > 0 {
				b.WriteString(colGap)
			}
			if ei < len(sec.Entries) {
				e := sec.Entries[ei]
				keyPad := keyWidths[si] - ansi.StringWidth(e.Key)
				cell := e.Key + strings.Repeat(" ", keyPad) + "  " + e.Desc
				if si < len(row)-1 {
					if cellPad := colWidths[si] - ansi.StringWidth(cell); cellPad > 0 {
						cell += strings.Repeat(" ", cellPad)
					}
				}
				b.WriteString(cell)
			} else if si < len(row)-1 {
				b.WriteString(strings.Repeat(" ", colWidths[si]))
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}

func helpNoteLines(row []HelpSection, colWidths []int, indent, colGap string) []string {
	maxNotes := 0
	for _, sec := range row {
		if n := len(sec.Notes); n > maxNotes {
			maxNotes = n
		}
	}
	lines := make([]string, 0, maxNotes)
	for ni := range maxNotes {
		var b strings.Builder
		b.WriteString(indent)
		for si, sec := range row {
			if si > 0 {
				b.WriteString(colGap)
			}
			if ni < len(sec.Notes) {
				note := sec.Notes[ni]
				if si < len(row)-1 {
					if pad := colWidths[si] - ansi.StringWidth(note); pad > 0 {
						note += strings.Repeat(" ", pad)
					}
				}
				b.WriteString(note)
			} else if si < len(row)-1 {
				b.WriteString(strings.Repeat(" ", colWidths[si]))
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}

func helpOverlayMaxScroll(width, height int, footer string, rows ...[]HelpSection) int {
	lines := buildHelpOverlayLines(footer, rows...)
	_, viewportHeight := helpOverlayViewport(width, height)
	if len(lines) <= viewportHeight {
		return 0
	}
	return len(lines) - viewportHeight
}

func buildHelpOverlay(width, height, scroll int, footer string, rows ...[]HelpSection) string {
	lines := buildHelpOverlayLines(footer, rows...)
	contentWidth, viewportHeight := helpOverlayViewport(width, height)
	if len(lines) == 0 {
		lines = []string{"  No key help available"}
	}

	maxScroll := max(0, len(lines)-viewportHeight)
	scroll = min(max(scroll, 0), maxScroll)
	start := scroll
	end := min(start+viewportHeight, len(lines))

	visibleLines := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		visibleLines = append(visibleLines, visualTruncate(line, contentWidth))
	}
	content := strings.Join(visibleLines, "\n")
	box := activeTheme.HelpOverlay.Width(contentWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
