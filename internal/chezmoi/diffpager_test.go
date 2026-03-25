package chezmoi

import (
	"strings"
	"testing"
)

func TestClientDiffConfig(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantPager string
		wantErr   string
	}{
		{
			name: "pager set",
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
			wantPager: "delta",
		},
		{
			name: "pager with quoted args",
			body: `
case "$1" in
dump-config)
	printf '%s\n' '{"diff":{"pager":"delta --syntax-theme=\"GitHub Dark\""}}'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
			wantPager: `delta --syntax-theme="GitHub Dark"`,
		},
		{
			name: "diff section missing",
			body: `
case "$1" in
dump-config)
	printf '%s\n' '{"color":{"ui":true}}'
	;;
*)
	echo "unexpected command: $*" >&2
	exit 1
	;;
esac
`,
			wantPager: "",
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

			cfg, err := client.DiffConfig()
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
				t.Fatalf("DiffConfig returned unexpected error: %v", err)
			}
			if cfg.Pager != tt.wantPager {
				t.Fatalf("Pager = %q, want %q", cfg.Pager, tt.wantPager)
			}
		})
	}
}
