package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/daptify14/chezit/internal/chezmoi"
)

const filesSearchDebounceDelay = 50 * time.Millisecond

func (m *Model) applyActiveFilter() {
	switch m.activeTabName() {
	case "Status":
		m.applyChezmoiFilter()
	case "Files":
		m.applyManagedFilter()
	}
}

func (m *Model) applyChezmoiFilter() {
	query := m.filterInput.Value()
	if query == "" {
		m.status.filteredFiles = m.status.files
		m.buildChangesRows()
		return
	}

	paths := make([]string, len(m.status.files))
	for i, f := range m.status.files {
		paths[i] = f.Path
	}

	matches := fuzzy.Find(query, paths)
	filtered := make([]chezmoi.FileStatus, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, m.status.files[match.Index])
	}
	m.status.filteredFiles = filtered
	m.buildChangesRows()
}

func (m *Model) applyManagedFilter() {
	query := m.filterInput.Value()
	homePrefix := m.targetPath
	if homePrefix != "" && !strings.HasSuffix(homePrefix, "/") {
		homePrefix += "/"
	}

	switch m.filesTab.viewMode {
	case managedViewIgnored:
		if query == "" {
			m.resetViewFilter(managedViewIgnored)
			return
		}
		m.filterSingleView(managedViewIgnored, query, homePrefix)

	case managedViewUnmanaged:
		if query == "" {
			m.resetViewFilter(managedViewUnmanaged)
			return
		}
		searchPaths := m.filesSearchPathsForMode(query, managedViewUnmanaged)
		filtered, _ := fuzzyMatchFiles(query, searchPaths, homePrefix)
		m.filesTab.views[managedViewUnmanaged].filteredFiles = filtered
		if m.filesTab.treeView {
			m.filesTab.views[managedViewUnmanaged].treeRows = projectFilteredTreeRows(filtered, m.targetPath)
		}
		m.filesTab.cursor = 0

	case managedViewAll:
		m.filterAllView(query, homePrefix)

	default: // managedViewManaged
		if query == "" {
			m.resetViewFilter(managedViewManaged)
			return
		}
		m.filterSingleView(managedViewManaged, query, homePrefix)
	}
}

// resetViewFilter resets a single view to its full file list and rebuilds tree rows.
func (m *Model) resetViewFilter(mode managedViewMode) {
	m.filesTab.views[mode].filteredFiles = m.filesTab.views[mode].files
	if m.filesTab.cursor >= len(m.filesTab.views[mode].filteredFiles) {
		m.filesTab.cursor = max(0, len(m.filesTab.views[mode].filteredFiles)-1)
	}
	if m.filesTab.treeView {
		m.filesTab.views[mode].treeRows = flattenManagedTree(m.filesTab.views[mode].tree)
	}
}

// filterSingleView fuzzy-filters a view's files using its existing tree and updates tree rows.
func (m *Model) filterSingleView(mode managedViewMode, query, homePrefix string) {
	filtered, matchSet := fuzzyMatchFiles(query, m.filesTab.views[mode].files, homePrefix)
	m.filesTab.views[mode].filteredFiles = filtered
	if m.filesTab.treeView {
		m.filesTab.views[mode].treeRows = filterManagedTree(m.filesTab.views[mode].tree, matchSet)
	}
	m.filesTab.cursor = 0
}

