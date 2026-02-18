package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// --- normalizePath tests ---

func TestNormalizePathCleansTrailingSlash(t *testing.T) {
	got := normalizePath("/home/user/")
	want := filepath.Clean("/home/user/")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePathCleansDotSegments(t *testing.T) {
	got := normalizePath("/home/user/../user/./docs")
	want := filepath.Clean("/home/user/docs")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePathEmptyReturnsEmpty(t *testing.T) {
	got := normalizePath("")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNormalizePathPreservesAbsolute(t *testing.T) {
	got := normalizePath("/usr/local/bin")
	if got != "/usr/local/bin" {
		t.Fatalf("expected /usr/local/bin, got %q", got)
	}
}

// --- walkPathsWithContext tests ---

// buildTestTree creates a temporary directory tree for testing.
// Returns the root path. Tree structure:
//
//	root/
//	  a.txt
//	  dir1/
//	    b.txt
//	    dir1a/
//	      c.txt
//	      deep/
//	        d.txt
//	  dir2/
//	    e.txt
//	  .git/
//	    config
//	  node_modules/
//	    pkg/
//	      index.js
func buildTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		"dir1/dir1a/deep",
		"dir2",
		".git",
		"node_modules/pkg",
	}
	files := map[string]string{
		"a.txt":                     "a",
		"dir1/b.txt":                "b",
		"dir1/dir1a/c.txt":          "c",
		"dir1/dir1a/deep/d.txt":     "d",
		"dir2/e.txt":                "e",
		".git/config":               "gitconfig",
		"node_modules/pkg/index.js": "module",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for f, content := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	return root
}

func TestWalkPathsBasicTraversal(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// Should include root, files, and non-skipped dirs.
	hasRoot := false
	hasATxt := false
	for _, r := range results {
		if r == root {
			hasRoot = true
		}
		if r == filepath.Join(root, "a.txt") {
			hasATxt = true
		}
	}
	if !hasRoot {
		t.Error("expected root directory in results")
	}
	if !hasATxt {
		t.Error("expected a.txt in results")
	}
}

func TestWalkPathsResultsAreSorted(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if !sort.StringsAreSorted(results) {
		t.Fatal("results are not sorted")
	}
}

func TestWalkPathsResultsAreDeduplicated(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// Walk the same root twice to exercise dedup.
	results, _, err := walkPathsWithContext(ctx, "", []string{root, root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	seen := make(map[string]struct{}, len(results))
	for _, r := range results {
		if _, exists := seen[r]; exists {
			t.Fatalf("duplicate result: %s", r)
		}
		seen[r] = struct{}{}
	}
}

func TestWalkPathsSkipDirectories(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, r := range results {
		rel, _ := filepath.Rel(root, r)
		for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
			if part == ".git" || part == "node_modules" {
				t.Fatalf("result %q should have been skipped (contains %q)", r, part)
			}
		}
	}
}

func TestWalkPathsDepthZeroMeansNoLimit(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// fastwalk: MaxDepth <= 0 means no depth limit (walk entire tree).
	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 0, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	// Should include deeply nested files like dir1/dir1a/deep/d.txt.
	deepPath := filepath.Join(root, "dir1", "dir1a", "deep", "d.txt")
	if !slices.Contains(results, deepPath) {
		t.Fatal("expected depth-0 (no limit) to include deeply nested files")
	}
}

func TestWalkPathsDepthOneDirectChildren(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 1, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, r := range results {
		if r == root {
			continue
		}
		rel, _ := filepath.Rel(root, r)
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		if depth > 1 {
			t.Fatalf("result %q at depth %d exceeds maxDepth 1", r, depth)
		}
	}

	// Should include a.txt and dir1, dir2 but not dir1/b.txt.
	hasDirChild := false
	for _, r := range results {
		if r == filepath.Join(root, "dir1") || r == filepath.Join(root, "dir2") {
			hasDirChild = true
		}
		if r == filepath.Join(root, "dir1", "b.txt") {
			t.Fatal("depth-1 walk should not include dir1/b.txt")
		}
	}
	if !hasDirChild {
		t.Fatal("expected direct child directories in depth-1 results")
	}
}

func TestWalkPathsDepthThree(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 3, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	// dir1/dir1a/c.txt is at depth 3 — should be included.
	hasC := false
	// dir1/dir1a/deep/d.txt is at depth 4 — should be excluded.
	hasD := false
	for _, r := range results {
		if r == filepath.Join(root, "dir1", "dir1a", "c.txt") {
			hasC = true
		}
		if r == filepath.Join(root, "dir1", "dir1a", "deep", "d.txt") {
			hasD = true
		}
	}
	if !hasC {
		t.Fatal("expected dir1/dir1a/c.txt at depth 3")
	}
	if hasD {
		t.Fatal("dir1/dir1a/deep/d.txt at depth 4 should be excluded at maxDepth 3")
	}
}

func TestWalkPathsMaxResultsCap(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 3)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	// Due to concurrency, we may get slightly more than maxResults.
	// But the result should be in the ballpark.
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results, got %d", len(results))
	}
	// Generous upper bound: fastwalk concurrency may slightly overshoot.
	if len(results) > 10 {
		t.Fatalf("expected results near cap of 3, got %d (too many)", len(results))
	}
}

func TestWalkPathsMaxResultsDoesNotReturnError(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	_, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1)
	if err != nil {
		t.Fatalf("max-results cap should not surface as error, got: %v", err)
	}
}

