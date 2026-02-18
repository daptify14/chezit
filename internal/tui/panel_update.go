package tui

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

const (
	panelPreviewMaxBytes    = 256 * 1024
	panelBinarySampleWindow = 4096
)

// handlePanelKeys routes key events when the panel has focus.
// Returns the updated model, command, and whether the key was handled.
func (m Model) handlePanelKeys(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, ChezPanelKeys.FocusList):
		m.panel.focusZone = panelFocusList
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.ContentMode):
		if m.panel.contentMode == panelModeDiff {
			m.panel.contentMode = panelModeContent
		} else {
			m.panel.contentMode = panelModeDiff
		}
		updated, cmd := m.panelLoadForCurrentTab()
		return updated, cmd, true

	case key.Matches(msg, ChezPanelKeys.ScrollDown):
		m = m.syncPanelViewportContent()
		m.panel.viewport.ScrollDown(navigationStepForKey(msg))
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.ScrollUp):
		m = m.syncPanelViewportContent()
		m.panel.viewport.ScrollUp(navigationStepForKey(msg))
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.HalfDown):
		m = m.syncPanelViewportContent()
		m.panel.viewport.HalfPageDown()
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.HalfUp):
		m = m.syncPanelViewportContent()
		m.panel.viewport.HalfPageUp()
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.Top):
		m = m.syncPanelViewportContent()
		m.panel.viewport.GotoTop()
		return m, nil, true

	case key.Matches(msg, ChezPanelKeys.Bottom):
		m = m.syncPanelViewportContent()
		m.panel.viewport.GotoBottom()
		return m, nil, true
	}

	return m, nil, false
}

// panelLoadForCurrentTab dispatches a panel load based on the active tab.
func (m Model) panelLoadForCurrentTab() (Model, tea.Cmd) {
	switch m.activeTabName() {
	case "Status":
		return m.panelLoadForChanges()
	case "Files":
		return m.panelLoadForManaged()
	}
	return m, nil
}

// panelLoadForChanges loads panel content for the currently selected changes row.
func (m Model) panelLoadForChanges() (Model, tea.Cmd) {
	row := m.currentChangesRow()
	if row.isHeader {
		m.panel.currentPath = ""
		m.panel.pendingLoad = false
		m = m.syncPanelViewportContent()
		return m, nil
	}

	var path string
	var section changesSection

	switch {
	case row.driftFile != nil:
		path = row.driftFile.Path
		section = row.section
	case row.gitFile != nil:
		path = row.gitFile.Path
		section = row.section
	case row.commit != nil:
		path = row.commit.Hash
		section = row.section
	default:
		return m, nil
	}

	if path == "" {
		m.panel.pendingLoad = false
		return m, nil
	}

	m.panel.currentPath = path
	m.panel.currentSection = section

	// Check cache — path is set so renderFilePanel can display it
	if _, ok := m.panel.cacheGet(path, m.panel.contentMode, section); ok {
		m = m.syncPanelViewportContent()
		return m, nil
	}

	// Queue the latest target while a load is in flight.
	if m.panel.loading {
		m.panel.pendingPath = path
		m.panel.pendingMode = m.panel.contentMode
		m.panel.pendingSection = section
		m.panel.pendingLoad = true
		m = m.syncPanelViewportContent()
		return m, nil
	}

	m.panel.pendingLoad = false
	m.panel.loading = true
	m = m.syncPanelViewportContent()
	return m, m.panelContentCmd(path, m.panel.contentMode, section)
}

// panelLoadForManaged loads panel content for the currently selected managed file.
func (m Model) panelLoadForManaged() (Model, tea.Cmd) {
	// In tree view, check if selected row is a directory
	if m.filesTab.treeView {
		rows := m.activeTreeRows()
		if m.filesTab.cursor >= 0 && m.filesTab.cursor < len(rows) {
			row := rows[m.filesTab.cursor]
			if row.node.isDir {
				m.panel.currentPath = ""
				m.panel.pendingLoad = false
				m = m.syncPanelViewportContent()
				return m, nil
			}
		}
	}

	path := m.selectedManagedPath()
	if path == "" {
		m.panel.currentPath = ""
		m.panel.pendingLoad = false
		m = m.syncPanelViewportContent()
		return m, nil
	}

	m.panel.currentPath = path
	m.panel.currentSection = changesSectionDrift // default section for managed

	// Check cache — path is set so renderFilePanel can display it
	if _, ok := m.panel.cacheGet(path, m.panel.contentMode, changesSectionDrift); ok {
		m = m.syncPanelViewportContent()
		return m, nil
	}

	// Queue the latest target while a load is in flight.
	if m.panel.loading {
		m.panel.pendingPath = path
		m.panel.pendingMode = m.panel.contentMode
		m.panel.pendingSection = changesSectionDrift
		m.panel.pendingLoad = true
		m = m.syncPanelViewportContent()
		return m, nil
	}

	m.panel.pendingLoad = false
	m.panel.loading = true
	m = m.syncPanelViewportContent()
	return m, m.panelContentCmd(path, m.panel.contentMode, changesSectionDrift)
}

