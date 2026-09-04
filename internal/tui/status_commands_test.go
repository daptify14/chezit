package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/daptify14/chezit/internal/chezmoi"
)

func TestResolveEditorOrder(t *testing.T) {
	tests := []struct {
		name   string
		editor string
		edit   chezmoi.EditConfig
		visual string
		env    string
		want   []string
	}{
		{
			name:   "option wins over chezmoi and environment",
			editor: "code --wait",
			edit:   chezmoi.EditConfig{Command: "vim", Args: []string{"-u", "NONE"}},
			visual: "nvim",
			env:    "vi",
			want:   []string{"code", "--wait"},
		},
		{
			name:   "chezmoi edit.command with edit.args",
			edit:   chezmoi.EditConfig{Command: "code", Args: []string{"--new-window", "--wait"}},
			visual: "nvim",
			env:    "vi",
			want:   []string{"code", "--new-window", "--wait"},
		},
		{
			name:   "edit.args follow the environment editor",
			edit:   chezmoi.EditConfig{Args: []string{"--wait"}},
			visual: "code",
			want:   []string{"code", "--wait"},
		},
		{
			name:   "edit.args follow the option when edit.command is unset",
			editor: "code",
			edit:   chezmoi.EditConfig{Args: []string{"--wait"}},
			visual: "nvim",
			want:   []string{"code", "--wait"},
		},
		{
			name:   "VISUAL before EDITOR",
			visual: "nvim",
			env:    "vi",
			want:   []string{"nvim"},
		},
		{
			name: "EDITOR alone is split on whitespace",
			env:  "vi -u NONE",
			want: []string{"vi", "-u", "NONE"},
		},
		{
			name: "nothing set falls back to vi",
			want: []string{"vi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.env)
			m := Model{opts: Options{Editor: tt.editor, EditConfig: tt.edit}}
			if got := m.resolveEditor(); !slices.Equal(got, tt.want) {
				t.Fatalf("resolveEditor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitEditorKeepsExecutablePathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "My Editor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "edit")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write editor: %v", err)
	}
	if got := splitEditor(path); !slices.Equal(got, []string{path}) {
		t.Fatalf("splitEditor(%q) = %q, want the path unsplit", path, got)
	}
}

func TestEditorCmdAppendsFile(t *testing.T) {
	t.Setenv("VISUAL", "code")
	t.Setenv("EDITOR", "")
	m := Model{opts: Options{EditConfig: chezmoi.EditConfig{Args: []string{"--wait"}}}}
	cmd := m.editorCmd("/tmp/notes.txt")
	want := []string{"code", "--wait", "/tmp/notes.txt"}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("editorCmd Args = %q, want %q", cmd.Args, want)
	}
}
