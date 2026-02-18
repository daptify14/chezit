package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	devicons "github.com/epilande/go-devicons"
)

// IconMode controls which icon set the TUI uses.
type IconMode string

// Icon mode values controlling which icon set is displayed.
const (
	IconModeNerdFont IconMode = "nerdfont"
	IconModeUnicode  IconMode = "unicode"
	IconModeNone     IconMode = "none"
)

var validIconModes = []IconMode{IconModeNerdFont, IconModeUnicode, IconModeNone}

// ParseIconMode validates and normalizes an icon mode string.
func ParseIconMode(s string) (IconMode, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return IconModeNerdFont, nil
	}
	for _, m := range validIconModes {
		if string(m) == s {
			return m, nil
		}
	}
	return "", fmt.Errorf("invalid icons mode %q (valid: nerdfont, unicode, none)", s)
}

// nerdFontDirIcon is the Nerd Font folder icon glyph.
const nerdFontDirIcon = "\uf115" // nf-fa-folder_open_o

// unicodeDirIcon is the standard Unicode folder icon.
const unicodeDirIcon = "\U0001F4C1" // 📁

// unicodeIcons maps file extensions to standard Unicode symbols.
var unicodeIcons = map[string]string{
	".md":   "\U0001F4DD", // 📝
	".txt":  "\U0001F4C4", // 📄
	".pdf":  "\U0001F4C4", // 📄
	".log":  "\U0001F4C4", // 📄
	".png":  "\U0001F5BC", // 🖼
	".jpg":  "\U0001F5BC", // 🖼
	".jpeg": "\U0001F5BC", // 🖼
	".gif":  "\U0001F5BC", // 🖼
	".svg":  "\U0001F5BC", // 🖼
	".webp": "\U0001F5BC", // 🖼
	".zip":  "\U0001F4E6", // 📦
	".tar":  "\U0001F4E6", // 📦
	".gz":   "\U0001F4E6", // 📦
	".bz2":  "\U0001F4E6", // 📦
	".xz":   "\U0001F4E6", // 📦
	".yaml": "\u2699",     // ⚙
	".yml":  "\u2699",     // ⚙
	".toml": "\u2699",     // ⚙
	".json": "\u2699",     // ⚙
	".ini":  "\u2699",     // ⚙
	".conf": "\u2699",     // ⚙
	".cfg":  "\u2699",     // ⚙
	".sh":   "\u25B6",     // ▶
	".bash": "\u25B6",     // ▶
	".zsh":  "\u25B6",     // ▶
	".fish": "\u25B6",     // ▶
}

// unicodeDefaultIcon is the fallback for files with unrecognized extensions.
const unicodeDefaultIcon = "\U0001F4C4" // 📄

// fileIcon returns the icon glyph for a file or directory.
func fileIcon(name string, isDir bool, mode IconMode) string {
	switch mode {
	case IconModeNone:
		return ""
	case IconModeUnicode:
		if isDir {
			return unicodeDirIcon
		}
		ext := strings.ToLower(filepath.Ext(name))
		if icon, ok := unicodeIcons[ext]; ok {
			return icon
		}
		return unicodeDefaultIcon
	case IconModeNerdFont:
		if isDir {
			return nerdFontDirIcon
		}
		style := devicons.IconForPath(name)
		return style.Icon
	default:
		return ""
	}
}

// fileIconColor returns the hex color string (e.g. "#61AFEF") for the icon.
// Returns empty string when no specific color applies (unicode/none modes, directories).
func fileIconColor(name string, isDir bool, mode IconMode) string {
	if mode != IconModeNerdFont || isDir {
		return ""
	}
	style := devicons.IconForPath(name)
	return style.Color
}

// renderFileIcon returns a styled icon string ready for display.
// When selected, icon color is omitted so it inherits the selection style.
// Returns empty string when icon mode is none.
func renderFileIcon(name string, isDir, selected bool, mode IconMode) string {
	icon := fileIcon(name, isDir, mode)
	if icon == "" {
		return ""
	}
	if selected {
		return icon + " "
	}
	hexColor := fileIconColor(name, isDir, mode)
	if hexColor != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hexColor)).Render(icon) + " "
	}
	if isDir {
		return activeTheme.PrimaryFg.Render(icon) + " "
	}
	return activeTheme.DimText.Render(icon) + " "
}
