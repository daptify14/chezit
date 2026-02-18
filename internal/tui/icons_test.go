package tui

import (
	"testing"
)

func TestParseIconMode(t *testing.T) {
	tests := []struct {
		input   string
		want    IconMode
		wantErr bool
	}{
		{"nerdfont", IconModeNerdFont, false},
		{"unicode", IconModeUnicode, false},
		{"none", IconModeNone, false},
		{"NERDFONT", IconModeNerdFont, false},
		{"  unicode  ", IconModeUnicode, false},
		{"", IconModeNerdFont, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseIconMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileIcon(t *testing.T) {
	t.Run("none mode returns empty", func(t *testing.T) {
		if got := fileIcon("main.go", false, IconModeNone); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
		if got := fileIcon("src", true, IconModeNone); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("unicode mode returns unicode glyphs", func(t *testing.T) {
		icon := fileIcon("readme.md", false, IconModeUnicode)
		if icon == "" {
			t.Fatal("expected non-empty icon for .md file in unicode mode")
		}
		if icon == fileIcon("image.png", false, IconModeUnicode) {
			t.Fatal("expected different icons for .md and .png")
		}
	})

	t.Run("unicode mode dir returns folder icon", func(t *testing.T) {
		icon := fileIcon("src", true, IconModeUnicode)
		if icon != unicodeDirIcon {
			t.Fatalf("expected unicode dir icon, got %q", icon)
		}
	})

	t.Run("unicode mode unknown extension returns default", func(t *testing.T) {
		icon := fileIcon("file.xyz", false, IconModeUnicode)
		if icon != unicodeDefaultIcon {
			t.Fatalf("expected default unicode icon, got %q", icon)
		}
	})

	t.Run("nerdfont mode returns non-empty for files", func(t *testing.T) {
		icon := fileIcon("main.go", false, IconModeNerdFont)
		if icon == "" {
			t.Fatal("expected non-empty icon for .go file in nerdfont mode")
		}
	})

	t.Run("nerdfont mode dir returns folder icon", func(t *testing.T) {
		icon := fileIcon("src", true, IconModeNerdFont)
		if icon != nerdFontDirIcon {
			t.Fatalf("expected nerdfont dir icon, got %q", icon)
		}
	})
}

func TestFileIconColor(t *testing.T) {
	t.Run("none mode returns empty", func(t *testing.T) {
		if got := fileIconColor("main.go", false, IconModeNone); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("unicode mode returns empty", func(t *testing.T) {
		if got := fileIconColor("main.go", false, IconModeUnicode); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("nerdfont dir returns empty", func(t *testing.T) {
		if got := fileIconColor("src", true, IconModeNerdFont); got != "" {
			t.Fatalf("expected empty for dir, got %q", got)
		}
	})

	t.Run("nerdfont file returns hex color", func(t *testing.T) {
		color := fileIconColor("main.go", false, IconModeNerdFont)
		if color == "" {
			t.Fatal("expected non-empty color for .go file in nerdfont mode")
		}
		if color[0] != '#' {
			t.Fatalf("expected hex color starting with #, got %q", color)
		}
	})
}

func TestRenderFileIcon(t *testing.T) {
	t.Run("none mode returns empty", func(t *testing.T) {
		if got := renderFileIcon("main.go", false, false, IconModeNone); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("nerdfont returns icon with trailing space", func(t *testing.T) {
		got := renderFileIcon("main.go", false, false, IconModeNerdFont)
		if got == "" {
			t.Fatal("expected non-empty rendered icon")
		}
		// Should end with a space for separation from filename
		if got[len(got)-1] != ' ' {
			t.Fatalf("expected trailing space, got %q", got)
		}
	})

	t.Run("unicode returns icon with trailing space", func(t *testing.T) {
		got := renderFileIcon("config.yaml", false, false, IconModeUnicode)
		if got == "" {
			t.Fatal("expected non-empty rendered icon")
		}
	})
}