// panelContentCmd returns a tea.Cmd that asynchronously loads content for the panel.
func (m Model) panelContentCmd(path string, mode panelContentMode, section changesSection) tea.Cmd {
	readOnly := m.service.IsReadOnly()

	return func() tea.Msg {
		var content string
		var err error

		switch mode {
		case panelModeDiff:
			switch section {
			case changesSectionDrift:
				content, err = m.service.Diff(path)
			case changesSectionUnstaged:
				if !readOnly {
					content, err = m.service.GitDiff(path, false)
				}
			case changesSectionStaged:
				if !readOnly {
					content, err = m.service.GitDiff(path, true)
				}
			case changesSectionUnpushed, changesSectionIncoming:
				content, err = m.service.GitShow(path)
			default:
				content, err = m.service.Diff(path)
			}

		case panelModeContent:
			if section == changesSectionUnpushed || section == changesSectionIncoming {
				return panelContentLoadedMsg{
					path: path, mode: mode, section: section,
					err: newPanelPreviewError("Use [diff] view to see commit changes"),
				}
			}
			content, err = m.panelLoadContentPreview(path, section)
		}

		return panelContentLoadedMsg{
			path:    path,
			mode:    mode,
			section: section,
			content: content,
			err:     err,
		}
	}
}

// handlePanelContentLoaded processes the async content result.
// It guards against stale results and updates the cache + viewport.
func (m Model) handlePanelContentLoaded(msg panelContentLoadedMsg) (Model, tea.Cmd) {
	m.panel.loading = false

	// Cache the result regardless of staleness
	lines := strings.Split(msg.content, "\n")
	m.panel.cachePut(msg.path, msg.mode, msg.section, panelCacheEntry{
		content: msg.content,
		lines:   lines,
		err:     msg.err,
	})

	// Stale async guard: only update viewport if this matches current state
	if msg.path == m.panel.currentPath && msg.mode == m.panel.contentMode && msg.section == m.panel.currentSection {
		// Reset viewport scroll position on new content
		if m.panel.viewportReady {
			m.panel.viewport.GotoTop()
		}
		m = m.syncPanelViewportContent()
	}

	if m.panel.pendingLoad {
		pendingPath := m.panel.pendingPath
		pendingMode := m.panel.pendingMode
		pendingSection := m.panel.pendingSection
		m.panel.pendingLoad = false

		if pendingPath == "" {
			return m, nil
		}
		m.panel.currentPath = pendingPath
		m.panel.currentSection = pendingSection
		m.panel.contentMode = pendingMode

		if _, ok := m.panel.cacheGet(pendingPath, pendingMode, pendingSection); ok {
			m = m.syncPanelViewportContent()
			return m, nil
		}

		m.panel.loading = true
		m = m.syncPanelViewportContent()
		return m, m.panelContentCmd(pendingPath, pendingMode, pendingSection)
	}

	return m, nil
}

func (m Model) panelViewportDimensions() (contentWidth, viewportHeight int, ok bool) {
	if !m.panel.shouldShow(m.width) {
		return 0, 0, false
	}

	panelW := panelWidthFor(m.width)
	contentWidth = max(panelW-4, 20) // border + padding

	var panelH int
	switch m.activeTabName() {
	case "Status":
		panelH = m.chezmoiChangesListHeight() + 4
	case "Files":
		panelH = m.chezmoiManagedListHeight() + 4
	default:
		panelH = max(m.height-2, 8)
	}
	viewportHeight = max(panelH-3, 5) // title + divider + status
	return contentWidth, viewportHeight, true
}

func (m Model) panelViewportContentForWidth(contentWidth int) string {
	switch {
	case m.panel.loading:
		return activeTheme.DimText.Render("  Loading...")
	case m.panel.currentPath == "":
		return activeTheme.DimText.Render("  Select an item to preview")
	default:
		entry, ok := m.panel.cacheGet(m.panel.currentPath, m.panel.contentMode, m.panel.currentSection)
		if !ok {
			return activeTheme.DimText.Render("  Loading...")
		}
		if entry.err != nil {
			return activeTheme.DimText.Render("  " + panelErrorText(entry.err))
		}
		return m.renderPanelViewportContent(entry.lines, contentWidth)
	}
}