// filterAllView handles the all-view case: merge sources, fuzzy match, partition back.
func (m *Model) filterAllView(query, homePrefix string) {
	if query == "" {
		m.filesTab.views[managedViewManaged].filteredFiles = m.filesTab.views[managedViewManaged].files
		m.filesTab.views[managedViewIgnored].filteredFiles = m.filesTab.views[managedViewIgnored].files
		m.filesTab.views[managedViewUnmanaged].filteredFiles = m.filesTab.views[managedViewUnmanaged].files
		allFiles := make([]string, 0,
			len(m.filesTab.views[managedViewManaged].files)+
				len(m.filesTab.views[managedViewIgnored].files)+
				len(m.filesTab.views[managedViewUnmanaged].files))
		allFiles = append(allFiles, m.filesTab.views[managedViewManaged].files...)
		allFiles = append(allFiles, m.filesTab.views[managedViewIgnored].files...)
		allFiles = append(allFiles, m.filesTab.views[managedViewUnmanaged].files...)
		if m.filesTab.cursor >= len(allFiles) {
			m.filesTab.cursor = max(0, len(allFiles)-1)
		}
		if m.filesTab.treeView {
			m.filesTab.views[managedViewAll].treeRows = flattenManagedTree(m.filesTab.views[managedViewAll].tree)
		}
		return
	}

	managedSearchPaths := m.filesSearchPathsForMode(query, managedViewManaged)
	ignoredSearchPaths := m.filesSearchPathsForMode(query, managedViewIgnored)
	unmanagedSearchPaths := m.filesSearchPathsForMode(query, managedViewUnmanaged)
	allSearchPaths := make([]string, 0, len(managedSearchPaths)+len(ignoredSearchPaths)+len(unmanagedSearchPaths))
	allSearchPaths = append(allSearchPaths, managedSearchPaths...)
	allSearchPaths = append(allSearchPaths, ignoredSearchPaths...)
	allSearchPaths = append(allSearchPaths, unmanagedSearchPaths...)

	_, matchSet := fuzzyMatchFiles(query, allSearchPaths, homePrefix)
	m.filesTab.views[managedViewManaged].filteredFiles = filterByMatchSet(managedSearchPaths, matchSet)
	m.filesTab.views[managedViewIgnored].filteredFiles = filterByMatchSet(ignoredSearchPaths, matchSet)
	m.filesTab.views[managedViewUnmanaged].filteredFiles = filterByMatchSet(unmanagedSearchPaths, matchSet)
	if m.filesTab.treeView {
		projectedPaths := make([]string, 0,
			len(m.filesTab.views[managedViewManaged].filteredFiles)+
				len(m.filesTab.views[managedViewIgnored].filteredFiles)+
				len(m.filesTab.views[managedViewUnmanaged].filteredFiles))
		projectedPaths = append(projectedPaths, m.filesTab.views[managedViewManaged].filteredFiles...)
		projectedPaths = append(projectedPaths, m.filesTab.views[managedViewIgnored].filteredFiles...)
		projectedPaths = append(projectedPaths, m.filesTab.views[managedViewUnmanaged].filteredFiles...)
		m.filesTab.views[managedViewAll].treeRows = projectFilteredTreeRows(projectedPaths, m.targetPath)
	}
	m.filesTab.cursor = 0
}

// projectSearchResults projects canonical search raw results into a view mode
// using the dataset classification map and unmanaged-dir fallback.
// Unknown paths are excluded by design.
func projectSearchResults(mode managedViewMode, rawResults []string, dataset filesDataset) []string {
	if len(rawResults) == 0 || !dataset.ready {
		return nil
	}

	projected := make([]string, 0, len(rawResults))
	seen := make(map[string]struct{}, len(rawResults))
	for _, p := range rawResults {
		normalized := normalizePath(p)
		if normalized == "" {
			continue
		}
		fc, ok := dataset.classifyKnown(normalized)
		if !ok {
			continue
		}
		if mode != managedViewAll && !projectModeIncludesClass(mode, fc) {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		projected = append(projected, normalized)
	}
	return projected
}

func projectModeIncludesClass(mode managedViewMode, class fileClass) bool {
	switch mode {
	case managedViewManaged:
		return class == fileClassManaged
	case managedViewIgnored:
		return class == fileClassIgnored
	case managedViewUnmanaged:
		return class == fileClassUnmanaged
	case managedViewAll:
		return true
	default:
		return false
	}
}

// filterByMatchSet returns the subset of paths present in the matchSet.
func filterByMatchSet(paths []string, matchSet map[string]bool) []string {
	result := make([]string, 0, len(paths))
	for _, f := range paths {
		if matchSet[f] {
			result = append(result, f)
		}
	}
	return result
}

// fuzzyMatchFiles performs fuzzy matching against relative paths (home prefix stripped)
// for better scoring, then returns the matched absolute paths and a matchSet for tree filtering.
func fuzzyMatchFiles(query string, absPaths []string, homePrefix string) (filtered []string, matchSet map[string]bool) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return nil, map[string]bool{}
	}
	queryLower := strings.ToLower(trimmedQuery)

	// Build relative paths for fuzzy matching (what the user sees in the tree)
	relPaths := make([]string, len(absPaths))
	for i, p := range absPaths {
		if strings.HasPrefix(p, homePrefix) {
			relPaths[i] = p[len(homePrefix):]
		} else {
			relPaths[i] = p
		}
	}

	// Tighten matching semantics:
	// require a contiguous substring hit first, then fuzzy-rank that subset.
	candidateRel := make([]string, 0, len(relPaths))
	candidateAbs := make([]string, 0, len(absPaths))
	for i, rel := range relPaths {
		if strings.Contains(strings.ToLower(rel), queryLower) {
			candidateRel = append(candidateRel, rel)
			candidateAbs = append(candidateAbs, absPaths[i])
		}
	}
	if len(candidateRel) == 0 {
		return nil, map[string]bool{}
	}

	matches := fuzzy.Find(trimmedQuery, candidateRel)
	filtered = make([]string, 0, len(matches))
	matchSet = make(map[string]bool, len(matches))
	for _, match := range matches {
		abs := candidateAbs[match.Index]
		filtered = append(filtered, abs)
		matchSet[abs] = true
	}
	return filtered, matchSet
}

