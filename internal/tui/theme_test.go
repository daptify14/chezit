package tui

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	catppuccin "github.com/catppuccin/go"
)

func TestThemeRoleProfilesMatchCanonicalMapping(t *testing.T) {
	tests := []struct {
		name   string
		flavor catppuccin.Flavor
		theme  Theme
	}{
		{name: "dark", flavor: catppuccin.Mocha, theme: ThemeDark()},
		{name: "light", flavor: catppuccin.Latte, theme: ThemeLight()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertColorHex(t, "PrimaryFg", tc.theme.PrimaryFg.GetForeground(), normalizeHex(tc.flavor.Sapphire().Hex))
			assertColorHex(t, "AccentFg", tc.theme.AccentFg.GetForeground(), normalizeHex(tc.flavor.Yellow().Hex))
			assertColorHex(t, "Selected.Background", tc.theme.Selected.GetBackground(), normalizeHex(tc.flavor.Surface0().Hex))
			assertColorHex(t, "StatusBar.Background", tc.theme.StatusBar.GetBackground(), normalizeHex(tc.flavor.Mantle().Hex))
			assertColorHex(t, "MenuTitle.Background", tc.theme.MenuTitle.GetBackground(), normalizeHex(tc.flavor.Mauve().Hex))
		})
	}
}

func TestThemeDarkAndLightAreDistinct(t *testing.T) {
	dark := ThemeDark()
	light := ThemeLight()

	darkPrimary := colorHex(dark.PrimaryFg.GetForeground())
	lightPrimary := colorHex(light.PrimaryFg.GetForeground())

	if darkPrimary == lightPrimary {
		t.Fatalf("dark and light themes should have distinct primary colors, both are %s", darkPrimary)
	}
}

func TestThemeDerivedStylesUseCanonicalColors(t *testing.T) {
	tests := []struct {
		name   string
		flavor catppuccin.Flavor
		theme  Theme
	}{
		{name: "dark", flavor: catppuccin.Mocha, theme: ThemeDark()},
		{name: "light", flavor: catppuccin.Latte, theme: ThemeLight()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			primary := normalizeHex(tc.flavor.Sapphire().Hex)
			accent := normalizeHex(tc.flavor.Yellow().Hex)

			assertColorHex(t, "BoldPrimary.Foreground", tc.theme.BoldPrimary.GetForeground(), primary)
			assertColorHex(t, "ActiveTab.Foreground", tc.theme.ActiveTab.GetForeground(), primary)
			assertColorHex(t, "Filter.BorderTopForeground", tc.theme.Filter.GetBorderTopForeground(), primary)
			assertColorHex(t, "PrimaryFg.Foreground", tc.theme.PrimaryFg.GetForeground(), primary)

			assertColorHex(t, "AccentFg.Foreground", tc.theme.AccentFg.GetForeground(), accent)

			assertColorHex(t, "SuccessFg.Foreground", tc.theme.SuccessFg.GetForeground(), normalizeHex(tc.flavor.Green().Hex))
			assertColorHex(t, "WarningFg.Foreground", tc.theme.WarningFg.GetForeground(), normalizeHex(tc.flavor.Peach().Hex))
			assertColorHex(t, "DangerFg.Foreground", tc.theme.DangerFg.GetForeground(), normalizeHex(tc.flavor.Red().Hex))
		})
	}
}

func TestThemeLightTextForegrounds(t *testing.T) {
	theme := ThemeLight()
	flavor := catppuccin.Latte

	assertColorHex(t, "Normal.Foreground", theme.Normal.GetForeground(), normalizeHex(flavor.Text().Hex))
	assertColorHex(t, "DimText.Foreground", theme.DimText.GetForeground(), normalizeHex(flavor.Subtext0().Hex))
	assertColorHex(t, "HintText.Foreground", theme.HintText.GetForeground(), normalizeHex(flavor.Overlay1().Hex))
	assertColorHex(t, "Selected.Foreground", theme.Selected.GetForeground(), normalizeHex(flavor.Text().Hex))
	assertColorHex(t, "StatusBar.Foreground", theme.StatusBar.GetForeground(), normalizeHex(flavor.Text().Hex))
}

func TestThemeForBackground(t *testing.T) {
	dark := ThemeForBackground(true)
	light := ThemeForBackground(false)

	assertColorHex(t, "dark primary", dark.PrimaryFg.GetForeground(), colorHex(ThemeDark().PrimaryFg.GetForeground()))
	assertColorHex(t, "light primary", light.PrimaryFg.GetForeground(), colorHex(ThemeLight().PrimaryFg.GetForeground()))
}

func assertColorHex(t *testing.T, field string, got color.Color, want string) {
	t.Helper()
	if gotHex := colorHex(got); gotHex != want {
		t.Fatalf("%s color mismatch: got %s want %s", field, gotHex, want)
	}
}

func colorHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", (r>>8)&0xFF, (g>>8)&0xFF, (b>>8)&0xFF)
}

func normalizeHex(hex string) string {
	hex = strings.ToLower(strings.TrimSpace(hex))
	if hex == "" {
		return ""
	}
	if !strings.HasPrefix(hex, "#") {
		return "#" + hex
	}
	return hex
}
