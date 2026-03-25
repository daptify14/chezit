package tui

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errPagerUnsupported = errors.New("pager not in supported list")

// pagerTimeout is the maximum time allowed for a pager subprocess.
const pagerTimeout = 3 * time.Second

// supportedPagerFlags maps pager binary basenames to mandatory safety flags.
// Only pagers in this table are activated; unsupported pagers are ignored.
var supportedPagerFlags = map[string][]string{
	"delta": {
		"--color-only",
		"--paging=never",
		"--detect-dark-light=never",
		"--width=variable",
		"--line-numbers=false",
		"--side-by-side=false",
	},
	"bat":           {"--paging=never", "--plain", "--color=always"},
	"diff-so-fancy": {},
}

// preparePagerArgs parses the user's pager command string, checks if the
// binary is supported, and appends safety flags. Returns the args slice
// and true if the pager is supported, or nil and false otherwise.
func preparePagerArgs(pagerCmd string, isDark bool) ([]string, bool) {
	if pagerCmd == "" {
		return nil, false
	}

	parts := strings.Fields(pagerCmd)
	if len(parts) == 0 {
		return nil, false
	}

	binary := filepath.Base(parts[0])
	flags, supported := supportedPagerFlags[binary]
	if !supported {
		return nil, false
	}

	args := make([]string, len(parts), len(parts)+len(flags)+1)
	copy(args, parts)
	args = append(args, flags...)

	if binary == "delta" {
		if isDark {
			args = append(args, "--dark")
		} else {
			args = append(args, "--light")
		}
	}

	return args, true
}

// pipeThroughDiffPager runs rawDiff through the pager command and returns
// ANSI-colored output. Returns an error if the pager is unsupported, not
// installed, or fails.
func pipeThroughDiffPager(rawDiff, pagerCmd string, isDark bool) (string, error) {
	args, supported := preparePagerArgs(pagerCmd, isDark)
	if !supported {
		return "", errPagerUnsupported
	}

	ctx, cancel := context.WithTimeout(context.Background(), pagerTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //#nosec G204 -- user-configured pager from chezmoi config
	cmd.Stdin = strings.NewReader(rawDiff)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return stdout.String(), nil
}

// diffRawLines returns the canonical raw diff lines for semantic operations
// like diffSummary. Falls back to rendered lines if rawLines is empty.
func (m Model) diffRawLines() []string {
	if len(m.diff.rawLines) > 0 {
		return m.diff.rawLines
	}
	return m.diff.lines
}

// renderDiffWithPager attempts to pipe rawDiff through the configured pager.
// Returns the rendered output and true if successful, or empty string and false
// if no pager is configured, unsupported, or execution fails.
func (m Model) renderDiffWithPager(rawDiff string) (string, bool) {
	if m.diffPagerCmd == "" {
		return "", false
	}
	rendered, err := pipeThroughDiffPager(rawDiff, m.diffPagerCmd, activeTheme.IsDark)
	if err != nil {
		return "", false
	}
	return rendered, true
}
