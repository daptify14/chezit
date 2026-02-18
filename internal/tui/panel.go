package tui

import "charm.land/bubbles/v2/viewport"

// panelContentMode distinguishes between diff and file content display.
type panelContentMode int

const (
	panelModeDiff panelContentMode = iota
	panelModeContent
)

// panelFocusZone tracks which zone has keyboard focus.
type panelFocusZone int

const (
	panelFocusList panelFocusZone = iota
	panelFocusPanel
)

// panelCacheKey uniquely identifies a cached panel content entry.
type panelCacheKey struct {
	path    string
	mode    panelContentMode
	section changesSection
}

// panelCacheEntry holds pre-loaded content for a file.
type panelCacheEntry struct {
	content string
	lines   []string
	err     error
}

// Panel auto-visibility constants.
const (
	panelAutoThreshold = 90
	panelMinWidth      = 60
	panelMaxCacheSize  = 200
)

// filePanel holds the state for the preview panel.
type filePanel struct {
	visible        bool
	manualOverride bool
	focusZone      panelFocusZone
	contentMode    panelContentMode

	viewport      viewport.Model
	viewportReady bool

	currentPath    string
	currentSection changesSection
	loading        bool

	pendingPath    string
	pendingMode    panelContentMode
	pendingSection changesSection
	pendingLoad    bool

	cache map[panelCacheKey]panelCacheEntry

	// Layout invalidation tracking.
	lastWidth int
}

// newFilePanel creates a zero-value panel with an initialised cache.
func newFilePanel(mode string) filePanel {
	p := filePanel{
		cache:       make(map[panelCacheKey]panelCacheEntry, 32),
		contentMode: panelModeDiff,
	}
	switch mode {
	case "show":
		p.manualOverride = true
		p.visible = true
	case "hide":
		p.manualOverride = true
		p.visible = false
	}
	return p
}

// shouldShow returns true if the panel should render at the given terminal width.
func (p *filePanel) shouldShow(termWidth int) bool {
	if termWidth < panelMinWidth {
		return false
	}
	if p.manualOverride {
		return p.visible
	}
	return termWidth >= panelAutoThreshold
}

// toggle flips the panel visibility manually.
func (p *filePanel) toggle(termWidth int) {
	if !p.manualOverride {
		p.manualOverride = true
		// First toggle: invert what auto-mode would do.
		p.visible = termWidth < panelAutoThreshold
	} else {
		p.visible = !p.visible
	}
}

// panelWidthFor computes the standard side-panel width (40%, min 30).
func panelWidthFor(width int) int {
	return max(width*40/100, 30)
}

// clearCache empties the content cache.
func (p *filePanel) clearCache() {
	p.cache = make(map[panelCacheKey]panelCacheEntry, 32)
}

// trimCache evicts the oldest half when the cap is exceeded.
// Since Go maps have no insertion order, we just clear the whole map
// when the cap is reached. This is simple and the cache refills quickly.
func (p *filePanel) trimCache() {
	if len(p.cache) > panelMaxCacheSize {
		p.cache = make(map[panelCacheKey]panelCacheEntry, 32)
	}
}

// cacheGet returns a cached entry and true if found.
func (p *filePanel) cacheGet(path string, mode panelContentMode, section changesSection) (panelCacheEntry, bool) {
	e, ok := p.cache[panelCacheKey{path: path, mode: mode, section: section}]
	return e, ok
}

// cachePut stores a content entry and trims if needed.
func (p *filePanel) cachePut(path string, mode panelContentMode, section changesSection, entry panelCacheEntry) {
	p.cache[panelCacheKey{path: path, mode: mode, section: section}] = entry
	p.trimCache()
}

// ensureViewport creates or resizes the viewport to the given dimensions.
func (p *filePanel) ensureViewport(width, height int) {
	if !p.viewportReady || p.lastWidth != width {
		p.viewport = viewport.New()
		p.viewport.SetWidth(width)
		p.viewport.SetHeight(height)
		p.viewportReady = true
		p.lastWidth = width
	}
	if p.viewport.Height() != height {
		p.viewport.SetHeight(height)
	}
	if p.viewport.Width() != width {
		p.viewport.SetWidth(width)
	}
}

// resetForTab resets the panel content mode to the tab's default and clears cache.
func (p *filePanel) resetForTab(tabName string) {
	switch tabName {
	case "Status":
		p.contentMode = panelModeDiff
	case "Files":
		p.contentMode = panelModeContent
	default:
		// Config, Commands: no panel
	}
	p.currentPath = ""
	p.currentSection = changesSectionDrift
	p.loading = false
	p.pendingPath = ""
	p.pendingMode = p.contentMode
	p.pendingSection = changesSectionDrift
	p.pendingLoad = false
	p.clearCache()
}
