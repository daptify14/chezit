package tui

import "github.com/daptify14/chezit/internal/chezmoi"

// ExitMsg is sent when the user wants to exit the TUI.
// EscQuit and EscBack both produce this message; the caller decides final handling.
type ExitMsg struct{}

// RefreshMsg is sent when the TUI has made changes that might affect
// external state (e.g., after apply, update, or git operations).
type RefreshMsg struct{}

// --- Messages ---

type chezmoiStatusLoadedMsg struct {
	files []chezmoi.FileStatus
	err   error
	gen   uint64
}

type chezmoiDiffLoadedMsg struct {
	path string
	diff string
	err  error
}

type chezmoiActionDoneMsg struct {
	action  chezmoiAction
	message string
	err     error
}

type chezmoiExecDoneMsg struct {
	action chezmoiAction
	err    error
}

type chezmoiManagedLoadedMsg struct {
	files []string
	err   error
	gen   uint64
}

type chezmoiIgnoredLoadedMsg struct {
	files []string
	err   error
	gen   uint64
}

type chezmoiUnmanagedLoadedMsg struct {
	files []string
	err   error
	gen   uint64
}

type chezmoiForgetDoneMsg struct {
	path string
	err  error
}

type chezmoiSourceContentMsg struct {
	path    string
	content string
	err     error
}

type chezmoiCapturedOutputMsg struct {
	action chezmoiAction
	label  string // display name, e.g., "chezmoi re-add"
	output string // captured stdout+stderr
	err    error
}

// infoContentLoadedMsg delivers loaded content for an Info sub-view.
type infoContentLoadedMsg struct {
	view    int // which sub-view this is for (infoViewConfig, etc.)
	content string
	err     error
	gen     uint64
}

type chezmoiGitStatusLoadedMsg struct {
	staged   []chezmoi.GitFile
	unstaged []chezmoi.GitFile
	info     chezmoi.GitInfo
	err      error
	gen      uint64
}

type chezmoiGitActionDoneMsg struct {
	action  chezmoiAction
	message string
	err     error
}

type chezmoiAddDoneMsg struct {
	path string
	err  error
}

type filesSearchDebouncedMsg struct {
	requestID uint64
}

type filesSearchCompletedMsg struct {
	gen       uint64
	requestID uint64
	query     string
	results   []string
	metrics   filesSearchMetrics
	err       error
}

type chezmoiArchiveDoneMsg struct {
	path string
	size int64
	err  error
}

type sourceDirResolvedMsg struct {
	path   string
	action chezmoiAction
	err    error
}

type opaqueDirPopulatedMsg struct {
	viewMode  managedViewMode
	relPath   string
	children  []*managedTreeNode
	gen       uint64
	requestID uint64
	err       error
}

// landingStatsReadyMsg is sent after all initial stats have loaded and debounced.
type landingStatsReadyMsg struct{}

type chezmoiGitCommitsLoadedMsg struct {
	unpushed []chezmoi.GitCommit
	incoming []chezmoi.GitCommit
	err      error
	gen      uint64
}

type chezmoiGitFetchDoneMsg struct {
	err error
	gen uint64
}

type templatePathsLoadedMsg struct {
	paths map[string]bool
	gen   uint64
}

// panelContentLoadedMsg is sent when async panel content loading completes.
type panelContentLoadedMsg struct {
	path    string
	mode    panelContentMode
	section changesSection
	content string
	err     error
}
