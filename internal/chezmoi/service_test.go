package chezmoi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	chezitconfig "github.com/daptify14/chezit/internal/config"
)

func TestServiceStatusReturnsFiles(t *testing.T) {
	binaryPath := writeFakeChezmoiBinary(t, `
case "$1" in
status)
	printf 'MM /home/test/.config/nvim/init.lua\nMM /home/test/.zshrc\n'
	;;
target-path)
	printf '/home/test\n'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	svc := NewService(client, chezitconfig.ModeWrite, "/home/test")

	files, err := svc.Status()
	if err != nil {
		t.Fatalf("Status returned unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestServiceManagedFilesReturnsFiles(t *testing.T) {
	binaryPath := writeFakeChezmoiBinary(t, `
case "$1" in
managed)
	printf '/home/test/.config/nvim/init.lua\n/home/test/.bashrc\n'
	;;
target-path)
	printf '/home/test\n'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	svc := NewService(client, chezitconfig.ModeWrite, "/home/test")

	files, err := svc.ManagedFiles()
	if err != nil {
		t.Fatalf("ManagedFiles returned unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 managed files, got %d", len(files))
	}
}

func TestServiceReadOnlyBlocksMutations(t *testing.T) {
	client := New(WithBinaryPath("/bin/true"))
	svc := NewService(client, chezitconfig.ModeReadOnly, "/home/test")

	if !svc.IsReadOnly() {
		t.Fatal("expected IsReadOnly=true")
	}

	if err := svc.ReAdd("/home/test/.bashrc"); err == nil {
		t.Fatal("expected ReAdd to return error in read-only mode")
	}
	if err := svc.Forget("/home/test/.bashrc"); err == nil {
		t.Fatal("expected Forget to return error in read-only mode")
	}
	if err := svc.GitAdd("/home/test/.bashrc"); err == nil {
		t.Fatal("expected GitAdd to return error in read-only mode")
	}
	if err := svc.GitCommit("test"); err == nil {
		t.Fatal("expected GitCommit to return error in read-only mode")
	}
	if err := svc.GitPush(); err == nil {
		t.Fatal("expected GitPush to return error in read-only mode")
	}
	if err := svc.GitPull(); err == nil {
		t.Fatal("expected GitPull to return error in read-only mode")
	}
}

func TestServiceInteractiveCmdsNilInReadOnly(t *testing.T) {
	client := New(WithBinaryPath("/bin/true"))
	svc := NewService(client, chezitconfig.ModeReadOnly, "/home/test")

	if svc.ApplyCmd("/home/test/.bashrc") != nil {
		t.Error("expected ApplyCmd nil in read-only mode")
	}
	if svc.ApplyAllCmd() != nil {
		t.Error("expected ApplyAllCmd nil in read-only mode")
	}
	if svc.UpdateCmd() != nil {
		t.Error("expected UpdateCmd nil in read-only mode")
	}
	if svc.InitCmd() != nil {
		t.Error("expected InitCmd nil in read-only mode")
	}
	if svc.EditCmd("/home/test/.bashrc") != nil {
		t.Error("expected EditCmd nil in read-only mode")
	}
}

func TestServiceArchiveNotBlockedByReadOnly(t *testing.T) {
	binaryPath := writeFakeChezmoiBinary(t, `
case "$1" in
archive)
	# Simulate writing an archive file to the --output path.
	for arg in "$@"; do
		case "$arg" in
		--output=*) outpath="${arg#--output=}"; printf 'fake-archive' > "$outpath" ;;
		esac
	done
	;;
target-path)
	printf '/home/test\n'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	svc := NewService(client, chezitconfig.ModeReadOnly, "/home/test")

	outputPath, err := svc.Archive()
	if err != nil {
		t.Fatalf("Archive should not be blocked in read-only mode, got: %v", err)
	}
	if outputPath == "" {
		t.Fatal("expected non-empty output path")
	}

	info, statErr := os.Stat(outputPath)
	if statErr != nil {
		t.Fatalf("archive file not created: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatal("archive file is empty")
	}
}

func TestServiceArchiveOutputDir(t *testing.T) {
	client := New(WithBinaryPath("/bin/true"))
	svc := NewService(client, chezitconfig.ModeWrite, "/home/test")

	dir := svc.ArchiveOutputDir()
	if !strings.Contains(dir, "chezit") || !strings.Contains(dir, "archives") {
		t.Errorf("unexpected archive dir: %s", dir)
	}
}

func TestServiceReadOnlyAllowsGitFetch(t *testing.T) {
	binaryPath := writeFakeChezmoiBinary(t, `
case "$1" in
git)
	# Simulate successful fetch
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	svc := NewService(client, chezitconfig.ModeReadOnly, "/home/test")

	if err := svc.GitFetch(); err != nil {
		t.Fatalf("expected GitFetch to succeed in read-only mode, got: %v", err)
	}
}

func writeFakeChezmoiBinary(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fake-chezmoi")
	// Skip leading --flags injected by Client.baseFlags() so the
	// case statements in test scripts can match on the subcommand.
	preamble := "#!/bin/sh\nset -eu\n" +
		"while [ $# -gt 0 ]; do case \"$1\" in --*) shift ;; *) break ;; esac; done\n"
	script := preamble + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake chezmoi binary: %v", err)
	}
	return path
}
