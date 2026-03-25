package chezmoi

import (
	"encoding/json"
	"testing"
)

func TestParseDiffConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPager string
		wantErr   bool
	}{
		{
			name:      "pager set",
			input:     `{"diff":{"pager":"delta"}}`,
			wantPager: "delta",
		},
		{
			name:      "pager with args",
			input:     `{"diff":{"pager":"delta --syntax-theme=Nord"}}`,
			wantPager: "delta --syntax-theme=Nord",
		},
		{
			name:      "pager empty",
			input:     `{"diff":{"pager":""}}`,
			wantPager: "",
		},
		{
			name:      "diff section missing",
			input:     `{"color":{"ui":true}}`,
			wantPager: "",
		},
		{
			name:      "empty object",
			input:     `{}`,
			wantPager: "",
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg struct {
				Diff struct {
					Pager string `json:"pager"`
				} `json:"diff"`
			}
			err := json.Unmarshal([]byte(tt.input), &cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Diff.Pager != tt.wantPager {
				t.Errorf("pager = %q, want %q", cfg.Diff.Pager, tt.wantPager)
			}
		})
	}
}
