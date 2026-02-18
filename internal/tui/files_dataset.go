package tui

import (
	"os"
	"path/filepath"
	"sort"
)

type fileClass uint8

const (
	fileClassManaged fileClass = iota
	fileClassIgnored
	fileClassUnmanaged
)

// filesDataset is the canonical classified index built from chezmoi command outputs.
type filesDataset struct {
	classMap          map[string]fileClass
	unmanagedDirRoots []string // sorted, for binary-search prefix matching
	allPaths          []string // sorted union of all abs paths
	ready             bool
}

// rebuildDataset constructs a new filesDataset from the current filesTab state.
func rebuildDataset(ft *filesTab) filesDataset {
	managedFiles := ft.views[managedViewManaged].files
	ignoredFiles := ft.views[managedViewIgnored].files
	unmanagedFiles := ft.views[managedViewUnmanaged].files

	totalEstimate := len(managedFiles) + len(ignoredFiles) + len(unmanagedFiles)
	classMap := make(map[string]fileClass, totalEstimate)

	for _, p := range managedFiles {
		classMap[p] = fileClassManaged
	}
	for _, p := range ignoredFiles {
		classMap[p] = fileClassIgnored
	}
	for _, p := range unmanagedFiles {
		// Keep ignored precedence deterministic when command outputs overlap.
		if classMap[p] == fileClassIgnored {
			continue
		}
		classMap[p] = fileClassUnmanaged
	}

	allPaths := make([]string, 0, len(classMap))
	for p := range classMap {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	var dirRoots []string
	for _, p := range unmanagedFiles {
		// Only index directories that are currently classified as unmanaged.
		if classMap[p] != fileClassUnmanaged {
			continue
		}
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		dirRoots = append(dirRoots, filepath.Clean(p))
	}
	sort.Strings(dirRoots)

	ready := managedFiles != nil && (ignoredFiles != nil || unmanagedFiles != nil)

	return filesDataset{
		classMap:          classMap,
		unmanagedDirRoots: dirRoots,
		allPaths:          allPaths,
		ready:             ready,
	}
}

// classify returns the pathClass for absPath using O(1) map lookup
// with O(log n) binary-search fallback for unmanaged directory descendants.
func (d *filesDataset) classify(absPath string) pathClass {
	if fc, ok := d.classifyKnown(absPath); ok {
		return fileClassToPathClass(fc)
	}
	return pathClassManaged
}

// classifyKnown returns the classified fileClass and whether the path is known
// to the dataset (including unmanaged descendants via directory-root fallback).
func (d *filesDataset) classifyKnown(absPath string) (fileClass, bool) {
	if absPath == "" {
		return fileClassManaged, false
	}

	if fc, ok := d.classMap[absPath]; ok {
		return fc, true
	}

	// Binary search for an unmanaged dir root that is a prefix of absPath.
	idx := sort.SearchStrings(d.unmanagedDirRoots, absPath)
	if idx > 0 {
		candidate := d.unmanagedDirRoots[idx-1]
		if absPath == candidate || hasPathPrefix(absPath, candidate) {
			return fileClassUnmanaged, true
		}
	}
	if idx < len(d.unmanagedDirRoots) {
		candidate := d.unmanagedDirRoots[idx]
		if absPath == candidate || hasPathPrefix(absPath, candidate) {
			return fileClassUnmanaged, true
		}
	}

	return fileClassManaged, false
}

// projectedPaths returns paths from allPaths matching the given class.
func (d *filesDataset) projectedPaths(class fileClass) []string {
	var result []string
	for _, p := range d.allPaths {
		if d.classMap[p] == class {
			result = append(result, p)
		}
	}
	return result
}

// isUnmanagedDirRoot reports whether absPath is a known unmanaged directory root.
func (d *filesDataset) isUnmanagedDirRoot(absPath string) bool {
	idx := sort.SearchStrings(d.unmanagedDirRoots, absPath)
	return idx < len(d.unmanagedDirRoots) && d.unmanagedDirRoots[idx] == absPath
}

func fileClassToPathClass(fc fileClass) pathClass {
	switch fc {
	case fileClassManaged:
		return pathClassManaged
	case fileClassIgnored:
		return pathClassIgnored
	case fileClassUnmanaged:
		return pathClassUnmanaged
	default:
		return pathClassManaged
	}
}

// rebuildDatasetAndAllView rebuilds the dataset and, if ready, the All-view tree.
func (m *Model) rebuildDatasetAndAllView() {
	m.filesTab.dataset = rebuildDataset(&m.filesTab)
	if m.filesTab.dataset.ready {
		m.rebuildFileViewTree(managedViewAll)
	}
}

func hasPathPrefix(child, root string) bool {
	return len(child) > len(root) &&
		child[len(root)] == filepath.Separator &&
		child[:len(root)] == root
}
