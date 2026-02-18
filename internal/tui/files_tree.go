package tui

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type managedTreeNode struct {
	name     string
	relPath  string
	absPath  string
	isDir    bool
	children []*managedTreeNode
	parent   *managedTreeNode
	expanded bool
	depth    int
	opaque   bool // directory detected via os.Stat, needs lazy population on expand
	loading  bool
	// loadingRequest tracks the active async request for this node.
	loadingRequest uint64
}

type flatTreeRow struct {
	node       *managedTreeNode
	depth      int
	isLast     bool
	prefixBits []bool
	fileCount  int
}

type managedTree struct {
	roots     []*managedTreeNode
	dirCount  int
	fileCount int
}

func buildManagedTree(absPaths []string, homeDir string) managedTree {
	paths := dedupeManagedTreePaths(absPaths)
	homePrefix := managedTreeHomePrefix(homeDir)

	lookup := make(map[string]*managedTreeNode, len(paths)*2)
	roots := make([]*managedTreeNode, 0, len(paths))
	leafNodes := make([]*managedTreeNode, 0, len(paths))
	dirCount := 0
	fileCount := 0

	for _, abs := range paths {
		rel := managedTreeRelativePath(abs, homePrefix)
		if rel == "" {
			continue
		}

		segments := strings.Split(rel, "/")
		var parent *managedTreeNode
		for i := range len(segments) - 1 {
			dirRel := strings.Join(segments[:i+1], "/")
			node, exists := lookup[dirRel]
			if !exists {
				node = &managedTreeNode{
					name:    segments[i],
					relPath: dirRel,
					isDir:   true,
					parent:  parent,
					depth:   i,
				}
				lookup[dirRel] = node
				dirCount++
				if parent == nil {
					roots = append(roots, node)
				} else {
					parent.children = append(parent.children, node)
				}
			}
			parent = node
		}

		leafRel := rel
		if _, exists := lookup[leafRel]; exists {
			// Deduplicate exact duplicate entries.
			continue
		}

		leaf := &managedTreeNode{
			name:    segments[len(segments)-1],
			relPath: leafRel,
			absPath: abs,
			isDir:   false,
			parent:  parent,
			depth:   len(segments) - 1,
		}
		lookup[leafRel] = leaf
		leafNodes = append(leafNodes, leaf)
		fileCount++
		if parent == nil {
			roots = append(roots, leaf)
		} else {
			parent.children = append(parent.children, leaf)
		}
	}

	markOpaqueLeafDirectories(leafNodes, &dirCount, &fileCount)
	sortManagedTreeNodesRecursively(roots)
	for _, root := range roots {
		autoExpand(root)
	}

	return managedTree{roots: roots, dirCount: dirCount, fileCount: fileCount}
}

// dedupeManagedTreePaths removes duplicate and parent-subsumed paths from a sorted list.
func dedupeManagedTreePaths(absPaths []string) []string {
	if len(absPaths) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(absPaths))
	for _, path := range absPaths {
		clean := normalizePath(path)
		if clean == "" || clean == "." {
			continue
		}
		normalized = append(normalized, clean)
	}
	if len(normalized) == 0 {
		return nil
	}

	sort.Strings(normalized)
	deduped := make([]string, 0, len(normalized))
	sep := string(filepath.Separator)
	for i, current := range normalized {
		if i > 0 && current == normalized[i-1] {
			continue
		}
		if i+1 < len(normalized) && strings.HasPrefix(normalized[i+1], current+sep) {
			continue
		}
		deduped = append(deduped, current)
	}
	return deduped
}

