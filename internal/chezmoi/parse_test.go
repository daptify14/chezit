package chezmoi

import (
	"strings"
	"testing"
)

func TestParseGitLogOneline(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []GitCommit
	}{
		{
			name:  "normal output with multiple commits",
			input: "abc1234 fix dotfiles config\ndef5678 add zshrc\n",
			want: []GitCommit{
				{Hash: "abc1234", Message: "fix dotfiles config"},
				{Hash: "def5678", Message: "add zshrc"},
			},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "  \n  \n",
			want:  nil,
		},
		{
			name:  "hash-only line without message",
			input: "abc1234\n",
			want: []GitCommit{
				{Hash: "abc1234", Message: ""},
			},
		},
		{
			name:  "multi-word message with colons",
			input: "abc1234 fix: update nvim config for lua migration\n",
			want: []GitCommit{
				{Hash: "abc1234", Message: "fix: update nvim config for lua migration"},
			},
		},
		{
			name:  "blank lines interspersed",
			input: "abc1234 first\n\ndef5678 second\n\n",
			want: []GitCommit{
				{Hash: "abc1234", Message: "first"},
				{Hash: "def5678", Message: "second"},
			},
		},
		{
			name:  "leading and trailing whitespace on lines",
			input: "  abc1234 fix config  \n  def5678 add rc  \n",
			want: []GitCommit{
				{Hash: "abc1234", Message: "fix config"},
				{Hash: "def5678", Message: "add rc"},
			},
		},
		{
			name:  "CRLF line endings",
			input: "abc1234 first commit\r\ndef5678 second commit\r\n",
			want: []GitCommit{
				{Hash: "abc1234", Message: "first commit"},
				{Hash: "def5678", Message: "second commit"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseGitLogOneline(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseGitLogOneline() returned %d commits, want %d\ngot: %#v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i].Hash != tt.want[i].Hash {
					t.Errorf("commit[%d].Hash = %q, want %q", i, got[i].Hash, tt.want[i].Hash)
				}
				if got[i].Message != tt.want[i].Message {
					t.Errorf("commit[%d].Message = %q, want %q", i, got[i].Message, tt.want[i].Message)
				}
			}
		})
	}
}

func TestIsValidGitHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid short hash", input: "abc1234", want: true},
		{name: "valid 8-char hash", input: "abcdef12", want: true},
		{name: "valid full SHA-1 (40 chars)", input: strings.Repeat("a1b2c3d4", 5), want: true},
		{name: "valid SHA-256 (64 chars)", input: strings.Repeat("abcdef01", 8), want: true},
		{name: "valid uppercase hex", input: "ABCDEF12", want: true},
		{name: "valid mixed case", input: "aBcDeF12", want: true},
		{name: "empty string", input: "", want: false},
		{name: "too short (2 chars)", input: "ab", want: false},
		{name: "too short (3 chars)", input: "abc", want: false},
		{name: "too long (65 chars)", input: strings.Repeat("a", 65), want: false},
		{name: "flag --help", input: "--help", want: false},
		{name: "flag --format=%H", input: "--format=%H", want: false},
		{name: "flag -v", input: "-v", want: false},
		{name: "non-hex chars", input: "xyz12345", want: false},
		{name: "space in hash", input: "abc 1234", want: false},
		{name: "newline in hash", input: "abc\n1234", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGitHash(tt.input)
			if got != tt.want {
				t.Errorf("isValidGitHash(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
