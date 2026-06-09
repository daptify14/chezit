package chezmoi

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

func TestClientBaseFlagsIncludesConfigPath(t *testing.T) {
	client := New(WithConfigPath("/tmp/chezmoi.toml"))
	flags := client.baseFlags()

	hasConfig := false
	for i := range len(flags) - 1 {
		if flags[i] == "--config" && flags[i+1] == "/tmp/chezmoi.toml" {
			hasConfig = true
			break
		}
	}

	if !hasConfig {
		t.Fatalf("expected --config flag with custom path, got %v", flags)
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

func TestClientCmdInjectsConfigPathFlag(t *testing.T) {
	binaryPath := writeFakeChezmoiRawArgsBinary(t)

	client := New(
		WithBinaryPath(binaryPath),
		WithConfigPath("/tmp/custom.toml"),
	)
	output, err := client.run("status")
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	args := strings.TrimSpace(string(output))
	if !strings.Contains(args, "--config /tmp/custom.toml") {
		t.Fatalf("expected custom config flag in args, got: %s", args)
	}
}

func TestCommandConstructorsIncludeConfigFlag(t *testing.T) {
	client := New(WithConfigPath("/tmp/custom.toml"))

	tests := []struct {
		name string
		cmd  *exec.Cmd
		want []string
	}{
		{name: "ApplyRefreshCmd", cmd: client.ApplyRefreshCmd(), want: []string{"--config", "/tmp/custom.toml", "apply", "--refresh-externals"}},
		{name: "ApplyCmd", cmd: client.ApplyCmd("/tmp/file"), want: []string{"--config", "/tmp/custom.toml", "apply", "/tmp/file"}},
		{name: "ApplyAllCmd", cmd: client.ApplyAllCmd(), want: []string{"--config", "/tmp/custom.toml", "apply"}},
		{name: "ApplyForceCmd", cmd: client.ApplyForceCmd("/tmp/file"), want: []string{"--config", "/tmp/custom.toml", "apply", "--force", "/tmp/file"}},
		{name: "ApplyAllForceCmd", cmd: client.ApplyAllForceCmd(), want: []string{"--config", "/tmp/custom.toml", "apply", "--force"}},
		{name: "ApplyDryRunCmd", cmd: client.ApplyDryRunCmd(), want: []string{"--config", "/tmp/custom.toml", "apply", "--dry-run", "-v"}},
		{name: "ApplyRefreshDryRunCmd", cmd: client.ApplyRefreshDryRunCmd(), want: []string{"--config", "/tmp/custom.toml", "apply", "--refresh-externals", "--dry-run", "-v"}},
		{name: "UpdateCmd", cmd: client.UpdateCmd(), want: []string{"--config", "/tmp/custom.toml", "update"}},
		{name: "EditCmd", cmd: client.EditCmd("/tmp/file"), want: []string{"--config", "/tmp/custom.toml", "edit", "/tmp/file"}},
		{name: "EditSourceCmd", cmd: client.EditSourceCmd(), want: []string{"--config", "/tmp/custom.toml", "edit"}},
		{name: "EditConfigCmd", cmd: client.EditConfigCmd(), want: []string{"--config", "/tmp/custom.toml", "edit-config"}},
		{name: "InitCmd", cmd: client.InitCmd(), want: []string{"--config", "/tmp/custom.toml", "init"}},
		{name: "EditConfigTemplateCmd", cmd: client.EditConfigTemplateCmd(), want: []string{"--config", "/tmp/custom.toml", "edit-config-template"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.cmd.Args[1:], tt.want) {
				t.Fatalf("args mismatch: got %v, want %v", tt.cmd.Args[1:], tt.want)
			}
		})
	}
}

func TestCommandConstructorsWithoutConfigFlag(t *testing.T) {
	client := New()

	tests := []struct {
		name string
		cmd  *exec.Cmd
	}{
		{name: "ApplyCmd", cmd: client.ApplyCmd("/tmp/file")},
		{name: "UpdateCmd", cmd: client.UpdateCmd()},
		{name: "EditConfigCmd", cmd: client.EditConfigCmd()},
		{name: "InitCmd", cmd: client.InitCmd()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if slices.Contains(tt.cmd.Args[1:], "--config") {
				t.Fatalf("did not expect --config in args: %v", tt.cmd.Args[1:])
			}
		})
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
		"while [ $# -gt 0 ]; do case \"$1\" in --config) shift 2 ;; --*) shift ;; *) break ;; esac; done\n"
	script := preamble + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake chezmoi binary: %v", err)
	}
	return path
}
