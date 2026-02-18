package tui

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- Files tab action menus ---

// openFilesActiveMenu dispatches to the correct actions menu based on view mode.
func (m *Model) openFilesActiveMenu() {
	switch m.filesTab.viewMode {
	case managedViewIgnored:
		m.openFilesIgnoredMenu()
	case managedViewUnmanaged:
		m.openFilesUnmanagedMenu()
	case managedViewAll:
		path := m.selectedManagedPathForOpen()
		switch m.classifyPath(path) {
		case pathClassIgnored:
			m.openFilesIgnoredMenu()
		case pathClassUnmanaged:
			m.openFilesUnmanagedMenu()
		default:
			m.openFilesActionsMenu()
		}
	default:
		m.openFilesActionsMenu()
	}
}

func (m *Model) openFilesIgnoredMenu() {
	path := m.selectedManagedPath()
	if path == "" {
		return
	}
	m.actions.managedItems = nil
	fmCap := fileManagerCapability()

	isDir := false
	rows := m.activeTreeRows()
	if m.filesTab.treeView && m.filesTab.cursor < len(rows) {
		isDir = rows[m.filesTab.cursor].node.isDir
	}

	if isDir {
		m.actions.managedItems = appendActionItemWithCapability(m.actions.managedItems, "Open in File Manager", chezmoiActionOpenFileManager, fmCap)
	} else {
		m.actions.managedItems = append(m.actions.managedItems,
			chezmoiActionItem{label: "View .chezmoiignore", action: chezmoiActionViewIgnoreFile},
			chezmoiActionItem{label: "Edit .chezmoiignore ($EDITOR)", action: chezmoiActionEditIgnoreFile},
			chezmoiActionItem{label: "──────────", action: chezmoiActionNone},
		)
		m.actions.managedItems = appendActionItemWithCapability(m.actions.managedItems, "Open in File Manager", chezmoiActionOpenFileManager, fmCap)
	}

	m.actions.managedCursor = firstSelectableCursor(m.actions.managedItems)
	m.actions.managedShow = true
}

func (m *Model) openFilesActionsMenu() {
	path := m.selectedManagedPath()
	if path == "" {
		return
	}
	m.actions.managedItems = nil
	fmCap := fileManagerCapability()

	isDir := false
	rows := m.activeTreeRows()
	if m.filesTab.treeView && m.filesTab.cursor < len(rows) {
		isDir = rows[m.filesTab.cursor].node.isDir
	}

	readOnly := m.service.IsReadOnly()
	if isDir {
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems,
			"Forget Directory",
			chezmoiActionForgetFile,
			"", !readOnly,
			"read-only mode",
		)
		m.actions.managedItems = append(m.actions.managedItems, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
		m.actions.managedItems = appendActionItemWithCapability(m.actions.managedItems, "Open in File Manager", chezmoiActionOpenFileManager, fmCap)
	} else {
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems,
			"View Source",
			chezmoiActionViewSource,
			"", !readOnly,
			"read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems,
			"Edit Source ($EDITOR)",
			chezmoiActionEditSource,
			"", !readOnly,
			"read-only mode",
		)
		m.actions.managedItems = append(m.actions.managedItems, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems,
			"Apply File",
			chezmoiActionApplyManaged,
			"", !readOnly,
			"read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems,
			"Forget File",
			chezmoiActionForgetFile,
			"", !readOnly,
			"read-only mode",
		)
		m.actions.managedItems = append(m.actions.managedItems, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
		m.actions.managedItems = appendActionItemWithCapability(m.actions.managedItems, "Open in File Manager", chezmoiActionOpenFileManager, fmCap)
	}

	m.actions.managedCursor = firstSelectableCursor(m.actions.managedItems)
	m.actions.managedShow = true
}

