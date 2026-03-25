package tui

import (
	"reflect"
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
				"--color-only", "--paging=never", "--detect-dark-light=never",
				"--width=variable", "--line-numbers=false", "--side-by-side=false",
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
				"--color-only", "--paging=never", "--detect-dark-light=never",
				"--width=variable", "--line-numbers=false", "--side-by-side=false",
				"--light",
			},
			wantOk: true,
		},
		{
			name:     "delta absolute path",
			pagerCmd: "/usr/local/bin/delta",
			isDark:   true,
			wantArgs: []string{
				"/usr/local/bin/delta",
				"--color-only", "--paging=never", "--detect-dark-light=never",
				"--width=variable", "--line-numbers=false", "--side-by-side=false",
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
