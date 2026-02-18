package tui

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m Model) renderManagedTabContent() string {
	return m.renderManagedTabContentWidth(m.effectiveWidth())
}

// renderManagedTabContentWidth renders the managed tab list at a specific width.
func (m Model) renderManagedTabContentWidth(maxWidth int) string {
	var b strings.Builder

	b.WriteString(m.renderChezmoiSearchBoxWidth(maxWidth))
	b.WriteString("\n")

	isLoading := m.filesTab.views[managedViewManaged].loading || m.filesTab.views[managedViewIgnored].loading || m.filesTab.views[managedViewUnmanaged].loading
	if isLoading {
		spinnerView := m.ui.loadingSpinner.View()
		var label string
		switch m.filesTab.viewMode {
		case managedViewIgnored:
			label = "Loading ignored files..."
		case managedViewUnmanaged:
			label = "Loading unmanaged files..."
		case managedViewAll:
			label = "Loading files..."
		default:
			label = "Loading managed files..."
		}
		fmt.Fprintf(&b, "  %s %s", spinnerView, label)
		return b.String()
	}

	tree := m.activeTree()
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		b.WriteString(activeTheme.DimText.Render(fmt.Sprintf("  %d ignored files in %d directories", tree.fileCount, tree.dirCount)))
	case managedViewUnmanaged:
		b.WriteString(activeTheme.DimText.Render(fmt.Sprintf("  %d unmanaged files in %d directories", tree.fileCount, tree.dirCount)))
	case managedViewAll:
		unmanagedCount := 0
		if m.filesTab.views[managedViewUnmanaged].files != nil {
			unmanagedCount = m.filesTab.views[managedViewUnmanaged].tree.fileCount
		}
		b.WriteString(activeTheme.DimText.Render(fmt.Sprintf("  %d managed, %d ignored, %d unmanaged in %d directories",
			m.filesTab.views[managedViewManaged].tree.fileCount, m.filesTab.views[managedViewIgnored].tree.fileCount, unmanagedCount, tree.dirCount)))
	default:
		b.WriteString(activeTheme.DimText.Render(fmt.Sprintf("  %d files in %d directories", tree.fileCount, tree.dirCount)))
	}
	b.WriteString("\n")

	if m.filesTab.treeView {
		b.WriteString(m.renderManagedTreeViewWidth(maxWidth))
	} else {
		b.WriteString(m.renderManagedFlatView(maxWidth))
	}

	if m.overlays.showViewPicker {
		b.WriteString("\n\n")
		b.WriteString(m.renderViewPickerMenu())
	} else if m.actions.managedShow {
		b.WriteString("\n\n")
		b.WriteString(m.renderManagedActionsMenu())
	}

	return b.String()
}

// emptyFilesLabel returns the appropriate "No ... files" label for the current view mode.
func emptyFilesLabel(mode managedViewMode) string {
	switch mode {
	case managedViewIgnored:
		return "  No ignored files"
	case managedViewUnmanaged:
		return "  No unmanaged files"
	case managedViewAll:
		return "  No files"
	default:
		return "  No managed files"
	}
}

