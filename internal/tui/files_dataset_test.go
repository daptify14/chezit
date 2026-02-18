package tui

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// helper: build a filesTab with the given file slices and call rebuildDataset.
func buildDataset(managed, ignored, unmanaged []string) filesDataset {
	ft := filesTab{}
	ft.views[managedViewManaged].files = managed
	ft.views[managedViewIgnored].files = ignored
	ft.views[managedViewUnmanaged].files = unmanaged
	return rebuildDataset(&ft)
}

func TestRebuildDataset_ManagedOnly(t *testing.T) {
	managed := []string{"/home/user/.bashrc", "/home/user/.gitconfig", "/home/user/.vimrc"}

	ds := buildDataset(managed, nil, nil)

	if ds.ready {
		t.Fatal("expected ready=false when only managed files are set")
	}

	for _, p := range managed {
		fc, ok := ds.classMap[p]
		if !ok {
			t.Fatalf("expected classMap to contain %q", p)
		}
		if fc != fileClassManaged {
			t.Fatalf("expected %q to be fileClassManaged, got %d", p, fc)
		}
	}

	if !sort.StringsAreSorted(ds.allPaths) {
		t.Fatal("expected allPaths to be sorted")
	}
	if len(ds.allPaths) != len(managed) {
		t.Fatalf("expected allPaths length %d, got %d", len(managed), len(ds.allPaths))
	}

	if len(ds.unmanagedDirRoots) != 0 {
		t.Fatalf("expected unmanagedDirRoots to be empty, got %d entries", len(ds.unmanagedDirRoots))
	}
}

func TestRebuildDataset_AllSources(t *testing.T) {
	managed := []string{"/home/user/.bashrc", "/home/user/.gitconfig"}
	ignored := []string{"/home/user/.cache/foo", "/home/user/.local/share/bar"}
	unmanaged := []string{"/home/user/.ssh/id_rsa", "/home/user/Documents/notes.txt"}

	ds := buildDataset(managed, ignored, unmanaged)

	if !ds.ready {
		t.Fatal("expected ready=true when all three sources are set")
	}

	// Verify classifications.
	for _, p := range managed {
		if ds.classMap[p] != fileClassManaged {
			t.Fatalf("expected %q classified as managed", p)
		}
	}
	for _, p := range ignored {
		if ds.classMap[p] != fileClassIgnored {
			t.Fatalf("expected %q classified as ignored", p)
		}
	}
	for _, p := range unmanaged {
		if ds.classMap[p] != fileClassUnmanaged {
			t.Fatalf("expected %q classified as unmanaged", p)
		}
	}

	// allPaths sorted.
	if !sort.StringsAreSorted(ds.allPaths) {
		t.Fatal("expected allPaths to be sorted")
	}

	// allPaths has no duplicates.
	expectedTotal := len(managed) + len(ignored) + len(unmanaged)
	if len(ds.allPaths) != expectedTotal {
		t.Fatalf("expected allPaths length %d, got %d", expectedTotal, len(ds.allPaths))
	}
	for i := 1; i < len(ds.allPaths); i++ {
		if ds.allPaths[i] == ds.allPaths[i-1] {
			t.Fatalf("duplicate found in allPaths at index %d: %q", i, ds.allPaths[i])
		}
	}
}

func TestDatasetClassify_UnknownPath(t *testing.T) {
	managed := []string{"/home/user/.bashrc"}
	ignored := []string{"/home/user/.cache/foo"}

	ds := buildDataset(managed, ignored, nil)

	// Unknown path should return pathClassManaged (fallback).
	result := ds.classify("/some/completely/unknown/path")
	if result != pathClassManaged {
		t.Fatalf("expected pathClassManaged for unknown path, got %d", result)
	}

	// Empty string should return pathClassManaged.
	result = ds.classify("")
	if result != pathClassManaged {
		t.Fatalf("expected pathClassManaged for empty string, got %d", result)
	}
}

func TestDatasetClassify_DirectLookup(t *testing.T) {
	managed := []string{"/home/user/.bashrc", "/home/user/.gitconfig"}
	ignored := []string{"/home/user/.cache/foo"}
	unmanaged := []string{"/home/user/.ssh/id_rsa"}

	ds := buildDataset(managed, ignored, unmanaged)

	tests := []struct {
		path string
		want pathClass
	}{
		{"/home/user/.bashrc", pathClassManaged},
		{"/home/user/.gitconfig", pathClassManaged},
		{"/home/user/.cache/foo", pathClassIgnored},
		{"/home/user/.ssh/id_rsa", pathClassUnmanaged},
	}

	for _, tt := range tests {
		got := ds.classify(tt.path)
		if got != tt.want {
			t.Fatalf("classify(%q) = %d, want %d", tt.path, got, tt.want)
		}
	}
}