func (m Model) executeFilesAction(action chezmoiAction) (tea.Model, tea.Cmd) {
	m.actions.managedShow = false
	path := m.selectedManagedPath()
	if path == "" {
		return m, nil
	}

	switch action {
	case chezmoiActionViewSource:
		if m.service.IsReadOnly() {
			m.ui.message = actionUnavailableMessage("read-only mode")
			return m, nil
		}
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadSourceContentCmd(path))

	case chezmoiActionEditSource:
		if m.service.IsReadOnly() {
			m.ui.message = actionUnavailableMessage("read-only mode")
			return m, nil
		}
		return m, m.editSourceCmd(path)

	case chezmoiActionForgetFile:
		if m.service.IsReadOnly() {
			m.ui.message = actionUnavailableMessage("read-only mode")
			return m, nil
		}
		// chezmoi forget requires an absolute path; selectedManagedPath()
		// returns relPath for directories, so resolve it.
		forgetPath := path
		if m.targetPath != "" && !filepath.IsAbs(forgetPath) {
			forgetPath = filepath.Join(m.targetPath, forgetPath)
		}
		m.overlays.confirmAction = chezmoiActionForgetFile
		m.overlays.confirmLabel = "forget " + path
		m.overlays.confirmPath = forgetPath
		m.view = ConfirmScreen
		return m, nil

	case chezmoiActionApplyManaged:
		if m.service.IsReadOnly() {
			m.ui.message = actionUnavailableMessage("read-only mode")
			return m, nil
		}
		m.overlays.confirmAction = chezmoiActionApplyManaged
		m.overlays.confirmLabel = "apply " + path
		m.overlays.confirmPath = path
		m.view = ConfirmScreen
		return m, nil

	case chezmoiActionOpenFileManager:
		openPathTarget := m.selectedManagedPathForOpen()
		if openPathTarget == "" {
			return m, nil
		}
		if err := openPath(openPathTarget); err != nil {
			m.ui.message = "Error: " + err.Error()
			return m, nil
		}
		m.ui.message = "Opened in file manager"
		return m, nil

	case chezmoiActionViewIgnoreFile:
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.loadIgnoreFileContentCmd())

	case chezmoiActionEditIgnoreFile:
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.resolveIgnoreFilePathCmd())

	case chezmoiActionEditTarget:
		absPath := m.selectedManagedPathForOpen()
		if absPath == "" {
			m.ui.message = "Error: no file selected"
			return m, nil
		}
		return m, m.editTargetCmd(absPath)

	case chezmoiActionAdd, chezmoiActionAddEncrypt, chezmoiActionAddTemplate,
		chezmoiActionAddAutoTemplate, chezmoiActionAddExact, chezmoiActionAddNoRecursive:
		if m.service.IsReadOnly() {
			m.ui.message = actionUnavailableMessage("read-only mode")
			return m, nil
		}
		absPath := m.selectedManagedPathForOpen()
		if absPath == "" {
			m.ui.message = "Error: no file selected"
			return m, nil
		}
		if err := m.service.Policy().ValidateTargetPath(absPath); err != nil {
			m.ui.message = "Error: " + err.Error()
			return m, nil
		}
		var opts chezmoi.AddOptions
		switch action {
		case chezmoiActionAddEncrypt:
			opts.Encrypt = true
		case chezmoiActionAddTemplate:
			opts.Template = true
		case chezmoiActionAddAutoTemplate:
			opts.AutoTemplate = true
		case chezmoiActionAddExact:
			opts.Exact = true
		case chezmoiActionAddNoRecursive:
			opts.NoRecursive = true
		}
		m.ui.busyAction = true
		return m, tea.Batch(m.ui.loadingSpinner.Tick, m.addFileCmd(absPath, opts))
	}

	return m, nil
}

// --- Unmanaged Actions Menu ---

func (m *Model) openFilesUnmanagedMenu() {
	path := m.selectedManagedPath()
	if path == "" {
		return
	}
	m.actions.managedItems = nil
	fmCap := fileManagerCapability()
	canAdd := !m.service.IsReadOnly()

	isDir := false
	rows := m.activeTreeRows()
	if m.filesTab.treeView && m.filesTab.cursor < len(rows) {
		isDir = rows[m.filesTab.cursor].node.isDir
	}

	if isDir {
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add", chezmoiActionAdd,
			"Add directory and all contents recursively\ncmd: chezmoi add <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add (Exact)", chezmoiActionAddExact,
			"Add directory exactly — untracked files inside will be removed on apply\ncmd: chezmoi add --exact <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add (Shallow)", chezmoiActionAddNoRecursive,
			"Add directory and immediate children only (non-recursive)\ncmd: chezmoi add --recursive=false <path>",
			canAdd, "read-only mode",
		)
	} else {
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add", chezmoiActionAdd,
			"Add file to source state\ncmd: chezmoi add <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add (Encrypted)", chezmoiActionAddEncrypt,
			"Add file encrypted with age/gpg\ncmd: chezmoi add --encrypt <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add (Template)", chezmoiActionAddTemplate,
			"Add file as a chezmoi template\ncmd: chezmoi add --template <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Add (Auto Template)", chezmoiActionAddAutoTemplate,
			"Add file and auto-detect template variables\ncmd: chezmoi add --autotemplate <path>",
			canAdd, "read-only mode",
		)
		m.actions.managedItems = append(m.actions.managedItems, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
		m.actions.managedItems = appendActionItem(
			m.actions.managedItems, "Open in Editor ($EDITOR)", chezmoiActionEditTarget,
			"Open file in your configured editor",
			true, "",
		)
	}

	m.actions.managedItems = append(m.actions.managedItems, chezmoiActionItem{label: "──────────", action: chezmoiActionNone})
	m.actions.managedItems = appendActionItemWithCapability(
		m.actions.managedItems, "Open in File Manager", chezmoiActionOpenFileManager, fmCap,
	)

	m.actions.managedCursor = firstSelectableCursor(m.actions.managedItems)
	m.actions.managedShow = true
}

// --- Files state helpers ---

// reflattenActiveTree re-flattens the tree for the current view mode and clamps the cursor.
func (m *Model) reflattenActiveTree() {
	fv := &m.filesTab.views[m.filesTab.viewMode]
	if strings.TrimSpace(m.filterInput.Value()) != "" {
		paths := m.activeFlatFiles()
		projected := buildManagedTree(paths, m.targetPath)
		for _, root := range projected.roots {
			expandAll(root)
		}
		collapsed := collapsedDirsFromRows(fv.treeRows)
		applyCollapsedDirs(projected.roots, collapsed)
		fv.treeRows = flattenManagedTree(projected)
	} else {
		fv.treeRows = flattenManagedTree(fv.tree)
	}
	if m.filesTab.cursor >= len(fv.treeRows) {
		m.filesTab.cursor = max(0, len(fv.treeRows)-1)
	}
}