func (m Model) syncPanelViewportContent() Model {
	contentWidth, viewportHeight, ok := m.panelViewportDimensions()
	if !ok {
		return m
	}
	m.panel.ensureViewport(contentWidth, viewportHeight)
	content := m.panelViewportContentForWidth(contentWidth)
	currentOffset := m.panel.viewport.YOffset()
	m.panel.viewport.SetContent(content)
	m.panel.viewport.SetYOffset(currentOffset)
	return m
}

// panelPreviewError is a user-facing, non-fatal panel message.
// It should render without an "Error:" prefix.
type panelPreviewError struct {
	msg string
}

func (e panelPreviewError) Error() string { return e.msg }

func (e panelPreviewError) UserMessage() string { return e.msg }

type panelUserMessageError interface {
	error
	UserMessage() string
}

func newPanelPreviewError(msg string) error {
	return panelPreviewError{msg: msg}
}

func panelErrorText(err error) string {
	if err == nil {
		return ""
	}
	if userErr, ok := errors.AsType[panelUserMessageError](err); ok {
		return userErr.UserMessage()
	}
	return "Error: " + err.Error()
}

func (m Model) panelLoadContentPreview(path string, section changesSection) (string, error) {
	switch section {
	case changesSectionUnstaged, changesSectionStaged:
		return m.panelReadSourceFile(path)
	default:
		// On the Status tab, drift items should show the actual local file
		// (not the chezmoi-rendered source state from CatTarget).
		if section == changesSectionDrift && m.activeTabName() == "Status" {
			return readPanelLocalFileWithNotFoundMessage(path, "File not found on disk")
		}
		return m.panelReadTargetFile(path)
	}
}

func (m Model) panelReadTargetFile(path string) (string, error) {
	content, err := m.service.CatTarget(path)
	if err != nil {
		mappedErr := mapPanelTargetPreviewError(err)
		if m.shouldFallbackToLocalTargetPreview(path, mappedErr) {
			localContent, localErr := readPanelLocalFileWithNotFoundMessage(path, "File not found on disk")
			if localErr == nil {
				return localContent, nil
			}
			return "", localErr
		}
		return "", mappedErr
	}
	data := []byte(content)
	if len(data) > panelPreviewMaxBytes {
		return "", newPanelPreviewError(
			fmt.Sprintf("File too large to preview (%s > %s)", panelFormatBytes(uint64(len(data))), panelFormatBytes(panelPreviewMaxBytes)),
		)
	}
	if isLikelyBinaryContent(data) {
		return "", newPanelPreviewError("Binary file (preview disabled)")
	}
	return content, nil
}

func (m Model) shouldFallbackToLocalTargetPreview(path string, err error) bool {
	if m.activeTabName() != "Files" {
		return false
	}
	if !filepath.IsAbs(path) {
		return false
	}
	return panelIsNotManagedPreviewError(err)
}

func panelIsNotManagedPreviewError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(panelErrorText(err)), "not managed")
}

func (m Model) panelReadSourceFile(path string) (string, error) {
	sourceDir, err := m.service.SourceDir()
	if err != nil {
		return "", newPanelPreviewError("Preview unavailable (cannot locate chezmoi source directory)")
	}
	sourcePath, err := resolvePanelSourcePath(sourceDir, path, m.targetPath)
	if err != nil {
		return "", err
	}
	return readPanelLocalFile(sourcePath)
}

func resolvePanelSourcePath(sourceDir, path, targetPath string) (string, error) {
	if sourceDir == "" {
		return "", newPanelPreviewError("Preview unavailable (cannot locate chezmoi source directory)")
	}
	clean, isAbs, err := normalizePanelSourceLookupPath(path)
	if err != nil {
		return "", err
	}

	candidates := panelSourceLookupCandidates(filepath.Clean(sourceDir), clean, isAbs, targetPath)
	for _, candidate := range candidates {
		if _, err := os.Lstat(candidate); err == nil {
			return candidate, nil
		}
	}
	if len(candidates) == 0 {
		return "", newPanelPreviewError("Preview unavailable for this path")
	}
	// Return the best-effort mapping for downstream error reporting.
	return candidates[0], nil
}

