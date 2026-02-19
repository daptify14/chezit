package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/daptify14/chezit/internal/chezmoi"
)

const (
	filesSearchTimeout    = 8 * time.Second
	filesSearchMaxResults = 4000
	filesSearchMaxDepth   = 3
)

// --- Files tab async command factories ---

func (m Model) loadManagedCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		var files []string
		var err error
		if m.filesTab.entryFilter.IsZero() {
			files, err = m.service.ManagedFiles()
		} else {
			files, err = m.service.ManagedFilesWithFilter(m.filesTab.entryFilter)
		}
		return chezmoiManagedLoadedMsg{files: files, err: err, gen: gen}
	}
}

func (m Model) loadIgnoredCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		files, err := m.service.IgnoredFiles()
		return chezmoiIgnoredLoadedMsg{files: files, err: err, gen: gen}
	}
}

func (m Model) loadUnmanagedCmd() tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		files, err := m.service.UnmanagedFiles(m.filesTab.entryFilter)
		return chezmoiUnmanagedLoadedMsg{files: files, err: err, gen: gen}
	}
}

func (m Model) runFilesSearchCmd(ctx context.Context, requestID uint64, query string, roots []string) tea.Cmd {
	gen := m.gen
	searchQuery := query
	searchRoots := append([]string(nil), roots...)
	return func() tea.Msg {
		results, metrics, err := walkPathsWithContext(ctx, searchQuery, searchRoots, filesSearchMaxDepth, filesSearchMaxResults)
		return filesSearchCompletedMsg{
			gen:       gen,
			requestID: requestID,
			query:     searchQuery,
			results:   results,
			metrics:   metrics,
			err:       err,
		}
	}
}

func (m Model) populateOpaqueDirCmd(
	viewMode managedViewMode,
	relPath, absPath string,
	depth int,
	requestID uint64,
) tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		children, err := readOpaqueDirChildren(relPath, absPath, depth)
		return opaqueDirPopulatedMsg{
			viewMode:  viewMode,
			relPath:   relPath,
			children:  children,
			gen:       gen,
			requestID: requestID,
			err:       err,
		}
	}
}

func (m Model) forgetFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		err := m.service.Forget(path)
		return chezmoiForgetDoneMsg{path: path, err: err}
	}
}

func (m Model) addFileCmd(path string, opts chezmoi.AddOptions) tea.Cmd {
	mgr := m.service
	return func() tea.Msg {
		err := mgr.Add(path, opts)
		return chezmoiAddDoneMsg{path: path, err: err}
	}
}

func (m Model) loadSourceContentCmd(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.service.CatTarget(path)
		return chezmoiSourceContentMsg{path: path, content: content, err: err}
	}
}