func TestWalkPathsContextCancellation(t *testing.T) {
	root := buildTestTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled walk should return context.Canceled, got: %v", err)
	}

	// Should have very few or no results since context was already cancelled.
	// Due to concurrency, a few results might slip through.
	if len(results) > 20 {
		t.Fatalf("expected few results after immediate cancellation, got %d", len(results))
	}
}

func TestWalkPathsContextTimeout(t *testing.T) {
	root := buildTestTree(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	// Let the timeout expire.
	time.Sleep(1 * time.Millisecond)

	_, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timed-out walk should return DeadlineExceeded, got: %v", err)
	}
}

func TestWalkPathsQueryMatchCaseInsensitive(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// Query "TXT" should match ".txt" files (case-insensitive).
	results, _, err := walkPathsWithContext(ctx, "TXT", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected case-insensitive matches for 'TXT'")
	}

	for _, r := range results {
		if !strings.Contains(strings.ToLower(r), "txt") {
			t.Fatalf("result %q does not contain 'txt'", r)
		}
	}
}

func TestWalkPathsQueryFiltersNonMatching(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "zzz_no_match_zzz", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected no results for non-matching query, got %d", len(results))
	}
}

func TestWalkPathsQuerySubstringMatch(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "dir1a", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected substring matches for 'dir1a'")
	}

	for _, r := range results {
		if !strings.Contains(strings.ToLower(r), "dir1a") {
			t.Fatalf("result %q does not contain 'dir1a'", r)
		}
	}
}

func TestWalkPathsEmptyQueryReturnsAll(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	withQuery, _, err := walkPathsWithContext(ctx, "b.txt", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	noQuery, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(noQuery) <= len(withQuery) {
		t.Fatalf("empty query should return more results (%d) than filtered (%d)", len(noQuery), len(withQuery))
	}
}

func TestWalkPathsWhitespaceQueryTreatedAsEmpty(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "   ", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	noQuery, _, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(results) != len(noQuery) {
		t.Fatalf("whitespace query (%d results) should match empty query (%d results)", len(results), len(noQuery))
	}
}

func TestWalkPathsMultipleRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(root1, "file1.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root2, "file2.txt"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	results, _, err := walkPathsWithContext(ctx, "", []string{root1, root2}, 3, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	hasFile1 := false
	hasFile2 := false
	for _, r := range results {
		if r == filepath.Join(root1, "file1.txt") {
			hasFile1 = true
		}
		if r == filepath.Join(root2, "file2.txt") {
			hasFile2 = true
		}
	}
	if !hasFile1 || !hasFile2 {
		t.Fatal("expected results from both roots")
	}
}

func TestWalkPathsEmptyRootsReturnsNil(t *testing.T) {
	ctx := context.Background()
	results, _, err := walkPathsWithContext(ctx, "", nil, 3, 1000)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got: %v", results)
	}
}

