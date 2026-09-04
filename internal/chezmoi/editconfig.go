package chezmoi

import "encoding/json"

// EditConfig holds the edit-related settings from chezmoi's resolved config.
type EditConfig struct {
	Command string   // edit.command, run verbatim
	Args    []string // edit.args, inserted before the file
}

// EditConfig parses edit.command and edit.args from chezmoi's resolved configuration.
func (c *Client) EditConfig() (EditConfig, error) {
	raw, err := c.DumpConfigJSON()
	if err != nil {
		return EditConfig{}, err
	}
	var cfg struct {
		Edit struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"edit"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return EditConfig{}, err
	}
	return EditConfig{Command: cfg.Edit.Command, Args: cfg.Edit.Args}, nil
}