func TestDatasetClassify_UnmanagedDescendant(t *testing.T) {
	tmpDir := t.TempDir()
	unmanagedDir := filepath.Join(tmpDir, "unmanaged-root")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	managed := []string{filepath.Join(tmpDir, ".bashrc")}
	unmanaged := []string{unmanagedDir}

	ds := buildDataset(managed, nil, unmanaged)

	// The directory should be detected as a dir root.
	if len(ds.unmanagedDirRoots) != 1 {
		t.Fatalf("expected 1 unmanagedDirRoot, got %d", len(ds.unmanagedDirRoots))
	}
	if ds.unmanagedDirRoots[0] != unmanagedDir {
		t.Fatalf("expected unmanagedDirRoot %q, got %q", unmanagedDir, ds.unmanagedDirRoots[0])
	}

	// A child path under the unmanaged directory should classify as unmanaged.
	childPath := filepath.Join(unmanagedDir, "subdir", "file.txt")
	got := ds.classify(childPath)
	if got != pathClassUnmanaged {
		t.Fatalf("classify(%q) = %d, want pathClassUnmanaged (%d)", childPath, got, pathClassUnmanaged)
	}

	// A path that is NOT under the directory should fall back to managed.
	siblingPath := filepath.Join(tmpDir, "other-dir", "file.txt")
	got = ds.classify(siblingPath)
	if got != pathClassManaged {
		t.Fatalf("classify(%q) = %d, want pathClassManaged (%d)", siblingPath, got, pathClassManaged)
	}
}

func TestDatasetProjectedPaths(t *testing.T) {
	managed := []string{"/a/managed1", "/a/managed2", "/a/managed3"}
	ignored := []string{"/a/ignored1", "/a/ignored2"}
	unmanaged := []string{"/a/unmanaged1"}

	ds := buildDataset(managed, ignored, unmanaged)

	gotManaged := ds.projectedPaths(fileClassManaged)
	if len(gotManaged) != 3 {
		t.Fatalf("projectedPaths(managed): expected 3, got %d", len(gotManaged))
	}
	for _, p := range gotManaged {
		if ds.classMap[p] != fileClassManaged {
			t.Fatalf("projectedPaths(managed) returned non-managed path %q", p)
		}
	}

	gotIgnored := ds.projectedPaths(fileClassIgnored)
	if len(gotIgnored) != 2 {
		t.Fatalf("projectedPaths(ignored): expected 2, got %d", len(gotIgnored))
	}
	for _, p := range gotIgnored {
		if ds.classMap[p] != fileClassIgnored {
			t.Fatalf("projectedPaths(ignored) returned non-ignored path %q", p)
		}
	}

	gotUnmanaged := ds.projectedPaths(fileClassUnmanaged)
	if len(gotUnmanaged) != 1 {
		t.Fatalf("projectedPaths(unmanaged): expected 1, got %d", len(gotUnmanaged))
	}
	if gotUnmanaged[0] != "/a/unmanaged1" {
		t.Fatalf("projectedPaths(unmanaged) returned %q, want /a/unmanaged1", gotUnmanaged[0])
	}
}