func managedTreeHomePrefix(homeDir string) string {
	if homeDir == "" {
		return ""
	}
	clean := normalizePath(homeDir)
	if clean == "" || clean == "." {
		return ""
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(clean, sep) {
		clean += sep
	}
	return clean
}

func managedTreeRelativePath(absPath, homePrefix string) string {
	rel := absPath
	if homePrefix != "" && strings.HasPrefix(absPath, homePrefix) {
		rel = absPath[len(homePrefix):]
	}
	return filepath.ToSlash(rel)
}

// markOpaqueLeafDirectories promotes leaf files that are actually directories on disk
// to opaque directory nodes, adjusting the dir/file counts accordingly.
func markOpaqueLeafDirectories(leafNodes []*managedTreeNode, dirCount, fileCount *int) {
	for _, node := range leafNodes {
		if node == nil || node.absPath == "" || node.isDir {
			continue
		}
		info, err := os.Stat(node.absPath)
		if err != nil || !info.IsDir() {
			continue
		}
		node.isDir = true
		node.opaque = true
		if dirCount != nil {
			(*dirCount)++
		}
		if fileCount != nil {
			(*fileCount)--
		}
	}
}

func sortManagedTreeNodesRecursively(nodes []*managedTreeNode) {
	sortTreeNodes(nodes)
	for _, node := range nodes {
		if node != nil && node.isDir && len(node.children) > 0 {
			sortManagedTreeNodesRecursively(node.children)
		}
	}
}

func sortChildren(node *managedTreeNode) {
	sortTreeNodes(node.children)
}

func sortTreeNodes(nodes []*managedTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		ci, cj := nodes[i], nodes[j]
		if ci.isDir != cj.isDir {
			return ci.isDir
		}
		return ci.name < cj.name
	})
}

func autoExpand(node *managedTreeNode) {
	if !node.isDir {
		return
	}
	if len(node.children) == 1 {
		node.expanded = true
	}
	for _, child := range node.children {
		autoExpand(child)
	}
}

// expandAll recursively expands all directory nodes in the tree.
// Used for the "all" view mode to show managed vs ignored files.
func expandAll(node *managedTreeNode) {
	if !node.isDir {
		return
	}
	node.expanded = true
	for _, child := range node.children {
		expandAll(child)
	}
}

func flattenManagedTree(tree managedTree) []flatTreeRow {
	var rows []flatTreeRow
	for i, root := range tree.roots {
		flattenNode(&rows, root, 0, i == len(tree.roots)-1, nil)
	}
	return rows
}

func flattenNode(rows *[]flatTreeRow, node *managedTreeNode, depth int, isLast bool, parentBits []bool) {
	bits := make([]bool, len(parentBits)+1)
	copy(bits, parentBits)
	bits[len(parentBits)] = isLast

	fc := 0
	if node.isDir {
		fc = countFiles(node)
	}

	*rows = append(*rows, flatTreeRow{
		node:       node,
		depth:      depth,
		isLast:     isLast,
		prefixBits: bits,
		fileCount:  fc,
	})

	if node.isDir && node.expanded {
		for i, child := range node.children {
			flattenNode(rows, child, depth+1, i == len(node.children)-1, bits)
		}
	}
}

func countFiles(node *managedTreeNode) int {
	if !node.isDir {
		return 0
	}
	count := 0
	for _, child := range node.children {
		if child.isDir {
			count += countFiles(child)
		} else {
			count++
		}
	}
	return count
}

func filterManagedTree(tree managedTree, matchSet map[string]bool) []flatTreeRow {
	var rows []flatTreeRow
	for i, root := range tree.roots {
		filterNode(&rows, root, 0, i == len(tree.roots)-1, nil, matchSet)
	}
	return rows
}