func (m Model) filesSearchPathsForMode(query string, mode managedViewMode) []string {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		switch mode {
		case managedViewManaged:
			return m.filesTab.views[managedViewManaged].files
		case managedViewIgnored:
			return m.filesTab.views[managedViewIgnored].files
		case managedViewUnmanaged:
			return m.filesTab.views[managedViewUnmanaged].files
		case managedViewAll:
			all := make([]string, 0,
				len(m.filesTab.views[managedViewManaged].files)+
					len(m.filesTab.views[managedViewIgnored].files)+
					len(m.filesTab.views[managedViewUnmanaged].files))
			all = append(all, m.filesTab.views[managedViewManaged].files...)
			all = append(all, m.filesTab.views[managedViewIgnored].files...)
			all = append(all, m.filesTab.views[managedViewUnmanaged].files...)
			return all
		default:
			return nil
		}
	}
	if m.filesTab.search.ready && m.filesTab.search.query == trimmedQuery {
		return projectSearchResults(mode, m.filesTab.search.rawResults, m.filesTab.dataset)
	}
	// Avoid provisional shallow-root matches while deep search is pending.
	// Showing only final deep-search results prevents list churn where entries
	// appear/disappear after the async walk completes.
	return nil
}

func (m *Model) cancelFilesSearch() {
	if m.filesTab.search.cancel != nil {
		m.filesTab.search.cancel()
		m.filesTab.search.cancel = nil
	}
}

func (m *Model) resetFilesSearch(incrementRequest bool) {
	m.cancelFilesSearch()
	if incrementRequest {
		m.filesTab.search.request++
	}
	m.filesTab.search.rawResults = nil
	m.filesTab.search.searching = false
	m.filesTab.search.paused = false
	m.filesTab.search.ready = false
	m.filesTab.search.query = ""
}

func (m *Model) pauseFilesSearch() {
	query := strings.TrimSpace(m.filterInput.Value())
	if query == "" {
		m.resetFilesSearch(true)
		return
	}

	if m.filesTab.search.searching {
		if m.filesTab.search.cancel != nil {
			m.filesTab.search.cancel()
			m.filesTab.search.cancel = nil
		} else {
			// Only debounce is in flight; bump request to ignore stale debounce tick.
			m.filesTab.search.request++
		}
		m.filesTab.search.searching = false
	}

	m.filesTab.search.query = query
	m.filesTab.search.paused = true
}

func (m *Model) triggerFilesSearchIfNeeded() tea.Cmd {
	query := strings.TrimSpace(m.filterInput.Value())
	if m.activeTabName() != "Files" {
		return nil
	}
	if query == "" {
		m.resetFilesSearch(true)
		return nil
	}
	if m.filesTab.viewMode != managedViewUnmanaged && m.filesTab.viewMode != managedViewAll {
		m.resetFilesSearch(false)
		return nil
	}
	if normalizePath(m.targetPath) == "" {
		m.resetFilesSearch(false)
		return nil
	}

	// If we already have final results for this query, avoid re-running.
	if m.filesTab.search.ready && m.filesTab.search.query == query {
		return nil
	}

	m.cancelFilesSearch()
	m.filesTab.search.request++
	requestID := m.filesTab.search.request
	m.filesTab.search.rawResults = nil
	m.filesTab.search.ready = false
	m.filesTab.search.searching = true
	m.filesTab.search.paused = false
	m.filesTab.search.query = query
	return tea.Tick(filesSearchDebounceDelay, func(time.Time) tea.Msg {
		return filesSearchDebouncedMsg{requestID: requestID}
	})
}
