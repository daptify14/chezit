package tui

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charlievieth/fastwalk"
)

// walkMaxWorkers is the default concurrency for fastwalk.
const walkMaxWorkers = 4

// errWalkMaxResults is a sentinel error used to stop walking after reaching
// the result cap. It is filtered from the returned error so callers never
// see it as a failure.
var errWalkMaxResults = errors.New("max results reached")

// errWalkCanceled signals context cancellation/deadline inside the walk callback.
// Unlike errWalkMaxResults, this sentinel is converted back into ctx.Err() and
// returned to callers.
var errWalkCanceled = errors.New("walk canceled")

// walkSkipDirs lists directory base names that the walker skips for
// performance safety. These cover common large trees that rarely contain
// user-managed dotfiles.
var walkSkipDirs = map[string]struct{}{
	".git":         {},
	".cache":       {},
	"node_modules": {},
	"vendor":       {},
	"__pycache__":  {},
	"build":        {},
	"dist":         {},
}

// normalizePath applies the project's single path normalization policy:
// filepath.Clean with empty/dot guard. All walker inputs, outputs, and
// filter comparisons use this helper.
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// walkPathsWithContext performs a bounded, concurrent filesystem walk across
// one or more root directories. It returns deduplicated, sorted absolute paths
// that contain the query as a case-insensitive substring.
//
// The walk uses charlievieth/fastwalk for concurrent directory reading with
// built-in depth enforcement, permission-error handling, and worker pooling.
//
// Context cancellation, max-result caps, and the skip-directory policy all
// cause early termination. Because fastwalk runs callbacks concurrently, a
// few extra results may arrive after a stop signal — this is normalized by
// the dedup+sort step.
func walkPathsWithContext(
	ctx context.Context,
	query string,
	roots []string,
	maxDepth int,
	maxResults int,
) ([]string, filesSearchMetrics, error) {
	startedAt := time.Now()
	finalizeMetrics := func(results []string, terminated string, rootsSearched int) filesSearchMetrics {
		return filesSearchMetrics{
			elapsed:    time.Since(startedAt),
			roots:      rootsSearched,
			matches:    len(results),
			terminated: terminated,
		}
	}

	if len(roots) == 0 || maxResults <= 0 {
		return nil, finalizeMetrics(nil, "complete", 0), nil
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))

	var (
		mu            sync.Mutex
		seen          = make(map[string]struct{}, 256)
		results       = make([]string, 0, 256)
		stopped       bool
		rootsSearched int
		reachedMax    bool
	)

	conf := &fastwalk.Config{
		NumWorkers: walkMaxWorkers,
		Follow:     false,
		Sort:       fastwalk.SortNone,
		MaxDepth:   maxDepth,
	}

	walkFn := func(rootClean string) fs.WalkDirFunc {
		return func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Check context cancellation.
			select {
			case <-ctx.Done():
				return errWalkCanceled
			default:
			}

			// Check if another root's walk already filled the cap.
			mu.Lock()
			alreadyStopped := stopped
			mu.Unlock()
			if alreadyStopped {
				return errWalkMaxResults
			}

			// Skip-directory policy.
			if d.IsDir() {
				base := filepath.Base(path)
				if _, skip := walkSkipDirs[base]; skip && path != rootClean {
					return fs.SkipDir
				}
			}

			// Case-insensitive substring match on full path.
			if queryLower != "" && !strings.Contains(strings.ToLower(path), queryLower) {
				return nil
			}

			normalized := normalizePath(path)
			if normalized == "" || normalized == "." {
				return nil
			}

			mu.Lock()
			defer mu.Unlock()

			if _, exists := seen[normalized]; exists {
				return nil
			}
			seen[normalized] = struct{}{}
			results = append(results, normalized)

			if len(results) >= maxResults {
				stopped = true
				return errWalkMaxResults
			}
			return nil
		}
	}

	// Walk each root sequentially. Each fastwalk.Walk call is internally
	// concurrent across directories within that root. Sequential iteration
	// across roots is sufficient given the shallow depth constraint and
	// typical small root count.
	for _, root := range roots {
		rootClean := normalizePath(root)
		if rootClean == "" {
			continue
		}
		rootsSearched++

		err := fastwalk.Walk(conf, rootClean, fastwalk.IgnorePermissionErrors(walkFn(rootClean)))
		if err != nil {
			switch {
			case errors.Is(err, errWalkMaxResults):
				// Expected bounded completion.
				reachedMax = true
			case errors.Is(err, errWalkCanceled):
				sort.Strings(results)
				if ctx.Err() != nil {
					switch {
					case errors.Is(ctx.Err(), context.DeadlineExceeded):
						return results, finalizeMetrics(results, "deadline", rootsSearched), ctx.Err()
					default:
						return results, finalizeMetrics(results, "canceled", rootsSearched), ctx.Err()
					}
				}
				return results, finalizeMetrics(results, "canceled", rootsSearched), context.Canceled
			default:
				// Non-sentinel errors from a single root are non-fatal.
				// Continue with remaining roots.
				continue
			}
		}

		mu.Lock()
		full := stopped
		mu.Unlock()
		if full {
			break
		}
	}

	sort.Strings(results)
	terminated := "complete"
	if reachedMax {
		terminated = "max-results"
	}
	return results, finalizeMetrics(results, terminated, rootsSearched), nil
}