func TestWalkPathsZeroMaxResultsReturnsNil(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{root}, 3, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got: %v", results)
	}
}

func TestWalkPathsInvalidRootSkipped(t *testing.T) {
	validRoot := buildTestTree(t)
	invalidRoot := filepath.Join(t.TempDir(), "nonexistent")

	ctx := context.Background()
	results, _, err := walkPathsWithContext(ctx, "", []string{invalidRoot, validRoot}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	// Should still get results from the valid root.
	if len(results) == 0 {
		t.Fatal("expected results from valid root despite invalid first root")
	}
}

func TestWalkPathsEmptyStringRootSkipped(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, _, err := walkPathsWithContext(ctx, "", []string{"", root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results from valid root despite empty string root")
	}
}

func TestWalkPathsSkipDirListContents(t *testing.T) {
	// Verify the skip list matches Phase 1 Step 1.5 spec.
	expected := []string{".git", ".cache", "node_modules", "vendor", "__pycache__", "build", "dist"}
	for _, dir := range expected {
		if _, exists := walkSkipDirs[dir]; !exists {
			t.Errorf("expected %q in walkSkipDirs", dir)
		}
	}
	if len(walkSkipDirs) != len(expected) {
		t.Errorf("walkSkipDirs has %d entries, expected %d", len(walkSkipDirs), len(expected))
	}
}

func TestWalkPathsSkipDirNotTraversed(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// Query for "config" which exists inside .git/config — it should NOT appear.
	results, _, err := walkPathsWithContext(ctx, "config", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, r := range results {
		rel, _ := filepath.Rel(root, r)
		if strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			t.Fatalf("result %q is inside skipped .git directory", r)
		}
	}
}

func TestWalkPathsSkipDirNodeModulesNotTraversed(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// Query for "index.js" which exists inside node_modules — should NOT appear.
	results, _, err := walkPathsWithContext(ctx, "index.js", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, r := range results {
		rel, _ := filepath.Rel(root, r)
		if strings.HasPrefix(rel, "node_modules"+string(filepath.Separator)) {
			t.Fatalf("result %q is inside skipped node_modules directory", r)
		}
	}
}

func TestWalkPathsNormalizedOutput(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	// Pass root with trailing slash to verify normalization.
	results, _, err := walkPathsWithContext(ctx, "", []string{root + "/"}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	for _, r := range results {
		if r != filepath.Clean(r) {
			t.Fatalf("result %q is not filepath.Clean'd", r)
		}
	}
}

func TestWalkPathsMetricsComplete(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, metrics, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	if metrics.terminated != "complete" {
		t.Fatalf("expected complete termination, got %q", metrics.terminated)
	}
	if metrics.roots != 1 {
		t.Fatalf("expected roots=1, got %d", metrics.roots)
	}
	if metrics.matches != len(results) {
		t.Fatalf("expected matches=%d, got %d", len(results), metrics.matches)
	}
	if metrics.elapsed <= 0 {
		t.Fatalf("expected positive elapsed duration, got %v", metrics.elapsed)
	}
}

func TestWalkPathsMetricsMaxResults(t *testing.T) {
	root := buildTestTree(t)
	ctx := context.Background()

	results, metrics, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1)
	if err != nil {
		t.Fatalf("max-results walk should not error, got: %v", err)
	}
	if metrics.terminated != "max-results" {
		t.Fatalf("expected max-results termination, got %q", metrics.terminated)
	}
	if metrics.matches != len(results) {
		t.Fatalf("expected matches=%d, got %d", len(results), metrics.matches)
	}
}

func TestWalkPathsMetricsCanceled(t *testing.T) {
	root := buildTestTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, metrics, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if metrics.terminated != "canceled" {
		t.Fatalf("expected canceled termination, got %q", metrics.terminated)
	}
}

func TestWalkPathsMetricsDeadline(t *testing.T) {
	root := buildTestTree(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	_, metrics, err := walkPathsWithContext(ctx, "", []string{root}, 10, 1000)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
	if metrics.terminated != "deadline" {
		t.Fatalf("expected deadline termination, got %q", metrics.terminated)
	}
}
