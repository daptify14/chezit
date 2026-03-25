package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPreparePagerArgs(t *testing.T) {
	tests := []struct {
		name     string
		pagerCmd string
		isDark   bool
		wantArgs []string
		wantOk   bool
	}{
		{
			name:     "empty pager",
			pagerCmd: "",
			wantOk:   false,
		},
		{
			name:     "unsupported pager",
			pagerCmd: "less",
			wantOk:   false,
		},
		{
			name:     "delta bare dark",
			pagerCmd: "delta",
			isDark:   true,
			wantArgs: []string{
				"delta",
				"--paging=never", "--detect-dark-light=never",
				"--dark",
			},
			wantOk: true,
		},
		{
			name:     "delta with user args light",
			pagerCmd: "delta --syntax-theme=Nord",
			isDark:   false,
			wantArgs: []string{
				"delta", "--syntax-theme=Nord",
				"--paging=never", "--detect-dark-light=never",
				"--light",
			},
			wantOk: true,
		},
		{
			name:     "delta with quoted arg containing spaces",
			pagerCmd: `delta --syntax-theme="GitHub Dark"`,
			isDark:   true,
			wantArgs: []string{
				"delta", "--syntax-theme=GitHub Dark",
				"--paging=never", "--detect-dark-light=never",
				"--dark",
			},
			wantOk: true,
		},
		{
			name:     "delta absolute path",
			pagerCmd: "/usr/local/bin/delta",
			isDark:   true,
			wantArgs: []string{
				"/usr/local/bin/delta",
				"--paging=never", "--detect-dark-light=never",
				"--dark",
			},
			wantOk: true,
		},
		{
			name:     "bat",
			pagerCmd: "bat",
			wantArgs: []string{"bat", "--paging=never", "--plain", "--color=always"},
			wantOk:   true,
		},
		{
			name:     "diff-so-fancy no extra flags",
			pagerCmd: "diff-so-fancy",
			wantArgs: []string{"diff-so-fancy"},
			wantOk:   true,
		},
		{
			name:     "unterminated quote",
			pagerCmd: `delta --syntax-theme="GitHub Dark`,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, ok := preparePagerArgs(tt.pagerCmd, tt.isDark)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if tt.wantOk && !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestPipeThroughDiffPager_Unsupported(t *testing.T) {
	_, err := pipeThroughDiffPager("diff content", "less", true)
	if err == nil {
		t.Fatal("expected error for unsupported pager")
	}
}

func TestPipeThroughDiffPager_Empty(t *testing.T) {
	_, err := pipeThroughDiffPager("diff content", "", true)
	if err == nil {
		t.Fatal("expected error for empty pager command")
	}
}

func TestPipeThroughDiffPager_SupportedPager(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "delta")
	script := strings.Join([]string{
		"#!/bin/sh",
		`printf 'ARGS:%s\n' "$*"`,
		"cat",
	}, "\n") + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake pager: %v", err)
	}

	rendered, err := pipeThroughDiffPager("+added\n-removed\n", scriptPath+` --syntax-theme="GitHub Dark"`, true)
	if err != nil {
		t.Fatalf("pipeThroughDiffPager returned error: %v", err)
	}

	if !strings.Contains(rendered, "ARGS:--syntax-theme=GitHub Dark --paging=never --detect-dark-light=never --dark") {
		t.Fatalf("rendered output missing expected args, got: %q", rendered)
	}
	if !strings.Contains(rendered, "+added\n-removed") {
		t.Fatalf("rendered output missing piped diff content, got: %q", rendered)
	}
}