// renderManagedTreeViewWidth renders the managed tree at a specific width.
func (m Model) renderManagedTreeViewWidth(maxWidth int) string {
	rows := m.activeTreeRows()
	if len(rows) == 0 {
		return activeTheme.DimText.Render(emptyFilesLabel(m.filesTab.viewMode))
	}

	var b strings.Builder
	listHeight := m.chezmoiManagedListHeight()
	start, end := visibleRange(len(rows), m.filesTab.cursor, listHeight)
	visible := rows[start:end]
	rowMaxWidth := maxWidth - 2

	for i, row := range visible {
		idx := start + i
		isSelected := idx == m.filesTab.cursor

		line := m.renderTreeRow(row, isSelected, rowMaxWidth)
		b.WriteString(line)
		if i < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderTreeRow(row flatTreeRow, selected bool, maxWidth int) string {
	isIgnored, isUnmanaged := m.treeNodeFileClass(row.node)
	isDimmed := isIgnored || isUnmanaged

	icon := renderFileIcon(row.node.name, row.node.isDir, selected, m.iconMode)

	var nameStr string
	if row.node.isDir {
		nameStr = renderDirName(row.node, icon, selected)
	} else {
		nameStr = icon + row.node.name
	}

	content := "  " + treeRowPrefix(row) + nameStr + treeRowSuffix(row, isIgnored, isUnmanaged, selected)
	content = visualTruncate(content, maxWidth)

	if selected {
		return activeTheme.Selected.Width(maxWidth).Render(content)
	}
	if isDimmed {
		return activeTheme.DimText.Render(content)
	}
	return content
}

// treeRowPrefix builds the tree connector string for a row based on its depth
// and sibling position. Returns an empty string for root-level rows.
func treeRowPrefix(row flatTreeRow) string {
	if row.depth == 0 {
		return ""
	}
	var b strings.Builder
	for i := 1; i < len(row.prefixBits)-1; i++ {
		if row.prefixBits[i] {
			b.WriteString("    ")
		} else {
			b.WriteString("│   ")
		}
	}
	if row.isLast {
		b.WriteString("└── ")
	} else {
		b.WriteString("├── ")
	}
	return b.String()
}

// treeNodeFileClass classifies a non-directory node for the current view mode.
// Returns (isIgnored, isUnmanaged). Both are false for directory nodes.
func (m Model) treeNodeFileClass(node *managedTreeNode) (isIgnored, isUnmanaged bool) {
	if node.isDir {
		return false, false
	}
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		return true, false
	case managedViewUnmanaged:
		return false, true
	case managedViewAll:
		class := m.classifyPath(node.absPath)
		return class == pathClassIgnored, class == pathClassUnmanaged
	}
	return false, false
}

// dirArrow returns the glyph for a directory's expand/collapse indicator.
func dirArrow(node *managedTreeNode) string {
	if node.loading {
		return "…"
	}
	if node.expanded {
		return "▼"
	}
	return "▶"
}

// renderDirName renders the display name for a directory node, including the
// expand/collapse arrow and icon.
func renderDirName(node *managedTreeNode, icon string, selected bool) string {
	arrow := dirArrow(node)
	if selected {
		return arrow + " " + icon + node.name
	}
	arrowStyle := activeTheme.PrimaryFg
	if node.loading {
		arrowStyle = activeTheme.DimText
	}
	return arrowStyle.Render(arrow) + " " + icon + activeTheme.BoldPrimary.Render(node.name)
}

// treeRowSuffix builds the count badge and classification tags for a tree row.
func treeRowSuffix(row flatTreeRow, isIgnored, isUnmanaged, selected bool) string {
	var suffix string
	if row.node.isDir && row.fileCount > 0 {
		countStr := fmt.Sprintf(" [%d files]", row.fileCount)
		if selected {
			suffix = countStr
		} else {
			suffix = activeTheme.DimText.Render(countStr)
		}
	}
	if isIgnored && !selected {
		suffix += activeTheme.DimText.Render(" [ignored]")
	}
	if isUnmanaged && !selected {
		suffix += activeTheme.DimText.Render(" [unmanaged]")
	}
	return suffix
}

// renderManagedFlatView renders the managed flat list at a specific width.
func (m Model) renderManagedFlatView(maxWidth int) string {
	files := m.activeFlatFiles()
	if len(files) == 0 {
		return activeTheme.DimText.Render(emptyFilesLabel(m.filesTab.viewMode))
	}

	var b strings.Builder
	listHeight := m.chezmoiManagedListHeight()
	start, end := visibleRange(len(files), m.filesTab.cursor, listHeight)
	visible := files[start:end]
	rowMaxWidth := maxWidth - 2

	for i, path := range visible {
		idx := start + i
		isSelected := idx == m.filesTab.cursor

		// Check if file is ignored or unmanaged (for dimmed styling)
		isIgnored := false
		isUnmanaged := false
		switch m.filesTab.viewMode {
		case managedViewIgnored:
			isIgnored = true
		case managedViewUnmanaged:
			isUnmanaged = true
		case managedViewAll:
			class := m.classifyPath(path)
			isIgnored = class == pathClassIgnored
			isUnmanaged = class == pathClassUnmanaged
		}

		cursor := "  "
		style := activeTheme.Normal
		if isSelected {
			cursor = "> "
			style = activeTheme.Selected
		} else if isIgnored || isUnmanaged {
			style = activeTheme.DimText
		}

		icon := renderFileIcon(filepath.Base(path), false, isSelected, m.iconMode)
		displayPath := shortenPath(path, m.targetPath)
		tag := ""
		if isIgnored && !isSelected {
			tag = " [ignored]"
		}
		if isUnmanaged && !isSelected {
			tag = " [unmanaged]"
		}
		displayPath = visualTruncate(icon+displayPath+tag, rowMaxWidth-2)
		b.WriteString(style.Render(cursor + displayPath))
		if i < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderManagedActionsMenu() string {
	if len(m.actions.managedItems) == 0 {
		return ""
	}

	var title string
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		title = " Ignored File "
	case managedViewUnmanaged:
		title = " Unmanaged File "
	case managedViewAll:
		path := m.selectedManagedPathForOpen()
		switch m.classifyPath(path) {
		case pathClassIgnored:
			title = " Ignored File "
		case pathClassUnmanaged:
			title = " Unmanaged File "
		default:
			title = " Managed File "
		}
	default:
		title = " Managed File "
	}
	path := m.selectedManagedPath()
	if path != "" {
		title = fmt.Sprintf(" %s ", filepath.Base(path))
	}

	items := make([]menuItem, len(m.actions.managedItems))
	for i, item := range m.actions.managedItems {
		items[i] = menuItem{
			label:       item.label,
			description: item.description,
			disabled:    item.disabled || item.action == chezmoiActionNone,
		}
	}
	return renderActionsMenu(title, items, m.actions.managedCursor)
}

func (m Model) renderManagedStatusBar() string {
	var total int
	if m.filesTab.treeView {
		total = len(m.activeTreeRows())
	} else {
		total = len(m.activeFlatFiles())
	}

	var status string
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		status = fmt.Sprintf(" %d ignored files | %d/%d ", m.filesTab.views[managedViewIgnored].tree.fileCount, m.filesTab.cursor+1, total)
	case managedViewUnmanaged:
		unmanagedCount := 0
		if m.filesTab.views[managedViewUnmanaged].files != nil {
			unmanagedCount = m.filesTab.views[managedViewUnmanaged].tree.fileCount
		}
		status = fmt.Sprintf(" %d unmanaged files | %d/%d ", unmanagedCount, m.filesTab.cursor+1, total)
	case managedViewAll:
		unmanagedCount := 0
		if m.filesTab.views[managedViewUnmanaged].files != nil {
			unmanagedCount = m.filesTab.views[managedViewUnmanaged].tree.fileCount
		}
		status = fmt.Sprintf(" %d managed + %d ignored + %d unmanaged | %d/%d ",
			m.filesTab.views[managedViewManaged].tree.fileCount, m.filesTab.views[managedViewIgnored].tree.fileCount, unmanagedCount, m.filesTab.cursor+1, total)
	default:
		status = fmt.Sprintf(" %d managed files | %d/%d ", m.filesTab.views[managedViewManaged].tree.fileCount, m.filesTab.cursor+1, total)
	}
	if total == 0 {
		switch m.filesTab.viewMode {
		case managedViewIgnored:
			status = " 0 ignored files "
		case managedViewUnmanaged:
			status = " 0 unmanaged files "
		case managedViewAll:
			status = " 0 files "
		default:
			status = " 0 managed files "
		}
	}

	path := m.selectedManagedPath()
	if path != "" {
		display := shortenPath(path, m.targetPath)
		status = fmt.Sprintf(" %s ", display)
	}

	if m.ui.message != "" {
		status = " " + m.ui.message + " "
	}

	filterChip := "[filter:all]"
	switch {
	case len(m.filesTab.entryFilter.Exclude) > 0:
		filterChip = fmt.Sprintf("[filter:%d excluded]", len(m.filesTab.entryFilter.Exclude))
	case len(m.filesTab.entryFilter.Include) > 0:
		filterChip = fmt.Sprintf("[filter:%d included]", len(m.filesTab.entryFilter.Include))
	}
	viewChip := fmt.Sprintf("[view:%s]", strings.ToLower(managedViewModeLabel(m.filesTab.viewMode)))
	status = strings.TrimRight(status, " ") + " " + viewChip + " " + filterChip + " "

	if m.filesTab.search.searching &&
		m.filterInput.Value() != "" &&
		(m.filesTab.viewMode == managedViewUnmanaged || m.filesTab.viewMode == managedViewAll) {
		status = strings.TrimRight(status, " ") + " [searching...] "
	} else if m.filesTab.search.paused &&
		m.filterInput.Value() != "" &&
		(m.filesTab.viewMode == managedViewUnmanaged || m.filesTab.viewMode == managedViewAll) {
		status = strings.TrimRight(status, " ") + " [search paused] "
	}

	statusBar := activeTheme.StatusBar.Width(m.effectiveWidth()).Render(status)

	var help string
	switch {
	case m.overlays.showViewPicker:
		help = m.helpHint("↑/↓ navigate | space toggle/select | 1-4 quick view | enter apply | esc back")
	case m.actions.managedShow:
		help = m.helpHint("↑/↓ navigate | enter select | esc back")
	case m.panel.shouldShow(m.width) && m.panel.focusZone == panelFocusPanel:
		help = m.helpHint("↑/↓ scroll | ^d/^u half | g/G top/bottom | v switch diff/content | h/← back to files | p hide preview | esc back")
	default:
		panelHint := m.listPreviewHint()
		clearHint := ""
		if m.filesTab.treeView && strings.TrimSpace(m.filterInput.Value()) != "" && !m.filterInput.Focused() {
			clearHint = " | c clear search"
		}
		if m.filesTab.treeView {
			help = m.helpHint("↑/↓ nav | enter open/toggle | a actions | t flat | f view/filter | r refresh" + clearHint + panelHint + " | ? keys | esc back")
		} else {
			help = m.helpHint("↑/↓ nav | enter actions | a actions | t tree | f view/filter | r refresh" + panelHint + " | ? keys | esc back")
		}
	}
	return statusBar + "\n" + help
}
