package tui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	"charm.land/huh/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// Screen represents the current top-level screen of the TUI.
type Screen int

// Screen values for top-level TUI screens.
const (
	LandingScreen Screen = iota // Welcome banner (standalone only)
	StatusScreen
	DiffScreen
	ConfirmScreen
	CommitScreen
)

type chezmoiAction int

const (
	chezmoiActionNone chezmoiAction = iota
	chezmoiActionViewDiff
	chezmoiActionReAdd
	chezmoiActionApplyFile
	chezmoiActionApplyAll
	chezmoiActionUpdate
	chezmoiActionCommit
	chezmoiActionPush
	chezmoiActionRefresh
	chezmoiActionInit

	chezmoiActionGitStage
	chezmoiActionGitUnstage
	chezmoiActionGitStageSelected
	chezmoiActionGitUnstageSelected
	chezmoiActionGitStageAll
	chezmoiActionGitUnstageAll
	chezmoiActionGitDiscard
	chezmoiActionGitDiscardSelected
	chezmoiActionGitUndoCommit

	chezmoiActionViewSource
	chezmoiActionEditSource
	chezmoiActionForgetFile
	chezmoiActionApplyManaged
	chezmoiActionOpenFileManager
	chezmoiActionEditTarget
	chezmoiActionFetch
	chezmoiActionPull

	// Ignored view actions
	chezmoiActionViewIgnoreFile
	chezmoiActionEditIgnoreFile

	// Unmanaged add actions
	chezmoiActionAdd
	chezmoiActionAddEncrypt
	chezmoiActionAddTemplate
	chezmoiActionAddAutoTemplate
	chezmoiActionAddExact
	chezmoiActionAddNoRecursive

	// Command tab actions
	chezmoiActionArchive
)

type changesSection int

const (
	changesSectionDrift changesSection = iota
	changesSectionUnstaged
	changesSectionStaged
	changesSectionUnpushed
	changesSectionIncoming
)

// Info sub-view indices.
const (
	infoViewConfig = iota // cat-config (user's config file)
	infoViewFull          // dump-config (full computed config)
	infoViewData          // template data
	infoViewDoctor        // health check
	infoViewCount
)

// infoTab manages the Info tab's own state.
type infoTab struct {
	activeView int                             // active sub-view index (0-3)
	viewNames  []string                        // ["Config", "Full", "Data", "Doctor"]
	views      [infoViewCount]infoSubViewState // per-sub-view state
	format     string                          // "yaml" or "json"
}

// commandsTab manages the Commands tab's own state.
type commandsTab struct {
	items  []chezmoiCommandItem
	cursor int
}

// statusTab manages the Status/Changes tab's own state.
type statusTab struct {
	files            []chezmoi.FileStatus
	filteredFiles    []chezmoi.FileStatus
	gitStagedFiles   []chezmoi.GitFile
	gitUnstagedFiles []chezmoi.GitFile
	gitInfo          chezmoi.GitInfo
	loadingGit       bool
	changesRows      []changesRow
	changesCursor    int
	selectionActive  bool // true when a range selection is active in the status list
	selectionAnchor  int  // anchor row index for status range selection
	sectionCollapsed [5]bool
	statusDeferred   bool // true if status load was deferred at startup
	gitDeferred      bool // true if git status load was deferred at startup

	unpushedCommits []chezmoi.GitCommit
	incomingCommits []chezmoi.GitCommit
	lastFetchTime   time.Time
	fetchInProgress bool
	templatePaths   map[string]bool // target paths of template-managed files
}

// managedViewMode selects which data the Managed tab displays.
type managedViewMode int

const (
	managedViewManaged managedViewMode = iota
	managedViewIgnored
	managedViewUnmanaged
	managedViewAll
)

// fileViewState holds the per-view-mode state for the Files tab.
// filesTab.views is indexed by managedViewMode (managed/ignored/unmanaged/all).
type fileViewState struct {
	files         []string
	filteredFiles []string
	tree          managedTree
	treeRows      []flatTreeRow
	loading       bool
}

// filesTab manages the Files tab's own state.
type filesTab struct {
	views       [4]fileViewState
	cursor      int
	viewMode    managedViewMode
	treeView    bool // true=tree (default), false=flat
	search      filesSearchData
	entryFilter chezmoi.EntryFilter

	dataset filesDataset

	managedDeferred bool // true if managed load was deferred at startup
}

// landingState groups fields for the landing page view.
type landingState struct {
	cursor     int  // 0-based index for landing page tab list
	statsReady bool // true when all initial stats (status, git, managed) are loaded
}

