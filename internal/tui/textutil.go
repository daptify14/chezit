package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func visualTruncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxWidth {
		return s
	}
	return ansi.Truncate(s, maxWidth, "…")
}

func visualPad(s string, targetWidth int) string {
	w := ansi.StringWidth(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}

func visibleRange(total, cursor, height int) (int, int) {
	if height <= 0 || total <= 0 {
		return 0, 0
	}
	if total <= height {
		return 0, total
	}
	if cursor < 0 {
		cursor = 0
	} else if cursor >= total {
		cursor = total - 1
	}
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	end := min(start+height, total)
	if start < 0 {
		start = 0
	}
	return start, end
}

func shortenPath(path, targetPath string) string {
	if targetPath != "" && strings.HasPrefix(path, targetPath+string(filepath.Separator)) {
		return "~/" + path[len(targetPath)+1:]
	}
	return path
}
