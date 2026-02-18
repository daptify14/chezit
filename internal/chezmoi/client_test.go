package chezmoi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientBaseFlagsContents(t *testing.T) {
	client := New()
	flags := client.baseFlags()

	want := []string{
		"--no-tty",
		"--color=false",
		"--no-pager",
		"--progress=false",
		"--use-builtin-diff",
	}

	if len(flags) != len(want) {
		t.Fatalf("expected %d base flags, got %d: %v", len(want), len(flags), flags)
	}
	for i, f := range want {
		if flags[i] != f {
			t.Errorf("baseFlags()[%d] = %q, want %q", i, flags[i], f)
		}
	}
}

func TestClientCmdInjectsBaseFlags(t *testing.T) {
	binaryPath := writeFakeChezmoiRawArgsBinary(t)

	client := New(WithBinaryPath(binaryPath))
	output, err := client.run("status", "--path-style=absolute")
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	args := strings.TrimSpace(string(output))
	for _, flag := range client.baseFlags() {
		if !strings.Contains(args, flag) {
			t.Errorf("expected base flag %q in command args, got: %s", flag, args)
		}
	}
	if !strings.Contains(args, "status") {
		t.Errorf("expected subcommand 'status' in args, got: %s", args)
	}
}

func TestClientTargetPathReturnsTrimmedValue(t *testing.T) {
	binaryPath := writeFakeChezmoiClientBinary(t, `
case "$1" in
target-path)
	printf '/home/custom\n'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	got, err := client.TargetPath()
	if err != nil {
		t.Fatalf("TargetPath returned unexpected error: %v", err)
	}
	if got != "/home/custom" {
		t.Fatalf("expected /home/custom, got %q", got)
	}
}

func TestClientTargetPathReturnsErrorWithCommandOutput(t *testing.T) {
	binaryPath := writeFakeChezmoiClientBinary(t, `
case "$1" in
target-path)
	echo "target-path failed" >&2
	exit 1
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	_, err := client.TargetPath()
	if err == nil {
		t.Fatal("expected TargetPath to fail")
	}
	if !strings.Contains(err.Error(), "target-path failed") {
		t.Fatalf("expected command output in error, got: %v", err)
	}
}

func TestClientIgnoredUsesResolvedTargetPath(t *testing.T) {
	binaryPath := writeFakeChezmoiClientBinary(t, `
case "$1" in
ignored)
	printf '.config/nvim/init.lua\n.ssh/config\n'
	;;
target-path)
	printf '/home/custom\n'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	got, err := client.Ignored()
	if err != nil {
		t.Fatalf("Ignored returned unexpected error: %v", err)
	}
	want := []string{
		filepath.Join("/home/custom", ".config/nvim/init.lua"),
		filepath.Join("/home/custom", ".ssh/config"),
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d paths, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ignored[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClientIgnoredReturnsErrorWhenTargetPathFails(t *testing.T) {
	binaryPath := writeFakeChezmoiClientBinary(t, `
case "$1" in
ignored)
	printf '.config/nvim/init.lua\n'
	;;
target-path)
	echo "cannot resolve target-path" >&2
	exit 1
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`)

	client := New(WithBinaryPath(binaryPath))
	_, err := client.Ignored()
	if err == nil {
		t.Fatal("expected Ignored to fail when target-path fails")
	}
	if !strings.Contains(err.Error(), "chezmoi target-path") {
		t.Fatalf("expected wrapped target-path error, got: %v", err)
	}
}

// writeFakeChezmoiRawArgsBinary creates a fake binary that echoes all received
// arguments, useful for verifying that baseFlags are injected by cmd().
func writeFakeChezmoiRawArgsBinary(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fake-chezmoi-args")
	script := "#!/bin/sh\necho \"$@\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake chezmoi args binary: %v", err)
	}
	return path
}

func writeFakeChezmoiClientBinary(t *testing.T, body string) string {
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
