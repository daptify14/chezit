package tui

import (
	"testing"
	"time"
)

func TestShortenPathWithTargetPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		targetPath string
		want       string
	}{
		{
			name:       "path inside target path",
			path:       "/home/test/.config/nvim/init.lua",
			targetPath: "/home/test",
			want:       "~/.config/nvim/init.lua",
		},
		{
			name:       "path shares prefix but is outside target path",
			path:       "/home/tester/.config/nvim/init.lua",
			targetPath: "/home/test",
			want:       "/home/tester/.config/nvim/init.lua",
		},
		{
			name:       "path exactly equal to target path",
			path:       "/home/test",
			targetPath: "/home/test",
			want:       "/home/test",
		},
		{
			name:       "empty target path leaves path unchanged",
			path:       "/home/test/.zshrc",
			targetPath: "",
			want:       "/home/test/.zshrc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenPath(tt.path, tt.targetPath)
			if got != tt.want {
				t.Fatalf("shortenPath(%q, %q) = %q, want %q", tt.path, tt.targetPath, got, tt.want)
			}
		})
	}
}

func TestHumanAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "just now"},
		{-5 * time.Second, "just now"},
		{59 * time.Second, "just now"},
		{60 * time.Second, "1m ago"},
		{59*time.Minute + 59*time.Second, "59m ago"},
		{time.Hour, "1h ago"},
		{25*time.Hour + 30*time.Minute, "25h ago"},
	}
	for _, tc := range cases {
		if got := humanAge(tc.d); got != tc.want {
			t.Errorf("humanAge(%s) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