func collapsedDirsFromRows(rows []flatTreeRow) map[string]struct{} {
	if len(rows) == 0 {
		return nil
	}

	collapsed := make(map[string]struct{})
	for _, row := range rows {
		if row.node == nil || !row.node.isDir || row.node.expanded || row.node.relPath == "" {
			continue
		}
		collapsed[row.node.relPath] = struct{}{}
	}
	if len(collapsed) == 0 {
		return nil
	}
	return collapsed
}

func applyCollapsedDirs(nodes []*managedTreeNode, collapsed map[string]struct{}) {
	if len(nodes) == 0 || len(collapsed) == 0 {
		return
	}

	var walk func(*managedTreeNode)
	walk = func(node *managedTreeNode) {
		if node == nil || !node.isDir {
			return
		}

		if _, shouldCollapse := collapsed[node.relPath]; shouldCollapse {
			node.expanded = false
		}
		for _, child := range node.children {
			walk(child)
		}
	}

	for _, root := range nodes {
		walk(root)
	}
}

func (m *Model) reflattenTreeForView(mode managedViewMode) {
	fv := &m.filesTab.views[mode]
	fv.treeRows = flattenManagedTree(fv.tree)
	if mode == m.filesTab.viewMode && m.filesTab.cursor >= len(fv.treeRows) {
		m.filesTab.cursor = max(0, len(fv.treeRows)-1)
	}
}

// rebuildFileViewTree rebuilds the tree and flat rows for the given view mode.
// For managedViewAll, it merges all three sources and builds classification sets.
func (m *Model) rebuildFileViewTree(mode managedViewMode) {
	if mode == managedViewAll {
		m.rebuildAllFileViewTree()
		return
	}
	fv := &m.filesTab.views[mode]
	fv.tree = buildManagedTree(fv.files, m.targetPath)
	fv.treeRows = flattenManagedTree(fv.tree)
}

// rebuildAllFileViewTree builds the combined All-view tree from the dataset.
func (m *Model) rebuildAllFileViewTree() {
	m.filesTab.views[managedViewAll].tree = buildManagedTree(m.filesTab.dataset.allPaths, m.targetPath)
	for _, root := range m.filesTab.views[managedViewAll].tree.roots {
		expandAll(root)
	}
	m.filesTab.views[managedViewAll].treeRows = flattenManagedTree(m.filesTab.views[managedViewAll].tree)
}

func (m Model) selectedManagedPath() string {
	if m.filesTab.treeView {
		rows := m.activeTreeRows()
		if m.filesTab.cursor >= 0 && m.filesTab.cursor < len(rows) {
			row := rows[m.filesTab.cursor]
			if row.node.isDir {
				return row.node.relPath
			}
			return row.node.absPath
		}
		return ""
	}
	files := m.activeFlatFiles()
	if m.filesTab.cursor >= 0 && m.filesTab.cursor < len(files) {
		return files[m.filesTab.cursor]
	}
	return ""
}

func (m Model) selectedManagedPathForOpen() string {
	if !m.filesTab.treeView {
		return m.selectedManagedPath()
	}
	rows := m.activeTreeRows()
	if m.filesTab.cursor < 0 || m.filesTab.cursor >= len(rows) {
		return ""
	}
	row := rows[m.filesTab.cursor]
	if !row.node.isDir {
		return row.node.absPath
	}
	if m.targetPath == "" {
		return ""
	}
	return filepath.Join(m.targetPath, row.node.relPath)
}

// --- Files tab query helpers ---

// activeTreeRows returns the tree rows for the current view mode.
func (m Model) activeTreeRows() []flatTreeRow {
	return m.filesTab.views[m.filesTab.viewMode].treeRows
}

// activeFlatFiles returns the flat file list for the current view mode.
func (m Model) activeFlatFiles() []string {
	switch m.filesTab.viewMode {
	case managedViewAll:
		merged := make([]string, 0, len(m.filesTab.views[managedViewManaged].filteredFiles)+len(m.filesTab.views[managedViewIgnored].filteredFiles)+len(m.filesTab.views[managedViewUnmanaged].filteredFiles))
		merged = append(merged, m.filesTab.views[managedViewManaged].filteredFiles...)
		merged = append(merged, m.filesTab.views[managedViewIgnored].filteredFiles...)
		merged = append(merged, m.filesTab.views[managedViewUnmanaged].filteredFiles...)
		return merged
	default:
		return m.filesTab.views[m.filesTab.viewMode].filteredFiles
	}
}

// activeTree returns the tree structure for the current view mode.
func (m Model) activeTree() managedTree {
	return m.filesTab.views[m.filesTab.viewMode].tree
}

func (m Model) classifyPath(absPath string) pathClass {
	return m.filesTab.dataset.classify(absPath)
}
