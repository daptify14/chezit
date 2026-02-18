package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	catppuccin "github.com/catppuccin/go"
)

// Theme holds all semantic colors and pre-computed styles for the TUI.
type Theme struct {
	Primary    color.Color
	Accent     color.Color
	Success    color.Color
	Warning    color.Color
	Danger     color.Color
	Dim        color.Color
	SubtleText color.Color

	Selected lipgloss.Style
	Normal   lipgloss.Style
	DimText  lipgloss.Style
	HintText lipgloss.Style

	BoldPrimary lipgloss.Style
	BoldAccent  lipgloss.Style
	BoldWarning lipgloss.Style
	BoldSuccess lipgloss.Style
	BoldOnly    lipgloss.Style

	PrimaryFg lipgloss.Style
	SuccessFg lipgloss.Style
	WarningFg lipgloss.Style
	DangerFg  lipgloss.Style
	AccentFg  lipgloss.Style

	Branch lipgloss.Style

	Filter lipgloss.Style

	StatusBar lipgloss.Style

	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style

	HelpOverlay lipgloss.Style

	CenteredBox lipgloss.Style

	MenuTitle lipgloss.Style

	Panel lipgloss.Style

	ChromaStyleName string
}

var activeTheme = ThemeDark()

// SetTheme sets the active global theme.
func SetTheme(t Theme) { activeTheme = t }

// ThemeDark returns the dark theme (Catppuccin Mocha palette).
func ThemeDark() Theme { return newTheme(catppuccin.Mocha, true) }

// ThemeLight returns the light theme (Catppuccin Latte palette).
func ThemeLight() Theme { return newTheme(catppuccin.Latte, false) }

// ThemeForBackground returns the appropriate theme for the terminal background.
func ThemeForBackground(isDark bool) Theme {
	if isDark {
		return ThemeDark()
	}
	return ThemeLight()
}

type roleProfile struct {
	primary     color.Color
	accent      color.Color
	selectedBg  color.Color
	statusBarBg color.Color
	menuTitleBg color.Color
}

type textProfile struct {
	normal color.Color
	dim    color.Color
	hint   color.Color
}

// canonicalRoleProfile maps semantic roles to Catppuccin palette tokens.
// The same token names are used for both dark and light palettes; the
// underlying hex values differ because each palette defines its own colors.
func canonicalRoleProfile(flavor catppuccin.Flavor) roleProfile {
	return roleProfile{
		primary:     lipgloss.Color(flavor.Sapphire().Hex),
		accent:      lipgloss.Color(flavor.Yellow().Hex),
		selectedBg:  lipgloss.Color(flavor.Surface0().Hex),
		statusBarBg: lipgloss.Color(flavor.Mantle().Hex),
		menuTitleBg: lipgloss.Color(flavor.Mauve().Hex),
	}
}

// textForegrounds returns text colors appropriate for the terminal polarity.
// Dark terminals use the palette's native content tokens; light terminals
// use darker content tokens to maintain contrast on inherited backgrounds.
func textForegrounds(flavor catppuccin.Flavor, isDark bool) textProfile {
	if isDark {
		return textProfile{
			normal: lipgloss.Color(flavor.Text().Hex),
			dim:    lipgloss.Color(flavor.Overlay1().Hex),
			hint:   lipgloss.Color(flavor.Subtext0().Hex),
		}
	}
	return textProfile{
		normal: lipgloss.Color(flavor.Text().Hex),
		dim:    lipgloss.Color(flavor.Subtext0().Hex),
		hint:   lipgloss.Color(flavor.Overlay1().Hex),
	}
}

// newTheme constructs a Theme from a Catppuccin flavor and terminal polarity.
func newTheme(flavor catppuccin.Flavor, isDark bool) Theme {
	profile := canonicalRoleProfile(flavor)
	tp := textForegrounds(flavor, isDark)

	primary := profile.primary
	secondary := lipgloss.Color(flavor.Overlay0().Hex)
	accent := profile.accent
	success := lipgloss.Color(flavor.Green().Hex)
	warning := lipgloss.Color(flavor.Peach().Hex)
	danger := lipgloss.Color(flavor.Red().Hex)
	dim := tp.dim
	selBg := profile.selectedBg
	selFg := lipgloss.Color(flavor.Text().Hex)
	statusBarBg := profile.statusBarBg
	menuTitleBg := profile.menuTitleBg

	chromaStyle := "catppuccin-mocha"
	if !isDark {
		chromaStyle = "catppuccin-latte"
	}

	t := Theme{
		Primary:    primary,
		Accent:     accent,
		Success:    success,
		Warning:    warning,
		Danger:     danger,
		Dim:        dim,
		SubtleText: lipgloss.Color(flavor.Subtext0().Hex),

		ChromaStyleName: chromaStyle,
	}

	t.Selected = lipgloss.NewStyle().
		Background(selBg).
		Foreground(selFg).
		Bold(true)
	t.Normal = lipgloss.NewStyle().Foreground(tp.normal)
	t.DimText = lipgloss.NewStyle().Foreground(dim)
	t.HintText = lipgloss.NewStyle().Foreground(tp.hint)

	t.BoldPrimary = lipgloss.NewStyle().Bold(true).Foreground(primary)
	t.BoldAccent = lipgloss.NewStyle().Bold(true).Foreground(accent)
	t.BoldWarning = lipgloss.NewStyle().Bold(true).Foreground(warning)
	t.BoldSuccess = lipgloss.NewStyle().Bold(true).Foreground(success)
	t.BoldOnly = lipgloss.NewStyle().Bold(true)

	t.PrimaryFg = lipgloss.NewStyle().Foreground(primary)
	t.SuccessFg = lipgloss.NewStyle().Foreground(success)
	t.WarningFg = lipgloss.NewStyle().Foreground(warning)
	t.DangerFg = lipgloss.NewStyle().Foreground(danger)
	t.AccentFg = lipgloss.NewStyle().Foreground(accent)

	t.Branch = lipgloss.NewStyle().Foreground(secondary)

	t.Filter = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(0, 1)

	t.StatusBar = lipgloss.NewStyle().
		Background(statusBarBg).
		Foreground(selFg).
		Padding(0, 1)

	t.ActiveTab = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		Underline(true)
	t.InactiveTab = lipgloss.NewStyle().Foreground(dim)

	t.HelpOverlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2)

	t.CenteredBox = lipgloss.NewStyle().
		Padding(1, 2)

	t.MenuTitle = lipgloss.NewStyle().
		Bold(true).
		Background(menuTitleBg).
		Foreground(lipgloss.Color(flavor.Crust().Hex)).
		Padding(0, 1)

	t.Panel = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(dim).
		Padding(0, 1)

	return t
}
