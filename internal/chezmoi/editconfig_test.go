package chezmoi

import (
	"slices"
	"strings"
	"testing"
)

func TestClientEditConfig(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantCommand string
		wantArgs    []string
		wantErr     string
	}{
		{
			name: "command and args set",
			body: `
case "$1" in
dump-config)
	printf '%s\n' '{"edit":{"command":"code","args":["--new-window","--wait"]}}'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
			wantCommand: "code",
			wantArgs:    []string{"--new-window", "--wait"},
		},
		{
			name: "edit section missing",
			body: `
case "$1" in
dump-config)
	printf '%s\n' '{"diff":{"pager":"delta"}}'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
		},
		{
			name: "invalid json",
			body: `
case "$1" in
dump-config)
	printf '%s\n' 'not json'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
			wantErr: "invalid character",
		},
		{
			name: "dump-config command failure",
			body: `
case "$1" in
dump-config)
	echo "dump-config failed" >&2
	exit 1
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
			wantErr: "chezmoi dump-config: dump-config failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(WithBinaryPath(writeFakeChezmoiClientBinary(t, tt.body)))

			cfg, err := client.EditConfig()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("EditConfig returned unexpected error: %v", err)
			}
			if cfg.Command != tt.wantCommand {
				t.Fatalf("Command = %q, want %q", cfg.Command, tt.wantCommand)
			}
			if !slices.Equal(cfg.Args, tt.wantArgs) {
				t.Fatalf("Args = %q, want %q", cfg.Args, tt.wantArgs)
			}
		})
	}
}