func TestDatasetIsUnmanagedDirRoot(t *testing.T) {
	tmpDir := t.TempDir()
	root1 := filepath.Join(tmpDir, "dir-a")
	root2 := filepath.Join(tmpDir, "dir-b")
	if err := os.MkdirAll(root1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(root2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ds := buildDataset(nil, nil, []string{root1, root2})

	// Exact matches should return true.
	if !ds.isUnmanagedDirRoot(root1) {
		t.Fatalf("expected isUnmanagedDirRoot(%q) = true", root1)
	}
	if !ds.isUnmanagedDirRoot(root2) {
		t.Fatalf("expected isUnmanagedDirRoot(%q) = true", root2)
	}

	// A path that is NOT a root should return false.
	if ds.isUnmanagedDirRoot("/nonexistent/path") {
		t.Fatal("expected isUnmanagedDirRoot for unknown path to be false")
	}

	// A child of a root is NOT itself a root.
	childPath := filepath.Join(root1, "child")
	if ds.isUnmanagedDirRoot(childPath) {
		t.Fatalf("expected isUnmanagedDirRoot(%q) = false for child of root", childPath)
	}
}

func TestRebuildDataset_Incremental(t *testing.T) {
	managed := []string{"/home/user/.bashrc", "/home/user/.vimrc"}
	ignored := []string{"/home/user/.cache/foo"}
	unmanaged := []string{"/home/user/Documents/notes.txt"}

	// Phase 1: only managed -> not ready.
	ds1 := buildDataset(managed, nil, nil)
	if ds1.ready {
		t.Fatal("phase 1: expected ready=false with only managed files")
	}
	if len(ds1.classMap) != len(managed) {
		t.Fatalf("phase 1: expected classMap length %d, got %d", len(managed), len(ds1.classMap))
	}

	// Phase 2: managed + ignored -> ready.
	ds2 := buildDataset(managed, ignored, nil)
	if !ds2.ready {
		t.Fatal("phase 2: expected ready=true with managed + ignored")
	}
	if len(ds2.classMap) != len(managed)+len(ignored) {
		t.Fatalf("phase 2: expected classMap length %d, got %d",
			len(managed)+len(ignored), len(ds2.classMap))
	}

	// Phase 3: managed + ignored + unmanaged -> all 3 classes present.
	ds3 := buildDataset(managed, ignored, unmanaged)
	if !ds3.ready {
		t.Fatal("phase 3: expected ready=true with all sources")
	}

	expectedTotal := len(managed) + len(ignored) + len(unmanaged)
	if len(ds3.classMap) != expectedTotal {
		t.Fatalf("phase 3: expected classMap length %d, got %d", expectedTotal, len(ds3.classMap))
	}

	// Verify each class is correctly assigned in phase 3.
	for _, p := range managed {
		if ds3.classMap[p] != fileClassManaged {
			t.Fatalf("phase 3: expected %q as managed", p)
		}
	}
	for _, p := range ignored {
		if ds3.classMap[p] != fileClassIgnored {
			t.Fatalf("phase 3: expected %q as ignored", p)
		}
	}
	for _, p := range unmanaged {
		if ds3.classMap[p] != fileClassUnmanaged {
			t.Fatalf("phase 3: expected %q as unmanaged", p)
		}
	}
}

func TestRebuildDataset_ReadyConditions(t *testing.T) {
	someFiles := []string{"/home/user/.bashrc"}

	tests := []struct {
		name      string
		managed   []string
		ignored   []string
		unmanaged []string
		wantReady bool
	}{
		{
			name:      "managed=nil, ignored=set -> ready=false",
			managed:   nil,
			ignored:   someFiles,
			unmanaged: nil,
			wantReady: false,
		},
		{
			name:      "managed=set, ignored=nil, unmanaged=nil -> ready=false",
			managed:   someFiles,
			ignored:   nil,
			unmanaged: nil,
			wantReady: false,
		},
		{
			name:      "managed=set, ignored=set, unmanaged=nil -> ready=true",
			managed:   someFiles,
			ignored:   someFiles,
			unmanaged: nil,
			wantReady: true,
		},
		{
			name:      "managed=set, ignored=nil, unmanaged=set -> ready=true",
			managed:   someFiles,
			ignored:   nil,
			unmanaged: someFiles,
			wantReady: true,
		},
		{
			name:      "managed=set, ignored=set, unmanaged=set -> ready=true",
			managed:   someFiles,
			ignored:   someFiles,
			unmanaged: someFiles,
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := buildDataset(tt.managed, tt.ignored, tt.unmanaged)
			if ds.ready != tt.wantReady {
				t.Fatalf("ready = %v, want %v", ds.ready, tt.wantReady)
			}
		})
	}
}

func TestHasPathPrefix(t *testing.T) {
	tests := []struct {
		name  string
		child string
		root  string
		want  bool
	}{
		{
			name:  "child under root directory",
			child: "/home/user/.config/file",
			root:  "/home/user/.config",
			want:  true,
		},
		{
			name:  "equal paths are not prefix (not a child)",
			child: "/home/user/.config",
			root:  "/home/user/.config",
			want:  false,
		},
		{
			name:  "no separator after root (shared prefix but not child)",
			child: "/home/user/.configextra",
			root:  "/home/user/.config",
			want:  false,
		},
		{
			name:  "child shorter than root",
			child: "/home",
			root:  "/home/user/.config",
			want:  false,
		},
		{
			name:  "deeply nested child",
			child: "/a/b/c/d/e/f",
			root:  "/a/b",
			want:  true,
		},
		{
			name:  "root is /",
			child: "/anything",
			root:  "/",
			want:  false, // len(child) > len(root) but child[1] is 'a' not separator
		},
		{
			name:  "empty root",
			child: "/home/user",
			root:  "",
			want:  true, // child[0] == '/' == separator, and child[:0] == "" == root
		},
		{
			name:  "both empty",
			child: "",
			root:  "",
			want:  false, // len(child) > len(root) fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPathPrefix(tt.child, tt.root)
			if got != tt.want {
				t.Fatalf("hasPathPrefix(%q, %q) = %v, want %v", tt.child, tt.root, got, tt.want)
			}
		})
	}
}

