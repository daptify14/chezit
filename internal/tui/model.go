package tui

import (
	"log/slog"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// --- Model ---

// Model is the main TUI model for dotfiles management.
type Model struct {
	service    *chezmoi.Service
	view       Screen
	opts       Options
	exited     bool   // Set to true when user requests exit
	targetPath string // chezmoi target-path, resolved once at init
	startupErr error  // startup failure shown in a dedicated fail-fast view
	gen        uint64 // generation counter for stale async message detection
	// opaquePopulateRequest increments for async opaque-dir population requests.
	opaquePopulateRequest uint64

	landing landingState

	activeTab int
	tabNames  []string

	status statusTab

	filesTab filesTab

	overlays overlayState

	actions actionsMenu

	info infoTab
	cmds commandsTab // named cmds to avoid shadowing exec.Cmd

	diff diffViewState

	commit commitState

	filterInput textinput.Model

	panel filePanel

	iconMode IconMode

	width  int
	height int

	ui       uiState
	debugLog *slog.Logger
}

func newFilterInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Prompt = "> "
	s := ti.Styles()
	s.Focused.Prompt = activeTheme.PrimaryFg
	s.Blurred.Prompt = activeTheme.PrimaryFg
	ti.SetStyles(s)
	ti.CharLimit = 120
	ti.SetWidth(40)
	return ti
}

// NewModel creates a new dotfiles TUI model with the given options.
func NewModel(opts Options) Model {
	// Use provided Service; panic if nil.
	svc := opts.Service
	if svc == nil {
		panic("NewModel: Service must be provided")
	}

	ti := newFilterInput()

	s := spinner.New(spinner.WithSpinner(spinner.Dot))

	presets := opts.CommitPresets
	if len(presets) == 0 {
		presets = defaultCommitPresets
	}

	tabs := []string{"Status", "Files", "Info"}

	// Build command list from policy.
	avail := svc.AvailableCommands()
	commands := make([]chezmoiCommandItem, 0, len(avail))
	for _, ac := range avail {
		commands = append(commands, chezmoiCommandItem{
			label:          ac.Label,
			description:    ac.Description,
			command:        ac.Command,
			id:             commandIDFromLabel(ac.Label),
			category:       ac.Category,
			available:      ac.Available,
			supportsDryRun: ac.SupportsDryRun,
		})
	}

	tabs = append(tabs, "Commands")

	startView := StatusScreen
	if opts.EscBehavior == EscQuit {
		startView = LandingScreen
	}

	// Resolve InitialTab to a tab index.
	initialTab := 0
	if opts.InitialTab != "" {
		for i, name := range tabs {
			if strings.EqualFold(name, opts.InitialTab) {
				initialTab = i
				break
			}
		}
		// Skip the landing screen when jumping to a specific tab.
		startView = StatusScreen
	}

	tp := svc.TargetPath()

	panel := newFilePanel(opts.PanelMode)
	panel.resetForTab(tabs[initialTab])

	// Determine which loads to defer based on the initial tab intent.
	var statusDeferred, gitDeferred, managedDeferred bool
	statusLoading := true
	gitLoading := true

	switch strings.ToLower(opts.InitialTab) {
	case "files":
		statusDeferred = true
		gitDeferred = true
		statusLoading = false
		gitLoading = false
	case "info":
		statusDeferred = true
		gitDeferred = true
		managedDeferred = true
		statusLoading = false
		gitLoading = false
	case "commands":
		statusDeferred = true
		gitDeferred = true
		managedDeferred = true
		statusLoading = false
		gitLoading = false
	default:
		// Default/Status: load status + git + managed for landing/summary
	}

	iconMode := opts.IconMode
	if iconMode == "" {
		iconMode = IconModeNerdFont
	}

	model := Model{
		service:     svc,
		targetPath:  tp,
		opts:        opts,
		iconMode:    iconMode,
		debugLog:    opts.DebugLog,
		view:        startView,
		activeTab:   initialTab,
		tabNames:    tabs,
		filterInput: ti,
		ui: uiState{
			loading:        statusLoading,
			loadingSpinner: s,
			mouseCapture:   true,
		},
		status:   statusTab{loadingGit: gitLoading, statusDeferred: statusDeferred, gitDeferred: gitDeferred},
		filesTab: filesTab{treeView: true, managedDeferred: managedDeferred},
		commit:   commitState{presets: presets},
		cmds: commandsTab{
			items: commands,
		},
		panel: panel,
		info: infoTab{
			viewNames: []string{"Config", "Full", "Data", "Doctor"},
			format:    "yaml",
		},
	}
	if !managedDeferred {
		model.filesTab.views[managedViewManaged].loading = true
	}
	if strings.EqualFold(opts.InitialTab, "info") {
		for i := range infoViewCount {
			model.info.views[i].loading = true
		}
	}
	return model
}