// uiState groups transient UI fields (loading, messages, busy indicator).
type uiState struct {
	message        string
	loading        bool
	loadingSpinner spinner.Model
	busyAction     bool
	mouseCapture   bool
}

// changesRow is a union row in the Status tab's changes list.
// Exactly one of driftFile, gitFile, or commit is non-nil; isHeader marks section headers.
type changesRow struct {
	isHeader  bool
	section   changesSection
	driftFile *chezmoi.FileStatus
	gitFile   *chezmoi.GitFile
	commit    *chezmoi.GitCommit
}

type chezmoiActionItem struct {
	label             string
	description       string
	action            chezmoiAction
	disabled          bool
	unavailableReason string
}

type chezmoiCommandID int

const (
	chezmoiCmdApply chezmoiCommandID = iota
	chezmoiCmdUpdate
	chezmoiCmdRefreshExternals
	chezmoiCmdReAddAll
	chezmoiCmdInit
	chezmoiCmdEditSource
	chezmoiCmdDoctor
	chezmoiCmdVerify
	chezmoiCmdStatus
	chezmoiCmdDiffAll
	chezmoiCmdCatConfig
	chezmoiCmdEditConfig
	chezmoiCmdEditConfigTemplate
	chezmoiCmdGitLog
	chezmoiCmdData
	chezmoiCmdArchive
)

type chezmoiCommandItem struct {
	label          string
	description    string
	command        string
	id             chezmoiCommandID
	category       string
	available      bool
	supportsDryRun bool
}

type viewPickerItem struct {
	mode  managedViewMode
	label string
	count int // file count for display (-1 = not yet loaded)
}

type filterCategory struct {
	entryType chezmoi.EntryType
	label     string
	enabled   bool
}

type pathClass uint8

const (
	pathClassManaged pathClass = iota
	pathClassIgnored
	pathClassUnmanaged
)

// commitState groups fields for the git commit view.
type commitState struct {
	presets     []string  // preset messages from config
	presetForm  *huh.Form // preset select + "Compose..."
	composeForm *huh.Form // free-text message input
	composing   bool      // true when compose form is active
}

// actionsMenu groups fields for the chezmoi and managed actions menus.
type actionsMenu struct {
	show          bool
	items         []chezmoiActionItem
	cursor        int
	managedShow   bool
	managedItems  []chezmoiActionItem
	managedCursor int
}

// overlayState groups fields for modal overlays (help, view picker, filter, confirm).
type overlayState struct {
	// Help
	showHelp   bool
	helpScroll int
	// View picker
	showViewPicker        bool
	viewPickerItems       []viewPickerItem
	viewPickerCursor      int
	viewPickerPendingMode managedViewMode
	// Entry filter
	showFilterOverlay bool
	filterCategories  []filterCategory
	filterCursor      int
	// Confirm dialog
	confirmAction chezmoiAction
	confirmLabel  string
	confirmPath   string
	confirmPaths  []string
}

// diffViewState groups fields for the full-screen diff overlay.
type diffViewState struct {
	content       string
	path          string
	lines         []string
	sourceSection changesSection
	previewApply  bool
	viewport      viewport.Model
	viewportReady bool
	lastWidth     int
}

// ensureViewport creates or resizes the viewport to the given dimensions.
func (d *diffViewState) ensureViewport(width, height int) {
	if !d.viewportReady || d.lastWidth != width {
		d.viewport = viewport.New()
		d.viewport.SetWidth(width)
		d.viewport.SetHeight(height)
		d.viewportReady = true
		d.lastWidth = width
	}
	if d.viewport.Height() != height {
		d.viewport.SetHeight(height)
	}
	if d.viewport.Width() != width {
		d.viewport.SetWidth(width)
	}
}

// resetViewport marks the viewport as uninitialized so it will be recreated on next use.
func (d *diffViewState) resetViewport() {
	d.viewportReady = false
}

// filesSearchMetrics captures per-run deep-search telemetry for debugging/tests.
// This is state-only telemetry (not rendered in UI in this rollout).
type filesSearchMetrics struct {
	elapsed    time.Duration
	roots      int
	matches    int
	terminated string // complete | max-results | canceled | deadline | error
}

// filesSearchData groups fields for the files-tab deep search (unmanaged/all views).
type filesSearchData struct {
	rawResults  []string
	searching   bool
	paused      bool
	ready       bool
	request     uint64
	query       string
	cancel      context.CancelFunc
	lastMetrics filesSearchMetrics
}