func filterNode(rows *[]flatTreeRow, node *managedTreeNode, depth int, isLast bool, parentBits []bool, matchSet map[string]bool) {
	if !node.isDir {
		if !matchSet[node.absPath] {
			return
		}
		bits := make([]bool, len(parentBits)+1)
		copy(bits, parentBits)
		bits[len(parentBits)] = isLast

		*rows = append(*rows, flatTreeRow{
			node:       node,
			depth:      depth,
			isLast:     isLast,
			prefixBits: bits,
		})
		return
	}

	if !hasMatchingDescendant(node, matchSet) {
		return
	}

	bits := make([]bool, len(parentBits)+1)
	copy(bits, parentBits)
	bits[len(parentBits)] = isLast

	fc := countMatchingFiles(node, matchSet)

	*rows = append(*rows, flatTreeRow{
		node:       node,
		depth:      depth,
		isLast:     isLast,
		prefixBits: bits,
		fileCount:  fc,
	})

	matchingChildren := filterMatchingChildren(node, matchSet)
	for i, child := range matchingChildren {
		filterNode(rows, child, depth+1, i == len(matchingChildren)-1, bits, matchSet)
	}
}

func hasMatchingDescendant(node *managedTreeNode, matchSet map[string]bool) bool {
	for _, child := range node.children {
		if !child.isDir && matchSet[child.absPath] {
			return true
		}
		if child.isDir && hasMatchingDescendant(child, matchSet) {
			return true
		}
	}
	return false
}

func countMatchingFiles(node *managedTreeNode, matchSet map[string]bool) int {
	count := 0
	for _, child := range node.children {
		if !child.isDir && matchSet[child.absPath] {
			count++
		}
		if child.isDir {
			count += countMatchingFiles(child, matchSet)
		}
	}
	return count
}

func filterMatchingChildren(node *managedTreeNode, matchSet map[string]bool) []*managedTreeNode {
	var result []*managedTreeNode
	for _, child := range node.children {
		if !child.isDir && matchSet[child.absPath] {
			result = append(result, child)
		} else if child.isDir && hasMatchingDescendant(child, matchSet) {
			result = append(result, child)
		}
	}
	return result
}

func readOpaqueDirChildren(parentRel, parentAbs string, parentDepth int) ([]*managedTreeNode, error) {
	if parentAbs == "" {
		return nil, errors.New("cannot read children for empty directory path")
	}
	entries, err := os.ReadDir(parentAbs)
	if err != nil {
		return nil, err
	}
	children := make([]*managedTreeNode, 0, len(entries))
	for _, entry := range entries {
		childRel := entry.Name()
		if parentRel != "" {
			childRel = parentRel + "/" + entry.Name()
		}
		children = append(children, &managedTreeNode{
			name:    entry.Name(),
			relPath: childRel,
			absPath: filepath.Join(parentAbs, entry.Name()),
			isDir:   entry.IsDir(),
			depth:   parentDepth + 1,
			opaque:  entry.IsDir(),
		})
	}
	sortTreeNodes(children)
	return children, nil
}

func findTreeNodeByRelPath(tree managedTree, relPath string) *managedTreeNode {
	for _, root := range tree.roots {
		if node := findTreeNodeByRelPathFromNode(root, relPath); node != nil {
			return node
		}
	}
	return nil
}

func findTreeNodeByRelPathFromNode(node *managedTreeNode, relPath string) *managedTreeNode {
	if node == nil {
		return nil
	}
	if node.relPath == relPath {
		return node
	}
	for _, child := range node.children {
		if found := findTreeNodeByRelPathFromNode(child, relPath); found != nil {
			return found
		}
	}
	return nil
}

func projectFilteredTreeRows(absPaths []string, homeDir string) []flatTreeRow {
	if len(absPaths) == 0 {
		return nil
	}
	tree := buildManagedTree(absPaths, homeDir)
	for _, root := range tree.roots {
		expandAll(root)
	}
	return flattenManagedTree(tree)
}

func findParentRow(rows []flatTreeRow, cursor int) int {
	if cursor < 0 || cursor >= len(rows) {
		return -1
	}
	target := rows[cursor].node.parent
	if target == nil {
		return -1
	}
	for i := cursor - 1; i >= 0; i-- {
		if rows[i].node == target {
			return i
		}
	}
	return -1
}