func TestFileClassToPathClass(t *testing.T) {
	tests := []struct {
		input fileClass
		want  pathClass
	}{
		{fileClassManaged, pathClassManaged},
		{fileClassIgnored, pathClassIgnored},
		{fileClassUnmanaged, pathClassUnmanaged},
		{fileClass(99), pathClassManaged}, // unknown defaults to managed
	}

	for _, tt := range tests {
		got := fileClassToPathClass(tt.input)
		if got != tt.want {
			t.Fatalf("fileClassToPathClass(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestRebuildDataset_DuplicatePathsLastWriteWins(t *testing.T) {
	// If the same path appears in both managed and ignored,
	// the later write (ignored) should win in classMap.
	managed := []string{"/home/user/.bashrc"}
	ignored := []string{"/home/user/.bashrc"}

	ds := buildDataset(managed, ignored, nil)

	fc, ok := ds.classMap["/home/user/.bashrc"]
	if !ok {
		t.Fatal("expected path in classMap")
	}
	// ignored is written after managed in rebuildDataset, so it wins.
	if fc != fileClassIgnored {
		t.Fatalf("expected fileClassIgnored for duplicate path, got %d", fc)
	}

	// allPaths should have no duplicates since it is built from map keys.
	if len(ds.allPaths) != 1 {
		t.Fatalf("expected allPaths length 1 for duplicate path, got %d", len(ds.allPaths))
	}
}

func TestRebuildDataset_IgnoredPrecedesUnmanagedOnOverlap(t *testing.T) {
	overlap := "/home/user/conflict.txt"
	ds := buildDataset(nil, []string{overlap}, []string{overlap})

	fc, ok := ds.classMap[overlap]
	if !ok {
		t.Fatal("expected overlap path in classMap")
	}
	if fc != fileClassIgnored {
		t.Fatalf("expected overlap path to remain ignored, got %d", fc)
	}
}

func TestRebuildDataset_UnmanagedFilesNotDirs(t *testing.T) {
	// Regular files in the unmanaged list should NOT appear as dir roots.
	tmpDir := t.TempDir()
	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ds := buildDataset(nil, nil, []string{regularFile})

	if len(ds.unmanagedDirRoots) != 0 {
		t.Fatalf("expected no dir roots for regular file, got %d", len(ds.unmanagedDirRoots))
	}

	// The file should still be in classMap as unmanaged.
	if ds.classMap[regularFile] != fileClassUnmanaged {
		t.Fatalf("expected regular file classified as unmanaged")
	}
}

func TestDatasetClassify_MultipleUnmanagedDirRoots(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "aaa")
	dirB := filepath.Join(tmpDir, "mmm")
	dirC := filepath.Join(tmpDir, "zzz")
	for _, d := range []string{dirA, dirB, dirC} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	ds := buildDataset(nil, nil, []string{dirC, dirA, dirB})

	// Dir roots should be sorted regardless of input order.
	if !sort.StringsAreSorted(ds.unmanagedDirRoots) {
		t.Fatal("expected unmanagedDirRoots to be sorted")
	}
	if len(ds.unmanagedDirRoots) != 3 {
		t.Fatalf("expected 3 dir roots, got %d", len(ds.unmanagedDirRoots))
	}

	// Children of each root should classify as unmanaged.
	for _, root := range []string{dirA, dirB, dirC} {
		child := filepath.Join(root, "subfile.txt")
		got := ds.classify(child)
		if got != pathClassUnmanaged {
			t.Fatalf("classify(%q) = %d, want pathClassUnmanaged", child, got)
		}
	}

	// A path between roots (lexicographically) but not a child of any root
	// should classify as managed.
	between := filepath.Join(tmpDir, "bbb-not-a-child")
	got := ds.classify(between)
	if got != pathClassManaged {
		t.Fatalf("classify(%q) = %d, want pathClassManaged", between, got)
	}
}

func TestDatasetProjectedPaths_Empty(t *testing.T) {
	ds := buildDataset(nil, nil, nil)

	got := ds.projectedPaths(fileClassManaged)
	if got != nil {
		t.Fatalf("expected nil for empty dataset projectedPaths, got %v", got)
	}
}

func TestDatasetProjectedPaths_Sorted(t *testing.T) {
	// projectedPaths iterates allPaths (which is sorted),
	// so the result should also be sorted.
	managed := []string{"/z/file", "/a/file", "/m/file"}
	ds := buildDataset(managed, []string{"/b/ignored"}, nil)

	got := ds.projectedPaths(fileClassManaged)
	if !sort.StringsAreSorted(got) {
		t.Fatalf("expected projectedPaths to return sorted slice, got %v", got)
	}
}
