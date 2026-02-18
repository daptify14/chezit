package chezmoi

import (
	"testing"
)

func TestParseStatusEmpty(t *testing.T) {
	files := ParseStatus("")
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestParseStatusSingleMM(t *testing.T) {
	files := ParseStatus("MM /home/user/.bashrc\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.SourceStatus != 'M' {
		t.Errorf("expected SourceStatus 'M', got %q", f.SourceStatus)
	}
	if f.DestStatus != 'M' {
		t.Errorf("expected DestStatus 'M', got %q", f.DestStatus)
	}
	if f.Path != "/home/user/.bashrc" {
		t.Errorf("expected path /home/user/.bashrc, got %q", f.Path)
	}
}

func TestParseStatusMixed(t *testing.T) {
	input := "A  /home/user/.config/newfile\n M /home/user/.zshrc\nD  /home/user/.bashrc\nR  /home/user/.chezmoiscripts/run_once.sh\n"
	files := ParseStatus(input)
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(files))
	}

	tests := []struct {
		src  rune
		dest rune
		path string
	}{
		{'A', ' ', "/home/user/.config/newfile"},
		{' ', 'M', "/home/user/.zshrc"},
		{'D', ' ', "/home/user/.bashrc"},
		{'R', ' ', "/home/user/.chezmoiscripts/run_once.sh"},
	}
	for i, tt := range tests {
		if files[i].SourceStatus != tt.src {
			t.Errorf("[%d] expected SourceStatus %q, got %q", i, tt.src, files[i].SourceStatus)
		}
		if files[i].DestStatus != tt.dest {
			t.Errorf("[%d] expected DestStatus %q, got %q", i, tt.dest, files[i].DestStatus)
		}
		if files[i].Path != tt.path {
			t.Errorf("[%d] expected path %q, got %q", i, tt.path, files[i].Path)
		}
	}
}

func TestParseStatusSpacesInPath(t *testing.T) {
	files := ParseStatus("MM /home/user/Library/Application Support/Code/settings.json\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "/home/user/Library/Application Support/Code/settings.json" {
		t.Errorf("unexpected path: %q", files[0].Path)
	}
}

func TestParseStatusTrailingNewlines(t *testing.T) {
	files := ParseStatus("MM /home/user/.bashrc\n\n\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestFileStatusSideLabel(t *testing.T) {
	tests := []struct {
		path      string
		src, dest rune
		label     string
	}{
		{path: "/home/user/.bashrc", src: 'M', dest: 'M', label: "diverged"},
		{path: "/home/user/.bashrc", src: 'A', dest: ' ', label: "pending apply"},
		{path: "/home/user/.bashrc", src: 'D', dest: ' ', label: "pending apply"},
		{path: "/home/user/.bashrc", src: 'R', dest: ' ', label: "pending apply"},
		{path: "/home/user/.chezmoiscripts/run_once.sh", src: 'R', dest: ' ', label: "pending script run"},
		{path: "/home/user/.bashrc", src: ' ', dest: 'M', label: "target changed"},
		{path: "/home/user/.bashrc", src: ' ', dest: 'D', label: "target changed"},
		{path: "/home/user/.bashrc", src: ' ', dest: ' ', label: ""},
	}
	for _, tt := range tests {
		f := FileStatus{Path: tt.path, SourceStatus: tt.src, DestStatus: tt.dest}
		if got := f.SideLabel(); got != tt.label {
			t.Errorf("SideLabel(%q,%c,%c) = %q, want %q", tt.path, tt.src, tt.dest, got, tt.label)
		}
	}
}

func TestFileStatusIsModified(t *testing.T) {
	if !(FileStatus{SourceStatus: 'M', DestStatus: ' '}).IsModified() {
		t.Error("expected IsModified=true for M ")
	}
	if (FileStatus{SourceStatus: ' ', DestStatus: ' '}).IsModified() {
		t.Error("expected IsModified=false for space-space")
	}
}

func TestParseGitPorcelainEmpty(t *testing.T) {
	staged, unstaged, err := ParseGitPorcelain("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(staged) != 0 || len(unstaged) != 0 {
		t.Fatalf("expected empty, got staged=%d unstaged=%d", len(staged), len(unstaged))
	}
}

func TestParseGitPorcelainMixed(t *testing.T) {
	input := "M  staged_file.txt\n M unstaged_file.txt\nMM both_file.txt\n?? untracked.txt\nR  old.txt -> new.txt\n"
	staged, unstaged, err := ParseGitPorcelain(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(staged) != 3 {
		t.Fatalf("expected 3 staged, got %d", len(staged))
	}
	if staged[0].Path != "staged_file.txt" || staged[0].StatusCode != "M" {
		t.Errorf("staged[0] = %+v", staged[0])
	}
	if staged[1].Path != "both_file.txt" || staged[1].StatusCode != "M" {
		t.Errorf("staged[1] = %+v", staged[1])
	}
	if staged[2].Path != "new.txt" || staged[2].StatusCode != "R" {
		t.Errorf("staged[2] = %+v", staged[2])
	}

	if len(unstaged) != 3 {
		t.Fatalf("expected 3 unstaged, got %d", len(unstaged))
	}
	if unstaged[0].Path != "unstaged_file.txt" || unstaged[0].StatusCode != "M" {
		t.Errorf("unstaged[0] = %+v", unstaged[0])
	}
	if unstaged[1].Path != "both_file.txt" || unstaged[1].StatusCode != "M" {
		t.Errorf("unstaged[1] = %+v", unstaged[1])
	}
	if unstaged[2].Path != "untracked.txt" || unstaged[2].StatusCode != "U" {
		t.Errorf("unstaged[2] = %+v", unstaged[2])
	}
}

func TestParseGitPorcelainQuotedPaths(t *testing.T) {
	input := "M  \"quoted file.txt\"\n"
	staged, _, err := ParseGitPorcelain(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(staged) != 1 {
		t.Fatalf("expected 1 staged, got %d", len(staged))
	}
	if staged[0].Path != "quoted file.txt" {
		t.Errorf("expected unquoted path, got %q", staged[0].Path)
	}
}
