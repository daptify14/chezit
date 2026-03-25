package chezmoi

import "encoding/json"

// DiffConfig holds the diff-related settings from chezmoi's resolved config.
type DiffConfig struct {
	Pager string // e.g., "delta", "diff-so-fancy"
}

// DiffConfig parses diff.pager from chezmoi's resolved configuration.
func (c *Client) DiffConfig() (DiffConfig, error) {
	raw, err := c.DumpConfigJSON()
	if err != nil {
		return DiffConfig{}, err
	}
	var cfg struct {
		Diff struct {
			Pager string `json:"pager"`
		} `json:"diff"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DiffConfig{}, err
	}
	return DiffConfig{Pager: cfg.Diff.Pager}, nil
}
