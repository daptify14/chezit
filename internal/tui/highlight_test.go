package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestDetectLexer(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "empty filename", filename: "", want: ""},
		{name: "go source file", filename: "main.go", want: "Go"},
		{name: "template file uses inner extension", filename: "config.yaml.tmpl", want: "YAML"},
		{name: "unknown file", filename: "config.chezitunknown", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lexer := detectLexer(tc.filename)
			if tc.want == "" {
				if lexer != nil {
					t.Fatalf("detectLexer(%q) = %q, want nil", tc.filename, lexer.Config().Name)
				}
				return
			}

			if lexer == nil {
				t.Fatalf("detectLexer(%q) = nil, want %q", tc.filename, tc.want)
			}
			if got := lexer.Config().Name; got != tc.want {
				t.Fatalf("detectLexer(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestHighlightCodeReturnsSourceWhenLexerMissing(t *testing.T) {
	source := "plain text\nline two"
	got := highlightCode(source, "README.chezitunknown")
	if got != source {
		t.Fatalf("highlightCode should return source unchanged when lexer is unknown:\nwant: %q\ngot:  %q", source, got)
	}
}

func TestHighlightCodePreservesTextAndTrailingNewline(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		endsWithN bool
	}{
		{name: "without trailing newline", source: "package main", endsWithN: false},
		{name: "with trailing newline", source: "package main\n", endsWithN: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := highlightCode(tc.source, "main.go")
			if plain := ansi.Strip(got); plain != tc.source {
				t.Fatalf("highlighted output should preserve source text:\nwant: %q\ngot:  %q", tc.source, plain)
			}
			if strings.HasSuffix(got, "\n") != tc.endsWithN {
				t.Fatalf("trailing newline mismatch for %q: got=%t want=%t", tc.name, strings.HasSuffix(got, "\n"), tc.endsWithN)
			}
		})
	}
}

func TestThemeProvidesChromaStyleName(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
		want  string
	}{
		{name: "dark", theme: ThemeDark(), want: "catppuccin-mocha"},
		{name: "light", theme: ThemeLight(), want: "catppuccin-latte"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.theme.ChromaStyleName; got != tc.want {
				t.Fatalf("theme style mismatch: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestHighlightCodeFallsBackWhenThemeStyleNameMissing(t *testing.T) {
	prev := activeTheme
	t.Cleanup(func() { activeTheme = prev })

	fallbackTheme := ThemeDark()
	fallbackTheme.ChromaStyleName = ""
	activeTheme = fallbackTheme

	source := "package main\n"
	got := highlightCode(source, "main.go")

	if plain := ansi.Strip(got); plain != source {
		t.Fatalf("fallback style should still preserve source text:\nwant: %q\ngot:  %q", source, plain)
	}
}
