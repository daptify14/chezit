package chezmoi

import (
	"errors"
	"testing"

	chezitconfig "github.com/daptify14/chezit/internal/config"
)

func TestIsReadOnly(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "/home/user")
	if p.IsReadOnly() {
		t.Fatal("expected IsReadOnly=false for write mode")
	}

	p = NewPolicy(chezitconfig.ModeReadOnly, "/home/user")
	if !p.IsReadOnly() {
		t.Fatal("expected IsReadOnly=true for read_only mode")
	}
}

func TestCheckMutation(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "/home/user")
	if err := p.CheckMutation(); err != nil {
		t.Fatalf("expected no error for write mode, got %v", err)
	}

	p = NewPolicy(chezitconfig.ModeReadOnly, "/home/user")
	if err := p.CheckMutation(); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected ErrReadOnly for read_only mode, got %v", err)
	}
}

func TestValidateTargetPath(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "/home/user")

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"empty path", "", ErrPathEmpty},
		{"relative path", "relative/path", ErrPathNotAbs},
		{"exact target", "/home/user", nil},
		{"within target", "/home/user/.config/nvim", nil},
		{"outside target", "/tmp/elsewhere", ErrOutsideTarget},
		{"prefix but not descendant", "/home/username", ErrOutsideTarget},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ValidateTargetPath(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateTargetPath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTargetPathEmptyTarget(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "")
	err := p.ValidateTargetPath("/some/path")
	if !errors.Is(err, ErrOutsideTarget) {
		t.Fatalf("expected ErrOutsideTarget when targetPath is empty, got %v", err)
	}
}

func TestAvailableCommandsReadOnly(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeReadOnly, "/home/user")
	cmds := p.AvailableCommands(true, true)

	labels := make(map[string]bool, len(cmds))
	for _, cmd := range cmds {
		labels[cmd.Label] = true
	}

	// Mutations must be hidden in read-only mode.
	forbidden := []string{"Apply", "Update", "Refresh Externals", "Re-Add All", "Init", "Edit Source"}
	for _, label := range forbidden {
		if labels[label] {
			t.Fatalf("read-only mode should not include %q", label)
		}
	}

	// Read-only info commands must still be visible.
	required := []string{"Status", "Diff All", "Doctor", "Verify", "Data", "Cat Config", "Git Log", "Archive"}
	for _, label := range required {
		if !labels[label] {
			t.Errorf("read-only mode should include %q", label)
		}
	}
}

func TestAvailableCommandsWriteMode(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "/home/user")
	cmds := p.AvailableCommands(true, true)

	labels := make(map[string]bool, len(cmds))
	for _, cmd := range cmds {
		labels[cmd.Label] = true
	}

	expected := []string{"Apply", "Update", "Refresh Externals", "Git Log", "Edit Source", "Edit Config", "Edit Config Template", "Archive"}
	for _, label := range expected {
		if !labels[label] {
			t.Errorf("expected command %q to be present", label)
		}
	}
}

func TestAvailableCommandsEditorAvailability(t *testing.T) {
	p := NewPolicy(chezitconfig.ModeWrite, "/home/user")

	cmds := p.AvailableCommands(false, false)
	labels := make(map[string]bool, len(cmds))
	for _, cmd := range cmds {
		labels[cmd.Label] = true
	}

	if labels["Edit Source"] {
		t.Error("Edit Source should not be present when hasEditSource=false")
	}
	if labels["Edit Config"] {
		t.Error("Edit Config should not be present when hasEditConfig=false")
	}
}