// Init implements tea.Model by returning the initial command batch.
func (m Model) Init() tea.Cmd {
	if m.startupErr != nil {
		return nil
	}

	cmds := []tea.Cmd{m.ui.loadingSpinner.Tick, tea.RequestBackgroundColor}

	tab := strings.ToLower(m.opts.InitialTab)

	switch tab {
	case "files":
		// Files-critical: load managed files immediately
		cmds = append(cmds, m.loadManagedCmd())
	case "info":
		// Info-critical: preload all sub-views so arrow navigation is instant.
		// Calls loadInfoSubViewCmd directly (not ensureInfoViewLoaded) because
		// NewModel already set loading=true for all views.
		for i := range infoViewCount {
			cmds = append(cmds, m.loadInfoSubViewCmd(i))
		}
	case "commands":
		// Commands: nothing critical to preload
	default:
		// Default/Status: load status + git + managed (for landing summary)
		cmds = append(cmds, m.loadStatusCmd(), m.loadManagedCmd(), m.loadTemplatePathsCmd(), m.loadGitStatusCmd(), m.loadGitCommitsCmd())
	}

	return tea.Batch(cmds...)
}

// loadDeferredForTab triggers any deferred data loads required by the given tab.
func (m *Model) loadDeferredForTab(tabName string) tea.Cmd {
	var cmds []tea.Cmd

	switch tabName {
	case "Status":
		if m.status.statusDeferred {
			m.status.statusDeferred = false
			m.ui.loading = true
			cmds = append(cmds, m.loadStatusCmd(), m.loadTemplatePathsCmd())
		}
		if m.status.gitDeferred {
			m.status.gitDeferred = false
			m.status.loadingGit = true
			cmds = append(cmds, m.loadGitStatusCmd(), m.loadGitCommitsCmd())
		}
		if m.filesTab.managedDeferred {
			m.filesTab.managedDeferred = false
			m.filesTab.views[managedViewManaged].loading = true
			cmds = append(cmds, m.loadManagedCmd())
		}
	case "Files":
		if m.filesTab.managedDeferred {
			m.filesTab.managedDeferred = false
			m.filesTab.views[managedViewManaged].loading = true
			cmds = append(cmds, m.loadManagedCmd())
		}
	case "Info":
		if cmd := m.preloadInfoViewsCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) > 0 {
		cmds = append(cmds, m.ui.loadingSpinner.Tick)
		return tea.Batch(cmds...)
	}
	return nil
}

// --- Generic utilities ---

// isAnyLoading reports whether any async operation needs the spinner active.
func (m Model) isAnyLoading() bool {
	return m.ui.loading ||
		m.ui.busyAction ||
		m.status.loadingGit ||
		m.status.fetchInProgress ||
		m.filesTab.views[managedViewManaged].loading ||
		m.filesTab.views[managedViewIgnored].loading ||
		m.filesTab.views[managedViewUnmanaged].loading ||
		m.info.views[m.info.activeView].loading
}

// allLandingStatsLoaded returns true when all three initial stats are done loading.
func (m Model) allLandingStatsLoaded() bool {
	statusReady := !m.ui.loading && !m.status.statusDeferred
	managedReady := !m.filesTab.views[managedViewManaged].loading && !m.filesTab.managedDeferred
	gitReady := !m.status.loadingGit && (!m.status.gitDeferred || m.service.IsReadOnly())
	return statusReady && managedReady && gitReady
}

// debounceLandingReadyCmd returns a command that fires landingStatsReadyMsg after a short delay.
// This prevents a jarring flash if loads complete at slightly different times.
func debounceLandingReadyCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return landingStatsReadyMsg{}
	})
}

// restyleFilterInputForTheme updates the filter input prompt styles to match
// the current activeTheme without resetting its value, focus, or cursor state.
func (m *Model) restyleFilterInputForTheme() {
	s := m.filterInput.Styles()
	s.Focused.Prompt = activeTheme.PrimaryFg
	s.Blurred.Prompt = activeTheme.PrimaryFg
	m.filterInput.SetStyles(s)
}

func (m Model) activeTabName() string {
	if m.activeTab >= 0 && m.activeTab < len(m.tabNames) {
		return m.tabNames[m.activeTab]
	}
	return ""
}

// Exited returns true if the user has requested to exit the TUI.
func (m Model) Exited() bool {
	return m.exited
}

// breadcrumbParts returns the breadcrumb trail for the current view.
func (m Model) breadcrumbParts() []string {
	if len(m.opts.Breadcrumb) > 0 {
		return m.opts.Breadcrumb
	}
	if tab := m.activeTabName(); tab != "" {
		return []string{appName, tab}
	}
	return []string{appName, "Chezmoi"}
}

// escCmd returns the appropriate model and command based on EscBehavior.
// With EscQuit, Esc returns to the landing page and q from landing quits.
// With EscBack, Esc emits ExitMsg.
func (m Model) escCmd() (Model, tea.Cmd) {
	if m.opts.EscBehavior == EscBack {
		return m, sendExitMsg()
	}
	m.view = LandingScreen
	return m, nil
}

// sendExitMsg returns a command that sends ExitMsg.
func sendExitMsg() tea.Cmd {
	return func() tea.Msg {
		return ExitMsg{}
	}
}

// sendRefreshMsg returns a command that sends RefreshMsg.
func sendRefreshMsg() tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{}
	}
}

// nextGen increments the generation counter, used when reloading data.
func (m *Model) nextGen() {
	m.gen++
}

func (m *Model) nextOpaquePopulateRequestID() uint64 {
	m.opaquePopulateRequest++
	return m.opaquePopulateRequest
}

func (m *Model) toggleMouseCapture() {
	m.ui.mouseCapture = !m.ui.mouseCapture
	if m.ui.mouseCapture {
		m.ui.message = "Mouse capture enabled (wheel + panel mouse)"
		return
	}
	m.ui.message = "Mouse capture disabled (drag to select/copy)"
}