func normalizePanelSourceLookupPath(path string) (clean string, isAbs bool, err error) {
	clean = filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == "" {
		return "", false, newPanelPreviewError("Preview unavailable for this path")
	}
	isAbs = filepath.IsAbs(clean)
	if !isAbs && (clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))) {
		return "", false, newPanelPreviewError("Preview unavailable for this path")
	}
	return clean, isAbs, nil
}

func panelSourceLookupCandidates(sourceDir, clean string, isAbs bool, targetPath string) []string {
	candidates := make([]string, 0, 12)
	seen := make(map[string]struct{}, 12)

	addCandidate := func(candidate string) {
		if candidate == "" {
			return
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	addRelativeCandidates := func(base, rel string) {
		if base == "" || rel == "" {
			return
		}
		addCandidate(filepath.Join(base, rel))
		homePrefix := "home" + string(filepath.Separator)
		if after, ok := strings.CutPrefix(rel, homePrefix); ok {
			addCandidate(filepath.Join(base, after))
		} else {
			addCandidate(filepath.Join(base, "home", rel))
		}
	}

	if isAbs {
		addCandidate(clean)
		if rel, ok := panelRelativeHomePathFromAbsolute(clean, targetPath); ok {
			addRelativeCandidates(sourceDir, rel)
			parent := filepath.Dir(sourceDir)
			if parent != sourceDir {
				addRelativeCandidates(parent, rel)
			}
		}
		return candidates
	}

	addRelativeCandidates(sourceDir, clean)
	parent := filepath.Dir(sourceDir)
	if parent != sourceDir {
		addRelativeCandidates(parent, clean)
		grandparent := filepath.Dir(parent)
		if grandparent != parent {
			addRelativeCandidates(grandparent, clean)
		}
	}
	return candidates
}

func panelRelativeHomePathFromAbsolute(absPath, targetPath string) (string, bool) {
	if targetPath != "" {
		rel, err := filepath.Rel(targetPath, absPath)
		if err == nil && rel != "." && rel != "" &&
			rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rel, true
		}
	}
	return "", false
}

func readPanelLocalFile(path string) (string, error) {
	return readPanelLocalFileWithNotFoundMessage(path, "File not found in chezmoi source")
}

func readPanelLocalFileWithNotFoundMessage(path, notFoundMessage string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", newPanelPreviewError(notFoundMessage)
		}
		return "", fmt.Errorf("preview stat failed for %q: %w", path, err)
	}
	if info.IsDir() {
		return "", newPanelPreviewError("Directory selected; preview skipped")
	}

	targetInfo := info
	if info.Mode()&os.ModeSymlink != 0 {
		targetInfo, err = os.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", newPanelPreviewError("Broken symlink; preview skipped")
			}
			return "", fmt.Errorf("preview stat failed for symlink target %q: %w", path, err)
		}
		if targetInfo.IsDir() {
			return "", newPanelPreviewError("Symlink points to a directory; preview skipped")
		}
	}

	if !targetInfo.Mode().IsRegular() {
		return "", newPanelPreviewError("Preview unavailable for this entry type")
	}
	var fileSize uint64
	if s := targetInfo.Size(); s > 0 {
		fileSize = uint64(s)
	}
	if fileSize > panelPreviewMaxBytes {
		return "", newPanelPreviewError(
			fmt.Sprintf("File too large to preview (%s > %s)", panelFormatBytes(fileSize), panelFormatBytes(panelPreviewMaxBytes)),
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("preview read failed for %q: %w", path, err)
	}
	if isLikelyBinaryContent(data) {
		return "", newPanelPreviewError("Binary file (preview disabled)")
	}
	return string(data), nil
}

func mapPanelTargetPreviewError(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not managed"):
		return newPanelPreviewError("File is not managed by chezmoi")
	case strings.Contains(msg, "not a file, script, or symlink"):
		return newPanelPreviewError("Entry is not previewable (not a file/script/symlink)")
	case strings.Contains(msg, "is a directory"):
		return newPanelPreviewError("Directory selected; preview skipped")
	default:
		return err
	}
}

func isLikelyBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > panelBinarySampleWindow {
		sample = sample[:panelBinarySampleWindow]
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if !utf8.Valid(sample) {
		return true
	}
	control := 0
	for _, b := range sample {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' && b != '\f' && b != '\b' {
			control++
		}
	}
	return control > len(sample)/10
}

func panelFormatBytes(size uint64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)
	switch {
	case size >= gib:
		return fmt.Sprintf("%.1f GiB", float64(size)/float64(gib))
	case size >= mib:
		return fmt.Sprintf("%.1f MiB", float64(size)/float64(mib))
	case size >= kib:
		return fmt.Sprintf("%.1f KiB", float64(size)/float64(kib))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
